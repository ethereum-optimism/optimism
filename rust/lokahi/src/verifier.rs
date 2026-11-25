//! The verification loop: one global timestamp cursor over every chain.

use crate::{
    source::{ChainSource, LocalSafeAt, SourceError, SourceProvider},
    state::{Verified, VerifiedResult},
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
    /// A block at the timestamp has an invalid executing message. Replacing it is not
    /// implemented, so the cursor stops here.
    #[error("invalid executing messages at {timestamp}: {reasons}")]
    InvalidMessages { timestamp: u64, reasons: String },
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
                        Wait::InvalidMessages { .. } if changed => {
                            tracing::error!(%reason, "Halting: block replacement is not implemented")
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
    async fn round(&self) -> Result<(), Wait> {
        self.refresh_finalized_l1().await?;
        let timestamp = self.state.read().expect("state lock").next_timestamp();

        let mut frontier = BTreeMap::new();
        for (&chain_id, source) in &self.sources {
            let progress = source.progress_at(timestamp).await?;
            self.state
                .write()
                .expect("state lock")
                .set_local_safe_timestamp(chain_id, progress.local_safe_timestamp);
            let at_timestamp =
                progress.at_timestamp.ok_or(Wait::NotLocalSafe { chain_id, timestamp })?;
            frontier.insert(chain_id, at_timestamp);
        }
        for (&chain_id, at) in &frontier {
            self.check_l1_inclusion(chain_id, at.l1_inclusion).await?;
        }
        self.check_messages(timestamp, &frontier).await?;

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

    /// Checks the executing messages of every block produced at exactly `timestamp`, with the
    /// shared message rules. A block carried over from an earlier timestamp was checked there.
    async fn check_messages(
        &self,
        timestamp: u64,
        frontier: &BTreeMap<u64, LocalSafeAt>,
    ) -> Result<(), Wait> {
        let mut blocks = BlocksWithReceipts::default();
        for (&chain_id, at) in frontier {
            let source = &self.sources[&chain_id];
            if source.has_block_at(timestamp) {
                blocks.insert(chain_id, source.block_with_receipts(at.block.hash).await?);
            }
        }
        if blocks.is_empty() {
            return Ok(());
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
        match graph.resolve().await {
            Ok(()) => Ok(()),
            Err(MessageGraphError::InvalidMessages(invalid)) => {
                let mut reasons = Vec::with_capacity(invalid.len());
                for (chain_id, err) in invalid {
                    // A provider failure is reported per message like an invalid one, but it only
                    // means the data was not available: retry rather than reject.
                    if let MessageGraphError::InteropProviderError(err) = err {
                        return Err(Wait::Source(err));
                    }
                    reasons.push(format!("chain {chain_id}: {err}"));
                }
                Err(Wait::InvalidMessages { timestamp, reasons: reasons.join("; ") })
            }
            Err(MessageGraphError::InteropProviderError(err)) => Err(Wait::Source(err)),
            Err(err) => Err(Wait::InvalidMessages { timestamp, reasons: err.to_string() }),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header as RpcHeader};
    use async_trait::async_trait;
    use jsonrpsee::{
        RpcModule,
        core::RpcResult,
        server::{Server, ServerHandle},
    };
    use kona_genesis::ChainGenesis;
    use kona_interop::ChainDependency;
    use kona_rpc::{
        DecimalChainId, OutputV0Response, OutputWithRequiredL1, SuperRootApiServer,
        SuperRootAtTimestampResponse,
    };
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

    /// A source CL answering one local-safe block, at genesis, or none.
    struct SourceCl {
        chain_id: u64,
        at_genesis: Option<(B256, BlockNumHash)>,
    }

    #[async_trait]
    impl SuperRootApiServer for SourceCl {
        async fn superroot_at_timestamp(
            &self,
            timestamp: alloy_primitives::U64,
        ) -> RpcResult<SuperRootAtTimestampResponse> {
            let mut optimistic_at_timestamp = BTreeMap::new();
            if let Some((hash, required_l1)) = self.at_genesis &&
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
                        output_root: B256::ZERO,
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

    async fn serve<C: Send + Sync + 'static>(module: RpcModule<C>) -> (Url, ServerHandle) {
        let server = Server::builder().build("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        (url, server.start(module))
    }

    /// The genesis block of a stubbed L2 chain.
    fn l2_genesis(chain_id: u64) -> Header {
        Header { extra_data: chain_id.to_be_bytes().to_vec().into(), ..Default::default() }
    }

    /// An execution RPC, for L1 and the L2 chains alike: a block by hash is an L2 genesis block
    /// with that hash, any other block is the L1 block, and no block has receipts.
    async fn serve_execution() -> (Url, ServerHandle) {
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
                let header = [CHAIN_A, CHAIN_B]
                    .map(l2_genesis)
                    .into_iter()
                    .find(|header| header.hash_slow() == hash)
                    .unwrap_or_else(l1_block);
                json(header)
            })
            .unwrap();
        module.register_method("eth_getBlockReceipts", |_, _, _| serde_json::json!([])).unwrap();
        serve(module).await
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

    /// A verifier over two chains whose source CLs answer `at_genesis`, with every block on L1.
    async fn verifier(
        at_genesis: [Option<(B256, BlockNumHash)>; 2],
        store_dir: &std::path::Path,
    ) -> (Verifier, Arc<RwLock<Verified>>, Vec<ServerHandle>) {
        let (execution, execution_handle) = serve_execution().await;
        let mut handles = vec![execution_handle];
        let mut sources = Vec::new();
        for (chain_id, at_genesis) in [CHAIN_A, CHAIN_B].into_iter().zip(at_genesis) {
            let (cl, handle) = serve(SourceCl { chain_id, at_genesis }.into_rpc()).await;
            handles.push(handle);
            sources.push(
                ChainSource::connect(rollup_config(chain_id), &cl, &execution).await.unwrap(),
            );
        }
        let dependency_set = DependencySet {
            dependencies: [(CHAIN_A, ChainDependency {}), (CHAIN_B, ChainDependency {})].into(),
            override_message_expiry_window: None,
        };
        let state = Arc::new(RwLock::new(crate::state::tests::open_state(
            store_dir,
            GENESIS_TIME,
            [(CHAIN_A, GENESIS_TIME), (CHAIN_B, GENESIS_TIME)].into(),
        )));
        let l1 = kona_sources::rpc::l1(execution);
        (Verifier::new(sources, dependency_set, l1, state.clone()), state, handles)
    }

    /// A timestamp at which every chain's block is local-safe, on L1, and without invalid
    /// messages is committed, and the authority answers its blocks.
    #[tokio::test]
    async fn round_commits_a_timestamp_every_chain_derived() {
        let (a, b) = (l2_genesis(CHAIN_A).hash_slow(), l2_genesis(CHAIN_B).hash_slow());
        let dir = tempfile::tempdir().unwrap();
        let (verifier, state, _servers) =
            verifier([Some((a, l1_inclusion())), Some((b, l1_inclusion()))], dir.path()).await;

        verifier.round().await.unwrap();

        let state = state.read().unwrap();
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
    async fn round_waits_for_every_chain(#[case] chain_b: Option<(B256, BlockNumHash)>) {
        let dir = tempfile::tempdir().unwrap();
        let (verifier, state, _servers) =
            verifier([Some((B256::repeat_byte(0xa), l1_inclusion())), chain_b], dir.path()).await;

        let wait = verifier.round().await.unwrap_err();

        assert!(
            matches!(
                wait,
                Wait::NotLocalSafe { chain_id: CHAIN_B, .. } |
                    Wait::L1Inconsistent { chain_id: CHAIN_B, .. }
            ),
            "got {wait:?}"
        );
        let state = state.read().unwrap();
        assert_eq!(state.next_timestamp(), GENESIS_TIME);
        assert_eq!(
            state.verified_head(CHAIN_A),
            Some(kona_engine::VerifierHead::Anchor { timestamp: GENESIS_TIME })
        );
    }
}
