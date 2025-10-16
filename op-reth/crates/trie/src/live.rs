//! Live trie collector for external proofs storage.

use crate::{
    api::{BlockStateDiff, OpProofsStorage},
    provider::OpProofsStateProviderRef,
};
use reth_evm::{execute::Executor, ConfigureEvm};
use reth_node_api::{FullNodeComponents, NodePrimitives, NodeTypes};
use reth_primitives_traits::{AlloyBlockHeader, RecoveredBlock};
use reth_provider::{
    DatabaseProviderFactory, HashedPostStateProvider, StateProviderFactory, StateReader,
    StateRootProvider,
};
use reth_revm::database::StateProviderDatabase;
use std::time::Instant;
use tracing::debug;

/// Live trie collector for external proofs storage.
#[derive(Debug)]
pub struct LiveTrieCollector<Node, PreimageStore>
where
    Node: FullNodeComponents,
    Node::Provider: StateReader + DatabaseProviderFactory + StateProviderFactory,
{
    evm_config: Node::Evm,
    provider: Node::Provider,
    storage: PreimageStore,
}

impl<Node, Store, Primitives> LiveTrieCollector<Node, Store>
where
    Node: FullNodeComponents<Types: NodeTypes<Primitives = Primitives>>,
    Primitives: NodePrimitives,
    Store: OpProofsStorage + Clone + 'static,
{
    /// Create a new `LiveTrieCollector` instance
    pub const fn new(evm_config: Node::Evm, provider: Node::Provider, storage: Store) -> Self {
        Self { evm_config, provider, storage }
    }

    /// Execute a block and store the updates in the storage.
    pub async fn execute_and_store_block_updates(
        &self,
        block: &RecoveredBlock<Primitives::Block>,
    ) -> eyre::Result<()> {
        let start = Instant::now();
        // ensure that we have the state of the parent block
        let (Some((earliest, _)), Some((latest, _))) = (
            self.storage.get_earliest_block_number().await?,
            self.storage.get_latest_block_number().await?,
        ) else {
            return Err(eyre::eyre!("No blocks stored"));
        };

        let fetch_block_duration = start.elapsed();

        let parent_block_number = block.number() - 1;
        if parent_block_number < earliest {
            return Err(eyre::eyre!(
                "Parent block number is less than earliest stored block number"
            ));
        }

        if parent_block_number > latest {
            return Err(eyre::eyre!(
                "Cannot execute block updates for block {} without parent state {} (latest stored block number: {})",
                block.number(),
                parent_block_number,
                latest
            ));
        }

        let block_number = block.number();

        // TODO: should we check block hash here?

        let state_provider = OpProofsStateProviderRef::new(
            self.provider.state_by_block_hash(block.parent_hash())?,
            self.storage.clone(),
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
            return Err(eyre::eyre!(
                "State root mismatch for block {} (have: {}, expected: {})",
                block.number(),
                state_root,
                block.state_root()
            ));
        }

        self.storage
            .store_trie_updates(
                block_number,
                BlockStateDiff { trie_updates, post_state: hashed_state },
            )
            .await?;

        let write_trie_updates_duration = start.elapsed() - calculate_state_root_duration;

        debug!("execute_and_store_block_updates duration: {:?}", start.elapsed());
        debug!("- fetch_block_duration: {:?}", fetch_block_duration);
        debug!("- init_provider_duration: {:?}", init_provider_duration);
        debug!("- execute_block_duration: {:?}", execute_block_duration);
        debug!("- calculate_state_root_duration: {:?}", calculate_state_root_duration);
        debug!("- write_trie_updates_duration: {:?}", write_trie_updates_duration);

        Ok(())
    }
}
