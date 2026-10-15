use super::super::state::EngineState;
use crate::{BlockStateDiff, OpProofsStore, engine::EngineError};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::Sender;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::{NodePrimitives, RecoveredBlock};
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use std::{sync::Arc, time::Instant};
use tracing::{debug, info};

/// A block of the chain a reorg switches to, with the way the engine takes it in.
#[derive(Debug)]
pub enum ReorgBlock<Block: reth_primitives_traits::Block> {
    /// Execute the block through the EVM, as `execute_block` does.
    Execute(Arc<RecoveredBlock<Block>>),
    /// Buffer reth's pre-computed trie data, as `index_block` does.
    Index {
        /// The block the trie data belongs to.
        block: BlockWithParent,
        /// The hashed post state of the block.
        hashed_state: Arc<HashedPostStateSorted>,
        /// The trie updates of the block.
        trie_updates: Arc<TrieUpdatesSorted>,
    },
}

impl<Block: reth_primitives_traits::Block> ReorgBlock<Block> {
    fn block_with_parent(&self) -> BlockWithParent {
        match self {
            Self::Execute(block) => block.block_with_parent(),
            Self::Index { block, .. } => *block,
        }
    }
}

pub(crate) struct ReorgTask<Block: reth_primitives_traits::Block> {
    pub(crate) blocks: Vec<ReorgBlock<Block>>,
    pub(crate) reply: Sender<Result<(), EngineError>>,
}

impl<Block: reth_primitives_traits::Block> ReorgTask<Block> {
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
        let _ = self.reply.send(run(state, self.blocks));
    }
}

fn run<Block, Evm, Provider, Store>(
    state: &mut EngineState<Evm, Provider, Store>,
    blocks: Vec<ReorgBlock<Block>>,
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
    let (Some(first), Some(last)) = (blocks.first(), blocks.last()) else {
        return Ok(());
    };
    let first = first.block_with_parent();
    let last_block_number = last.block_with_parent().block.number;
    let tip = state.get_tip()?;

    if first.block.number > tip.number {
        // Reorg originates beyond the stored tip — the engine is still catching up.
        // Sync batches will fetch post-reorg blocks from the provider, so nothing to unwind.
        debug!(
            target: "trie::engine::task",
            first_block = first.block.number,
            tip = tip.number,
            "Reorg starts beyond stored tip, skipping"
        );
        return Ok(());
    }

    let start = Instant::now();
    let common_ancestor_number = first.block.number.saturating_sub(1);

    info!(
        target: "trie::engine::task",
        reorg_depth = blocks.len(),
        common_ancestor = common_ancestor_number,
        "Handling reorg: unwinding and buffering new path"
    );

    state.unwind(first)?;

    for block in blocks {
        match block {
            ReorgBlock::Execute(block) => super::execute_block(&block, state)?,
            ReorgBlock::Index { block, hashed_state, trie_updates } => state.memory.insert(
                block,
                BlockStateDiff {
                    sorted_trie_updates: Arc::unwrap_or_clone(trie_updates),
                    sorted_post_state: Arc::unwrap_or_clone(hashed_state),
                },
            ),
        }
    }

    let total_duration = start.elapsed();

    #[cfg(feature = "metrics")]
    state.metrics.reorg_duration_seconds.record(total_duration);

    info!(
        target: "trie::engine::task",
        start_block_number = first.block.number,
        end_block_number = last_block_number,
        ?total_duration,
        "Trie updates rewound and buffered successfully",
    );

    Ok(())
}
