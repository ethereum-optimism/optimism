//! The supernode query API: `supernode_syncStatus` and `superroot_atTimestamp`.
//!
//! These are the two questions the rest of the OP Stack asks a supernode. op-proposer reads the
//! super root it proposes from `superroot_atTimestamp`; op-challenger reads the same method at
//! every step of a dispute; op-interop-mon and `kona-sp1-proposer` read both. None of them knows
//! or cares which supernode implementation answered, which is the whole requirement: the answers
//! have to be the ones op-supernode would have given, byte for byte where they are commitments
//! and field for field where they are status.
//!
//! The three pieces are kept apart deliberately. [`wire`] is the JSON, [`aggregate`] is the
//! arithmetic, [`chain`] is the reads, and this module is the composition — which is where the
//! branch structure of op-supernode's `superroot.atTimestamp` lives, because that is the part
//! where being *nearly* right produces a well-formed super root over the wrong branch.
//!
//! # Where the answers come from
//!
//! Both methods are served from the chains' own queues and from the interop actor, not from a
//! second view assembled here:
//!
//! - each chain's sync status is the one that chain's `optimism_syncStatus` serves,
//! - each chain's optimistic output comes from one borrow of its engine state,
//! - the verified frontier comes from the actor that owns the verified store, answered inside one
//!   turn of its loop so the frontier and the L1 block it was derived from cannot straddle a
//!   commit.
//!
//! # Binding before composition
//!
//! The RPC server binds before the chains are composed, because a harness that launched this
//! process needs an address to wait for. A call that arrives in that window is answered with
//! [`QueryError::Starting`] rather than with an empty chain set, which would be a well-formed
//! statement that the supernode hosts nothing.

mod aggregate;
mod chain;
mod error;
mod wire;

pub(crate) use chain::QueryChain;

use crate::{
    interop::{InteropReader, Verdict, VerifiedAt},
    query::{
        aggregate::Aggregate,
        chain::OptimisticOutput,
        error::QueryError,
        wire::{WireChainId, WireSuperRootAtTimestamp, WireSuperRootData, WireSyncStatus, WireU64},
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::ChainId;
use jsonrpsee::{RpcModule, core::RpcResult, proc_macros::rpc};
use kona_interop::OutputRootWithChain;
use lokahi_interop::VerifiedResult;
use std::{
    collections::BTreeMap,
    sync::{Arc, OnceLock},
};
use tracing::warn;

/// The `supernode` namespace: what the whole chain set has derived.
#[rpc(server, namespace = "supernode")]
pub(crate) trait SupernodeQueryApi {
    /// Every hosted chain's sync status, and the conservative minima over the set.
    ///
    /// op-supernode: `supernode_syncStatus`, `activity/supernode`.
    #[method(name = "syncStatus")]
    async fn supernode_sync_status(&self) -> RpcResult<WireSyncStatus>;
}

/// The `superroot` namespace: the super root at a timestamp.
#[rpc(server, namespace = "superroot")]
pub(crate) trait SuperrootQueryApi {
    /// The super root at `timestamp`, the optimistic branch under it, and the set's progress.
    ///
    /// op-supernode: `superroot_atTimestamp`, `activity/superroot`.
    #[method(name = "atTimestamp")]
    async fn superroot_at_timestamp(
        &self,
        timestamp: WireU64,
    ) -> RpcResult<WireSuperRootAtTimestamp>;
}

/// The chains and the verifier the query API answers from.
#[derive(Debug)]
pub(crate) struct QueryState {
    /// One entry per hosted chain, in ascending chain-id order.
    chains: Vec<QueryChain>,
    /// The read handle on the interop verifier, or [`None`] when interop is off.
    ///
    /// [`None`] is not a degraded mode: a supernode whose chain set does not schedule interop is
    /// answering about pre-interop consensus, where the optimistic outputs *are* the canonical
    /// ones. That is op-supernode's `ErrNotActive` branch, reached there through a verifier that
    /// exists and reports interop inactive, and reached here by there being no verifier at all.
    interop: Option<InteropReader>,
}

impl QueryState {
    /// Reduces every chain's sync status to the set's, folding in the verifier's L1 progress.
    ///
    /// The verifier is folded in whenever one exists, for both methods. op-supernode does it
    /// inside `syncstatus.Aggregate` through `ChainContainer.VerifierCurrentL1`, which is
    /// registered exactly when interop is configured.
    async fn aggregate(&self) -> Result<Aggregate, QueryError> {
        let mut statuses = BTreeMap::new();
        for chain in &self.chains {
            statuses.insert(chain.chain_id(), chain.sync_status().await?);
        }

        let aggregate = Aggregate::of(statuses);
        Ok(match &self.interop {
            Some(reader) => aggregate.with_verifier_l1(reader.current_l1()),
            None => aggregate,
        })
    }

    /// Gathers every chain's optimistic output at `timestamp`.
    ///
    /// A chain that has not derived that far is left out, which is op-supernode's `NotFound`
    /// skip. Every other failure fails the whole call, and that is load-bearing: op-challenger
    /// reads this map at step > 0, so a silently partial map would produce a permanent
    /// `InvalidTransition` commitment on chain.
    async fn optimistic_branch(
        &self,
        timestamp: u64,
    ) -> Result<BTreeMap<ChainId, OptimisticOutput>, QueryError> {
        let mut out = BTreeMap::new();
        for chain in &self.chains {
            if let Some(output) = chain.optimistic_at(timestamp).await? {
                out.insert(chain.chain_id(), output);
            }
        }
        Ok(out)
    }

    /// Reads each chain's output root at the head the verifier committed to.
    ///
    /// op-supernode reads the same roots by block *hash*, and an execution layer that no longer
    /// has that hash on its canonical chain answers `NotFound`, which fails the call. Here the
    /// read is by number and the hash is checked against it, which fails the same case for the
    /// same reason: a chain that reorged below a verified block would otherwise contribute the
    /// output root of a block nothing ever verified, and the super root over it would be
    /// well-formed and wrong.
    async fn verified_roots(
        &self,
        timestamp: u64,
        heads: &BTreeMap<ChainId, BlockNumHash>,
    ) -> Result<Vec<OutputRootWithChain>, QueryError> {
        let mut roots = Vec::with_capacity(self.chains.len());
        for chain in &self.chains {
            let chain_id = chain.chain_id();
            let head = heads
                .get(&chain_id)
                .copied()
                .ok_or(QueryError::ChainNotVerified { timestamp, chain_id })?;
            let root = chain.output_root_at(timestamp, head).await?;
            roots.push(OutputRootWithChain::new(chain_id, root));
        }
        Ok(roots)
    }
}

/// The handle the RPC server holds on state that does not exist when it binds.
///
/// A [`OnceLock`] rather than a lock that can be rewritten: the chain set is fixed for the life of
/// the process, so the only transition is "not composed yet" to "composed", and a type that cannot
/// express a second transition cannot accidentally serve two different chain sets.
#[derive(Debug, Clone, Default)]
pub(crate) struct QueryHandle(Arc<OnceLock<QueryState>>);

impl QueryHandle {
    /// Publishes the composed chains and the verifier read handle to the RPC server.
    pub(crate) fn compose(&self, chains: Vec<QueryChain>, interop: Option<InteropReader>) {
        if self.0.set(QueryState { chains, interop }).is_err() {
            // Unreachable: `run` composes once. Reported rather than panicked on, because a
            // supernode that is already answering correctly should not be stopped by it.
            warn!(target: "lokahi", "The supernode query API was composed twice; keeping the first chain set");
        }
    }

    /// Returns the composed state, or says the supernode is still starting.
    fn state(&self) -> Result<&QueryState, QueryError> {
        self.0.get().ok_or(QueryError::Starting)
    }

    /// Builds the RPC module serving both namespaces from this handle.
    pub(crate) fn into_rpc_module(
        self,
    ) -> Result<RpcModule<()>, jsonrpsee::core::RegisterMethodError> {
        let rpc = QueryRpc { handle: self };
        let mut module = RpcModule::new(());
        module.merge(SupernodeQueryApiServer::into_rpc(rpc.clone()))?;
        module.merge(SuperrootQueryApiServer::into_rpc(rpc))?;
        Ok(module)
    }
}

/// The server behind both namespaces.
#[derive(Debug, Clone)]
struct QueryRpc {
    /// The chains and verifier to answer from.
    handle: QueryHandle,
}

/// Which of op-supernode's `atTimestamp` branches answers, and what the verifier said while
/// deciding.
///
/// Named rather than inlined because this mapping *is* the correctness question. Every branch
/// produces a well-formed response, so a wrong one is not a failure a consumer can notice: a
/// timestamp before interop activated that answered `Waiting` would look to a proposer like a
/// supernode that is merely behind, forever.
#[derive(Debug, Clone, PartialEq, Eq)]
enum Branch {
    /// A frontier is committed at the timestamp: the super root is over the blocks it names.
    /// op-supernode: `VerifiedResultAtTimestamp` returning a result.
    Verified {
        /// The committed frontier.
        result: VerifiedResult,
        /// The L1 block the verifier had considered, read in the same turn as the frontier.
        verifier_l1: BlockNumHash,
    },
    /// Interop covers the timestamp and the verifier will reach it, but has not. No super root.
    /// op-supernode: `ethereum.NotFound`.
    Waiting {
        /// The L1 block the verifier had considered, read in the same turn as the miss.
        verifier_l1: BlockNumHash,
    },
    /// The optimistic outputs are the canonical ones, so the super root is composed from them.
    /// op-supernode: `ErrNotActive`, `ErrBeforeVerifiedDB`, `ErrNotStarted` — and, here, a chain
    /// set that schedules no interop at all, which has no verifier to ask.
    Handoff,
}

impl Branch {
    /// Maps what the verifier said onto the branch that answers.
    ///
    /// [`None`] is a chain set with no interop scheduled. op-supernode always has a verifier to
    /// ask and hears `ErrNotActive` from it; there is nothing to ask here, and the answer is the
    /// same one.
    fn of(verified: Option<VerifiedAt>) -> Self {
        match verified {
            None => Self::Handoff,
            Some(VerifiedAt { current_l1, verdict }) => match verdict {
                Verdict::Verified(result) => Self::Verified { result, verifier_l1: current_l1 },
                Verdict::NotYet => Self::Waiting { verifier_l1: current_l1 },
                Verdict::NotActive | Verdict::BeforeVerified | Verdict::NotStarted => Self::Handoff,
            },
        }
    }
}

impl From<Aggregate> for WireSyncStatus {
    fn from(aggregate: Aggregate) -> Self {
        let chain_ids = aggregate.chain_ids();
        let Aggregate {
            chains,
            current_l1,
            safe_timestamp,
            local_safe_timestamp,
            finalized_timestamp,
        } = aggregate;

        Self {
            chains: chains.into_iter().map(|(id, status)| (WireChainId(id), status)).collect(),
            chain_ids,
            current_l1: current_l1.into(),
            safe_timestamp,
            local_safe_timestamp,
            finalized_timestamp,
        }
    }
}

impl WireSuperRootAtTimestamp {
    /// Renders the response from the pieces the branch decided on.
    ///
    /// `current_l1` is passed in rather than taken from `aggregate` because two branches lower it
    /// to the verifier's snapshot value, and `data` is passed in because what it holds — or that it
    /// holds nothing — is the branch's whole answer.
    fn new(
        aggregate: &Aggregate,
        optimistic: &BTreeMap<ChainId, OptimisticOutput>,
        current_l1: BlockNumHash,
        data: Option<WireSuperRootData>,
    ) -> Self {
        Self {
            current_l1: current_l1.into(),
            safe_timestamp: aggregate.safe_timestamp,
            local_safe_timestamp: aggregate.local_safe_timestamp,
            finalized_timestamp: aggregate.finalized_timestamp,
            optimistic_at_timestamp: OptimisticOutput::branch(optimistic),
            chain_ids: aggregate.chain_ids(),
            data,
        }
    }
}

impl QueryRpc {
    /// Answers `supernode_syncStatus`.
    async fn sync_status(&self) -> Result<WireSyncStatus, QueryError> {
        Ok(self.handle.state()?.aggregate().await?.into())
    }

    /// Answers `superroot_atTimestamp`.
    ///
    /// The order is op-supernode's `Superroot.atTimestamp`: aggregate the set, ask the verifier
    /// about the timestamp, build the optimistic branch, and only then decide what — if anything —
    /// can be said about the super root. Reading the verifier before the optimistic branch is what
    /// makes the L1 block it reports usable: it was observed in the same turn as the frontier, so
    /// reporting it closes the race where the aggregate's own read straddled a commit or a rewind.
    async fn at_timestamp(&self, timestamp: u64) -> Result<WireSuperRootAtTimestamp, QueryError> {
        let state = self.handle.state()?;
        let aggregate = state.aggregate().await?;

        let verified = match &state.interop {
            Some(reader) => Some(reader.verified_at(timestamp).await?),
            None => None,
        };
        let optimistic = state.optimistic_branch(timestamp).await?;

        let (current_l1, data) = match Branch::of(verified) {
            Branch::Verified { result, verifier_l1 } => {
                aggregate.require_same_chain_set(timestamp, &result.l2_heads)?;
                let roots = state.verified_roots(timestamp, &result.l2_heads).await?;
                (
                    Aggregate::lower(aggregate.current_l1, verifier_l1),
                    Some(WireSuperRootData::verified(timestamp, result.l1_inclusion, roots)),
                )
            }
            // No frontier yet, so no super root. The verifier's write gate guarantees the reported
            // L1 has not reached the block this timestamp would need, so a consumer that waits on
            // L1 progress waits rather than concluding the super root is missing for good.
            Branch::Waiting { verifier_l1 } => {
                (Aggregate::lower(aggregate.current_l1, verifier_l1), None)
            }
            Branch::Handoff => (
                aggregate.current_l1,
                WireSuperRootData::from_handoff(timestamp, &aggregate, &optimistic),
            ),
        };

        Ok(WireSuperRootAtTimestamp::new(&aggregate, &optimistic, current_l1, data))
    }
}

#[async_trait::async_trait]
impl SupernodeQueryApiServer for QueryRpc {
    async fn supernode_sync_status(&self) -> RpcResult<WireSyncStatus> {
        Ok(self.sync_status().await?)
    }
}

#[async_trait::async_trait]
impl SuperrootQueryApiServer for QueryRpc {
    async fn superroot_at_timestamp(
        &self,
        timestamp: WireU64,
    ) -> RpcResult<WireSuperRootAtTimestamp> {
        Ok(self.at_timestamp(timestamp.0).await?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::query::wire::WireOutputV0;
    use alloy_primitives::B256;
    use kona_protocol::{BlockInfo, L2BlockInfo, OutputRoot, SyncStatus};
    use serde_json::{Value, json};

    /// The cross-language parity fixture, read from where the Go side keeps it.
    ///
    /// One file, two readers. The Go test in `op-devstack/sysgo` decodes it with
    /// `eth.SuperNodeSyncStatusResponse` and `eth.SuperRootAtTimestampResponse` — the very types
    /// op-proposer and op-challenger decode with — and recomputes both commitments in it with
    /// `eth.OutputRoot` and `eth.SuperRoot`. This test asserts that lokahi *produces* that file.
    /// Between them the two tests say something neither can say alone: what lokahi serves is what
    /// the Go consumers read, and the commitments in it are the ones Go computes.
    ///
    /// It lives in the Go package's `testdata` because that is where `go test` can find it without
    /// a path relative to a working directory, and it is pulled in here with [`include_str`], so a
    /// file that moves or disappears breaks the build rather than skipping the test.
    const FIXTURE: &str =
        include_str!("../../../../op-devstack/sysgo/testdata/lokahi_supernode_query.json");

    /// The timestamp the fixture's super root is at.
    const FIXTURE_TIMESTAMP: u64 = 1_700_000_040;

    /// Returns the fixture's expected JSON for one method.
    fn expected(method: &str) -> Value {
        let doc: Value = serde_json::from_str(FIXTURE).expect("the fixture is valid JSON");
        doc.get(method).unwrap_or_else(|| panic!("the fixture has no {method} response")).clone()
    }

    /// Serializes a response and compares it with the fixture as JSON values, so the comparison is
    /// about field names and encodings rather than about key order or indentation.
    fn assert_matches_fixture<T: serde::Serialize>(method: &str, response: &T) {
        let actual = serde_json::to_value(response).expect("the response serializes");
        assert_eq!(
            expected(method),
            actual,
            "\n{method} does not match the fixture. Serialized:\n{}\n",
            serde_json::to_string_pretty(&actual).unwrap_or_default()
        );
    }

    fn block(seed: u8, number: u64, timestamp: u64) -> BlockInfo {
        BlockInfo {
            hash: B256::repeat_byte(seed),
            number,
            parent_hash: B256::repeat_byte(seed.wrapping_sub(1)),
            timestamp,
        }
    }

    fn l2(seed: u8, number: u64, timestamp: u64, l1: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: block(seed, number, timestamp),
            l1_origin: BlockNumHash { number: l1, hash: B256::repeat_byte(seed ^ 0xf0) },
            seq_num: 3,
        }
    }

    /// One chain's sync status, spread out from a single seed so no two fields collide.
    fn status(seed: u8, l1: u64, safe: u64, local_safe: u64, finalized: u64) -> SyncStatus {
        let s = |n: u8| seed.wrapping_add(n);
        SyncStatus {
            current_l1: block(seed, l1, 1_700_000_000),
            current_l1_finalized: block(s(1), l1 - 30, 1_699_999_000),
            head_l1: block(s(2), l1 + 4, 1_700_000_050),
            safe_l1: block(s(3), l1 - 6, 1_699_999_900),
            finalized_l1: block(s(4), l1 - 30, 1_699_999_000),
            unsafe_l2: l2(s(5), 5_000, local_safe + 4, l1),
            safe_l2: l2(s(6), 4_900, safe, l1 - 6),
            finalized_l2: l2(s(7), 4_800, finalized, l1 - 30),
            local_safe_l2: l2(s(8), 4_950, local_safe, l1 - 2),
        }
    }

    /// The fixture's two chains, and the verifier that is behind both of them.
    fn fixture_aggregate() -> Aggregate {
        let chains = BTreeMap::from([
            (901, status(0x11, 900, 1_700_000_040, 1_700_000_050, 1_700_000_000)),
            (902, status(0x21, 898, 1_700_000_030, 1_700_000_046, 1_700_000_010)),
        ]);
        Aggregate::of(chains)
            .with_verifier_l1(BlockNumHash { number: 896, hash: B256::repeat_byte(0x77) })
    }

    fn optimistic(state: u8, bridge: u8, hash: u8, l1: u8, l1_number: u64) -> OptimisticOutput {
        OptimisticOutput {
            block: BlockNumHash { number: 4_900, hash: B256::repeat_byte(hash) },
            output: OutputRoot::from_parts(
                B256::repeat_byte(state),
                B256::repeat_byte(bridge),
                B256::repeat_byte(hash),
            ),
            required_l1: BlockNumHash { number: l1_number, hash: B256::repeat_byte(l1) },
        }
    }

    fn fixture_optimistic() -> BTreeMap<ChainId, OptimisticOutput> {
        BTreeMap::from([
            (901, optimistic(0x31, 0x32, 0x33, 0x34, 880)),
            (902, optimistic(0x41, 0x42, 0x43, 0x44, 884)),
        ])
    }

    /// The roots the verified branch reads at the heads the verifier committed to. Deliberately
    /// different preimages from the optimistic ones: if the two branches were ever confused, the
    /// fixture's super root would move.
    fn fixture_verified_roots() -> Vec<OutputRootWithChain> {
        vec![
            OutputRootWithChain::new(
                901,
                OutputRoot::from_parts(
                    B256::repeat_byte(0x51),
                    B256::repeat_byte(0x52),
                    B256::repeat_byte(0x53),
                )
                .hash(),
            ),
            OutputRootWithChain::new(
                902,
                OutputRoot::from_parts(
                    B256::repeat_byte(0x61),
                    B256::repeat_byte(0x62),
                    B256::repeat_byte(0x63),
                )
                .hash(),
            ),
        ]
    }

    /// `supernode_syncStatus`, field for field, against what the Go consumers decode.
    #[test]
    fn the_sync_status_response_matches_the_go_wire_shape() {
        assert_matches_fixture("supernode_syncStatus", &WireSyncStatus::from(fixture_aggregate()));
    }

    /// `superroot_atTimestamp` on the verified branch — the branch that publishes a commitment.
    #[test]
    fn the_superroot_response_matches_the_go_wire_shape() {
        let aggregate = fixture_aggregate();
        let optimistic = fixture_optimistic();
        let data = WireSuperRootData::verified(
            FIXTURE_TIMESTAMP,
            BlockNumHash { number: 890, hash: B256::repeat_byte(0x55) },
            fixture_verified_roots(),
        );
        let response = WireSuperRootAtTimestamp::new(
            &aggregate,
            &optimistic,
            aggregate.current_l1,
            Some(data),
        );

        assert_matches_fixture("superroot_atTimestamp", &response);
    }

    /// A chain id is a decimal string on this wire, as a map key and as an array element alike.
    /// Rendering it as a JSON number is the mistake that makes every Go consumer fail to decode.
    #[test]
    fn chain_ids_are_decimal_strings() {
        let response = WireSyncStatus::from(fixture_aggregate());
        let value = serde_json::to_value(&response).expect("serializes");

        assert_eq!(value["chain_ids"], json!(["901", "902"]));
        assert!(value["chains"].get("901").is_some(), "the map key is the decimal chain id");
    }

    /// `super.timestamp` is hexadecimal because `eth.SuperV1` marshals it through
    /// `hexutil.Uint64`, while every other timestamp in these responses is a plain number.
    #[test]
    fn only_the_super_root_timestamp_is_hexadecimal() {
        let aggregate = fixture_aggregate();
        let data = WireSuperRootData::verified(
            FIXTURE_TIMESTAMP,
            BlockNumHash::default(),
            fixture_verified_roots(),
        );
        let value = serde_json::to_value(WireSuperRootAtTimestamp::new(
            &aggregate,
            &fixture_optimistic(),
            aggregate.current_l1,
            Some(data),
        ))
        .expect("serializes");

        assert_eq!(value["data"]["super"]["timestamp"], json!("0x6553f128"));
        assert_eq!(value["safe_timestamp"], json!(1_700_000_030u64));
    }

    /// An absent super root is an absent `data` key, not `"data": null`: the Go field is
    /// `omitempty`, and a consumer that distinguishes the two would see a shape op-supernode
    /// never sends.
    #[test]
    fn an_absent_super_root_omits_the_data_key() {
        let aggregate = fixture_aggregate();
        let value = serde_json::to_value(WireSuperRootAtTimestamp::new(
            &aggregate,
            &fixture_optimistic(),
            aggregate.current_l1,
            None,
        ))
        .expect("serializes");

        assert!(value.get("data").is_none(), "data must be omitted, not null: {value}");
    }

    /// The empty branches are still objects and arrays, never `null`. op-supernode allocates both
    /// before filling them, so a chain set that has derived nothing serves `{}` and `[]`.
    #[test]
    fn empty_collections_serialize_as_empty_rather_than_null() {
        let aggregate = Aggregate::of(BTreeMap::new());
        let value = serde_json::to_value(WireSuperRootAtTimestamp::new(
            &aggregate,
            &BTreeMap::new(),
            aggregate.current_l1,
            None,
        ))
        .expect("serializes");

        assert_eq!(value["optimistic_at_timestamp"], json!({}));
        assert_eq!(value["chain_ids"], json!([]));
        assert_eq!(value["current_l1"], json!({ "hash": B256::ZERO, "number": 0 }));

        let status = serde_json::to_value(WireSyncStatus::from(Aggregate::of(BTreeMap::new())))
            .expect("serializes");
        assert_eq!(status["chains"], json!({}));
        assert_eq!(status["chain_ids"], json!([]));
    }

    /// The optimistic entry carries the preimage and its hash, and the hash is the preimage's.
    #[test]
    fn an_optimistic_entry_carries_the_hash_of_the_preimage_it_shows() {
        let entry = optimistic(0x31, 0x32, 0x33, 0x34, 880);
        let rendered = OptimisticOutput::branch(&BTreeMap::from([(901, entry)]));
        let wire = rendered.get(&WireChainId(901)).expect("the chain is present");

        assert_eq!(wire.output, WireOutputV0::from(entry.output));
        assert_eq!(wire.output_root, entry.output.hash());
    }

    /// Every verdict maps to the branch op-supernode answers that timestamp from. Getting one of
    /// these wrong produces a response that is well-formed and says something untrue.
    #[test]
    fn each_verdict_selects_op_supernodes_branch() {
        let verifier_l1 = BlockNumHash { number: 5, hash: B256::repeat_byte(0x05) };
        let at = |verdict| Some(VerifiedAt { current_l1: verifier_l1, verdict });
        let result = VerifiedResult {
            timestamp: FIXTURE_TIMESTAMP,
            l1_inclusion: verifier_l1,
            l2_heads: BTreeMap::new(),
        };

        assert_eq!(
            Branch::of(at(Verdict::Verified(result.clone()))),
            Branch::Verified { result, verifier_l1 }
        );
        assert_eq!(Branch::of(at(Verdict::NotYet)), Branch::Waiting { verifier_l1 });
        assert_eq!(Branch::of(at(Verdict::NotActive)), Branch::Handoff);
        assert_eq!(Branch::of(at(Verdict::BeforeVerified)), Branch::Handoff);
        assert_eq!(Branch::of(at(Verdict::NotStarted)), Branch::Handoff);
        // A chain set with no interop scheduled: there is no verifier to ask, and pre-interop
        // consensus covers every timestamp.
        assert_eq!(Branch::of(None), Branch::Handoff);
    }

    /// A query that arrives before the chains are composed is told so, rather than being answered
    /// with an empty chain set — which would be a well-formed claim that this supernode hosts
    /// nothing.
    #[test]
    fn a_query_before_composition_says_the_supernode_is_starting() {
        let handle = QueryHandle::default();
        assert!(matches!(handle.state(), Err(QueryError::Starting)));

        handle.compose(Vec::new(), None);
        assert!(handle.state().is_ok());
    }

    /// The timestamp parameter accepts what the Go clients send.
    #[test]
    fn the_timestamp_parameter_accepts_hex_and_decimal() {
        let parse = |raw: &str| serde_json::from_str::<WireU64>(raw).map(|value| value.0);

        // `hexutil.Uint64`, which is what op-service/sources.SuperNodeClient sends.
        assert_eq!(parse("\"0x6553f128\"").expect("hex"), FIXTURE_TIMESTAMP);
        assert_eq!(parse("1700000040").expect("number"), FIXTURE_TIMESTAMP);
        assert_eq!(parse("\"1700000040\"").expect("decimal string"), FIXTURE_TIMESTAMP);
        assert!(parse("\"0xnope\"").is_err());
    }
}
