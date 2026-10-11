use super::super::state::EngineState;
use crate::OpProofsStore;
use crossbeam_channel::Sender;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{BlockHashReader, BlockReader, DatabaseProviderFactory, StateReader};

pub(crate) struct FlushTask {
    pub(crate) reply: Sender<()>,
}

impl FlushTask {
    pub(crate) fn execute<Evm, Provider, Store>(self, state: &mut EngineState<Evm, Provider, Store>)
    where
        Evm: ConfigureEvm,
        Provider: BlockHashReader
            + StateReader
            + DatabaseProviderFactory
            + BlockReader<Block = BlockTy<Evm::Primitives>>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        state.drain_persistence();
        let _ = self.reply.send(());
    }
}
