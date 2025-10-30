//! Live trie collector for external proofs storage.

use crate::{
    api::OperationDurations, provider::OpProofsStateProviderRef, BlockStateDiff, OpProofsStorage,
    OpProofsStorageError, OpProofsStore,
};
use alloy_eips::{eip1898::BlockWithParent, NumHash};
use derive_more::Constructor;
use reth_evm::{execute::Executor, ConfigureEvm};
use reth_primitives_traits::{AlloyBlockHeader, BlockTy, RecoveredBlock};
use reth_provider::{
    DatabaseProviderFactory, HashedPostStateProvider, StateProviderFactory, StateReader,
    StateRootProvider,
};
use reth_revm::database::StateProviderDatabase;
use std::time::Instant;
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
    ) -> eyre::Result<()> {
        let mut operation_durations = OperationDurations::default();

        let start = Instant::now();
        // ensure that we have the state of the parent block
        let (Some((earliest, _)), Some((latest, _))) = (
            self.storage.get_earliest_block_number().await?,
            self.storage.get_latest_block_number().await?,
        ) else {
            return Err(OpProofsStorageError::NoBlocksFound.into());
        };

        let parent_block_number = block.number() - 1;
        if parent_block_number < earliest {
            return Err(OpProofsStorageError::UnknownParent.into());
        }

        if parent_block_number > latest {
            return Err(OpProofsStorageError::MissingParentBlock {
                block_number: block.number(),
                parent_block_number,
                latest_block_number: latest,
            }
            .into());
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

        let execution_result =
            block_executor.execute(&(*block).clone()).map_err(|err| eyre::eyre!(err))?;

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
            }
            .into());
        }

        let update_result = self
            .storage
            .store_trie_updates(
                block_ref,
                BlockStateDiff { trie_updates, post_state: hashed_state },
            )
            .await?;

        operation_durations.total_duration_seconds = start.elapsed();
        operation_durations.write_duration_seconds = operation_durations.total_duration_seconds -
            operation_durations.state_root_duration_seconds;

        #[cfg(feature = "metrics")]
        {
            let block_metrics = self.storage.metrics().block_metrics();
            block_metrics.record_operation_durations(&operation_durations);
            block_metrics.increment_write_counts(&update_result);
        }

        info!(
            block_number = block.number(),
            ?operation_durations,
            "Trie updates stored successfully",
        );

        Ok(())
    }
}
