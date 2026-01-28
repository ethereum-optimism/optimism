use crate::{EngineActorRequest, EngineClientError, EngineClientResult};
use async_trait::async_trait;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use tokio::sync::mpsc;

/// Client used to interact with the Engine.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait NetworkEngineClient: Debug + Send + Sync {
    /// Note: a successful response does not mean the block was successfully inserted.
    /// This function just sends the message to the engine. It does not wait for a response.
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()>;
}

/// Client to use to send unsafe blocks to the Engine's inbound channel.
#[derive(Debug)]
pub struct QueuedNetworkEngineClient {
    /// A channel to use to send the EngineActor requests.
    pub engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
}

#[async_trait]
impl NetworkEngineClient for QueuedNetworkEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        trace!(target: "network", ?block, "Sending unsafe block to engine.");
        Ok(self
            .engine_actor_request_tx
            .send(EngineActorRequest::ProcessUnsafeL2BlockRequest(Box::new(block)))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?)
    }
}
