//! The verification loop: one global timestamp cursor over every chain.

use crate::{
    source::{ChainSource, LocalSafeAt, SourceError, SourceProvider},
    state::{DeniedRecord, Verified, VerifiedResult},
    store::StoreError,
};
use alloy_eips::{BlockNumHash, BlockNumberOrTag};
use alloy_primitives::map::HashMap;
use alloy_provider::{Provider, RootProvider};
use kona_genesis::RollupConfig;
use kona_interop::{BlocksWithReceipts, DependencySet, MessageGraph, MessageGraphError};
use std::{
    collections::BTreeMap,
    sync::{Arc, RwLock},
    time::Duration,
};
use thiserror::Error;
use tracing::{debug, info, warn};

/// How long the loop waits before looking again when a round cannot advance.
const IDLE_INTERVAL: Duration = Duration::from_millis(500);
/// The bound on one round, so a hanging RPC holds the loop for no longer.
const ROUND_TIMEOUT: Duration = Duration::from_secs(10);

/// Why a round did not advance the cursor.
#[derive(Debug, Error)]
pub(crate) enum Wait {
    /// A chain has not derived its block at the timestamp yet.
    #[error("chain {chain_id} has no local-safe block at {timestamp} yet")]
    NotLocalSafe { chain_id: u64, timestamp: u64 },
    /// An L1 block a chain derived from is not canonical on the verifier's L1.
    #[error("L1 inclusion {l1:?} of chain {chain_id} is not on the canonical L1 chain")]
    L1Inconsistent { chain_id: u64, l1: BlockNumHash },
    /// A source or the L1 RPC failed.
    #[error(transparent)]
    Source(#[from] SourceError),
    /// The L1 RPC failed.
    #[error("L1 RPC: {0}")]
    L1(#[from] alloy_provider::transport::TransportError),
    /// Blocks at the timestamp have invalid executing messages, so verification denied them.
    /// Their chains replace them, and the timestamp is verified again.
    #[error("denied blocks at {timestamp}: {reasons}")]
    Denied { timestamp: u64, reasons: String },
    /// A chain's block at the timestamp is denied, and its source has not replaced it yet.
    #[error("chain {chain_id} has not replaced denied block {block:?} at {timestamp} yet")]
    AwaitingReplacement { chain_id: u64, block: BlockNumHash, timestamp: u64 },
    /// A source answered another block, or L1 inclusion, at the timestamp while its messages were
    /// checked, so the check is not trusted to deny anything.
    #[error("chain {chain_id} changed its block at {timestamp} while it was checked")]
    FrontierChanged { chain_id: u64, timestamp: u64 },
    /// The message graph could not be checked at all, which no source can fix.
    #[error("message graph at {timestamp} failed: {reason}")]
    GraphFailed { timestamp: u64, reason: String },
    /// The round ran out of time.
    #[error("round timed out")]
    TimedOut,
    /// The store failed to record a result. The verifier stops rather than answer what it
    /// could not record.
    #[error(transparent)]
    Store(#[from] StoreError),
}

/// The verifier.
#[derive(Debug)]
pub(crate) struct Verifier {
    sources: BTreeMap<u64, ChainSource>,
    rollup_configs: HashMap<u64, RollupConfig>,
    dependency_set: DependencySet,
    l1: RootProvider,
    state: Arc<RwLock<Verified>>,
}

impl Verifier {
    pub(crate) fn new(
        sources: Vec<ChainSource>,
        dependency_set: DependencySet,
        l1: RootProvider,
        state: Arc<RwLock<Verified>>,
    ) -> Self {
        let rollup_configs = sources
            .iter()
            .map(|source| (source.chain_id(), source.rollup_config.clone()))
            .collect();
        let sources = sources.into_iter().map(|source| (source.chain_id(), source)).collect();
        Self { sources, rollup_configs, dependency_set, l1, state }
    }

    /// Runs the loop until the task is dropped or the store fails.
    pub(crate) async fn run(self) -> Result<(), StoreError> {
        let mut last_wait = None;
        loop {
            let round = tokio::time::timeout(ROUND_TIMEOUT, self.round())
                .await
                .unwrap_or(Err(Wait::TimedOut));
            match round {
                Ok(()) => last_wait = None,
                Err(Wait::Store(err)) => return Err(err),
                Err(wait) => {
                    let reason = wait.to_string();
                    let changed = last_wait.as_deref() != Some(reason.as_str());
                    match &wait {
                        Wait::NotLocalSafe { .. } => debug!(%reason, "Waiting"),
                        Wait::Denied { .. } => {
                            warn!(%reason, "Denied blocks with invalid executing messages")
                        }
                        Wait::AwaitingReplacement { .. } if changed => {
                            info!(%reason, "Waiting for a denied block to be replaced")
                        }
                        Wait::GraphFailed { .. } if changed => {
                            tracing::error!(%reason, "Halting: the message graph cannot be checked")
                        }
                        _ if changed => warn!(%reason, "Round did not advance"),
                        _ => debug!(%reason, "Round did not advance"),
                    }
                    last_wait = Some(reason);
                    tokio::time::sleep(IDLE_INTERVAL).await;
                }
            }
        }
    }

    /// Verifies the next timestamp on every chain, committing it if every chain has its block
    /// local-safe, on the canonical L1 chain, and with valid executing messages.
    ///
    /// A block with an invalid executing message is denied instead, and the timestamp is not
    /// committed. Its chain replaces it with the deposits-only block, and a later round verifies
    /// the timestamp again with the replacement, which can in turn invalidate a block on another
    /// chain that executed a message from the denied block.
    async fn round(&self) -> Result<(), Wait> {
        self.refresh_finalized_l1().await?;
        let timestamp = self.state.read().expect("state lock").next_timestamp();

        let mut frontier = BTreeMap::new();
        for (&chain_id, source) in &self.sources {
            let progress = source.progress_at(timestamp).await?;
            let mut state = self.state.write().expect("state lock");
            state.set_local_safe_timestamp(chain_id, progress.local_safe_timestamp);
            let at = progress.at_timestamp.ok_or(Wait::NotLocalSafe { chain_id, timestamp })?;
            if state.is_denied(chain_id, at.block)? {
                return Err(Wait::AwaitingReplacement { chain_id, block: at.block, timestamp });
            }
            frontier.insert(chain_id, at);
        }
        for (&chain_id, at) in &frontier {
            self.check_l1_inclusion(chain_id, at.l1_inclusion).await?;
        }
        let invalid = self.check_messages(timestamp, &frontier).await?;
        if !invalid.is_empty() {
            self.confirm_frontier(timestamp, &frontier).await?;
            return Err(self.deny(timestamp, &frontier, invalid)?);
        }

        let result = VerifiedResult {
            l1_inclusion: frontier
                .values()
                .map(|at| at.l1_inclusion)
                .max_by_key(|l1| l1.number)
                .expect("at least one chain"),
            heads: frontier.iter().map(|(&chain_id, at)| (chain_id, at.block)).collect(),
        };
        info!(timestamp, heads = ?result.heads, l1_inclusion = result.l1_inclusion.number, "Verified timestamp");
        self.state.write().expect("state lock").commit(timestamp, result)?;
        Ok(())
    }

    async fn refresh_finalized_l1(&self) -> Result<(), Wait> {
        if let Some(block) = self.l1.get_block_by_number(BlockNumberOrTag::Finalized).await? {
            let finalized = BlockNumHash { number: block.header.number, hash: block.header.hash };
            self.state.write().expect("state lock").set_finalized_l1(finalized)?;
        }
        Ok(())
    }

    /// Checks that `l1` is the canonical L1 block at its height. Genesis carries no L1 block.
    async fn check_l1_inclusion(&self, chain_id: u64, l1: BlockNumHash) -> Result<(), Wait> {
        if l1.hash.is_zero() {
            return Ok(());
        }
        let canonical = self.l1.get_block_by_number(l1.number.into()).await?;
        match canonical {
            Some(block) if block.header.hash == l1.hash => Ok(()),
            _ => Err(Wait::L1Inconsistent { chain_id, l1 }),
        }
    }

    /// Confirms that every source still reports the `frontier` it was checked at: the same block
    /// at `timestamp`, from the same L1 block, which is still canonical. A denial is irreversible
    /// for the chain that replaces the block, so a read that changed while the messages were
    /// checked makes the round retry rather than deny.
    async fn confirm_frontier(
        &self,
        timestamp: u64,
        frontier: &BTreeMap<u64, LocalSafeAt>,
    ) -> Result<(), Wait> {
        for (&chain_id, at) in frontier {
            let now = self.sources[&chain_id].progress_at(timestamp).await?.at_timestamp;
            if now.as_ref() != Some(at) {
                return Err(Wait::FrontierChanged { chain_id, timestamp });
            }
            self.check_l1_inclusion(chain_id, at.l1_inclusion).await?;
        }
        Ok(())
    }

    /// Records each chain's block at `timestamp` that has an invalid executing message, with the
    /// reason, as denied, durably before the authority answers it.
    fn deny(
        &self,
        timestamp: u64,
        frontier: &BTreeMap<u64, LocalSafeAt>,
        invalid: BTreeMap<u64, String>,
    ) -> Result<Wait, StoreError> {
        let state = self.state.read().expect("state lock");
        let mut reasons = Vec::with_capacity(invalid.len());
        for (chain_id, reason) in invalid {
            let at = frontier[&chain_id];
            warn!(chain_id, timestamp, block = ?at.block, output_root = %at.output_root, %reason, "Denying a block with an invalid executing message");
            state.deny(&DeniedRecord {
                chain_id,
                block: at.block,
                output_root: at.output_root,
                timestamp,
            })?;
            reasons.push(format!("chain {chain_id} block {}: {reason}", at.block.number));
        }
        Ok(Wait::Denied { timestamp, reasons: reasons.join("; ") })
    }

    /// Checks the executing messages of every block produced at exactly `timestamp`, with the
    /// shared message rules, and returns the chains whose block is invalid, with the reason. A
    /// block carried over from an earlier timestamp was checked there.
    ///
    /// Only a message the rules reject makes a block invalid. Any failure to read a block or its
    /// logs, including a source answering another block than the hash asked for, retries the
    /// round instead, even when other messages are invalid.
    async fn check_messages(
        &self,
        timestamp: u64,
        frontier: &BTreeMap<u64, LocalSafeAt>,
    ) -> Result<BTreeMap<u64, String>, Wait> {
        let mut blocks = BlocksWithReceipts::default();
        for (&chain_id, at) in frontier {
            let source = &self.sources[&chain_id];
            if source.has_block_at(timestamp) {
                blocks.insert(chain_id, source.block_with_receipts(at.block.hash).await?);
            }
        }
        if blocks.is_empty() {
            return Ok(BTreeMap::new());
        }
        let frontier_blocks: HashMap<u64, BlockNumHash> =
            frontier.iter().map(|(&chain_id, at)| (chain_id, at.block)).collect();
        let provider = SourceProvider {
            sources: &self.sources,
            frontier: &frontier_blocks,
            verified: &self.state,
        };
        let graph = MessageGraph::derive(
            &blocks,
            &provider,
            &self.rollup_configs,
            &self.dependency_set,
            self.dependency_set.get_message_expiry_window(),
        );
        invalid_chains(timestamp, graph.resolve().await)
    }
}

/// The chains whose block the message rules reject in `resolved`, with the reason, or the wait
/// for a round that could not decide.
fn invalid_chains(
    timestamp: u64,
    resolved: Result<(), MessageGraphError<SourceError>>,
) -> Result<BTreeMap<u64, String>, Wait> {
    match resolved {
        Ok(()) => Ok(BTreeMap::new()),
        Err(MessageGraphError::InvalidMessages(invalid)) => {
            let mut reasons = BTreeMap::new();
            for (chain_id, err) in invalid {
                // A provider failure is reported per message like an invalid one, but it only
                // means the data was not available: retry rather than reject.
                if let MessageGraphError::InteropProviderError(err) = err {
                    return Err(Wait::Source(err));
                }
                reasons.insert(chain_id, err.to_string());
            }
            Ok(reasons)
        }
        // As the proof program does, every block in a cycle of same-timestamp messages is
        // invalid.
        Err(MessageGraphError::CyclicDependency { chain_ids }) => {
            let reason = format!("cyclic dependency among chains {chain_ids:?}");
            Ok(chain_ids.into_iter().map(|chain_id| (chain_id, reason.clone())).collect())
        }
        Err(MessageGraphError::InteropProviderError(err)) => Err(Wait::Source(err)),
        Err(err) => Err(Wait::GraphFailed { timestamp, reason: err.to_string() }),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Header, Receipt, ReceiptWithBloom};
    use alloy_primitives::{Address, B256};
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header as RpcHeader};
    use async_trait::async_trait;
    use jsonrpsee::{
        RpcModule,
        core::RpcResult,
        server::{Server, ServerHandle},
    };
    use kona_genesis::ChainGenesis;
    use kona_interop::{ChainDependency, ExecutingMessageBuilder, SuperchainBuilder};
    use kona_rpc::{
        DecimalChainId, OutputV0Response, OutputWithRequiredL1, SuperRootApiServer,
        SuperRootAtTimestampResponse,
    };
    use op_alloy_consensus::OpReceipt;
    use op_alloy_rpc_types::OpTransactionReceipt;
    use std::sync::Mutex;
    use url::Url;

    const CHAIN_A: u64 = 901;
    const CHAIN_B: u64 = 902;
    const GENESIS_TIME: u64 = 1000;

    /// The L1 block every stubbed chain derived from.
    fn l1_block() -> Header {
        Header { number: 5, ..Default::default() }
    }

    fn l1_inclusion() -> BlockNumHash {
        BlockNumHash { number: 5, hash: l1_block().hash_slow() }
    }

    /// A block of a stubbed L2 chain at genesis, told apart by `tag`.
    fn l2_genesis_tagged(chain_id: u64, tag: &[u8]) -> Header {
        Header {
            extra_data: [&chain_id.to_be_bytes()[..], tag].concat().into(),
            ..Default::default()
        }
    }

    /// The genesis block of a stubbed L2 chain.
    fn l2_genesis(chain_id: u64) -> Header {
        l2_genesis_tagged(chain_id, b"")
    }

    /// Chain B's block at genesis that has an invalid executing message.
    fn invalid_block() -> Header {
        l2_genesis_tagged(CHAIN_B, b"invalid")
    }

    /// Chain B's deposits-only replacement of [`invalid_block`].
    fn replacement_block() -> Header {
        l2_genesis_tagged(CHAIN_B, b"replacement")
    }

    /// A chain's local-safe block at genesis, by hash, and the L1 block that made it safe.
    type AtGenesis = Option<(B256, BlockNumHash)>;

    fn at_genesis(header: &Header) -> AtGenesis {
        Some((header.hash_slow(), l1_inclusion()))
    }

    /// A source CL answering one local-safe block, at genesis, or none. Each request takes the
    /// first of its answers, until only the last one is left.
    struct SourceCl {
        chain_id: u64,
        answers: Arc<Mutex<Vec<AtGenesis>>>,
    }

    #[async_trait]
    impl SuperRootApiServer for SourceCl {
        async fn superroot_at_timestamp(
            &self,
            timestamp: alloy_primitives::U64,
        ) -> RpcResult<SuperRootAtTimestampResponse> {
            let answer = {
                let mut answers = self.answers.lock().unwrap();
                if answers.len() > 1 { answers.remove(0) } else { answers[0] }
            };
            let mut optimistic_at_timestamp = BTreeMap::new();
            if let Some((hash, required_l1)) = answer &&
                timestamp.to::<u64>() == GENESIS_TIME
            {
                optimistic_at_timestamp.insert(
                    DecimalChainId(self.chain_id),
                    OutputWithRequiredL1 {
                        output: OutputV0Response {
                            state_root: B256::ZERO,
                            message_passer_storage_root: B256::ZERO,
                            block_hash: hash,
                        },
                        output_root: output_root(hash),
                        required_l1,
                    },
                );
            }
            Ok(SuperRootAtTimestampResponse {
                current_l1: BlockNumHash::default(),
                current_safe_timestamp: 0,
                current_local_safe_timestamp: GENESIS_TIME,
                current_finalized_timestamp: 0,
                optimistic_at_timestamp,
                chain_ids: vec![DecimalChainId(self.chain_id)],
                data: None,
            })
        }
    }

    /// The output root the stubbed sources report for the block `hash`.
    fn output_root(hash: B256) -> B256 {
        B256::from(alloy_primitives::U256::from_be_bytes(hash.0) ^ alloy_primitives::U256::MAX)
    }

    async fn serve<C: Send + Sync + 'static>(module: RpcModule<C>) -> (Url, ServerHandle) {
        let server = Server::builder().build("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        (url, server.start(module))
    }

    /// An execution RPC, for L1 and the L2 chains alike: a block by hash is the L2 block of the
    /// stubbed chains with that hash, any other block is the L1 block, and a block has the
    /// receipts `receipts` holds for its hash, or none.
    async fn serve_execution(receipts: HashMap<B256, serde_json::Value>) -> (Url, ServerHandle) {
        let json = |header: Header| {
            serde_json::to_value(Block::<()>::new(
                RpcHeader::new(header),
                BlockTransactions::Hashes(vec![]),
            ))
            .unwrap()
        };
        let mut module = RpcModule::new(());
        module.register_method("eth_getBlockByNumber", move |_, _, _| json(l1_block())).unwrap();
        module
            .register_method("eth_getBlockByHash", move |params, _, _| {
                let (hash, _full): (B256, bool) = params.parse().unwrap();
                let header = [
                    l2_genesis(CHAIN_A),
                    l2_genesis(CHAIN_B),
                    invalid_block(),
                    replacement_block(),
                ]
                .into_iter()
                .find(|header| header.hash_slow() == hash)
                .unwrap_or_else(l1_block);
                json(header)
            })
            .unwrap();
        module
            .register_method("eth_getBlockReceipts", move |params, _, _| {
                let block: alloy_eips::BlockId = params.one().unwrap();
                block
                    .as_block_hash()
                    .and_then(|hash| receipts.get(&hash).cloned())
                    .unwrap_or_else(|| serde_json::json!([]))
            })
            .unwrap();
        serve(module).await
    }

    /// The receipts of a block with one transaction, executing a message from a chain outside the
    /// dependency set, which is invalid.
    fn invalid_executing_receipts(block_hash: B256) -> serde_json::Value {
        let mut superchain = SuperchainBuilder::new();
        superchain.chain(CHAIN_B).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_origin_chain_id(999)
                .with_origin_timestamp(GENESIS_TIME),
        );
        let (blocks, _, _) = superchain.build();
        let logs = blocks[&CHAIN_B].receipts[0]
            .logs()
            .iter()
            .cloned()
            .map(|inner| alloy_rpc_types_eth::Log { inner, ..Default::default() })
            .collect();
        let receipt = OpTransactionReceipt {
            inner: alloy_rpc_types_eth::TransactionReceipt {
                inner: ReceiptWithBloom {
                    receipt: OpReceipt::Eip1559(Receipt {
                        status: true.into(),
                        cumulative_gas_used: 0,
                        logs,
                    }),
                    logs_bloom: Default::default(),
                },
                transaction_hash: B256::ZERO,
                transaction_index: Some(0),
                block_hash: Some(block_hash),
                block_number: Some(0),
                gas_used: 0,
                effective_gas_price: 0,
                blob_gas_used: None,
                blob_gas_price: None,
                from: Address::ZERO,
                to: None,
                contract_address: None,
            },
            l1_block_info: Default::default(),
            op_gas_refund: None,
        };
        serde_json::to_value(vec![receipt]).unwrap()
    }

    fn rollup_config(chain_id: u64) -> RollupConfig {
        let mut config = RollupConfig {
            l2_chain_id: chain_id.into(),
            block_time: 2,
            genesis: ChainGenesis { l2_time: GENESIS_TIME, ..Default::default() },
            ..Default::default()
        };
        config.hardforks.lagoon_time = Some(0);
        config
    }

    /// A verifier over two chains and its stubs.
    struct Fixture {
        verifier: Verifier,
        state: Arc<RwLock<Verified>>,
        /// What each chain's source answers at genesis, by chain.
        answers: BTreeMap<u64, Arc<Mutex<Vec<AtGenesis>>>>,
        _servers: Vec<ServerHandle>,
        _store_dir: tempfile::TempDir,
    }

    impl Fixture {
        /// Makes chain `chain_id`'s source answer `answer` from now on.
        fn answer(&self, chain_id: u64, answer: AtGenesis) {
            *self.answers[&chain_id].lock().unwrap() = vec![answer];
        }
    }

    /// A verifier over two chains whose source CLs answer `answers` at genesis, with every block
    /// on L1 and with chain B's [`invalid_block`] having an invalid executing message.
    async fn fixture(answers: [Vec<AtGenesis>; 2]) -> Fixture {
        let invalid = invalid_block().hash_slow();
        let (execution, execution_handle) =
            serve_execution(HashMap::from_iter([(invalid, invalid_executing_receipts(invalid))]))
                .await;
        let mut servers = vec![execution_handle];
        let mut sources = Vec::new();
        let mut chain_answers = BTreeMap::new();
        for (chain_id, answers) in [CHAIN_A, CHAIN_B].into_iter().zip(answers) {
            let answers = Arc::new(Mutex::new(answers));
            chain_answers.insert(chain_id, answers.clone());
            let (cl, handle) = serve(SourceCl { chain_id, answers }.into_rpc()).await;
            servers.push(handle);
            sources.push(
                ChainSource::connect(rollup_config(chain_id), &cl, &execution).await.unwrap(),
            );
        }
        let dependency_set = DependencySet {
            dependencies: [(CHAIN_A, ChainDependency {}), (CHAIN_B, ChainDependency {})].into(),
            override_message_expiry_window: None,
        };
        let store_dir = tempfile::tempdir().unwrap();
        let state = Arc::new(RwLock::new(crate::state::tests::open_state(
            store_dir.path(),
            GENESIS_TIME,
            [(CHAIN_A, GENESIS_TIME), (CHAIN_B, GENESIS_TIME)].into(),
        )));
        let l1 = kona_sources::rpc::l1(execution);
        Fixture {
            verifier: Verifier::new(sources, dependency_set, l1, state.clone()),
            state,
            answers: chain_answers,
            _servers: servers,
            _store_dir: store_dir,
        }
    }

    fn chain_a_at_genesis() -> Vec<AtGenesis> {
        vec![at_genesis(&l2_genesis(CHAIN_A))]
    }

    /// A timestamp at which every chain's block is local-safe, on L1, and without invalid
    /// messages is committed, and the authority answers its blocks.
    #[tokio::test]
    async fn round_commits_a_timestamp_every_chain_derived() {
        let (a, b) = (l2_genesis(CHAIN_A).hash_slow(), l2_genesis(CHAIN_B).hash_slow());
        let fixture = fixture([chain_a_at_genesis(), vec![at_genesis(&l2_genesis(CHAIN_B))]]).await;

        fixture.verifier.round().await.unwrap();

        let state = fixture.state.read().unwrap();
        assert_eq!(state.next_timestamp(), GENESIS_TIME + 1);
        assert_eq!(
            state.verified_head(CHAIN_B),
            Some(kona_engine::VerifierHead::Verified {
                block: BlockNumHash { number: 0, hash: b },
                timestamp: GENESIS_TIME,
            })
        );
        assert_eq!(
            state.finalized_head(CHAIN_A),
            Some(kona_engine::VerifierHead::Verified {
                block: BlockNumHash { number: 0, hash: a },
                timestamp: GENESIS_TIME,
            })
        );
    }

    /// The cursor waits while one chain has not derived its block, or derived it from an L1 block
    /// that is not canonical.
    #[rstest::rstest]
    #[case::not_local_safe(None)]
    #[case::not_on_l1(Some((B256::repeat_byte(0xb), BlockNumHash { number: 5, hash: B256::repeat_byte(0xee) })))]
    #[tokio::test]
    async fn round_waits_for_every_chain(#[case] chain_b: AtGenesis) {
        let fixture = fixture([chain_a_at_genesis(), vec![chain_b]]).await;

        let wait = fixture.verifier.round().await.unwrap_err();

        assert!(
            matches!(
                wait,
                Wait::NotLocalSafe { chain_id: CHAIN_B, .. } |
                    Wait::L1Inconsistent { chain_id: CHAIN_B, .. }
            ),
            "got {wait:?}"
        );
        let state = fixture.state.read().unwrap();
        assert_eq!(state.next_timestamp(), GENESIS_TIME);
        assert_eq!(
            state.verified_head(CHAIN_A),
            Some(kona_engine::VerifierHead::Anchor { timestamp: GENESIS_TIME })
        );
    }

    /// A block with an invalid executing message is denied with its output root, and the cursor
    /// holds while its source still reports it. Once the source reports another block at the
    /// timestamp, that block is verified like any other and the cursor moves on.
    #[tokio::test]
    async fn round_denies_an_invalid_block_and_verifies_its_replacement() {
        let fixture = fixture([chain_a_at_genesis(), vec![at_genesis(&invalid_block())]]).await;
        let denied = BlockNumHash { number: 0, hash: invalid_block().hash_slow() };

        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Denied { timestamp: GENESIS_TIME, .. }), "got {wait:?}");
        {
            let state = fixture.state.read().unwrap();
            assert_eq!(
                state.denied_blocks(CHAIN_B, 0).unwrap(),
                vec![DeniedRecord {
                    chain_id: CHAIN_B,
                    block: denied,
                    output_root: output_root(denied.hash),
                    timestamp: GENESIS_TIME,
                }]
            );
            assert!(state.denied_blocks(CHAIN_A, 0).unwrap().is_empty());
            assert_eq!(state.next_timestamp(), GENESIS_TIME);
        }

        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(
            matches!(wait, Wait::AwaitingReplacement { chain_id: CHAIN_B, block, .. } if block == denied),
            "got {wait:?}"
        );

        let replacement = BlockNumHash { number: 0, hash: replacement_block().hash_slow() };
        fixture.answer(CHAIN_B, at_genesis(&replacement_block()));
        fixture.verifier.round().await.unwrap();

        let state = fixture.state.read().unwrap();
        assert_eq!(state.next_timestamp(), GENESIS_TIME + 1);
        assert_eq!(
            state.verified_head(CHAIN_B),
            Some(kona_engine::VerifierHead::Verified {
                block: replacement,
                timestamp: GENESIS_TIME
            })
        );
        assert_eq!(state.denied_blocks(CHAIN_B, 0).unwrap().len(), 1);
    }

    /// A source that answers another block at the timestamp while its messages are checked, as a
    /// reorg of the source does, denies nothing: the round is retried with what the source
    /// answers then.
    #[tokio::test]
    async fn a_frontier_that_changes_during_the_check_is_not_denied() {
        let fixture = fixture([
            chain_a_at_genesis(),
            vec![at_genesis(&invalid_block()), at_genesis(&replacement_block())],
        ])
        .await;

        let wait = fixture.verifier.round().await.unwrap_err();

        assert!(
            matches!(wait, Wait::FrontierChanged { chain_id: CHAIN_B, timestamp: GENESIS_TIME }),
            "got {wait:?}"
        );
        assert!(fixture.state.read().unwrap().denied_blocks(CHAIN_B, 0).unwrap().is_empty());
        fixture.verifier.round().await.unwrap();
        assert_eq!(fixture.state.read().unwrap().next_timestamp(), GENESIS_TIME + 1);
    }

    /// The frontier is confirmed against its sources and the canonical L1 chain before a denial:
    /// an L1 inclusion that is no longer canonical denies nothing.
    #[tokio::test]
    async fn a_frontier_off_the_canonical_l1_chain_is_not_confirmed() {
        let off_l1 = BlockNumHash { number: 5, hash: B256::repeat_byte(0xee) };
        let invalid = invalid_block().hash_slow();
        let fixture = fixture([chain_a_at_genesis(), vec![Some((invalid, off_l1))]]).await;
        let frontier = BTreeMap::from([(
            CHAIN_B,
            LocalSafeAt {
                block: BlockNumHash { number: 0, hash: invalid },
                l1_inclusion: off_l1,
                output_root: output_root(invalid),
            },
        )]);

        let wait = fixture.verifier.confirm_frontier(GENESIS_TIME, &frontier).await.unwrap_err();

        assert!(matches!(wait, Wait::L1Inconsistent { chain_id: CHAIN_B, .. }), "got {wait:?}");
    }

    /// A failed read is never a reason to deny: when any message could not be checked, the
    /// round is retried even though another chain's message is invalid.
    #[test]
    fn a_failed_read_among_invalid_messages_retries_the_round() {
        let invalid = HashMap::from_iter([
            (CHAIN_A, MessageGraphError::ChainNotInDependencySet(999)),
            (
                CHAIN_B,
                MessageGraphError::InteropProviderError(SourceError::WrongBlock {
                    requested: B256::repeat_byte(1),
                    actual: B256::repeat_byte(2),
                }),
            ),
        ]);

        let decided =
            invalid_chains(GENESIS_TIME, Err(MessageGraphError::InvalidMessages(invalid)));

        assert!(
            matches!(decided, Err(Wait::Source(SourceError::WrongBlock { .. }))),
            "got {decided:?}"
        );
    }
}
