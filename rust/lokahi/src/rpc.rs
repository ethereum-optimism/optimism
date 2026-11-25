//! lokahi's RPC: the authority the nodes poll, op-supernode's super root and sync status for proofs
//! tooling and tests, and, behind a flag, a test control.

use crate::{
    source::{SourceError, SourceProgress, Sources},
    state::Verified,
    store::StoreError,
    verifier::lowest,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Bytes, FixedBytes, U64};
use async_trait::async_trait;
use jsonrpsee::{
    RpcModule,
    core::{RegisterMethodError, RpcResult},
    proc_macros::rpc,
    types::{ErrorCode, ErrorObject},
};
use kona_engine::VerifierStatus;
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_protocol::OutputRoot;
use kona_rpc::{
    DecimalChainId, InteropAuthorityApiServer, OutputV0Response, OutputWithRequiredL1,
    SuperRootApiServer, SuperRootAtTimestampResponse, SuperRootResponseData, SuperV1Response,
};
use serde::{Deserialize, Serialize};
use std::{
    collections::BTreeMap,
    sync::{Arc, RwLock},
};
use tracing::info;

/// The response of `supernode_syncStatus`, in op-supernode's shape.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub(crate) struct SyncStatusResponse {
    /// Each chain's `optimism_syncStatus`, as its source node answered it.
    pub chains: BTreeMap<DecimalChainId, serde_json::Value>,
    /// The chains, ascending.
    pub chain_ids: Vec<DecimalChainId>,
    /// The lowest L1 block a source or verification is at: every L1 block strictly below it is
    /// fully processed.
    pub current_l1: BlockNumHash,
    /// The lowest safe head timestamp of the chains.
    pub safe_timestamp: u64,
    /// The lowest local-safe head timestamp of the chains.
    pub local_safe_timestamp: u64,
    /// The lowest finalized head timestamp of the chains.
    pub finalized_timestamp: u64,
}

/// op-supernode's `supernode` namespace.
#[rpc(server, namespace = "supernode")]
pub(crate) trait SupernodeApi {
    /// Every chain's sync status, and the lowest progress across them.
    #[method(name = "syncStatus")]
    async fn sync_status(&self) -> RpcResult<SyncStatusResponse>;
}

/// op-supernode's liveness probe.
#[rpc(server, namespace = "heartbeat")]
pub(crate) trait HeartbeatApi {
    /// Four random bytes.
    #[method(name = "check")]
    async fn check(&self) -> RpcResult<Bytes>;
}

/// Controls and reads for tests, served only with `--test-control`. Timestamps are plain JSON
/// numbers.
#[rpc(server, namespace = "lokahitest")]
pub(crate) trait TestControlApi {
    /// Pauses verification before `timestamp` and every later one, also if verification is past
    /// it already. Zero resumes. A restart resumes too.
    #[method(name = "pauseAt")]
    async fn pause_at(&self, timestamp: u64) -> RpcResult<()>;

    /// Resumes verification.
    #[method(name = "resume")]
    async fn resume(&self) -> RpcResult<()>;

    /// The first timestamp interop is active on every chain.
    #[method(name = "activationTimestamp")]
    async fn activation_timestamp(&self) -> RpcResult<u64>;

    /// The first timestamp lokahi verifies: the activation timestamp, since lokahi always
    /// verifies from activation.
    #[method(name = "firstVerifiableTimestamp")]
    async fn first_verifiable_timestamp(&self) -> RpcResult<u64>;

    /// The timestamp this run started verifying at: the activation timestamp on an empty store,
    /// the one after the last stored result on a resume.
    #[method(name = "verificationStartTimestamp")]
    async fn verification_start_timestamp(&self) -> RpcResult<u64>;

    /// Whether `timestamp` is verified. Every timestamp before activation is.
    #[method(name = "verifiedAtTimestamp")]
    async fn verified_at_timestamp(&self, timestamp: u64) -> RpcResult<bool>;

    /// Whether verification reached the sources' local-safe heads since the start. Until it has,
    /// the `interop` heads are an error.
    #[method(name = "caughtUp")]
    async fn caught_up(&self) -> RpcResult<bool>;
}

/// Serves every namespace from the verifier's state and the chains' sources.
///
/// `interop_chainStatus` answers the heads only once verification has caught up with the sources'
/// local-safe heads since the start: before that, a store that is empty or behind would answer
/// heads far below the ones the nodes were told before, and the nodes would drop their labels to
/// them. Without heads the nodes hold their labels instead. The epoch and the denied blocks are
/// answered at once: every denial is in the store before it is answered, and a node must take a
/// new epoch on before verification reads it again, also while it catches up.
#[derive(Debug, Clone)]
pub(crate) struct Rpc {
    pub state: Arc<RwLock<Verified>>,
    pub sources: Arc<Sources>,
}

fn unknown_chain(chain_id: u64) -> ErrorObject<'static> {
    ErrorObject::owned(
        ErrorCode::InvalidParams.code(),
        format!("chain {chain_id} is not verified here"),
        None::<()>,
    )
}

fn internal(err: impl std::fmt::Display) -> ErrorObject<'static> {
    ErrorObject::owned(ErrorCode::InternalError.code(), err.to_string(), None::<()>)
}

fn source_error(chain_id: u64, err: SourceError) -> ErrorObject<'static> {
    internal(format!("chain {chain_id}: {err}"))
}

impl Rpc {
    /// The module serving every namespace, with the test control if `test_control`.
    pub(crate) fn into_module(
        self,
        test_control: bool,
    ) -> Result<RpcModule<()>, RegisterMethodError> {
        let mut module = RpcModule::new(());
        module.merge(InteropAuthorityApiServer::into_rpc(self.clone()))?;
        module.merge(SuperRootApiServer::into_rpc(self.clone()))?;
        module.merge(SupernodeApiServer::into_rpc(self.clone()))?;
        module.merge(HeartbeatApiServer::into_rpc(self.clone()))?;
        if test_control {
            module.merge(TestControlApiServer::into_rpc(self))?;
        }
        Ok(module)
    }
}

/// `superroot_atTimestamp` at `timestamp`, from every source's `progress` at it, read before the
/// state as op-supernode reads its chains before its verifier.
///
/// A verified timestamp is answered from the store alone, but for the progress fields: its super
/// root, and each chain's optimistic output, which is the denied block where the verified block
/// replaced one. A timestamp that is not verified yet has no super root, and each chain's
/// optimistic output is its source's local-safe block, or the denied block the source replaced.
/// Before activation the super root is made of the sources' outputs, once every chain has one.
fn super_root_at(
    state: &Verified,
    sources: &Sources,
    timestamp: u64,
    progress: &BTreeMap<u64, SourceProgress>,
) -> Result<SuperRootAtTimestampResponse, StoreError> {
    let least =
        |field: fn(&SourceProgress) -> u64| progress.values().map(field).min().unwrap_or_default();
    let mut response = SuperRootAtTimestampResponse {
        current_l1: lowest(
            progress.values().map(|progress| progress.current_l1).chain([state.current_l1()]),
        ),
        current_safe_timestamp: least(|progress| progress.safe_timestamp),
        current_local_safe_timestamp: least(|progress| progress.local_safe_timestamp),
        current_finalized_timestamp: least(|progress| progress.finalized_timestamp),
        optimistic_at_timestamp: BTreeMap::new(),
        chain_ids: progress.keys().copied().map(DecimalChainId).collect(),
        data: None,
    };
    let active = timestamp >= state.activation_timestamp();
    if active && let Some(result) = state.result_at(timestamp)? {
        for (&chain_id, chain) in &result.chains {
            response.optimistic_at_timestamp.insert(
                DecimalChainId(chain_id),
                optimistic(chain.optimistic(), chain.required_l1),
            );
        }
        let outputs =
            result.chains.iter().map(|(&chain_id, chain)| (chain_id, chain.output.hash()));
        response.data = Some(super_root_data(timestamp, result.l1_inclusion, outputs));
        return Ok(response);
    }

    for (&chain_id, progress) in progress {
        let Some(at) = progress.at_timestamp else { continue };
        let output = if active && sources[&chain_id].has_block_at(timestamp) {
            replaced_output(state, chain_id, at.block)?.unwrap_or(at.output)
        } else {
            at.output
        };
        response
            .optimistic_at_timestamp
            .insert(DecimalChainId(chain_id), optimistic(&output, at.l1_inclusion));
    }
    if !active && response.optimistic_at_timestamp.len() == progress.len() {
        let required_l1 = response
            .optimistic_at_timestamp
            .values()
            .map(|output| output.required_l1)
            .max_by_key(|l1| l1.number)
            .unwrap_or_default();
        let outputs =
            response.optimistic_at_timestamp.iter().map(|(id, output)| (id.0, output.output_root));
        response.data = Some(super_root_data(timestamp, required_l1, outputs));
    }
    Ok(response)
}

/// The denied block the source's local-safe `block` of `chain_id` replaced, if it replaced one:
/// one denied at its height, unless `block` is itself denied and not replaced yet. With several,
/// after an L1 reorg, the one decided on the highest L1 block.
fn replaced_output(
    state: &Verified,
    chain_id: u64,
    block: BlockNumHash,
) -> Result<Option<OutputRoot>, StoreError> {
    let denials = state.denials_at(chain_id, block.number)?;
    if denials.iter().any(|denial| denial.block == block) {
        return Ok(None);
    }
    Ok(denials.into_iter().max_by_key(|denial| denial.l1.number).map(|denial| denial.output))
}

fn optimistic(output: &OutputRoot, required_l1: BlockNumHash) -> OutputWithRequiredL1 {
    OutputWithRequiredL1 {
        output: OutputV0Response::from(output),
        output_root: output.hash(),
        required_l1,
    }
}

fn super_root_data(
    timestamp: u64,
    required_l1: BlockNumHash,
    outputs: impl Iterator<Item = (u64, alloy_primitives::B256)>,
) -> SuperRootResponseData {
    let super_root = SuperRoot::new(
        timestamp,
        outputs
            .map(|(chain_id, output_root)| OutputRootWithChain::new(chain_id, output_root))
            .collect(),
    );
    SuperRootResponseData {
        verified_required_l1: required_l1,
        super_v1: SuperV1Response::from(&super_root),
        super_root: super_root.hash(),
    }
}

#[async_trait]
impl InteropAuthorityApiServer for Rpc {
    async fn chain_status(&self, chain_id: u64, denied_from: u64) -> RpcResult<VerifierStatus> {
        let state = self.state.read().expect("state lock");
        let verified = state.verified_head(chain_id).ok_or_else(|| unknown_chain(chain_id))?;
        let finalized = state.finalized_head(chain_id).ok_or_else(|| unknown_chain(chain_id))?;
        let caught_up = state.is_caught_up();
        let denied = state.denied_blocks(chain_id, denied_from).map_err(internal)?;
        Ok(VerifierStatus {
            epoch: state.epoch(),
            verified: caught_up.then_some(verified),
            finalized: caught_up.then_some(finalized),
            denied: denied.into_iter().map(|record| record.block).collect(),
        })
    }
}

#[async_trait]
impl SuperRootApiServer for Rpc {
    async fn superroot_at_timestamp(
        &self,
        timestamp: U64,
    ) -> RpcResult<SuperRootAtTimestampResponse> {
        let timestamp = timestamp.to::<u64>();
        let mut progress = BTreeMap::new();
        for (&chain_id, source) in self.sources.iter() {
            let at =
                source.progress_at(timestamp).await.map_err(|err| source_error(chain_id, err))?;
            progress.insert(chain_id, at);
        }
        let state = self.state.read().expect("state lock");
        super_root_at(&state, &self.sources, timestamp, &progress).map_err(internal)
    }
}

#[async_trait]
impl SupernodeApiServer for Rpc {
    async fn sync_status(&self) -> RpcResult<SyncStatusResponse> {
        let mut chains = BTreeMap::new();
        let mut statuses = Vec::with_capacity(self.sources.len());
        for (&chain_id, source) in self.sources.iter() {
            let (raw, status) =
                source.sync_status().await.map_err(|err| source_error(chain_id, err))?;
            chains.insert(DecimalChainId(chain_id), raw);
            statuses.push(status);
        }
        let verifier_l1 = self.state.read().expect("state lock").current_l1();
        let least = |field: fn(&kona_protocol::SyncStatus) -> u64| {
            statuses.iter().map(field).min().unwrap_or_default()
        };
        Ok(SyncStatusResponse {
            chain_ids: chains.keys().copied().collect(),
            current_l1: lowest(
                statuses.iter().map(|status| status.current_l1.id()).chain([verifier_l1]),
            ),
            safe_timestamp: least(|status| status.safe_l2.block_info.timestamp),
            local_safe_timestamp: least(|status| status.local_safe_l2.block_info.timestamp),
            finalized_timestamp: least(|status| status.finalized_l2.block_info.timestamp),
            chains,
        })
    }
}

#[async_trait]
impl HeartbeatApiServer for Rpc {
    async fn check(&self) -> RpcResult<Bytes> {
        Ok(Bytes::copy_from_slice(FixedBytes::<4>::random().as_slice()))
    }
}

#[async_trait]
impl TestControlApiServer for Rpc {
    async fn pause_at(&self, timestamp: u64) -> RpcResult<()> {
        info!(timestamp, "Pausing verification");
        self.state.write().expect("state lock").set_pause_at((timestamp != 0).then_some(timestamp));
        Ok(())
    }

    async fn resume(&self) -> RpcResult<()> {
        info!("Resuming verification");
        self.state.write().expect("state lock").set_pause_at(None);
        Ok(())
    }

    async fn activation_timestamp(&self) -> RpcResult<u64> {
        Ok(self.state.read().expect("state lock").activation_timestamp())
    }

    async fn first_verifiable_timestamp(&self) -> RpcResult<u64> {
        Ok(self.state.read().expect("state lock").activation_timestamp())
    }

    async fn verification_start_timestamp(&self) -> RpcResult<u64> {
        Ok(self.state.read().expect("state lock").start_timestamp())
    }

    async fn verified_at_timestamp(&self, timestamp: u64) -> RpcResult<bool> {
        Ok(self.state.read().expect("state lock").is_verified_at(timestamp))
    }

    async fn caught_up(&self) -> RpcResult<bool> {
        Ok(self.state.read().expect("state lock").is_caught_up())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        source::{ChainSource, by_chain},
        state::{ChainResult, VerifiedResult, tests::open_state},
        verifier::{
            Wait,
            tests::{
                CHAIN_A, CHAIN_B, Fixture, GENESIS_TIME, at_genesis, chain_a_at_genesis, fixture,
                fixture_activating_at, invalid_block, l1_inclusion, l2_genesis, output,
                replacement_block, rollup_config, serve,
            },
        },
    };
    use alloy_primitives::B256;

    const CHAIN: u64 = 10;

    fn rpc(fixture: &Fixture) -> Rpc {
        Rpc { state: fixture.state.clone(), sources: fixture.sources.clone() }
    }

    async fn at(rpc: &Rpc, timestamp: u64) -> SuperRootAtTimestampResponse {
        rpc.superroot_at_timestamp(U64::from(timestamp)).await.unwrap()
    }

    /// Each chain's optimistic block hash.
    fn optimistic_blocks(response: &SuperRootAtTimestampResponse) -> Vec<(u64, B256)> {
        response
            .optimistic_at_timestamp
            .iter()
            .map(|(chain_id, output)| (chain_id.0, output.output.block_hash))
            .collect()
    }

    /// The super root of the chains' blocks `blocks` at `timestamp`.
    fn super_root(timestamp: u64, blocks: [(u64, B256); 2]) -> B256 {
        let outputs = blocks
            .map(|(chain_id, hash)| OutputRootWithChain::new(chain_id, output(hash).hash()))
            .to_vec();
        SuperRoot::new(timestamp, outputs).hash()
    }

    fn valid_at_genesis() -> [Vec<crate::verifier::tests::AtGenesis>; 2] {
        [chain_a_at_genesis(), vec![at_genesis(&l2_genesis(CHAIN_B))]]
    }

    /// A block that was denied and replaced stays the chain's optimistic output at its timestamp:
    /// while the source still reports it, once the source replaced it, and once the replacement
    /// is verified, when the super root has the replacement. A restart answers the same from the
    /// store, but for the L1 block verification is at, which only the next round decides.
    #[tokio::test]
    async fn a_denied_and_replaced_block_stays_the_optimistic_output() {
        let fixture = fixture([chain_a_at_genesis(), vec![at_genesis(&invalid_block())]]).await;
        let rpc = rpc(&fixture);
        let a = l2_genesis(CHAIN_A).hash_slow();
        let (invalid, replacement) = (invalid_block().hash_slow(), replacement_block().hash_slow());

        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Denied { .. }), "got {wait:?}");
        let denied = at(&rpc, GENESIS_TIME).await;
        assert_eq!(denied.data, None);
        assert_eq!(optimistic_blocks(&denied), [(CHAIN_A, a), (CHAIN_B, invalid)]);

        fixture.answer(CHAIN_B, at_genesis(&replacement_block()));
        let replaced = at(&rpc, GENESIS_TIME).await;
        assert_eq!(replaced.data, None);
        assert_eq!(optimistic_blocks(&replaced), [(CHAIN_A, a), (CHAIN_B, invalid)]);
        let chain_b = &replaced.optimistic_at_timestamp[&DecimalChainId(CHAIN_B)];
        assert_eq!(chain_b.output_root, output(invalid).hash());

        fixture.verifier.round().await.unwrap();
        let verified = at(&rpc, GENESIS_TIME).await;
        let data = verified.data.clone().expect("verified");
        assert_eq!(
            data.super_root,
            super_root(GENESIS_TIME, [(CHAIN_A, a), (CHAIN_B, replacement)])
        );
        assert_eq!(data.verified_required_l1, l1_inclusion());
        assert_eq!(optimistic_blocks(&verified), [(CHAIN_A, a), (CHAIN_B, invalid)]);
        assert_eq!(verified.current_l1, l1_inclusion(), "below the sources' L1 block");
        assert_eq!(rpc.verification_start_timestamp().await.unwrap(), GENESIS_TIME);
        drop(rpc);

        let Fixture { verifier, state, sources, store_dir, .. } = fixture;
        drop((verifier, state));
        let state = Arc::new(RwLock::new(crate::verifier::tests::open_state(
            store_dir.path(),
            GENESIS_TIME,
        )));
        let rpc = Rpc { state, sources };
        let restarted = at(&rpc, GENESIS_TIME).await;
        assert_eq!(restarted.current_l1, BlockNumHash::default());
        assert_eq!(
            SuperRootAtTimestampResponse { current_l1: verified.current_l1, ..restarted },
            verified
        );
        assert_eq!(rpc.verification_start_timestamp().await.unwrap(), GENESIS_TIME + 1);
        assert_eq!(rpc.first_verifiable_timestamp().await.unwrap(), GENESIS_TIME);
        assert!(rpc.verified_at_timestamp(GENESIS_TIME).await.unwrap());
        assert!(!rpc.verified_at_timestamp(GENESIS_TIME + 1).await.unwrap());
    }

    /// A timestamp beyond the verified head has no super root, and the L1 block verification is
    /// at holds the current L1 block down: to none before any round.
    #[tokio::test]
    async fn a_timestamp_beyond_the_verified_head_has_no_super_root() {
        let fixture = fixture(valid_at_genesis()).await;
        let rpc = rpc(&fixture);

        let unverified = at(&rpc, GENESIS_TIME).await;
        assert_eq!(unverified.data, None);
        assert_eq!(optimistic_blocks(&unverified).len(), 2, "both chains have their block");
        assert_eq!(unverified.current_l1, BlockNumHash::default());
        assert_eq!(unverified.chain_ids, [DecimalChainId(CHAIN_A), DecimalChainId(CHAIN_B)]);

        fixture.verifier.round().await.unwrap();
        let beyond = at(&rpc, GENESIS_TIME + 1).await;
        assert_eq!(beyond.data, None);
        assert!(beyond.optimistic_at_timestamp.is_empty());
        assert_eq!(beyond.current_l1, l1_inclusion());
        assert_eq!(beyond.current_local_safe_timestamp, GENESIS_TIME);
        assert!(at(&rpc, GENESIS_TIME).await.data.is_some());
    }

    /// Before activation the super root is made of the sources' outputs, once every chain has
    /// one, and needs the highest L1 block any of them needs.
    #[tokio::test]
    async fn before_activation_the_super_root_is_the_sources_outputs() {
        let fixture = fixture_activating_at(valid_at_genesis(), GENESIS_TIME + 2).await;
        let rpc = rpc(&fixture);
        let (a, b) = (l2_genesis(CHAIN_A).hash_slow(), l2_genesis(CHAIN_B).hash_slow());

        let data = at(&rpc, GENESIS_TIME).await.data.expect("every chain has its block");
        assert_eq!(data.super_root, super_root(GENESIS_TIME, [(CHAIN_A, a), (CHAIN_B, b)]));
        assert_eq!(data.verified_required_l1, l1_inclusion());

        fixture.answer(CHAIN_B, None);
        assert_eq!(at(&rpc, GENESIS_TIME).await.data, None);
    }

    fn fixture_block(tag: u8, number: u64) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(tag) }
    }

    /// The output the Go fixtures build from `tag`.
    fn fixture_output(tag: u8) -> OutputRoot {
        OutputRoot::from_parts(
            B256::repeat_byte(tag + 1),
            B256::repeat_byte(tag + 2),
            B256::repeat_byte(tag + 3),
        )
    }

    fn progress(current_l1: BlockNumHash, timestamps: [u64; 3]) -> SourceProgress {
        let [safe_timestamp, local_safe_timestamp, finalized_timestamp] = timestamps;
        SourceProgress {
            at_timestamp: None,
            current_l1,
            safe_timestamp,
            local_safe_timestamp,
            finalized_timestamp,
        }
    }

    /// A verified timestamp of two chains, the second of which replaced a denied block, answers
    /// exactly what op-supernode's Go types serialize for it
    /// (`op-service/eth/supernode_wire_fixture_test.go`), its output and super roots included.
    #[test]
    fn a_verified_timestamp_answers_in_op_supernode_wire_form() {
        let expected: serde_json::Value =
            serde_json::from_str(include_str!("../testdata/superroot_at_timestamp.json")).unwrap();
        let dir = tempfile::tempdir().unwrap();
        let mut state = open_state(dir.path(), 1003, BTreeMap::from([(901, 1000), (902, 1000)]));
        let chain = |tag: u8, required_l1, replaced| ChainResult {
            block: BlockNumHash { number: 1, hash: B256::repeat_byte(tag + 3) },
            output: fixture_output(tag),
            required_l1,
            replaced,
        };
        let result = VerifiedResult {
            l1_inclusion: fixture_block(0x22, 6),
            chains: BTreeMap::from([
                (901, chain(0xa0, fixture_block(0x21, 5), None)),
                (902, chain(0xc0, fixture_block(0x22, 6), Some(fixture_output(0xb0)))),
            ]),
        };
        state.commit(1003, result).unwrap();
        state.set_current_l1(fixture_block(0x13, 9));
        let progress = BTreeMap::from([
            (901, progress(fixture_block(0x11, 7), [1002, 1004, 1000])),
            (902, progress(fixture_block(0x12, 8), [1006, 1005, 1001])),
        ]);

        let response = super_root_at(&state, &Sources::new(), 1003, &progress).unwrap();

        assert_eq!(serde_json::to_value(&response).unwrap(), expected);
        assert_eq!(
            serde_json::from_value::<SuperRootAtTimestampResponse>(expected).unwrap(),
            response
        );
    }

    /// Each chain's sync status is its source's, passed through, and the aggregate is the lowest
    /// of each field across the chains and, for the L1 block, verification, as op-supernode's Go
    /// types serialize it.
    #[tokio::test]
    async fn the_sync_status_is_each_sources_in_op_supernode_wire_form() {
        let expected: serde_json::Value =
            serde_json::from_str(include_str!("../testdata/supernode_sync_status.json")).unwrap();
        let mut servers = Vec::new();
        let mut sources = Vec::new();
        for chain_id in [CHAIN_A, CHAIN_B] {
            let status = expected["chains"][chain_id.to_string()].clone();
            let mut module = RpcModule::new(());
            module.register_method("optimism_syncStatus", move |_, _, _| status.clone()).unwrap();
            let (url, server) = serve(module).await;
            servers.push(server);
            sources.push(ChainSource::connect(rollup_config(chain_id), &url, &url).await.unwrap());
        }
        let dir = tempfile::tempdir().unwrap();
        let mut state =
            open_state(dir.path(), GENESIS_TIME, BTreeMap::from([(CHAIN_A, 0), (CHAIN_B, 0)]));
        state.set_current_l1(fixture_block(0x99, 100));
        let rpc = Rpc { state: Arc::new(RwLock::new(state)), sources: Arc::new(by_chain(sources)) };

        let response = rpc.sync_status().await.unwrap();
        assert_eq!(serde_json::to_value(&response).unwrap(), expected);

        let behind = fixture_block(0x98, 3);
        rpc.state.write().unwrap().set_current_l1(behind);
        assert_eq!(rpc.sync_status().await.unwrap().current_l1, behind);
        for server in servers {
            server.stop().unwrap();
        }
    }

    /// The test control pauses verification at a timestamp, also one that every chain derived,
    /// and resumes it.
    #[tokio::test]
    async fn the_test_control_pauses_and_resumes_verification() {
        let fixture = fixture(valid_at_genesis()).await;
        let rpc = rpc(&fixture);

        rpc.pause_at(GENESIS_TIME).await.unwrap();
        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Paused { timestamp: GENESIS_TIME }), "got {wait:?}");
        assert!(!rpc.verified_at_timestamp(GENESIS_TIME).await.unwrap());
        assert!(rpc.verified_at_timestamp(GENESIS_TIME - 1).await.unwrap(), "before activation");

        rpc.resume().await.unwrap();
        fixture.verifier.round().await.unwrap();
        assert!(rpc.verified_at_timestamp(GENESIS_TIME).await.unwrap());
        assert_eq!(rpc.activation_timestamp().await.unwrap(), GENESIS_TIME);
    }

    #[tokio::test]
    async fn the_heartbeat_answers_four_random_bytes() {
        let dir = tempfile::tempdir().unwrap();
        let rpc = Rpc {
            state: Arc::new(RwLock::new(open_state(dir.path(), 100, BTreeMap::new()))),
            sources: Default::default(),
        };
        let (first, second) = (rpc.check().await.unwrap(), rpc.check().await.unwrap());
        assert_eq!(first.len(), 4);
        assert_ne!(first, second);
    }

    /// The test control is served only when asked for.
    #[test]
    fn the_test_control_is_served_only_with_its_flag() {
        let dir = tempfile::tempdir().unwrap();
        let rpc = Rpc {
            state: Arc::new(RwLock::new(open_state(dir.path(), 100, BTreeMap::new()))),
            sources: Default::default(),
        };
        let methods = |test_control| {
            rpc.clone()
                .into_module(test_control)
                .unwrap()
                .method_names()
                .map(String::from)
                .collect::<Vec<_>>()
        };
        let without = methods(false);
        for method in [
            "superroot_atTimestamp",
            "supernode_syncStatus",
            "heartbeat_check",
            "interop_chainStatus",
        ] {
            assert!(without.iter().any(|name| name == method), "{method} missing from {without:?}");
        }
        assert!(!without.iter().any(|name| name.starts_with("lokahitest_")));
        assert!(methods(true).iter().any(|name| name == "lokahitest_pauseAt"));
    }

    /// Until verification caught up the status carries no heads, which makes the nodes hold their
    /// labels, while the epoch and the denied blocks are answered from the start, the denied
    /// blocks from the height asked for.
    #[tokio::test]
    async fn the_status_answers_heads_once_caught_up_and_denials_at_once() {
        let dir = tempfile::tempdir().unwrap();
        let state = open_state(dir.path(), 100, BTreeMap::from([(CHAIN, 100)]));
        let epoch = state.epoch();
        let rpc = Rpc { state: Arc::new(RwLock::new(state)), sources: Default::default() };

        let status = rpc.chain_status(CHAIN, 0).await.unwrap();
        assert_eq!(
            status,
            VerifierStatus { epoch, verified: None, finalized: None, denied: vec![] }
        );
        assert_eq!(
            rpc.chain_status(CHAIN + 1, 0).await.unwrap_err().code(),
            ErrorCode::InvalidParams.code()
        );

        let denied = |number| BlockNumHash { number, hash: B256::repeat_byte(number as u8) };
        for number in [3, 5] {
            let record = crate::state::DeniedRecord {
                chain_id: CHAIN,
                block: denied(number),
                output: crate::state::tests::output_of(denied(number)),
                timestamp: 100 + number,
                l1: BlockNumHash { number: 7, hash: B256::repeat_byte(7) },
            };
            rpc.state.read().unwrap().deny(&record).unwrap();
        }
        assert_eq!(rpc.chain_status(CHAIN, 0).await.unwrap().denied, vec![denied(3), denied(5)]);
        assert_eq!(rpc.chain_status(CHAIN, 4).await.unwrap().denied, vec![denied(5)]);

        rpc.state.write().unwrap().set_caught_up();
        let status = rpc.chain_status(CHAIN, 0).await.unwrap();
        assert_eq!(status.verified, Some(kona_engine::VerifierHead::Anchor { timestamp: 100 }));
        assert_eq!(status.finalized, Some(kona_engine::VerifierHead::Anchor { timestamp: 100 }));
        assert_eq!(status.epoch, epoch);
    }
}
