use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use tokio::sync::mpsc::{self, error::TrySendError};

/// The engine, as the network actor uses it to insert the unsafe blocks it receives.
pub trait Engine: Debug + Send + Sync {
    /// Queues `block` for insertion without waiting, so that a backed-up engine never stops the
    /// network.
    ///
    /// Success means the block was queued, not that it was inserted. Fails with
    /// [`TrySendError::Full`] while the engine's queue is full, and with [`TrySendError::Closed`]
    /// once the engine has stopped.
    fn try_insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> Result<(), TrySendError<()>>;
}

impl Engine for mpsc::Sender<OpExecutionPayloadEnvelope> {
    fn try_insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> Result<(), TrySendError<()>> {
        self.try_send(block).map_err(crate::without_value)
    }
}
