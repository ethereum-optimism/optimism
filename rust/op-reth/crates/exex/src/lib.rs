//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_consensus::BlockHeader;
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use futures_util::TryStreamExt;
use reth_execution_types::Chain;
use reth_exex::{ExExContext, ExExEvent, ExExNotification};
use reth_node_api::{FullNodeComponents, NodePrimitives, NodeTypes};
use reth_optimism_trie::{
    CanonicalInMemoryStateProvider, OpProofStoragePruner, OpProofsProviderRO, OpProofsProviderRw,
    OpProofsStore, ProofWindowRange, engine::EngineHandle, proof::DatabaseStateRoot,
};
use reth_provider::{BlockHashReader, BlockNumReader, HeaderProvider};
use reth_trie::{
    HashedPostState, HashedPostStateSorted, SortedTrieData, StateRoot, updates::TrieUpdatesSorted,
};
use std::sync::Arc;
use tracing::{debug, info, warn};

// Safety threshold for maximum blocks to prune automatically on startup.
// If the required prune exceeds this, the node will error out and require manual pruning. Default
// is 1000 blocks.
const MAX_PRUNE_BLOCKS_STARTUP: u64 = 1000;

/// Default proofs history window: 1 month of blocks at 2s block time
const DEFAULT_PROOFS_HISTORY_WINDOW: u64 = 1_296_000;

/// Default verification interval: disabled (0 = no periodic re-execution)
const DEFAULT_VERIFICATION_INTERVAL: u64 = 0;

/// Number of blocks behind the chain tip within which we consider the ExEx to be in real-time
/// operation. Notifications further behind than this are treated as sync catch-up and handled
/// asynchronously by the engine.
const REAL_TIME_BLOCKS_THRESHOLD: u64 = 64;

/// Builder for [`OpProofsExEx`].
#[derive(Debug)]
pub struct OpProofsExExBuilder<Node, Storage>
where
    Node: FullNodeComponents,
{
    ctx: ExExContext<Node>,
    storage: Storage,
    proofs_history_window: u64,
    verification_interval: u64,
}

impl<Node, Storage> OpProofsExExBuilder<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// Create a new builder with required parameters and defaults.
    pub const fn new(ctx: ExExContext<Node>, storage: Storage) -> Self {
        Self {
            ctx,
            storage,
            proofs_history_window: DEFAULT_PROOFS_HISTORY_WINDOW,
            verification_interval: DEFAULT_VERIFICATION_INTERVAL,
        }
    }

    /// Sets the window to span blocks for proofs history.
    pub const fn with_proofs_history_window(mut self, window: u64) -> Self {
        self.proofs_history_window = window;
        self
    }

    /// Sets the interval at which blocks are re-executed to verify pre-computed trie data.
    ///
    /// Every `interval`-th block (by block number) will be executed in full even when
    /// pre-computed trie data is available, allowing detection of any divergence.
    /// Set to `0` (the default) to disable periodic verification.
    pub const fn with_verification_interval(mut self, interval: u64) -> Self {
        self.verification_interval = interval;
        self
    }

    /// Builds the [`OpProofsExEx`].
    pub fn build(self) -> OpProofsExEx<Node, Storage> {
        OpProofsExEx {
            ctx: self.ctx,
            storage: self.storage,
            proofs_history_window: self.proofs_history_window,
            verification_interval: self.verification_interval,
        }
    }
}

/// OP Proofs ExEx - processes blocks and tracks state changes within fault proof window.
///
/// Saves and serves trie nodes to make proofs faster. This handles the process of
/// saving the current state, new blocks as they're added, and serving proof RPCs
/// based on the saved data.
///
/// # Examples
///
/// The following example shows how to install the ExEx with either in-memory or persistent storage.
/// This can be used when launching an OP-Reth node via a binary.
///
/// ```
/// use futures_util::FutureExt;
/// use reth_db::test_utils::create_test_rw_db;
/// use reth_node_api::NodeTypesWithDBAdapter;
/// use reth_node_builder::{NodeBuilder, NodeConfig};
/// use reth_optimism_chainspec::OP_DEV;
/// use reth_optimism_exex::OpProofsExEx;
/// use reth_optimism_node::{OpNode, args::RollupArgs};
/// use reth_optimism_trie::{InMemoryProofsStorage, OpProofsStorage, db::MdbxProofsStorage};
/// use reth_provider::providers::BlockchainProvider;
/// use std::{sync::Arc, time::Duration};
///
/// let config = NodeConfig::new(OP_DEV.clone());
/// let db = create_test_rw_db();
/// let args = RollupArgs::default();
/// let op_node = OpNode::new(args);
///
/// // Create in-memory or persistent storage
/// let storage: OpProofsStorage<Arc<InMemoryProofsStorage>> =
///     Arc::new(InMemoryProofsStorage::new()).into();
///
/// // Example for creating persistent storage
/// # let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
/// # let storage_path = temp_dir.path().join("proofs_storage");
///
/// # let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = Arc::new(
/// #    MdbxProofsStorage::new(&storage_path).expect("Failed to create MdbxProofsStorage"),
/// # ).into();
///
/// let storage_exec = storage.clone();
/// let proofs_history_window = 1_296_000u64;
///
/// // Verification interval: perform full execution every N blocks
/// let verification_interval = 0; // 0 = disabled, 100 = verify every 100 blocks
///
/// // Can also use install_exex_if along with a boolean flag
/// // Set this based on your configuration or CLI args
/// let _builder = NodeBuilder::new(config)
///     .with_database(db)
///     .with_types_and_provider::<OpNode, BlockchainProvider<NodeTypesWithDBAdapter<OpNode, _>>>()
///     .with_components(op_node.components())
///     .install_exex("proofs-history", move |exex_context| async move {
///         Ok(OpProofsExEx::builder(exex_context, storage_exec)
///             .with_proofs_history_window(proofs_history_window)
///             .with_verification_interval(verification_interval)
///             .build()
///             .run()
///             .boxed())
///     })
///     .on_node_started(|_full_node| Ok(()))
///     .check_launch();
/// ```
#[derive(Debug)]
pub struct OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// The ExEx context containing the node related utilities e.g. provider, notifications,
    /// events.
    ctx: ExExContext<Node>,
    /// The type of storage DB.
    storage: Storage,
    /// The window to span blocks for proofs history. Value is the number of blocks, received as
    /// cli arg.
    proofs_history_window: u64,
    /// How often (in blocks) to re-execute a block for verification even when pre-computed trie
    /// data is available. `0` disables periodic verification.
    verification_interval: u64,
}

impl<Node, Storage> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents,
{
    /// Create a new `OpProofsExEx` instance.
    pub fn new(ctx: ExExContext<Node>, storage: Storage) -> Self {
        OpProofsExExBuilder::new(ctx, storage).build()
    }

    /// Create a new builder for `OpProofsExEx`.
    pub const fn builder(
        ctx: ExExContext<Node>,
        storage: Storage,
    ) -> OpProofsExExBuilder<Node, Storage> {
        OpProofsExExBuilder::new(ctx, storage)
    }
}

impl<Node, Storage, Primitives> OpProofsExEx<Node, Storage>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Node::Provider: CanonicalInMemoryStateProvider,
    Primitives: NodePrimitives,
    Primitives::Block: Clone,
    Storage: OpProofsStore + Clone + 'static,
{
    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        self.ensure_initialized()?;
        self.reconcile_with_canonical_chain()?;

        let pruner = OpProofStoragePruner::new(
            self.storage.clone(),
            self.ctx.provider().clone(),
            self.proofs_history_window,
        );

        let engine_handle = EngineHandle::spawn(
            self.ctx.evm_config().clone(),
            self.ctx.provider().clone(),
            self.storage.clone(),
            pruner,
        );

        engine_handle.sync_to(self.ctx.provider().best_block_number()?)?;

        while let Some(notification) = self.ctx.notifications.try_next().await? {
            self.handle_notification(notification, &engine_handle)?;
        }

        Ok(())
    }

    /// Ensure proofs storage is initialized
    fn ensure_initialized(&self) -> eyre::Result<()> {
        // Check if proofs storage is initialized. An empty proof window returns NoBlocksFound;
        let provider_ro = self.storage.provider_ro()?;
        let window = provider_ro.get_proof_window().map_err(|_| eyre::eyre!(
            "Proofs storage not initialized. Please run 'op-reth proofs init --proofs-history.storage-path <PATH>' first."
        ))?;
        let earliest_block_number = window.earliest.number;
        let latest_block_number = window.latest.number;

        // Check if we have accumulated too much history for the configured window.
        // If the gap between what we have and what we want to keep is too large, the auto-pruner
        // will stall the node.
        let target_earliest = latest_block_number.saturating_sub(self.proofs_history_window);
        if target_earliest > earliest_block_number {
            let blocks_to_prune = target_earliest - earliest_block_number;
            if blocks_to_prune > MAX_PRUNE_BLOCKS_STARTUP {
                return Err(eyre::eyre!(
                    "Configuration requires pruning {} blocks, which exceeds the safety threshold of {}. \
                     Huge prune operations can stall the node. \
                     Please run 'op-reth proofs prune' manually before starting the node.",
                    blocks_to_prune,
                    MAX_PRUNE_BLOCKS_STARTUP
                ));
            }
        }

        Ok(())
    }

    /// Unwinds the proofs storage to the highest stored block that is on reth's canonical chain.
    ///
    /// reth can start below the proofs tip, or on another fork: after an unclean shutdown it
    /// unwinds to its last persisted state trie without notifying ExExes, and the dropped blocks
    /// may be replaced by another fork. Indexing on top of a stale tip would keep serving the
    /// dropped blocks and reject reth's next block with a parent hash mismatch.
    ///
    /// Returns an error if no block of the proofs window is on reth's canonical chain.
    fn reconcile_with_canonical_chain(&self) -> eyre::Result<()> {
        let provider = self.ctx.provider();
        let best_block = provider.best_block_number()?;
        let storage_provider = self.storage.provider_ro()?;
        let ProofWindowRange { earliest, latest } = storage_provider.get_proof_window()?;

        if latest.number <= best_block && provider.block_hash(latest.number)? == Some(latest.hash) {
            return Ok(());
        }

        // The stored blocks form a single chain, so if none of them is canonical, the earliest
        // is not either.
        if earliest.number > best_block ||
            provider.block_hash(earliest.number)? != Some(earliest.hash)
        {
            return Err(eyre::eyre!(
                "Proofs storage (blocks {}..={}, tip {}) shares no block with reth's canonical \
                 chain (best block {best_block}). Please re-initialize it with \
                 'op-reth proofs init --proofs-history.storage-path <PATH>'.",
                earliest.number,
                latest.number,
                latest.hash,
            ));
        }

        // The storage records block hashes only for the window bounds. Blocks in between are
        // matched by their state root instead, which pins the state proofs are served from.
        let highest_candidate =
            if latest.number > best_block { best_block } else { latest.number - 1 };
        let mut common = earliest;
        for number in (earliest.number + 1..=highest_candidate).rev() {
            let header = provider
                .sealed_header(number)?
                .ok_or_else(|| eyre::eyre!("canonical header {number} not found"))?;
            let state_root =
                StateRoot::overlay_root(&storage_provider, number, HashedPostState::default())?;
            if state_root == header.state_root() {
                common = NumHash::new(number, header.hash());
                break;
            }
        }
        drop(storage_provider);

        warn!(
            target: "optimism::exex",
            proofs_tip = latest.number,
            proofs_tip_hash = ?latest.hash,
            best_block,
            unwind_to = common.number,
            unwind_to_hash = ?common.hash,
            "Proofs storage is not on reth's canonical chain, unwinding it",
        );
        let storage_provider = self.storage.provider_rw()?;
        // Only the number of the first removed block is read; the storage does not record its
        // hash.
        storage_provider.unwind_history(BlockWithParent::new(
            common.hash,
            NumHash::new(common.number + 1, Default::default()),
        ))?;
        storage_provider.commit()?;
        Ok(())
    }

    fn handle_notification(
        &self,
        notification: ExExNotification<Primitives>,
        engine_handle: &EngineHandle<Primitives::Block>,
    ) -> eyre::Result<()> {
        match &notification {
            ExExNotification::ChainCommitted { new } => {
                self.handle_chain_committed(new.clone(), engine_handle)?
            }
            ExExNotification::ChainReorged { old, new } => {
                self.handle_chain_reorged(old.clone(), new.clone(), engine_handle)?
            }
            ExExNotification::ChainReverted { old } => {
                self.handle_chain_reverted(old.clone(), engine_handle)?
            }
        }

        if let Some(committed_chain) = notification.committed_chain() {
            self.ctx.events.send(ExExEvent::FinishedHeight(committed_chain.tip().num_hash()))?;
        }

        Ok(())
    }

    fn handle_chain_committed(
        &self,
        new: Arc<Chain<Primitives>>,
        engine_handle: &EngineHandle<Primitives::Block>,
    ) -> eyre::Result<()> {
        debug!(
            target: "optimism::exex",
            block_number = new.tip().number(),
            block_hash = ?new.tip().hash(),
            "ChainCommitted notification received",
        );

        let best_block = self.ctx.provider().best_block_number()?;
        let is_near_tip =
            best_block.saturating_sub(new.tip().number()) < REAL_TIME_BLOCKS_THRESHOLD;
        if !is_near_tip {
            engine_handle.sync_to(new.tip().number())?;
            return Ok(());
        }

        // `Chain::blocks()` is a BTreeMap so iteration is already ordered oldest → newest.
        for (&block_number, block) in new.blocks() {
            // Fast path: use pre-computed trie data only when verification is not due.
            let should_verify = self.verification_interval > 0 &&
                block_number.is_multiple_of(self.verification_interval);
            let precomputed = (!should_verify).then(|| new.trie_data_at(block_number)).flatten();

            if let Some(d) = precomputed {
                let SortedTrieData { hashed_state, trie_updates } = &d.get().sorted;
                engine_handle.index_block(
                    block.block_with_parent(),
                    (**trie_updates).clone(),
                    (**hashed_state).clone(),
                )?;
            } else if let Err(err) = engine_handle.execute_block(block) {
                // Slow path: execute the block in full (no trie data, or verification interval
                // hit). Execution reads bytecode reth has not persisted from reth's in-memory
                // chain, so a block from a fork reth has since left can fail. Leave that block to
                // the sync loop, which executes reth's canonical block at this height.
                if self.ctx.provider().block_hash(block_number)? == Some(block.hash()) {
                    return Err(err.into());
                }
                debug!(
                    target: "optimism::exex",
                    block_number,
                    block_hash = ?block.hash(),
                    %err,
                    "Block left reth's chain during proofs execution; syncing to reth's chain",
                );
                engine_handle.sync_to(new.tip().number())?;
                return Ok(());
            }
        }

        Ok(())
    }

    fn handle_chain_reorged(
        &self,
        old: Arc<Chain<Primitives>>,
        new: Arc<Chain<Primitives>>,
        engine_handle: &EngineHandle<Primitives::Block>,
    ) -> eyre::Result<()> {
        info!(
            old_block_number = old.tip().number(),
            old_block_hash = ?old.tip().hash(),
            new_block_number = new.tip().number(),
            new_block_hash = ?new.tip().hash(),
            "ChainReorged notification received",
        );

        if old.fork_block() != new.fork_block() {
            return Err(eyre::eyre!(
                "Fork blocks do not match: old fork block {:?}, new fork block {:?}",
                old.fork_block(),
                new.fork_block()
            ));
        }

        let mut block_updates: Vec<(
            BlockWithParent,
            Arc<TrieUpdatesSorted>,
            Arc<HashedPostStateSorted>,
        )> = Vec::with_capacity(new.len());
        for block_number in new.blocks().keys() {
            let block = new
                .blocks()
                .get(block_number)
                .ok_or_else(|| eyre::eyre!("Missing block {} in new chain", block_number))?;
            let trie_data = new
                .trie_data_at(*block_number)
                .ok_or_else(|| {
                    eyre::eyre!("Missing Trie data for block {} in new chain", block_number)
                })?
                .get();
            let trie_updates = &trie_data.sorted.trie_updates;
            let hashed_state = &trie_data.sorted.hashed_state;

            block_updates.push((
                block.block_with_parent(),
                trie_updates.clone(),
                hashed_state.clone(),
            ));
        }

        engine_handle.reorg(block_updates)?;

        Ok(())
    }

    fn handle_chain_reverted(
        &self,
        old: Arc<Chain<Primitives>>,
        engine_handle: &EngineHandle<Primitives::Block>,
    ) -> eyre::Result<()> {
        info!(
            target: "optimism::exex",
            old_block_number = old.tip().number(),
            old_block_hash = ?old.tip().hash(),
            "ChainReverted notification received",
        );

        engine_handle.unwind(old.first().block_with_parent())?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Header, private::alloy_primitives::B256};
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use reth_db::test_utils::tempdir_path;
    use reth_ethereum_primitives::{Block, Receipt};
    use reth_execution_types::{Chain, ExecutionOutcome};
    use reth_exex_test_utils::TestExExHandle;
    use reth_optimism_trie::{
        BlockStateDiff, OpProofsInitProvider, OpProofsProviderRO, OpProofsProviderRw,
        OpProofsStore, db::MdbxProofsStorage, engine::EngineHandle,
    };
    use reth_primitives_traits::{Account, RecoveredBlock, SealedBlock, SealedHeader};
    use reth_provider::{BlockWriter, CanonChainTracker};
    use reth_trie::{
        ComputedTrieData, HashedPostState, HashedPostStateSorted, LazyTrieData,
        updates::TrieUpdatesSorted,
    };
    use std::{collections::BTreeMap, default::Default, sync::Arc};

    // -------------------------------------------------------------------------
    // Helpers: deterministic blocks and deterministic Chain with precomputed updates
    // -------------------------------------------------------------------------
    fn b256(byte: u8) -> B256 {
        B256::new([byte; 32])
    }

    // deterministic hash from block number: 0 -> 0x00.., 1 -> 0x01.., etc.
    fn hash_for_num(num: u64) -> B256 {
        // if you only care about small test numbers, this is enough:
        b256(num as u8)

        // If you want to avoid wrapping when num > 255, use something like:
        // let mut out = [0u8; 32];
        // out[0..8].copy_from_slice(&num.to_be_bytes());
        // B256::new(out)
    }

    fn mk_block(num: u64) -> RecoveredBlock<Block> {
        let mut b: RecoveredBlock<Block> = Default::default();
        b.set_block_number(num);
        b.set_hash(hash_for_num(num));
        b.set_parent_hash(hash_for_num(num - 1));
        b
    }

    fn mk_chain_with_updates(
        from: u64,
        to: u64,
        hash_override: Option<B256>,
    ) -> Chain<reth_ethereum_primitives::EthPrimitives> {
        let mut blocks: Vec<RecoveredBlock<Block>> = Vec::new();
        let mut trie_data = BTreeMap::new();

        for n in from..=to {
            let mut b = mk_block(n);
            if let Some(hash) = hash_override {
                b.set_hash(hash);
            }
            blocks.push(b);

            let data = LazyTrieData::ready(ComputedTrieData::new(
                Arc::new(HashedPostStateSorted::default()),
                Arc::new(TrieUpdatesSorted::default()),
            ));
            trie_data.insert(n, data);
        }

        mk_chain(blocks, from, trie_data)
    }

    fn mk_chain_without_updates(
        from: u64,
        to: u64,
    ) -> Chain<reth_ethereum_primitives::EthPrimitives> {
        mk_chain((from..=to).map(mk_block).collect(), from, BTreeMap::new())
    }

    fn mk_chain(
        blocks: Vec<RecoveredBlock<Block>>,
        first_block: u64,
        trie_data: BTreeMap<u64, LazyTrieData>,
    ) -> Chain<reth_ethereum_primitives::EthPrimitives> {
        let execution_outcome: ExecutionOutcome<Receipt> = ExecutionOutcome {
            bundle: Default::default(),
            receipts: Vec::new(),
            requests: Vec::new(),
            first_block,
        };

        Chain::new(blocks, execution_outcome, trie_data)
    }

    // Bootstrap storage to the genesis block via the init flow (sets earliest = latest = genesis).
    fn init_storage<S: OpProofsStore>(storage: S) {
        init_storage_at(storage, NumHash::new(0, b256(0x00)));
    }

    fn init_storage_at<S: OpProofsStore>(storage: S, genesis: NumHash) {
        let init = storage.initialization_provider().expect("init");
        init.set_initial_state_anchor(genesis).expect("anchor");
        init.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(init).expect("commit");
    }

    fn store_block<S: OpProofsStore>(
        storage: &S,
        parent: B256,
        block: NumHash,
        diff: BlockStateDiff,
    ) {
        let rw = storage.provider_rw().expect("provider rw");
        rw.store_trie_updates(BlockWithParent::new(parent, block), diff).expect("store block");
        OpProofsProviderRw::commit(rw).expect("commit");
    }

    // A state diff that makes the block's state root differ from an empty block's.
    fn diff_with_account() -> BlockStateDiff {
        let mut post_state = HashedPostState::default();
        post_state.accounts.insert(b256(0xaa), Some(Account { nonce: 1, ..Default::default() }));
        BlockStateDiff { sorted_post_state: post_state.into_sorted(), ..Default::default() }
    }

    // Inserts empty blocks 1..=tip on top of reth's genesis and makes the last one reth's head.
    // Returns the canonical headers indexed by block number, genesis included.
    fn insert_canonical_chain<P>(
        handle: &TestExExHandle,
        provider: &P,
        tip: u64,
    ) -> Vec<SealedHeader>
    where
        P: CanonChainTracker<Header = Header>,
    {
        let mut headers = vec![handle.genesis.clone_sealed_header()];
        let rw = handle.provider_factory.provider_rw().expect("provider rw");
        for number in 1..=tip {
            let parent_hash = headers.last().expect("genesis").hash();
            let header = Header { number, parent_hash, ..Default::default() };
            let block = SealedBlock::seal_slow(Block { header, body: Default::default() })
                .try_recover()
                .expect("empty block");
            rw.insert_block(&block).expect("insert block");
            headers.push(block.clone_sealed_header());
        }
        rw.commit().expect("commit");
        provider.set_canonical_head(headers.last().expect("genesis").clone());
        headers
    }

    /// Waits until the proofs storage's latest block is `expected`.
    async fn wait_for_latest_block<S: OpProofsStore>(store: &S, expected: NumHash) {
        tokio::time::timeout(std::time::Duration::from_secs(5), async {
            loop {
                let provider = store.provider_ro().expect("provider ro");
                if provider.get_latest_block().expect("get latest block") == expected {
                    return;
                }
                tokio::time::sleep(std::time::Duration::from_millis(10)).await;
            }
        })
        .await
        .expect("proofs storage should be unwound to reth's chain within 5s");
    }

    // Initialize exex with config
    fn build_test_exex<NodeT, Store>(
        ctx: ExExContext<NodeT>,
        storage: Store,
    ) -> OpProofsExEx<NodeT, Store>
    where
        NodeT: FullNodeComponents,
        Store: OpProofsStore + Clone + 'static,
    {
        OpProofsExEx::builder(ctx, storage)
            .with_proofs_history_window(20)
            .with_verification_interval(1000)
            .build()
    }

    #[tokio::test]
    async fn handle_notification_chain_committed() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );
        let exex = build_test_exex(ctx, store.clone());

        // Notification: chain committed 1..5
        let new_chain = Arc::new(mk_chain_with_updates(1, 1, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        exex.handle_notification(notif, &engine_handle).expect("handle chain commit");

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 1);
    }

    #[tokio::test]
    async fn handle_notification_chain_committed_skips_already_processed() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        // Process blocks 1..5 sequentially to trigger real-time path (synchronous)
        for i in 1..=5 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };
            exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        }

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 5);

        // Try to handle already processed notification
        let new_chain = Arc::new(mk_chain_with_updates(5, 5, Some(hash_for_num(10))));
        let notif = ExExNotification::ChainCommitted { new: new_chain };
        exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        engine_handle.flush();
        let latest =
            store.provider_ro().expect("provider ro").get_latest_block().expect("get latest block");
        assert_eq!(latest.number, 5);
        assert_eq!(latest.hash, hash_for_num(5)); // block was not updated
    }

    #[tokio::test]
    async fn handle_notification_chain_reorged() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };
            exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        }

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to reorg from block 6..12
        let old_chain = Arc::new(mk_chain_with_updates(6, 10, None));
        let new_chain = Arc::new(mk_chain_with_updates(6, 12, None));

        // Notification: chain reorged 6..12
        let notif = ExExNotification::ChainReorged { new: new_chain, old: old_chain };

        exex.handle_notification(notif, &engine_handle).expect("handle chain re-orged");
        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 12);
    }

    #[tokio::test]
    async fn handle_notification_chain_reorged_skips_beyond_stored_blocks() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        }

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to reorg starting at block 12 (beyond stored tip).
        // Both chains share the same fork point (block 11), so this is a valid reorg notification
        // that starts beyond what we've indexed — the engine should skip it.
        let old_chain = Arc::new(mk_chain_with_updates(12, 15, None));
        let new_chain = Arc::new(mk_chain_with_updates(12, 20, None));

        // Notification: chain reorged 12..20, fork at 11
        let notif = ExExNotification::ChainReorged { new: new_chain, old: old_chain };

        exex.handle_notification(notif, &engine_handle).expect("handle chain re-orged");
        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 10);
    }

    #[tokio::test]
    async fn handle_notification_chain_reverted() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        for i in 1..=10 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        }

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 10);

        // Now the tip is 10, and we want to revert from block 9..10
        let old_chain = Arc::new(mk_chain_with_updates(9, 10, None));

        // Notification: chain reverted 9..10
        let notif = ExExNotification::ChainReverted { old: old_chain };

        exex.handle_notification(notif, &engine_handle).expect("handle chain reverted");
        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 8);
    }

    #[tokio::test]
    async fn handle_notification_chain_reverted_skips_beyond_stored_blocks() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        for i in 1..=5 {
            let new_chain = Arc::new(mk_chain_with_updates(i, i, None));
            let notif = ExExNotification::ChainCommitted { new: new_chain };

            exex.handle_notification(notif, &engine_handle).expect("handle chain commit");
        }

        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 5);

        // Now the tip is 10, and we want to revert from block 9..10
        let old_chain = Arc::new(mk_chain_with_updates(9, 10, None));

        // Notification: chain reverted 9..10
        let notif = ExExNotification::ChainReverted { old: old_chain };

        exex.handle_notification(notif, &engine_handle).expect("handle chain reverted");
        engine_handle.flush();
        let latest = store
            .provider_ro()
            .expect("provider ro")
            .get_latest_block()
            .expect("get latest block")
            .number;
        assert_eq!(latest, 5);
    }

    #[tokio::test]
    async fn ensure_initialized_errors_on_storage_not_initialized() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, store.clone());
        let _ = exex.ensure_initialized().expect_err("should return error");
    }

    #[tokio::test]
    async fn ensure_initialized_errors_when_prune_exceeds_threshold() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        for i in 1..1100 {
            let rw = store.provider_rw().expect("provider rw");
            rw.store_trie_updates(
                BlockWithParent::new(hash_for_num(i - 1), BlockNumHash::new(i, hash_for_num(i))),
                BlockStateDiff::default(),
            )
            .expect("store trie update");
            OpProofsProviderRw::commit(rw).expect("commit");
        }

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, store.clone());
        let _ = exex.ensure_initialized().expect_err("should return error");
    }

    #[tokio::test]
    async fn ensure_initialized_succeeds() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let exex = build_test_exex(ctx, store.clone());
        exex.ensure_initialized().expect("should not return error");
    }

    #[tokio::test]
    async fn handle_notification_errors_on_empty_storage() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        // Any notification will do
        let new_chain = Arc::new(mk_chain_with_updates(1, 5, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        let err = exex.handle_notification(notif, &engine_handle).unwrap_err();
        // Error now comes from the engine layer (storage not initialised).
        assert_eq!(err.to_string(), "No blocks found");
    }

    #[tokio::test]
    async fn handle_notification_schedules_execution_on_gap_without_trie_data() {
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");
        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );
        let exex = build_test_exex(ctx, store.clone());

        // Without pre-computed trie data this notification needs EVM execution. The proofs DB
        // is still at genesis, so blocks 1..4 must be indexed before executing this range. The
        // engine schedules catch-up; the empty test provider has no blocks to replay.
        let notif =
            ExExNotification::ChainCommitted { new: Arc::new(mk_chain_without_updates(5, 10)) };
        exex.handle_notification(notif, &engine_handle)
            .expect("missing predecessor blocks should schedule catch-up");

        engine_handle.flush();
        let latest =
            store.provider_ro().expect("provider ro").get_latest_block().expect("get").number;
        assert_eq!(latest, 0);
    }

    /// Returns block 1 sealed with `state_root`, built on the genesis block the tests' proofs store
    /// is initialized with.
    fn block_one_with_state_root(state_root: B256) -> RecoveredBlock<Block> {
        let header = alloy_consensus::Header {
            number: 1,
            parent_hash: hash_for_num(0),
            state_root,
            ..Default::default()
        };
        reth_primitives_traits::SealedBlock::seal_slow(Block { header, body: Default::default() })
            .try_recover()
            .expect("empty block")
    }

    /// Makes `block` reth's canonical block at its height.
    fn make_canonical(
        handle: &reth_exex_test_utils::TestExExHandle,
        provider: &impl reth_provider::CanonChainTracker<Header = alloy_consensus::Header>,
        block: &RecoveredBlock<Block>,
    ) {
        use reth_provider::BlockWriter;

        let rw = handle.provider_factory.provider_rw().expect("provider rw");
        rw.insert_block(block).expect("insert block");
        rw.commit().expect("commit");
        provider.set_canonical_head(block.clone_sealed_header());
    }

    #[tokio::test]
    async fn handle_notification_hands_block_reth_replaced_to_sync_loop() {
        use reth_optimism_trie::engine::EngineError;
        use std::time::{Duration, Instant};

        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        init_storage(store.clone());

        let (ctx, handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");
        let provider = ctx.components.provider.clone();
        let pruner = OpProofStoragePruner::new(store.clone(), provider.clone(), 20);
        // The test context's EVM config is a no-op that cannot execute blocks.
        let engine_handle = EngineHandle::spawn_with_thresholds(
            reth_evm_ethereum::EthEvmConfig::mainnet(),
            provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );
        let exex = build_test_exex(ctx, store.clone());

        // The notification's block 1 builds on the proofs tip, but its header's state root is not
        // the one execution produces, so its execution fails.
        let err = engine_handle.execute_block(&mk_block(1)).expect_err("state root should differ");
        let EngineError::StateRootMismatch { current_state_hash, .. } = err else {
            panic!("expected a state root mismatch, got {err}");
        };

        // reth's canonical block 1 has the state root execution produces, standing in for the
        // block that replaced the notification's block in a reorg.
        let canonical = block_one_with_state_root(current_state_hash);
        make_canonical(&handle, &provider, &canonical);

        let notif =
            ExExNotification::ChainCommitted { new: Arc::new(mk_chain_without_updates(1, 1)) };
        exex.handle_notification(notif, &engine_handle)
            .expect("a failed block reth has replaced should not stop the ExEx");

        // Only the sync loop can index reth's block 1: the ExEx stopped at the failed block.
        let deadline = Instant::now() + Duration::from_secs(30);
        loop {
            let latest = store.provider_ro().expect("provider ro").get_latest_block().expect("get");
            if latest.number == 1 {
                assert_eq!(latest.hash, canonical.hash());
                break;
            }
            assert!(Instant::now() < deadline, "sync loop did not index reth's block 1");
            std::thread::sleep(Duration::from_millis(100));
        }
    }

    #[tokio::test]
    async fn handle_notification_propagates_failed_execution_of_canonical_block() {
        assert_failure_of_canonical_block_stops_exex(0, false).await;
    }

    #[tokio::test]
    async fn handle_notification_propagates_failed_verification_of_canonical_block() {
        assert_failure_of_canonical_block_stops_exex(1, true).await;
    }

    /// Asserts that the ExEx stops when reth's canonical block 1 fails proofs execution, both on
    /// the execution path and at the verification interval.
    async fn assert_failure_of_canonical_block_stops_exex(
        verification_interval: u64,
        with_trie_data: bool,
    ) {
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        init_storage(store.clone());

        let (ctx, handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");
        let provider = ctx.components.provider.clone();
        let pruner = OpProofStoragePruner::new(store.clone(), provider.clone(), 20);
        // The test context's EVM config is a no-op that cannot execute blocks.
        let engine_handle = EngineHandle::spawn_with_thresholds(
            reth_evm_ethereum::EthEvmConfig::mainnet(),
            provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );
        let exex = OpProofsExEx::builder(ctx, store.clone())
            .with_proofs_history_window(20)
            .with_verification_interval(verification_interval)
            .build();

        // reth's canonical block 1 is the notification's block, whose header's state root is not
        // the one execution produces. Executing it again would fail the same way, so the failure
        // stops the ExEx.
        let block = mk_block(1);
        make_canonical(&handle, &provider, &block);
        let chain = if with_trie_data {
            mk_chain_with_updates(1, 1, None)
        } else {
            mk_chain_without_updates(1, 1)
        };
        let notif = ExExNotification::ChainCommitted { new: Arc::new(chain) };
        let err = exex
            .handle_notification(notif, &engine_handle)
            .expect_err("a failed block reth still has should stop the ExEx");
        assert!(err.to_string().contains("State root mismatch for block 1"), "{err:#}");

        engine_handle.flush();
        let latest =
            store.provider_ro().expect("provider ro").get_latest_block().expect("get").number;
        assert_eq!(latest, 0);
    }

    #[tokio::test]
    async fn run_unwinds_proofs_storage_ahead_of_reth() {
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let (ctx, handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let canonical = insert_canonical_chain(&handle, &ctx.components.provider, 3);
        init_storage_at(store.clone(), canonical[0].num_hash());
        for header in &canonical[1..] {
            store_block(&store, header.parent_hash, header.num_hash(), Default::default());
        }
        // Blocks 4 and 5 were indexed before reth dropped them at restart.
        let mut parent = canonical[3].hash();
        for number in 4..=5 {
            store_block(
                &store,
                parent,
                NumHash::new(number, hash_for_num(number)),
                diff_with_account(),
            );
            parent = hash_for_num(number);
        }

        let run = tokio::spawn(build_test_exex(ctx, store.clone()).run());

        wait_for_latest_block(&store, canonical[3].num_hash()).await;

        run.abort();
        let _ = run.await;
    }

    #[tokio::test]
    async fn run_unwinds_proofs_storage_on_diverged_fork() {
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        let (ctx, handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let canonical = insert_canonical_chain(&handle, &ctx.components.provider, 3);
        init_storage_at(store.clone(), canonical[0].num_hash());
        store_block(&store, canonical[0].hash(), canonical[1].num_hash(), Default::default());
        // The proofs storage followed a fork from block 2 on that reth no longer has. The stored
        // tip is at reth's height but has another hash.
        store_block(&store, canonical[1].hash(), NumHash::new(2, b256(0xb2)), diff_with_account());
        store_block(&store, b256(0xb2), NumHash::new(3, b256(0xb3)), diff_with_account());

        let run = tokio::spawn(build_test_exex(ctx, store.clone()).run());

        wait_for_latest_block(&store, canonical[1].num_hash()).await;

        run.abort();
        let _ = run.await;
    }

    #[tokio::test]
    async fn run_fails_without_canonical_block_in_proofs_window() {
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
        // The proofs storage is anchored on a genesis that is not reth's.
        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let err = tokio::time::timeout(
            std::time::Duration::from_secs(5),
            build_test_exex(ctx, store).run(),
        )
        .await
        .expect("run should fail at startup")
        .expect_err("no proofs block is canonical");
        assert!(err.to_string().contains("op-reth proofs init"), "{err:#}");
    }

    #[tokio::test]
    async fn handle_notification_schedules_async_on_gap() {
        // MDBX proofs storage
        let dir = tempdir_path();
        let store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));

        init_storage(store.clone());

        let (ctx, _handle) =
            reth_exex_test_utils::test_exex_context().await.expect("exex test context");

        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        let engine_handle = EngineHandle::spawn_with_thresholds(
            ctx.components.components.evm_config.clone(),
            ctx.components.provider.clone(),
            store.clone(),
            pruner,
            1,
            2,
        );

        let exex = build_test_exex(ctx, store.clone());

        // Notification: chain committed 5..10 (Blocks 1,2,3,4 are missing from storage)
        let new_chain = Arc::new(mk_chain_with_updates(5, 10, None));
        let notif = ExExNotification::ChainCommitted { new: new_chain };

        // Process notification — should return immediately (gap detected, deferred to engine).
        exex.handle_notification(notif, &engine_handle)
            .expect("handle chain commit should return ok immediately");

        // Verify the notification handler did NOT process blocks synchronously.
        // The engine has a sync target set but no blocks in the provider, so its catch-up
        // will error out without writing anything. Storage stays at block 0.
        engine_handle.flush();
        let latest =
            store.provider_ro().expect("provider ro").get_latest_block().expect("get").number;
        assert_eq!(latest, 0, "Main thread should not have processed the blocks synchronously");
    }
}
