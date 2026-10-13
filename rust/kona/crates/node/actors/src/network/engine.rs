use crate::EngineClientResult;
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;

/// The engine, as the network actor uses it to insert the unsafe blocks it receives.
#[async_trait]
pub trait Engine: Debug + Send + Sync {
    /// Queues `block` for insertion.
    ///
    /// Success means the block was queued, not that it was inserted.
    async fn insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> EngineClientResult<()>;
}
