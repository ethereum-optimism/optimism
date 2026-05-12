use super::super::state::EngineState;
use crate::{
    BlockStateDiff, OpProofsStore, engine::EngineError, provider::OpProofsStateProviderRef,
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use crossbeam_channel::Sender;
use reth_evm::{ConfigureEvm, execute::Executor};
use reth_primitives_traits::{AlloyBlockHeader, NodePrimitives, RecoveredBlock};
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, HashedPostStateProvider,
    StateProviderFactory, StateReader, StateRootProvider,
};
use reth_revm::database::StateProviderDatabase;
use std::time::Instant;
use tracing::{debug, info};

pub(crate) struct ExecuteBlockTask<Block: reth_primitives_traits::Block> {
    pub(crate) block: RecoveredBlock<Block>,
    pub(crate) reply: Sender<Result<(), EngineError>>,
}

impl<Block: reth_primitives_traits::Block> ExecuteBlockTask<Block> {
    pub(crate) fn execute<Evm, Provider, Store>(self, state: &mut EngineState<Evm, Provider, Store>)
    where
        Evm: ConfigureEvm<Primitives: NodePrimitives<Block = Block>>,
        Provider: BlockHashReader
            + StateReader
            + DatabaseProviderFactory
            + StateProviderFactory
            + BlockReader<Block = Block>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        let result = run(&self.block, state);
        let _ = self.reply.send(result);
    }
}

pub(crate) fn run<Block, Evm, Provider, Store>(
    block: &RecoveredBlock<Block>,
    state: &mut EngineState<Evm, Provider, Store>,
) -> Result<(), EngineError>
where
    Block: reth_primitives_traits::Block,
    Evm: ConfigureEvm<Primitives: NodePrimitives<Block = Block>>,
    Provider: BlockHashReader
        + StateReader
        + DatabaseProviderFactory
        + StateProviderFactory
        + BlockReader<Block = Block>
        + Clone
        + 'static,
    Store: OpProofsStore + Clone + 'static,
{
    let start = Instant::now();
    let tip = state.get_tip()?;
    let parent_block_number = block.number().saturating_sub(1);

    if block.number() <= tip.number {
        debug!(
            block_number = block.number(),
            tip_number = tip.number,
            "Block already covered by tip, skipping execute_and_store",
        );
        return Ok(());
    }

    if block.number() > tip.number.saturating_add(1) {
        debug!(
            block_number = block.number(),
            tip_number = tip.number,
            "Gap detected, updating sync target",
        );
        state.update_sync_target(block.number());
        return Ok(());
    }

    if block.parent_hash() != tip.hash {
        return Err(EngineError::ParentHashMismatch {
            block_number: block.number(),
            expected_parent_hash: tip.hash,
            actual_parent_hash: block.parent_hash(),
        });
    }

    let block_ref =
        BlockWithParent::new(block.parent_hash(), NumHash::new(block.number(), block.hash()));

    let inner_provider = OpProofsStateProviderRef::new(
        state.provider.state_by_block_hash(block.parent_hash())?,
        state.storage.provider_ro()?,
        parent_block_number,
    );
    let state_provider = state.memory.state_provider(block.parent_hash(), inner_provider);

    let db = StateProviderDatabase::new(&state_provider);
    let block_executor = state.evm_config.batch_executor(db);
    let execution_result = block_executor.execute(block)?;
    let execution_duration = start.elapsed();

    let hashed_state = state_provider.hashed_post_state(&execution_result.state);
    let (state_root, trie_updates) =
        state_provider.state_root_with_updates(hashed_state.clone())?;
    let state_root_duration = start.elapsed() - execution_duration;

    if state_root != block.state_root() {
        return Err(EngineError::StateRootMismatch {
            block_number: block.number(),
            current_state_hash: state_root,
            expected_state_hash: block.state_root(),
        });
    }

    let sorted_trie_updates = trie_updates.into_sorted();
    let sorted_post_state = hashed_state.into_sorted();

    state.memory.insert(block_ref, BlockStateDiff { sorted_trie_updates, sorted_post_state });

    let total_duration = start.elapsed();

    #[cfg(feature = "metrics")]
    {
        state.metrics.execute_block_duration_seconds.record(total_duration);
        state.metrics.execution_duration_seconds.record(execution_duration);
        state.metrics.state_root_duration_seconds.record(state_root_duration);
    }

    info!(
        block_number = block.number(),
        ?total_duration,
        ?execution_duration,
        ?state_root_duration,
        "Block executed and trie updates buffered successfully",
    );

    Ok(())
}
