//! A task making derivation derive again from the finalized head.

use crate::{
    EngineClient, EngineState, ExecutableEngineTask, UndoTaskError, state::EngineSyncStateUpdate,
    task_queue::synchronize,
};
use async_trait::async_trait;
use derive_more::Constructor;
use std::sync::Arc;
use tokio::sync::mpsc;

/// The [`UndoTask`] makes derivation derive again from the finalized head, after a cross-chain
/// verifier undid verifications the node may have built on: it deleted a denial, so a block the
/// node replaced on it may have to come back, or it lost its store.
///
/// The task moves the local-safe head and the safe label back to the finalized head and fails
/// with [`UndoTaskError::Undone`]: derivation restarts from the finalized head, which truncates
/// the safe head database, and derives every later height again with the verifier's new denied
/// blocks. Where a block the node replaced on a deleted denial is derived again, the derived
/// attributes differ from the replacement on the execution layer, so consolidation builds the
/// derived block and reorgs the unsafe chain onto it. Only the labels move; the execution layer
/// keeps its head until derivation reorgs it.
///
/// `done` hears once the forkchoice update moving the labels back succeeded. From then on the
/// execution layer's safe block is the finalized head, so a restart also derives again from it.
#[derive(Debug, Clone, Constructor)]
pub struct UndoTask {
    /// The engine client.
    pub client: Arc<EngineClient>,
    /// Where to report that the labels moved back.
    pub done: mpsc::Sender<()>,
}

#[async_trait]
impl ExecutableEngineTask for UndoTask {
    type Output = ();

    type Error = UndoTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), UndoTaskError> {
        let Some(_) = state.sync_state.established_finalized_head() else {
            return Err(UndoTaskError::NoFinalizedHead);
        };
        let finalized = state.sync_state.finalized_head();
        synchronize(
            self.client.as_ref(),
            state,
            EngineSyncStateUpdate {
                local_safe_head: Some(finalized),
                safe_head: Some(finalized),
                ..Default::default()
            },
        )
        .await?;
        let _ = self.done.try_send(());
        let local_safe = finalized.block_info.id();
        warn!(target: "engine", ?local_safe, "Moved the local-safe head and the safe label back to the finalized head to derive again after the verifier undid verifications");
        Err(UndoTaskError::Undone { local_safe })
    }
}
