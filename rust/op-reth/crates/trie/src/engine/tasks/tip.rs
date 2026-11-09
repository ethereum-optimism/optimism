use super::super::state::EngineState;
use crate::{
    OpProofsStore,
    engine::{CanonicalInMemoryStateProvider, EngineError},
};
use alloy_eips::NumHash;
use crossbeam_channel::Sender;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{BlockHashReader, BlockReader, DatabaseProviderFactory, StageCheckpointReader};

pub(crate) struct TipTask {
    pub(crate) reply: Sender<Result<NumHash, EngineError>>,
}

impl TipTask {
    pub(crate) fn execute<Evm, Provider, Store>(self, state: &EngineState<Evm, Provider, Store>)
    where
        Evm: ConfigureEvm,
        Provider: BlockHashReader
            + CanonicalInMemoryStateProvider
            + StageCheckpointReader
            + DatabaseProviderFactory
            + BlockReader<Block = BlockTy<Evm::Primitives>>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        let _ = self.reply.send(state.get_tip());
    }
}
