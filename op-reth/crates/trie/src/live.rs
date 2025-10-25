//! Live trie collector for external proofs storage.

use crate::{
    api::{BlockStateDiff, OpProofsStorageError, OpProofsStore},
    provider::OpProofsStateProviderRef,
    OpProofsStorage,
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
        let start = Instant::now();
        // ensure that we have the state of the parent block
        let (Some((earliest, _)), Some((latest, _))) = (
            self.storage.get_earliest_block_number().await?,
            self.storage.get_latest_block_number().await?,
        ) else {
            return Err(OpProofsStorageError::NoBlocksFound.into());
        };

        let fetch_block_duration = start.elapsed();

        let parent_block_number = block.number() - 1;
        if parent_block_number < earliest {
            return Err(OpProofsStorageError::UnknownParent.into());
        }

        if parent_block_number > latest {
            return Err(OpProofsStorageError::BlockUpdateFailed(
                block.number(),
                parent_block_number,
                latest,
            )
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

        let init_provider_duration = start.elapsed() - fetch_block_duration;

        let db = StateProviderDatabase::new(&state_provider);
        let block_executor = self.evm_config.batch_executor(db);

        let execution_result =
            block_executor.execute(&(*block).clone()).map_err(|err| eyre::eyre!(err))?;

        let execute_block_duration = start.elapsed() - init_provider_duration;

        let hashed_state = state_provider.hashed_post_state(&execution_result.state);
        let (state_root, trie_updates) =
            state_provider.state_root_with_updates(hashed_state.clone())?;

        let calculate_state_root_duration = start.elapsed() - execute_block_duration;

        if state_root != block.state_root() {
            return Err(OpProofsStorageError::StateRootMismatch(
                block.number(),
                state_root,
                block.state_root(),
            )
            .into());
        }

        self.storage
            .store_trie_updates(
                block_ref,
                BlockStateDiff { trie_updates, post_state: hashed_state },
            )
            .await?;

        let write_trie_updates_duration = start.elapsed() - calculate_state_root_duration;
        let execute_and_store_total_duration = start.elapsed();

        info!(
            block_number = block.number(),
            ?execute_and_store_total_duration,
            ?fetch_block_duration,
            ?init_provider_duration,
            ?execute_block_duration,
            ?calculate_state_root_duration,
            ?write_trie_updates_duration,
            "Stored trie updates",
        );

        Ok(())
    }
}
