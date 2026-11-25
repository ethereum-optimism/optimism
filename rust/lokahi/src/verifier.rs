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
    /// An L1 reorg removed the L1 inclusion of committed results, so the results from `from` on
    /// were deleted and are verified again.
    #[error("L1 inclusion {l1:?} is not canonical any more; rewound the results from {from} on")]
    Rewound { from: u64, l1: BlockNumHash },
    /// The L1 inclusion of the finalized result is not canonical: L1 reorged below its finalized
    /// head, or the L1 RPC serves another chain. Finality cannot move back, so the verifier
    /// stops.
    #[error("L1 inclusion {l1:?} of the finalized result at {timestamp} is not canonical")]
    FinalityViolation { timestamp: u64, l1: BlockNumHash },
    /// The store failed to record a result. The verifier stops rather than answer what it
    /// could not record.
    #[error(transparent)]
    Store(#[from] StoreError),
}

impl Wait {
    /// Whether the verifier stops rather than retry.
    const fn is_fatal(&self) -> bool {
        matches!(self, Self::Store(_) | Self::FinalityViolation { .. })
    }
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

    /// Runs the loop until the task is dropped, the store fails, or L1 violates finality.
    pub(crate) async fn run(self) -> Result<(), Wait> {
        let mut last_wait = None;
        loop {
            let round = tokio::time::timeout(ROUND_TIMEOUT, self.round())
                .await
                .unwrap_or(Err(Wait::TimedOut));
            match round {
                Ok(()) => last_wait = None,
                Err(wait) if wait.is_fatal() => {
                    tracing::error!(reason = %wait, "Halting");
                    return Err(wait);
                }
                // Verify the rewound timestamps again right away.
                Err(wait @ Wait::Rewound { .. }) => {
                    warn!(reason = %wait, "Rewound after an L1 reorg");
                    last_wait = None;
                }
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
    ///
    /// The committed results are checked against L1 after the frontier is read and before it is
    /// used, so a frontier read after an L1 reorg is never committed on top of results from the
    /// old L1 chain.
    async fn round(&self) -> Result<(), Wait> {
        self.refresh_finalized().await?;
        let timestamp = self.state.read().expect("state lock").next_timestamp();

        let frontier = self.read_frontier(timestamp).await;
        // A rewind takes priority over any wait: an L1 reorg often makes the frontier
        // unavailable by moving the sources' local-safe heads back.
        self.rewind_reorged_results().await?;
        if let Err(Wait::NotLocalSafe { .. }) = &frontier {
            self.state.write().expect("state lock").set_caught_up();
        }
        let frontier = frontier?;
        for (&chain_id, at) in &frontier {
            self.check_l1_inclusion(chain_id, at.l1_inclusion).await?;
        }
        let invalid = self.check_messages(timestamp, &frontier).await?;
        if !invalid.is_empty() {
            self.confirm_frontier(timestamp, &frontier).await?;
            return Err(self.deny(timestamp, &frontier, invalid)?);
        }

        let result = VerifiedResult {
            l1_inclusion: frontier_l1_inclusion(&frontier),
            heads: frontier.iter().map(|(&chain_id, at)| (chain_id, at.block)).collect(),
        };
        info!(timestamp, heads = ?result.heads, l1_inclusion = result.l1_inclusion.number, "Verified timestamp");
        self.state.write().expect("state lock").commit(timestamp, result)?;
        Ok(())
    }

    /// Every chain's local-safe block at `timestamp`, unless one has not derived it yet or its
    /// block there is denied and not replaced yet.
    ///
    /// A denial holds only on the L1 chain it was decided on, and the nodes apply it only while
    /// its L1 block is canonical. One whose L1 block was reorged out is deleted instead of waited
    /// on, and the block is verified again.
    async fn read_frontier(&self, timestamp: u64) -> Result<BTreeMap<u64, LocalSafeAt>, Wait> {
        let mut frontier = BTreeMap::new();
        for (&chain_id, source) in &self.sources {
            let progress = source.progress_at(timestamp).await?;
            let at = progress.at_timestamp.ok_or(Wait::NotLocalSafe { chain_id, timestamp })?;
            let denial = self.state.read().expect("state lock").denial(chain_id, at.block)?;
            if let Some(denial) = denial {
                if self.is_canonical(denial.l1).await? {
                    return Err(Wait::AwaitingReplacement { chain_id, block: at.block, timestamp });
                }
                warn!(chain_id, block = ?at.block, l1 = ?denial.l1, "Deleting a denial decided on an L1 block that was reorged out");
                self.state.read().expect("state lock").undeny(&denial)?;
            }
            frontier.insert(chain_id, at);
        }
        Ok(frontier)
    }

    /// Moves the finalized head to the latest result whose L1 inclusion is on the finalized L1
    /// chain: at or below the finalized L1 block by number, and the canonical block at its height
    /// by hash. A result whose inclusion was reorged out is not final; the reorg check rewinds
    /// it.
    async fn refresh_finalized(&self) -> Result<(), Wait> {
        let Some(block) = self.l1.get_block_by_number(BlockNumberOrTag::Finalized).await? else {
            return Ok(());
        };
        let finalized_l1 = BlockNumHash { number: block.header.number, hash: block.header.hash };
        let (candidate, known_final) = {
            let state = self.state.read().expect("state lock");
            let candidate = state.latest_included_by(finalized_l1.number)?.filter(|(ts, _)| {
                state.finalized().is_none_or(|(finalized_ts, _)| finalized_ts < ts)
            });
            (candidate, state.finalized().map(|(_, result)| result.l1_inclusion))
        };
        let Some(candidate) = candidate else { return Ok(()) };
        let l1 = candidate.1.l1_inclusion;
        let on_finalized_chain = if Some(l1) == known_final || l1.hash.is_zero() {
            true
        } else if l1.number == finalized_l1.number {
            l1.hash == finalized_l1.hash
        } else {
            // The canonical block at a height below the finalized block is finalized.
            self.is_canonical(l1).await?
        };
        if !on_finalized_chain {
            debug!(
                timestamp = candidate.0,
                ?l1,
                ?finalized_l1,
                "L1 inclusion is not on the finalized L1 chain; not finalizing it"
            );
            return Ok(());
        }
        self.state.write().expect("state lock").set_finalized(candidate);
        Ok(())
    }

    /// Rewinds the committed results whose L1 inclusion is no longer canonical.
    ///
    /// L1 inclusions never decrease with the timestamp, and a reorg replaces the L1 chain above
    /// a fork point, so the results off the canonical L1 chain are the ones from the first whose
    /// inclusion is off it: a check of the latest result finds whether there are any, and a
    /// binary search the first. The results from it on are deleted with the denials decided on
    /// them, in one write, and verified again. A finalized result is never deleted: one whose
    /// inclusion is off the canonical chain stops the verifier.
    async fn rewind_reorged_results(&self) -> Result<(), Wait> {
        let (latest, floor) = {
            let state = self.state.read().expect("state lock");
            (state.latest().cloned(), state.first_unfinalized_timestamp())
        };
        let Some((latest_ts, latest)) = latest else { return Ok(()) };
        if self.is_canonical(latest.l1_inclusion).await? {
            return Ok(());
        }
        if latest_ts < floor {
            return Err(Wait::FinalityViolation { timestamp: latest_ts, l1: latest.l1_inclusion });
        }
        // The first result off the canonical chain is in `lo..=hi`.
        let (mut lo, mut hi) = (floor, latest_ts);
        while lo < hi {
            let mid = lo + (hi - lo) / 2;
            let result = self
                .state
                .read()
                .expect("state lock")
                .result_at(mid)?
                .ok_or(StoreError::Missing(mid))?;
            if self.is_canonical(result.l1_inclusion).await? {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }
        if lo == floor {
            let finalized = self.state.read().expect("state lock").finalized().cloned();
            if let Some((timestamp, result)) = finalized &&
                !self.is_canonical(result.l1_inclusion).await?
            {
                return Err(Wait::FinalityViolation { timestamp, l1: result.l1_inclusion });
            }
        }
        let (results, denied) = self.state.write().expect("state lock").rewind(lo)?;
        warn!(from = lo, results, denied, reorged = ?latest.l1_inclusion, "Rewound the results whose L1 inclusion was reorged out");
        Err(Wait::Rewound { from: lo, l1: latest.l1_inclusion })
    }

    /// Whether `l1` is the canonical L1 block at its height. Genesis carries no L1 block.
    async fn is_canonical(&self, l1: BlockNumHash) -> Result<bool, Wait> {
        if l1.hash.is_zero() {
            return Ok(true);
        }
        let canonical = self.l1.get_block_by_number(l1.number.into()).await?;
        Ok(canonical.is_some_and(|block| block.header.hash == l1.hash))
    }

    /// Checks that `l1` is the canonical L1 block at its height.
    async fn check_l1_inclusion(&self, chain_id: u64, l1: BlockNumHash) -> Result<(), Wait> {
        if self.is_canonical(l1).await? {
            Ok(())
        } else {
            Err(Wait::L1Inconsistent { chain_id, l1 })
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
        let l1 = state.next_l1_inclusion(frontier_l1_inclusion(frontier));
        let mut reasons = Vec::with_capacity(invalid.len());
        for (chain_id, reason) in invalid {
            let at = frontier[&chain_id];
            warn!(chain_id, timestamp, block = ?at.block, output_root = %at.output_root, ?l1, %reason, "Denying a block with an invalid executing message");
            state.deny(&DeniedRecord {
                chain_id,
                block: at.block,
                output_root: at.output_root,
                timestamp,
                l1,
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

/// The latest L1 block any chain in `frontier` derived its block from.
fn frontier_l1_inclusion(frontier: &BTreeMap<u64, LocalSafeAt>) -> BlockNumHash {
    frontier
        .values()
        .map(|at| at.l1_inclusion)
        .max_by_key(|l1| l1.number)
        .expect("at least one chain")
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
        l1_header(5, 0)
    }

    /// L1 block `number` of the fork `fork`: fork 0 is the chain before any reorg.
    fn l1_header(number: u64, fork: u8) -> Header {
        let extra_data = if fork == 0 { Default::default() } else { vec![fork].into() };
        Header { number, extra_data, ..Default::default() }
    }

    fn l1_id(number: u64, fork: u8) -> BlockNumHash {
        BlockNumHash { number, hash: l1_header(number, fork).hash_slow() }
    }

    /// An L1 chain a test can reorg: its canonical block at a height is of fork 1 from the
    /// reorg's fork point on and of fork 0 below it.
    #[derive(Debug, Clone)]
    struct L1Chain(Arc<Mutex<L1ChainState>>);

    #[derive(Debug)]
    struct L1ChainState {
        /// The first height of fork 1, once L1 reorged.
        reorged_from: Option<u64>,
        /// The height of the finalized block.
        finalized: u64,
    }

    impl L1Chain {
        fn new(finalized: u64) -> Self {
            Self(Arc::new(Mutex::new(L1ChainState { reorged_from: None, finalized })))
        }

        fn reorg_from(&self, number: u64) {
            self.0.lock().unwrap().reorged_from = Some(number);
        }

        fn finalize(&self, number: u64) {
            self.0.lock().unwrap().finalized = number;
        }

        fn canonical(&self, number: u64) -> Header {
            let state = self.0.lock().unwrap();
            l1_header(number, u8::from(state.reorged_from.is_some_and(|from| number >= from)))
        }

        /// Serves the chain's `eth_getBlockByNumber`.
        async fn serve(&self) -> (Url, ServerHandle) {
            let chain = self.clone();
            let mut module = RpcModule::new(());
            module
                .register_method("eth_getBlockByNumber", move |params, _, _| {
                    let (tag, _full): (BlockNumberOrTag, bool) = params.parse().unwrap();
                    let number = match tag {
                        BlockNumberOrTag::Number(number) => number,
                        BlockNumberOrTag::Finalized => chain.0.lock().unwrap().finalized,
                        tag => panic!("unexpected L1 block tag {tag}"),
                    };
                    serde_json::to_value(Block::<()>::new(
                        RpcHeader::new(chain.canonical(number)),
                        BlockTransactions::Hashes(vec![]),
                    ))
                    .unwrap()
                })
                .unwrap();
            serve(module).await
        }
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
        l1: L1Chain,
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
        let l1_chain = L1Chain::new(l1_block().number);
        let (l1, l1_handle) = l1_chain.serve().await;
        servers.push(l1_handle);
        Fixture {
            verifier: Verifier::new(
                sources,
                dependency_set,
                kona_sources::rpc::l1(l1),
                state.clone(),
            ),
            state,
            l1: l1_chain,
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
        // The next round finalizes the result, whose L1 inclusion is the finalized L1 block.
        fixture.verifier.refresh_finalized().await.unwrap();

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
        // Verification caught up with a source that has not derived the timestamp, not with one
        // whose block there is off L1.
        assert_eq!(state.is_caught_up(), chain_b.is_none());
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
                    l1: l1_inclusion(),
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

    /// A verifier over two chains whose state holds a result at each timestamp from genesis, of
    /// L1 inclusion `inclusions[i]` on fork 0 at `GENESIS_TIME + i`, and with a denial decided at
    /// each of `denials`. Nothing on L1 is finalized.
    async fn fixture_with_results(inclusions: &[u64], denials: &[u64]) -> Fixture {
        let fixture = fixture([chain_a_at_genesis(), vec![at_genesis(&l2_genesis(CHAIN_B))]]).await;
        fixture.l1.finalize(0);
        {
            let mut state = fixture.state.write().unwrap();
            for (i, &l1) in inclusions.iter().enumerate() {
                let block = BlockNumHash { number: i as u64, hash: B256::with_last_byte(i as u8) };
                let result = VerifiedResult {
                    l1_inclusion: l1_id(l1, 0),
                    heads: [(CHAIN_A, block), (CHAIN_B, block)].into(),
                };
                state.commit(GENESIS_TIME + i as u64, result).unwrap();
            }
            for &timestamp in denials {
                state.deny(&denied_at(timestamp)).unwrap();
            }
        }
        fixture
    }

    fn denied_at(timestamp: u64) -> DeniedRecord {
        DeniedRecord {
            chain_id: CHAIN_B,
            block: BlockNumHash { number: timestamp - GENESIS_TIME, hash: B256::repeat_byte(0xdd) },
            output_root: B256::repeat_byte(0xdd),
            timestamp,
            l1: l1_id(4, 0),
        }
    }

    fn finalized_timestamp(fixture: &Fixture) -> Option<u64> {
        fixture.state.read().unwrap().finalized().map(|(ts, _)| *ts)
    }

    /// An L1 reorg that removes the L1 inclusion of committed results rewinds them, from the
    /// first one on the reorged blocks, with the denials decided on them, and keeps the earlier
    /// ones and the finalized head.
    #[tokio::test]
    async fn a_reorged_l1_inclusion_rewinds_the_results_on_it() {
        let fixture =
            fixture_with_results(&[4, 4, 5, 6, 6, 7], &[GENESIS_TIME + 1, GENESIS_TIME + 4]).await;
        fixture.l1.finalize(4);
        fixture.verifier.refresh_finalized().await.unwrap();
        assert_eq!(finalized_timestamp(&fixture), Some(GENESIS_TIME + 1));
        fixture.verifier.rewind_reorged_results().await.unwrap();

        fixture.l1.reorg_from(6);
        let wait = fixture.verifier.rewind_reorged_results().await.unwrap_err();

        assert!(
            matches!(wait, Wait::Rewound { from, .. } if from == GENESIS_TIME + 3),
            "got {wait:?}"
        );
        {
            let state = fixture.state.read().unwrap();
            assert_eq!(state.next_timestamp(), GENESIS_TIME + 3);
            assert_eq!(state.denied_blocks(CHAIN_B, 0).unwrap(), vec![denied_at(GENESIS_TIME + 1)]);
        }
        assert_eq!(finalized_timestamp(&fixture), Some(GENESIS_TIME + 1));
        fixture.verifier.rewind_reorged_results().await.unwrap();
    }

    /// The rewind takes priority over waiting for a frontier the reorg made unavailable, and the
    /// timestamps are verified again on the new L1 chain.
    #[tokio::test]
    async fn a_round_rewinds_before_it_waits() {
        let fixture = fixture_with_results(&[6], &[]).await;
        fixture.l1.reorg_from(6);

        let wait = fixture.verifier.round().await.unwrap_err();

        assert!(matches!(wait, Wait::Rewound { from: GENESIS_TIME, .. }), "got {wait:?}");
        fixture.verifier.round().await.unwrap();
        let state = fixture.state.read().unwrap();
        assert_eq!(state.next_timestamp(), GENESIS_TIME + 1);
        assert_eq!(state.latest().unwrap().1.l1_inclusion, l1_inclusion());
    }

    /// A reorg of the finalized result's L1 inclusion stops the verifier rather than rewind
    /// finality.
    #[rstest::rstest]
    #[case::below_later_results(&[4, 5, 6])]
    #[case::of_the_latest_result(&[4, 5])]
    #[tokio::test]
    async fn a_reorg_below_the_finalized_result_halts(#[case] inclusions: &[u64]) {
        let fixture = fixture_with_results(inclusions, &[]).await;
        fixture.l1.finalize(5);
        fixture.verifier.refresh_finalized().await.unwrap();
        assert_eq!(finalized_timestamp(&fixture), Some(GENESIS_TIME + 1));
        fixture.l1.reorg_from(5);

        let wait = fixture.verifier.rewind_reorged_results().await.unwrap_err();

        assert!(
            matches!(wait, Wait::FinalityViolation { timestamp, .. } if timestamp == GENESIS_TIME + 1),
            "got {wait:?}"
        );
        assert!(wait.is_fatal());
        assert_eq!(
            fixture.state.read().unwrap().next_timestamp(),
            GENESIS_TIME + inclusions.len() as u64
        );
    }

    /// A result is finalized only if its L1 inclusion is on the finalized L1 chain: at or below
    /// the finalized L1 block by number is not enough.
    #[rstest::rstest]
    #[case::at_the_finalized_height(7)]
    #[case::below_the_finalized_height(8)]
    #[tokio::test]
    async fn finality_needs_the_l1_inclusion_on_the_finalized_chain(#[case] finalized: u64) {
        let fixture = fixture_with_results(&[5, 6, 7], &[]).await;
        fixture.l1.reorg_from(6);
        fixture.l1.finalize(finalized);

        fixture.verifier.refresh_finalized().await.unwrap();
        assert_eq!(finalized_timestamp(&fixture), None);

        // Once the reorged results are rewound, the one before them is final.
        fixture.verifier.rewind_reorged_results().await.unwrap_err();
        fixture.verifier.refresh_finalized().await.unwrap();
        assert_eq!(finalized_timestamp(&fixture), Some(GENESIS_TIME));
    }

    /// A rewind deletes a denial decided on the reorged L1 data with the results, so a source that
    /// derives the denied block again on the new L1 chain has it checked, and denied, again.
    #[tokio::test]
    async fn a_denial_a_reorg_rewound_is_decided_again() {
        let invalid = invalid_block().hash_slow();
        let fixture = fixture([chain_a_at_genesis(), vec![Some((invalid, l1_id(6, 0)))]]).await;
        let denied = || fixture.state.read().unwrap().denied_blocks(CHAIN_B, 0).unwrap();
        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Denied { .. }), "got {wait:?}");
        fixture.answer(CHAIN_B, Some((replacement_block().hash_slow(), l1_id(6, 0))));
        fixture.verifier.round().await.unwrap();

        // L1 reorgs out the block both the denial and the replacement were derived from, and the
        // source derives the invalid block again from an earlier one.
        fixture.l1.reorg_from(6);
        fixture.answer(CHAIN_B, at_genesis(&invalid_block()));
        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Rewound { from: GENESIS_TIME, .. }), "got {wait:?}");
        assert_eq!(denied(), vec![]);

        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Denied { timestamp: GENESIS_TIME, .. }), "got {wait:?}");
        assert_eq!(
            denied().iter().map(|record| record.block.hash).collect::<Vec<_>>(),
            vec![invalid]
        );
    }

    /// A denial holds only on the L1 chain it was decided on: while its L1 block is canonical
    /// the round waits for the replacement, and once a reorg removed that block, even with every
    /// committed result still canonical, the denial is deleted and the block is checked, and
    /// here denied, again, on the new L1 chain.
    #[tokio::test]
    async fn a_denial_decided_on_a_reorged_l1_block_is_decided_again() {
        let invalid = invalid_block().hash_slow();
        let fixture = fixture([chain_a_at_genesis(), vec![Some((invalid, l1_id(6, 0)))]]).await;
        let denials = || fixture.state.read().unwrap().denied_blocks(CHAIN_B, 0).unwrap();
        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::Denied { .. }), "got {wait:?}");
        assert_eq!(denials()[0].l1, l1_id(6, 0), "decided on the frontier's L1 inclusion");
        let wait = fixture.verifier.round().await.unwrap_err();
        assert!(matches!(wait, Wait::AwaitingReplacement { .. }), "got {wait:?}");

        fixture.l1.reorg_from(6);
        fixture.answer(CHAIN_B, at_genesis(&invalid_block()));
        let wait = fixture.verifier.round().await.unwrap_err();

        assert!(matches!(wait, Wait::Denied { timestamp: GENESIS_TIME, .. }), "got {wait:?}");
        let records = denials();
        assert_eq!(records.len(), 1);
        assert_eq!((records[0].block.hash, records[0].l1), (invalid, l1_inclusion()));
    }
}
