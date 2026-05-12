use super::super::state::EngineState;
use crate::{BlockStateDiff, OpProofsStore, engine::EngineError};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::Sender;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use std::{sync::Arc, time::Instant};
use tracing::{debug, info};

pub(crate) struct ReorgTask {
    pub(crate) block_updates:
        Vec<(BlockWithParent, Arc<TrieUpdatesSorted>, Arc<HashedPostStateSorted>)>,
    pub(crate) reply: Sender<Result<(), EngineError>>,
}

impl ReorgTask {
    pub(crate) fn execute<Evm, Provider, Store>(self, state: &mut EngineState<Evm, Provider, Store>)
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
        let _ = self.reply.send(run(state, self.block_updates));
    }
}

fn run<Evm, Provider, Store>(
    state: &mut EngineState<Evm, Provider, Store>,
    block_updates: Vec<(BlockWithParent, Arc<TrieUpdatesSorted>, Arc<HashedPostStateSorted>)>,
) -> Result<(), EngineError>
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
    if block_updates.is_empty() {
        return Ok(());
    }

    let first = &block_updates[0].0;
    let tip = state.get_tip()?;

    if first.block.number > tip.number {
        // Reorg originates beyond the stored tip — the engine is still catching up.
        // Sync batches will fetch post-reorg blocks from the provider, so nothing to unwind.
        debug!(
            target: "live-trie::engine",
            first_block = first.block.number,
            tip = tip.number,
            "Reorg starts beyond stored tip, skipping"
        );
        return Ok(());
    }

    let start = Instant::now();
    let common_ancestor_number = first.block.number.saturating_sub(1);

    info!(
        target: "live-trie::engine",
        reorg_depth = block_updates.len(),
        common_ancestor = common_ancestor_number,
        "Handling reorg: unwinding and buffering new path"
    );

    state.unwind(*first)?;

    for (block, trie_updates, hashed_state) in &block_updates {
        state.memory.insert(
            *block,
            BlockStateDiff {
                sorted_trie_updates: (**trie_updates).clone(),
                sorted_post_state: (**hashed_state).clone(),
            },
        );
    }

    let total_duration = start.elapsed();

    #[cfg(feature = "metrics")]
    state.metrics.reorg_duration_seconds.record(total_duration);

    info!(
        start_block_number = block_updates.first().map(|(b, _, _)| b.block.number),
        end_block_number = block_updates.last().map(|(b, _, _)| b.block.number),
        ?total_duration,
        "Trie updates rewound and buffered successfully",
    );

    Ok(())
}
