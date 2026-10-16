//! [`EngineState`] — all mutable engine state in one place.

#[cfg(feature = "metrics")]
use super::metrics::EngineMetrics;
use super::{
    DEFAULT_PERSISTENCE_TIMEOUT_SECS,
    buffer::state::TrieBufferState,
    error::EngineError,
    persistence::{PersistenceHandle, error::PersistenceError},
};
use crate::{OpProofStoragePruner, OpProofsProviderRO, OpProofsStorageError, OpProofsStore};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use crossbeam_channel::{Receiver, RecvError, RecvTimeoutError, bounded};
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
};
use std::time::Duration;
#[cfg(feature = "metrics")]
use std::time::Instant;
use tracing::{error, info, warn};

/// Tracks all in-flight state for background persistence.
pub(crate) struct PersistenceState {
    /// Handle to the persistence service.
    handle: PersistenceHandle,
    /// Reply channel for the in-flight save. Present only while a save is running.
    ///
    /// Exposed `pub(crate)` so the engine loop can include it in a `select!`.
    pub(crate) in_flight: Option<Receiver<Result<Option<u64>, PersistenceError>>>,
}

impl PersistenceState {
    const fn new(handle: PersistenceHandle) -> Self {
        Self { handle, in_flight: None }
    }

    /// Blocking: wait for the in-flight save to finish and prune `memory`.
    ///
    /// Used during shutdown and before unwind to quiesce the persistence layer.
    pub(crate) fn wait(&mut self, memory: &TrieBufferState) {
        let Some(rx) = self.in_flight.take() else { return };

        match rx.recv_timeout(Duration::from_secs(DEFAULT_PERSISTENCE_TIMEOUT_SECS)) {
            Ok(Ok(Some(last_persisted))) => {
                info!(
                    target: "trie::engine::state",
                    block_number = last_persisted,
                    "Persistence completed (waited), pruning memory"
                );
                memory.prune(last_persisted + 1);
            }
            Ok(Ok(None)) => {}
            Ok(Err(e)) => {
                error!(target: "trie::engine::state", ?e, "Persistence save failed while waiting");
            }
            Err(RecvTimeoutError::Timeout) => {
                error!(target: "trie::engine::state", "Persistence timeout while waiting");
            }
            Err(RecvTimeoutError::Disconnected) => {
                error!(target: "trie::engine::state", "Persistence service disconnected while waiting");
            }
        }
    }

    /// Handle a completed background save received via `select!`: clear `in_flight` and prune
    /// memory.
    pub(crate) fn on_complete(
        &mut self,
        result: Result<Result<Option<u64>, PersistenceError>, RecvError>,
        memory: &TrieBufferState,
    ) {
        self.in_flight = None;

        match result {
            Ok(Ok(Some(last_persisted))) => {
                info!(
                    target: "trie::engine::state",
                    block_number = last_persisted,
                    "Background persistence completed, pruning memory"
                );
                memory.prune(last_persisted + 1);
            }
            Ok(Ok(None)) => {}
            Ok(Err(e)) => {
                error!(target: "trie::engine::state", ?e, "Background persistence save failed");
            }
            Err(_) => {
                error!(target: "trie::engine::state", "Persistence service disconnected unexpectedly");
            }
        }
    }

    /// Start a background save if no save is already running.
    ///
    /// The caller is responsible for checking the threshold before calling this.
    /// Completion is handled reactively by the engine loop via `select!` on [`Self::in_flight`].
    pub(crate) fn advance_persistence(
        &mut self,
        memory: &TrieBufferState,
    ) -> Result<(), EngineError> {
        if self.in_flight.is_some() {
            return Ok(());
        }

        let blocks = memory.blocks_ordered();
        if blocks.is_empty() {
            return Ok(());
        }

        info!(
            target: "trie::engine::state",
            count = blocks.len(),
            start_block = blocks.first().map(|arc| arc.0.block.number),
            end_block = blocks.last().map(|arc| arc.0.block.number),
            "Persistence threshold reached: sending to persistence service"
        );

        let (tx, rx) = bounded(1);
        self.handle.save_updates(blocks, tx)?;
        self.in_flight = Some(rx);

        Ok(())
    }

    /// Send an unwind to the persistence service and block until it completes. The caller
    /// drains any in-flight save first.
    pub(crate) fn unwind(&self, to: BlockWithParent) -> Result<(), EngineError> {
        let (tx, rx) = bounded(1);
        self.handle.unwind(to, tx)?;

        match rx.recv_timeout(Duration::from_secs(DEFAULT_PERSISTENCE_TIMEOUT_SECS)) {
            Ok(Ok(())) => Ok(()),
            Ok(Err(e)) => Err(e.into()),
            Err(RecvTimeoutError::Timeout) => Err(EngineError::PersistenceTimeout),
            Err(RecvTimeoutError::Disconnected) => Err(EngineError::PersistenceDisconnected),
        }
    }
}

/// All mutable state owned by the engine.
pub(crate) struct EngineState<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: StateReader + DatabaseProviderFactory + StateProviderFactory + BlockReader,
{
    /// The highest block number the engine should sync to between processing actions.
    pub(crate) sync_target: u64,

    pub(crate) evm_config: Evm,
    pub(crate) provider: Provider,
    pub(crate) storage: Store,

    pub(crate) memory: TrieBufferState,
    pub(crate) persistence: PersistenceState,

    #[cfg(feature = "metrics")]
    pub(crate) metrics: EngineMetrics,
}

impl<Evm, Provider, Store> EngineState<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: BlockHashReader
        + StateReader
        + DatabaseProviderFactory
        + StateProviderFactory
        + BlockReader<Block = BlockTy<Evm::Primitives>>
        + Clone
        + 'static,
    Store: OpProofsStore + Clone + 'static,
{
    pub(crate) fn new(
        evm_config: Evm,
        provider: Provider,
        storage: Store,
        pruner: OpProofStoragePruner<Store, Provider>,
    ) -> Self {
        let persistence_handle = PersistenceHandle::spawn(pruner, storage.clone());
        Self {
            evm_config,
            provider,
            storage,
            memory: TrieBufferState::new(),
            persistence: PersistenceState::new(persistence_handle),
            sync_target: 0,
            #[cfg(feature = "metrics")]
            metrics: EngineMetrics::new_with_labels(&[] as &[(&str, &str)]),
        }
    }

    /// Start a background save if no save is already running.
    ///
    /// The caller is responsible for checking the persistence threshold before
    /// calling this.
    pub(crate) fn advance_persistence(&mut self) -> Result<(), EngineError> {
        self.persistence.advance_persistence(&self.memory)
    }

    /// Block until any in-flight background save finishes and the memory buffer is pruned.
    pub(crate) fn drain_persistence(&mut self) {
        self.persistence.wait(&self.memory);
    }

    /// Drain any in-flight save, check `to` against the proof window and the stored block below
    /// it, unwind the persistence service to `to`, then unwind the in-memory buffer to match.
    pub(crate) fn unwind(&mut self, to: BlockWithParent) -> Result<(), EngineError> {
        #[cfg(feature = "metrics")]
        let start = Instant::now();
        // A save in flight can prune the window, so the checks run once it is done.
        self.drain_persistence();
        self.check_unwind_target(to)?;
        self.persistence.unwind(to)?;
        self.memory.unwind(to.block.number);
        // Clamp `sync_target` to the post-unwind tip. Without this, a stale target left over
        // from before a deep FCU rewind keeps `needs_sync()` perpetually true and spins the
        // runner on `BlockNotFound` for blocks the provider no longer has.
        self.sync_target = self.sync_target.min(to.block.number.saturating_sub(1));
        #[cfg(feature = "metrics")]
        self.metrics.unwind_duration_seconds.record(start.elapsed());
        Ok(())
    }

    /// Fails an unwind that would remove the window's earliest block, or whose target does not
    /// build on the stored block below it, before anything is unwound.
    ///
    /// Only the blocks this engine has buffered and the window's ends have a known hash: a block
    /// persisted before a restart has none, so an unwind onto it goes unverified.
    fn check_unwind_target(&self, to: BlockWithParent) -> Result<(), EngineError> {
        let window = self.storage.provider_ro()?.get_proof_window()?;
        if to.block.number <= window.earliest.number {
            warn!(
                target: "trie::engine::state",
                to_block = to.block.number,
                earliest = window.earliest.number,
                latest = window.latest.number,
                "Refusing to unwind at or below the earliest block"
            );
            return Err(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number: to.block.number,
                earliest_block_number: window.earliest.number,
            }
            .into());
        }
        let fork_number = to.block.number - 1;
        let stored_fork_hash = self.memory.hash_at(fork_number).or_else(|| {
            [window.latest, window.earliest]
                .into_iter()
                .find(|block| block.number == fork_number)
                .map(|block| block.hash)
        });
        match stored_fork_hash {
            Some(stored) if stored != to.parent => {
                warn!(
                    target: "trie::engine::state",
                    to_block = to.block.number,
                    stored_parent = ?stored,
                    target_parent = ?to.parent,
                    "Refusing to unwind onto a block the target does not build on"
                );
                Err(EngineError::ParentHashMismatch {
                    block_number: to.block.number,
                    expected_parent_hash: stored,
                    actual_parent_hash: to.parent,
                })
            }
            _ => Ok(()),
        }
    }

    /// Advances `sync_target` to `block_number` if it is higher than the current target.
    pub(crate) const fn update_sync_target(&mut self, block_number: u64) {
        if block_number > self.sync_target {
            self.sync_target = block_number;
        }
    }

    /// Returns the current tip: the in-memory tip if present, otherwise the latest persisted block.
    pub(crate) fn get_tip(&self) -> Result<NumHash, EngineError> {
        if let Some(tip) = self.memory.tip() {
            return Ok(tip);
        }

        Ok(self.storage.provider_ro()?.get_latest_block()?)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{OpProofsInitProvider, db::MdbxProofsStorage};
    use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
    use alloy_primitives::B256;
    use reth_chainspec::MAINNET;
    use reth_db_common::init::init_genesis;
    use reth_evm_ethereum::EthEvmConfig;
    use reth_provider::{
        providers::BlockchainProvider,
        test_utils::{MockNodeTypesWithDB, create_test_provider_factory_with_chain_spec},
    };
    use std::sync::Arc;
    use tempfile::TempDir;

    type TestEngineState =
        EngineState<EthEvmConfig, BlockchainProvider<MockNodeTypesWithDB>, Arc<MdbxProofsStorage>>;

    fn bootstrap_storage() -> Arc<MdbxProofsStorage> {
        let dir = TempDir::new().unwrap().keep();
        let store = Arc::new(MdbxProofsStorage::new(&dir).unwrap());
        let init = store.initialization_provider().expect("init provider");
        init.set_initial_state_anchor(BlockNumHash { number: 0, hash: B256::ZERO })
            .expect("set anchor");
        init.commit_initial_state().expect("commit initial state");
        OpProofsInitProvider::commit(init).expect("commit tx");
        store
    }

    fn make_engine_state() -> TestEngineState {
        let chain_spec = MAINNET.clone();
        let factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
        init_genesis(&factory).expect("init genesis");
        let blockchain_db = BlockchainProvider::new(factory).expect("blockchain provider");
        let storage = bootstrap_storage();
        let pruner = OpProofStoragePruner::new(storage.clone(), blockchain_db.clone(), 1000);
        let evm_config = EthEvmConfig::ethereum(chain_spec);
        EngineState::new(evm_config, blockchain_db, storage, pruner)
    }

    /// An unwind target whose parent is unknown to the engine: the store holds only the anchor,
    /// so nothing checks the parent of a block far above it.
    fn unwind_target(block_number: u64) -> BlockWithParent {
        BlockWithParent::new(
            B256::repeat_byte(0xAA),
            NumHash::new(block_number, B256::repeat_byte(0x50)),
        )
    }

    /// An unwind target right above the anchor, whose parent the anchor's hash verifies.
    fn unwind_target_above_anchor(parent: B256) -> BlockWithParent {
        BlockWithParent::new(parent, NumHash::new(1, B256::repeat_byte(0x50)))
    }

    #[test]
    fn unwind_clamps_stale_sync_target_down_to_new_tip() {
        let mut state = make_engine_state();
        state.sync_target = 100;
        state.unwind(unwind_target(50)).expect("unwind");
        assert_eq!(state.sync_target, 49);
    }

    #[test]
    fn unwind_leaves_sync_target_alone_when_at_new_tip() {
        // sync_target already at the new tip: idempotent.
        let mut state = make_engine_state();
        state.sync_target = 49;
        state.unwind(unwind_target(50)).expect("unwind");
        assert_eq!(state.sync_target, 49);
    }

    #[test]
    fn unwind_leaves_sync_target_alone_when_below_new_tip() {
        // Engine was behind both before and after the unwind. Don't bump it down further.
        let mut state = make_engine_state();
        state.sync_target = 10;
        state.unwind(unwind_target(50)).expect("unwind");
        assert_eq!(state.sync_target, 10);
    }

    #[test]
    fn unwind_to_block_one_clamps_target_to_zero_without_underflow() {
        // The original bug scenario at its sharpest: deep rewind all the way back.
        // `saturating_sub(1)` must not underflow when unwinding to block 1.
        let mut state = make_engine_state();
        state.sync_target = 1_000_000;
        state.unwind(unwind_target_above_anchor(B256::ZERO)).expect("unwind");
        assert_eq!(state.sync_target, 0);
    }

    #[test]
    fn unwind_at_the_anchor_fails_before_touching_anything() {
        let mut state = make_engine_state();
        state.sync_target = 100;
        let anchor = BlockWithParent::new(B256::ZERO, NumHash::new(0, B256::ZERO));
        let err = state.unwind(anchor).expect_err("the anchor is the window's earliest block");
        assert!(
            matches!(
                err,
                EngineError::Storage(OpProofsStorageError::UnwindBeyondEarliest {
                    unwind_block_number: 0,
                    earliest_block_number: 0,
                })
            ),
            "got {err:?}"
        );
        assert_eq!(state.sync_target, 100);
    }

    #[test]
    fn unwind_onto_a_parent_the_window_end_contradicts_fails() {
        let mut state = make_engine_state();
        let err = state
            .unwind(unwind_target_above_anchor(B256::repeat_byte(0xAA)))
            .expect_err("the anchor is the stored block below the target");
        assert!(
            matches!(
                err,
                EngineError::ParentHashMismatch { block_number: 1, expected_parent_hash, .. }
                    if expected_parent_hash == B256::ZERO
            ),
            "got {err:?}"
        );
    }
}
