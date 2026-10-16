use super::super::state::EngineState;
use crate::{OpProofsStore, engine::CanonicalInMemoryStateProvider};
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{BlockHashReader, BlockReader, DatabaseProviderFactory};
use tracing::debug;

pub(crate) struct SyncToTask {
    pub(crate) target: u64,
}

impl SyncToTask {
    pub(crate) fn execute<Evm, Provider, Store>(self, state: &mut EngineState<Evm, Provider, Store>)
    where
        Evm: ConfigureEvm,
        Provider: BlockHashReader
            + CanonicalInMemoryStateProvider
            + DatabaseProviderFactory
            + BlockReader<Block = BlockTy<Evm::Primitives>>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        state.update_sync_target(self.target);
        debug!(target: "trie::engine::task", sync_target = self.target, "Sync target updated");
    }
}
