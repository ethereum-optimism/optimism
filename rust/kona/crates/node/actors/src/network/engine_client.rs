use crate::EngineClientResult;
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// Client used to interact with the Engine.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait NetworkEngineClient: Debug + Send + Sync {
    /// Note: a successful response does not mean the block was successfully inserted.
    /// This function just sends the message to the engine. It does not wait for a response.
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()>;
}
