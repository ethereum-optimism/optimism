//! Live trie collector for external proofs storage.

use crate::{
    api::OperationDurations, provider::OpProofsStateProviderRef, BlockStateDiff, OpProofsStorage,
    OpProofsStorageError, OpProofsStore,
};
use alloy_eips::{eip1898::BlockWithParent, NumHash};
use alloy_primitives::map::{DefaultHashBuilder, HashMap};
use derive_more::Constructor;
use reth_evm::{execute::Executor, ConfigureEvm};
use reth_primitives_traits::{AlloyBlockHeader, BlockTy, RecoveredBlock};
use reth_provider::{
    DatabaseProviderFactory, HashedPostStateProvider, StateProviderFactory, StateReader,
    StateRootProvider,
};
use reth_revm::database::StateProviderDatabase;
use reth_trie::{updates::TrieUpdatesSorted, HashedPostStateSorted};
use std::{sync::Arc, time::Instant};
use tracing::info;

/// Live trie collector for external proofs storage.
#[derive(Debug, Constructor)]
pub struct LiveTrieCollector<'tx, Evm, Provider, PreimageStore>
where
    Evm: ConfigureEvm,
    Provider: StateReader + DatabaseProviderFactory + StateProviderFactory,
{
    evm_config: Evm,
    provider: Provider,
    storage: &'tx OpProofsStorage<PreimageStore>,
}

impl<'tx, Evm, Provider, Store> LiveTrieCollector<'tx, Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: StateReader + DatabaseProviderFactory + StateProviderFactory,
    Store: 'tx + OpProofsStore + Clone + 'static,
{
    /// Execute a block and store the updates in the storage.
    pub async fn execute_and_store_block_updates(
        &self,
        block: &RecoveredBlock<BlockTy<Evm::Primitives>>,
    ) -> Result<(), OpProofsStorageError> {
        let mut operation_durations = OperationDurations::default();

        let start = Instant::now();
        // ensure that we have the state of the parent block
        let (Some((earliest, _)), Some((latest, _))) = (
            self.storage.get_earliest_block_number().await?,
            self.storage.get_latest_block_number().await?,
        ) else {
            return Err(OpProofsStorageError::NoBlocksFound);
        };

        let parent_block_number = block.number() - 1;
        if parent_block_number < earliest {
            return Err(OpProofsStorageError::UnknownParent);
        }

        if parent_block_number > latest {
            return Err(OpProofsStorageError::MissingParentBlock {
                block_number: block.number(),
                parent_block_number,
                latest_block_number: latest,
            });
        }

        let block_ref =
            BlockWithParent::new(block.parent_hash(), NumHash::new(block.number(), block.hash()));

        // TODO: should we check block hash here?

        let state_provider = OpProofsStateProviderRef::new(
            self.provider.state_by_block_hash(block.parent_hash())?,
            self.storage,
            parent_block_number,
        );

        let db = StateProviderDatabase::new(&state_provider);
        let block_executor = self.evm_config.batch_executor(db);

        let execution_result = block_executor.execute(&(*block).clone())?;

        operation_durations.execution_duration_seconds = start.elapsed();

        let hashed_state = state_provider.hashed_post_state(&execution_result.state);
        let (state_root, trie_updates) =
            state_provider.state_root_with_updates(hashed_state.clone())?;

        operation_durations.state_root_duration_seconds =
            start.elapsed() - operation_durations.execution_duration_seconds;

        if state_root != block.state_root() {
            return Err(OpProofsStorageError::StateRootMismatch {
                block_number: block.number(),
                current_state_hash: state_root,
                expected_state_hash: block.state_root(),
            });
        }

        let update_result = self
            .storage
            .store_trie_updates(
                block_ref,
                BlockStateDiff {
                    sorted_trie_updates: trie_updates.into_sorted(),
                    sorted_post_state: hashed_state.into_sorted(),
                },
            )
            .await?;

        operation_durations.total_duration_seconds = start.elapsed();
        operation_durations.write_duration_seconds = operation_durations.total_duration_seconds -
            operation_durations.state_root_duration_seconds -
            operation_durations.execution_duration_seconds;

        #[cfg(feature = "metrics")]
        {
            let block_metrics = self.storage.metrics().block_metrics();
            block_metrics.record_operation_durations(&operation_durations);
            block_metrics.increment_write_counts(&update_result);
        }

        info!(
            block_number = block.number(),
            ?operation_durations,
            ?update_result,
            "Block executed and trie updates stored successfully",
        );

        Ok(())
    }

    /// Store trie updates for a given block.
    pub async fn store_block_updates(
        &self,
        block: BlockWithParent,
        sorted_trie_updates: TrieUpdatesSorted,
        sorted_post_state: HashedPostStateSorted,
    ) -> Result<(), OpProofsStorageError> {
        let start = Instant::now();
        let mut operation_durations = OperationDurations::default();

        let storage_result = self
            .storage
            .store_trie_updates(block, BlockStateDiff { sorted_trie_updates, sorted_post_state })
            .await?;

        let write_duration = start.elapsed();
        operation_durations.total_duration_seconds = write_duration;
        operation_durations.write_duration_seconds = write_duration;

        #[cfg(feature = "metrics")]
        {
            let block_metrics = self.storage.metrics().block_metrics();
            block_metrics.record_operation_durations(&operation_durations);
            block_metrics.increment_write_counts(&storage_result);
        }

        info!(
            block_number = block.block.number,
            ?operation_durations,
            ?storage_result,
            "Trie updates stored successfully",
        );

        Ok(())
    }

    /// Handles chain reorganizations by replacing block updates after a common ancestor.
    ///
    /// This method removes all block updates after the latest common ancestor (the block before
    /// the first block in `new_blocks`) and replaces them with the updates from the provided new
    /// chain.
    ///
    /// # Arguments
    ///
    /// * `new_blocks` - A vector of references to `RecoveredBlock` instances representing the new
    ///   blocks to be added to the trie storage.
    pub async fn unwind_and_store_block_updates(
        &self,
        block_updates: Vec<(BlockWithParent, Arc<TrieUpdatesSorted>, Arc<HashedPostStateSorted>)>,
    ) -> eyre::Result<()> {
        if block_updates.is_empty() {
            return Ok(());
        }

        let start = Instant::now();
        let mut operation_durations = OperationDurations::default();
        let latest_common_block_number = block_updates[0].0.block.number.saturating_sub(1);

        let mut block_trie_updates: HashMap<BlockWithParent, BlockStateDiff> =
            HashMap::with_capacity_and_hasher(block_updates.len(), DefaultHashBuilder::default());

        for (block, trie_updates, hashed_state) in &block_updates {
            block_trie_updates.insert(
                *block,
                BlockStateDiff {
                    sorted_trie_updates: (**trie_updates).clone(),
                    sorted_post_state: (**hashed_state).clone(),
                },
            );
        }

        self.storage.replace_updates(latest_common_block_number, block_trie_updates).await?;
        let write_duration = start.elapsed();
        operation_durations.total_duration_seconds = write_duration;
        operation_durations.write_duration_seconds = write_duration;

        #[cfg(feature = "metrics")]
        {
            let block_metrics = self.storage.metrics().block_metrics();
            block_metrics.record_operation_durations(&operation_durations);
        }

        info!(
            start_block_number = block_updates.first().map(|(b, _, _)| b.block.number),
            end_block_number = block_updates.last().map(|(b, _, _)| b.block.number),
            ?operation_durations,
            "Trie updates rewound and stored successfully",
        );
        Ok(())
    }

    /// Remove account, storage and trie updates from historical storage for all blocks from
    /// the specified block (inclusive).
    pub async fn unwind_history(&self, to: BlockWithParent) -> Result<(), OpProofsStorageError> {
        self.storage.unwind_history(to).await
    }
}
