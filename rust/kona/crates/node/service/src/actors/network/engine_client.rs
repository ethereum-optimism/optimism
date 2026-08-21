use crate::{ChainControllerClientError, ChainControllerClientResult, ChainControllerRequest};
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
    async fn send_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> ChainControllerClientResult<()>;
}

/// Client to use to send unsafe blocks to the [`crate::ChainController`]'s inbound channel.
#[derive(Debug)]
pub struct QueuedNetworkEngineClient {
    /// A channel to use to send the `ChainController` requests.
    pub controller_request_tx: mpsc::Sender<ChainControllerRequest>,
}

#[async_trait]
impl NetworkEngineClient for QueuedNetworkEngineClient {
    async fn send_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> ChainControllerClientResult<()> {
        trace!(target: "network", ?block, "Sending unsafe block to engine.");
        Ok(self
            .controller_request_tx
            .send(ChainControllerRequest::ProcessUnsafeL2Block(Box::new(block)))
            .await
            .map_err(|_| {
                ChainControllerClientError::RequestError("request channel closed.".to_string())
            })?)
    }
}
