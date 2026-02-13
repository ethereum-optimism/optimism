//! Development RPC API for exposing internal engine state and task queue information.
//!
//! This module provides development and debugging endpoints that allow introspection
//! of the engine's internal state, task queue, and operations.

use async_trait::async_trait;
use jsonrpsee::{
    PendingSubscriptionSink, SubscriptionSink,
    core::{RpcResult, SubscriptionResult},
    types::ErrorCode,
};

use crate::{DevEngineApiServer, EngineRpcClient};
use jsonrpsee::core::to_json_raw_value;

/// Implementation of the development RPC API.
#[derive(Debug)]
pub struct DevEngineRpc<EngineRpcClient_> {
    /// The engine query sender.
    engine_client: EngineRpcClient_,
}

impl<EngineRpcClient_: EngineRpcClient> DevEngineRpc<EngineRpcClient_> {
    /// Creates a new [`DevEngineRpc`] instance.
    pub const fn new(engine_client: EngineRpcClient_) -> Self {
        Self { engine_client }
    }

    /// Gets an engine queue length watcher for subscriptions.
    async fn engine_queue_length_watcher(
        &self,
    ) -> Result<tokio::sync::watch::Receiver<usize>, jsonrpsee::core::SubscriptionError> {
        self.engine_client.dev_subscribe_to_engine_queue_length().await.map_err(|_| jsonrpsee::core::SubscriptionError::from("Internal error. Failed to receive engine task receiver query. The engine query handler is likely closed."))
    }

    async fn send_queue_length_update(
        sink: &SubscriptionSink,
        queue_length: &usize,
    ) -> Result<(), jsonrpsee::core::SubscriptionError> {
        sink.send(to_json_raw_value(queue_length).map_err(|_| {
            jsonrpsee::core::SubscriptionError::from(
                "Internal error. Impossible to convert engine queue length to json",
            )
        })?)
        .await
        .map_err(|_| {
            jsonrpsee::core::SubscriptionError::from(
                "Failed to send engine queue length update. Subscription likely dropped.",
            )
        })
    }
}

#[async_trait]
impl<EngineRpcClient_: EngineRpcClient + 'static> DevEngineApiServer
    for DevEngineRpc<EngineRpcClient_>
{
    async fn dev_task_queue_length(&self) -> RpcResult<usize> {
        self.engine_client.dev_get_task_queue_length().await.map_err(|_| {
            jsonrpsee::types::ErrorObjectOwned::owned(
                ErrorCode::InternalError.code(),
                "Failed to receive task queue length",
                None::<()>,
            )
        })
    }

    async fn dev_subscribe_engine_queue_length(
        &self,
        sink: PendingSubscriptionSink,
    ) -> SubscriptionResult {
        let sink = sink.accept().await?;

        let mut subscription = self.engine_queue_length_watcher().await?;

        let mut current_queue_length = *subscription.borrow();

        Self::send_queue_length_update(&sink, &current_queue_length).await?;

        while let Ok(new_queue_length) = subscription
            .wait_for(|queue_length| queue_length != &current_queue_length)
            .await
            .map(|state| *state)
        {
            Self::send_queue_length_update(&sink, &new_queue_length).await?;
            current_queue_length = new_queue_length;
        }

        tracing::warn!(target: "rpc::dev::engine_queue_size", "Subscription to engine queue size has been closed.");
        Ok(())
    }
}
