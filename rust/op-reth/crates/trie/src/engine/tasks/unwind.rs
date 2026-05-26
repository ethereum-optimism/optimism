use super::super::state::EngineState;
use crate::{OpProofsStore, engine::EngineError};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::Sender;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
};
use tracing::{debug, info};

pub(crate) struct UnwindTask {
    pub(crate) to: BlockWithParent,
    pub(crate) reply: Sender<Result<(), EngineError>>,
}

impl UnwindTask {
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
        let _ = self.reply.send(run(state, self.to));
    }
}

fn run<Evm, Provider, Store>(
    state: &mut EngineState<Evm, Provider, Store>,
    to: BlockWithParent,
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
    let tip = state.get_tip()?;
    if to.block.number > tip.number {
        debug!(
            target: "trie::engine::task",
            to_block = to.block.number,
            tip = tip.number,
            "Unwind target beyond stored tip, skipping"
        );
        return Ok(());
    }

    info!(target: "trie::engine::task", to_block = to.block.number, "Unwinding history");
    state.unwind(to)?;
    Ok(())
}
