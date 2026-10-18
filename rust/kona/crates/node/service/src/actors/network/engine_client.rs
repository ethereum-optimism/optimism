use crate::{EngineClientError, EngineClientResult};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// Client used to interact with the Engine.
#[cfg_attr(test, mockall::automock)]
pub trait NetworkEngineClient: Debug + Send + Sync {
    /// Best-effort gossip admission; a busy receiver may drop the block without blocking.
    fn try_send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()>;
}

impl NetworkEngineClient for crate::EngineHandle {
    fn try_send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        match self.try_insert_unsafe(block) {
            Ok(()) | Err(crate::UnsafeAdmissionError::Full) => Ok(()),
            Err(crate::UnsafeAdmissionError::Closed) => Err(EngineClientError::Closed),
        }
    }
}
