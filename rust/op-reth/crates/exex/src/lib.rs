//! ExEx unique for OP-Reth. See also [`reth_exex`] for more op-reth execution extensions.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_consensus::BlockHeader;
use futures_util::TryStreamExt;
use reth_execution_types::Chain;
use reth_exex::{ExExContext, ExExEvent, ExExNotification};
use reth_node_api::{FullNodeComponents, NodePrimitives, NodeTypes};
use reth_optimism_trie::{
    OpProofStoragePruner, OpProofsProviderRO, OpProofsStore,
    engine::{EngineHandle, ReorgBlock},
};
use reth_provider::BlockNumReader;
use reth_trie::{ComputedTrieData, LazyTrieData, SortedTrieData};
use std::sync::Arc;
use tracing::{debug, info};

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
/// We are currently using it in optimism/bin/src/main.rs.
///
/// ```
/// use futures_util::FutureExt;
/// use reth_db::test_utils::create_test_rw_db;
/// use reth_node_api::NodeTypesWithDBAdapter;
/// use reth_node_builder::{NodeBuilder, NodeConfig};
/// use reth_optimism_chainspec::OP_MAINNET;
/// use reth_optimism_exex::OpProofsExEx;
/// use reth_optimism_node::{OpNode, args::RollupArgs};
/// use reth_optimism_trie::{InMemoryProofsStorage, OpProofsStorage, db::MdbxProofsStorage};
/// use reth_provider::providers::BlockchainProvider;
/// use std::{sync::Arc, time::Duration};
///
/// let config = NodeConfig::new(OP_MAINNET.clone());
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
    Primitives: NodePrimitives,
    Primitives::Block: Clone,
    Storage: OpProofsStore + Clone + 'static,
{
    /// Main execution loop for the ExEx
    pub async fn run(mut self) -> eyre::Result<()> {
        self.ensure_initialized()?;

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
            if let Some(trie_data) = self.precomputed_trie_data(&new, block_number) {
                let SortedTrieData { hashed_state, trie_updates } = &trie_data.sorted;
                engine_handle.index_block(
                    block.block_with_parent(),
                    (**trie_updates).clone(),
                    (**hashed_state).clone(),
                )?;
            } else {
                engine_handle.execute_block(block)?;
            }
        }

        Ok(())
    }

    /// reth's trie data for `block_number`, unless the block has none or its verification is due,
    /// in which case the engine executes the block instead.
    fn precomputed_trie_data<'chain>(
        &self,
        chain: &'chain Chain<Primitives>,
        block_number: u64,
    ) -> Option<&'chain ComputedTrieData> {
        let verification_due = self.verification_interval > 0 &&
            block_number.is_multiple_of(self.verification_interval);
        (!verification_due)
            .then(|| chain.trie_data_at(block_number))
            .flatten()
            .map(LazyTrieData::get)
    }

    /// Switches the store to `new`: the engine unwinds to the fork block and takes the new
    /// blocks in the way [`Self::handle_chain_committed`] would, indexing reth's trie data or
    /// executing the block. reth notifies a reorg with both chains on one fork block; two
    /// different fork blocks mean the notification is not a reorg the store can follow, so it
    /// fails before the engine touches anything.
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

        let blocks = new
            .blocks()
            .iter()
            .map(|(&block_number, block)| {
                self.precomputed_trie_data(&new, block_number).map_or_else(
                    || ReorgBlock::Execute(Arc::clone(block)),
                    |trie_data| ReorgBlock::Index {
                        block: block.block_with_parent(),
                        hashed_state: trie_data.sorted.hashed_state.clone(),
                        trie_updates: trie_data.sorted.trie_updates.clone(),
                    },
                )
            })
            .collect();

        engine_handle.reorg(blocks)?;

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
    use alloy_consensus::private::alloy_primitives::B256;
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use reth_chain_state::{ExecutedBlock, NewCanonicalChain};
    use reth_chainspec::{ChainSpec, ChainSpecBuilder, MAINNET};
    use reth_db::test_utils::tempdir_path;
    use reth_ethereum_primitives::{Block, Receipt};
    use reth_evm_ethereum::EthEvmConfig;
    use reth_execution_types::{Chain, ExecutionOutcome};
    use reth_exex_test_utils::{Adapter, TestExExHandle, test_exex_context_with_chain_spec};
    use reth_optimism_trie::{
        BlockStateDiff, InMemoryProofsStorage, OpProofsInitProvider, OpProofsProviderRO,
        OpProofsProviderRw, OpProofsStore, db::MdbxProofsStorage, engine::EngineHandle,
    };
    use reth_primitives_traits::{RecoveredBlock, SealedHeader};
    use reth_trie::{
        ComputedTrieData, HashedPostStateSorted, LazyTrieData, updates::TrieUpdatesSorted,
    };
    use rstest::rstest;
    use std::{collections::BTreeMap, default::Default, ops::RangeInclusive, sync::Arc};

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

        let execution_outcome: ExecutionOutcome<Receipt> = ExecutionOutcome {
            bundle: Default::default(),
            receipts: Vec::new(),
            requests: Vec::new(),
            first_block: from,
        };

        Chain::new(blocks, execution_outcome, trie_data)
    }

    // Bootstrap storage to the genesis block via the init flow (sets earliest = latest = genesis).
    fn init_storage<S: OpProofsStore>(storage: S) {
        let genesis = NumHash::new(0, b256(0x00));
        let init = storage.initialization_provider().expect("init");
        init.set_initial_state_anchor(genesis).expect("anchor");
        init.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(init).expect("commit");
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

    /// Which store implementation a case runs over.
    #[derive(Clone, Copy, Debug)]
    enum Backend {
        InMemory,
        Mdbx,
    }

    /// A mainnet-shaped chain past the merge from genesis, so that an empty block changes no
    /// state when executed: no block reward, no withdrawals and no beacon root.
    fn merged_chain_spec() -> Arc<ChainSpec> {
        Arc::new(
            ChainSpecBuilder::default()
                .chain(MAINNET.chain)
                .genesis(MAINNET.genesis.clone())
                .paris_activated()
                .build(),
        )
    }

    /// An empty block `number` on `parent`, with a hash only `tag` and `number` decide. Its
    /// header carries the empty state root, which is what executing it on [`merged_chain_spec`]
    /// over a store initialized without trie data computes.
    fn empty_block(number: u64, parent: B256, tag: u8) -> RecoveredBlock<Block> {
        let mut block: RecoveredBlock<Block> = Default::default();
        block.set_block_number(number);
        block.set_parent_hash(parent);
        block.set_hash(B256::new([[tag; 16], [number as u8; 16]].concat().try_into().unwrap()));
        block
    }

    /// Empty blocks `numbers` chained on `parent`, which reth's canonical in-memory state takes
    /// in as `update` so that the engine finds their parents' state when it executes them.
    fn canonical_empty_chain(
        ctx: &ExExContext<Adapter>,
        parent: BlockNumHash,
        numbers: RangeInclusive<u64>,
        tag: u8,
        update: impl FnOnce(Vec<ExecutedBlock>) -> NewCanonicalChain,
    ) -> Vec<RecoveredBlock<Block>> {
        let mut blocks: Vec<RecoveredBlock<Block>> = Vec::new();
        let mut parent_hash = parent.hash;
        for number in numbers {
            let block = empty_block(number, parent_hash, tag);
            parent_hash = block.hash();
            blocks.push(block);
        }
        let executed = blocks
            .iter()
            .map(|block| ExecutedBlock {
                recovered_block: Arc::new(block.clone()),
                ..Default::default()
            })
            .collect();
        let tip = blocks.last().expect("a chain has a tip");
        let in_memory = ctx.components.provider.canonical_in_memory_state();
        in_memory.update_chain(update(executed));
        in_memory.set_canonical_head(SealedHeader::new(tip.header().clone(), tip.hash()));
        blocks
    }

    /// A chain of `blocks`, with empty trie data for each when `with_trie_data`.
    fn chain_of(
        blocks: Vec<RecoveredBlock<Block>>,
        with_trie_data: bool,
    ) -> Chain<reth_ethereum_primitives::EthPrimitives> {
        let first_block = blocks.first().expect("a chain has a first block").number();
        let trie_data = blocks
            .iter()
            .filter(|_| with_trie_data)
            .map(|block| (block.number(), LazyTrieData::ready(ComputedTrieData::default())))
            .collect();
        let execution_outcome: ExecutionOutcome<Receipt> = ExecutionOutcome {
            bundle: Default::default(),
            receipts: Vec::new(),
            requests: Vec::new(),
            first_block,
        };
        Chain::new(blocks, execution_outcome, trie_data)
    }

    fn latest_block<S: OpProofsStore>(store: &S) -> NumHash {
        store.provider_ro().expect("provider ro").get_latest_block().expect("latest block")
    }

    /// Anchors `store` at the test context's genesis and spawns an engine flushing every block,
    /// executing with a real EVM over `chain_spec` since the context's EVM refuses every call.
    fn engine_over<S: OpProofsStore + Clone + 'static>(
        ctx: &ExExContext<Adapter>,
        handle: &TestExExHandle,
        chain_spec: Arc<ChainSpec>,
        store: S,
    ) -> EngineHandle<Block> {
        let init = store.initialization_provider().expect("init");
        init.set_initial_state_anchor(handle.genesis.num_hash()).expect("anchor");
        init.commit_initial_state().expect("commit init");
        OpProofsInitProvider::commit(init).expect("commit");
        let pruner = OpProofStoragePruner::new(store.clone(), ctx.components.provider.clone(), 20);
        EngineHandle::spawn_with_thresholds(
            EthEvmConfig::ethereum(chain_spec),
            ctx.components.provider.clone(),
            store,
            pruner,
            1,
            2,
        )
    }

    /// Commits empty blocks 1 through 3 on genesis, with trie data, through the ExEx.
    fn commit_three_blocks<S: OpProofsStore + Clone + 'static>(
        exex: &OpProofsExEx<Adapter, S>,
        handle: &mut TestExExHandle,
        engine: &EngineHandle<Block>,
    ) -> Vec<RecoveredBlock<Block>> {
        let blocks =
            canonical_empty_chain(&exex.ctx, handle.genesis.num_hash(), 1..=3, 0x10, |new| {
                NewCanonicalChain::Commit { new }
            });
        let notification =
            ExExNotification::ChainCommitted { new: Arc::new(chain_of(blocks.clone(), true)) };
        exex.handle_notification(notification, engine).expect("handle chain commit");
        handle.assert_event_finished_height(blocks[2].num_hash()).expect("finished height");
        blocks
    }

    async fn run_reorg_without_trie_data_case<S: OpProofsStore + Clone + 'static>(store: S) {
        let chain_spec = merged_chain_spec();
        let (ctx, mut handle) =
            test_exex_context_with_chain_spec(chain_spec.clone()).await.expect("exex test context");
        let engine = engine_over(&ctx, &handle, chain_spec, store.clone());
        let exex = build_test_exex(ctx, store.clone());
        let old = commit_three_blocks(&exex, &mut handle, &engine);
        // reth switches from blocks 2 and 3 to blocks 2' through 4' on block 1.
        let old_from_fork = old[1..].to_vec();
        let new = canonical_empty_chain(&exex.ctx, old[0].num_hash(), 2..=4, 0x20, |new| {
            NewCanonicalChain::Reorg {
                new,
                old: old_from_fork
                    .iter()
                    .map(|block| ExecutedBlock {
                        recovered_block: Arc::new(block.clone()),
                        ..Default::default()
                    })
                    .collect(),
            }
        });
        let new_tip = new[2].num_hash();
        let notification = ExExNotification::ChainReorged {
            old: Arc::new(chain_of(old_from_fork, true)),
            new: Arc::new(chain_of(new, false)),
        };

        exex.handle_notification(notification, &engine)
            .expect("a reorg without trie data executes the new chain");

        engine.flush();
        assert_eq!(latest_block(&store), new_tip);
        handle.assert_event_finished_height(new_tip).expect("finished height");
    }

    /// A reorg whose new chain carries no trie data executes the new blocks, as a commit without
    /// trie data does, instead of failing.
    #[rstest]
    #[tokio::test]
    async fn chain_reorged_without_trie_data_executes_the_new_chain(
        #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    ) {
        match backend {
            Backend::InMemory => {
                run_reorg_without_trie_data_case(InMemoryProofsStorage::new()).await
            }
            Backend::Mdbx => {
                let dir = tempdir_path();
                run_reorg_without_trie_data_case(Arc::new(
                    MdbxProofsStorage::new(dir.as_path()).expect("env"),
                ))
                .await
            }
        }
    }

    /// A reorg whose old and new chains do not share a fork block fails before the engine
    /// touches the store, and acknowledges no height.
    #[tokio::test]
    async fn chain_reorged_with_different_fork_blocks_fails_and_changes_nothing() {
        let store = InMemoryProofsStorage::new();
        let chain_spec = merged_chain_spec();
        let (ctx, mut handle) =
            test_exex_context_with_chain_spec(chain_spec.clone()).await.expect("exex test context");
        let engine = engine_over(&ctx, &handle, chain_spec, store.clone());
        let exex = build_test_exex(ctx, store.clone());
        let old = commit_three_blocks(&exex, &mut handle, &engine);
        let stored_tip = old[2].num_hash();
        // The old chain forks at block 1, the new chain at block 2.
        let mut new = Vec::new();
        let mut parent_hash = old[1].hash();
        for number in 3..=4 {
            let block = empty_block(number, parent_hash, 0x20);
            parent_hash = block.hash();
            new.push(block);
        }
        let notification = ExExNotification::ChainReorged {
            old: Arc::new(chain_of(old[1..].to_vec(), true)),
            new: Arc::new(chain_of(new, true)),
        };

        let err = exex
            .handle_notification(notification, &engine)
            .expect_err("the chains have different fork blocks");

        assert!(err.to_string().starts_with("Fork blocks do not match"), "got {err}");
        engine.flush();
        assert_eq!(latest_block(&store), stored_tip);
        handle.assert_events_empty();
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
