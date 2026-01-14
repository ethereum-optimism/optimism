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
use kona_engine::{EngineQueries, EngineQuerySender};

use crate::DevEngineApiServer;
use jsonrpsee::core::to_json_raw_value;

/// Implementation of the development RPC API.
#[derive(Debug)]
pub struct DevEngineRpc {
    /// The engine query sender.
    engine_query_sender: EngineQuerySender,
}

impl DevEngineRpc {
    /// Creates a new [`DevEngineRpc`] instance.
    pub const fn new(engine_query_sender: EngineQuerySender) -> Self {
        Self { engine_query_sender }
    }

    /// Gets an engine queue length watcher for subscriptions.
    async fn engine_queue_length_watcher(
        &self,
    ) -> Result<tokio::sync::watch::Receiver<usize>, jsonrpsee::core::SubscriptionError> {
        let (query_tx, query_rx) = tokio::sync::oneshot::channel();

        if let Err(e) =
            self.engine_query_sender.send(EngineQueries::QueueLengthReceiver(query_tx)).await
        {
            tracing::warn!(target: "rpc::dev", ?e, "Failed to send engine state receiver query. The engine query handler is likely closed.");
            return Err(jsonrpsee::core::SubscriptionError::from(
                "Internal error. Failed to send engine state receiver query. The engine query handler is likely closed.",
            ));
        }

        query_rx.await.map_err(|_| jsonrpsee::core::SubscriptionError::from("Internal error. Failed to receive engine task receiver query. The engine query handler is likely closed."))
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
impl DevEngineApiServer for DevEngineRpc {
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

    async fn dev_task_queue_length(&self) -> RpcResult<usize> {
        let (query_tx, query_rx) = tokio::sync::oneshot::channel();

        self.engine_query_sender.send(EngineQueries::TaskQueueLength(query_tx)).await.map_err(
            |_| {
                jsonrpsee::types::ErrorObjectOwned::owned(
                    ErrorCode::InternalError.code(),
                    "Engine query channel closed",
                    None::<()>,
                )
            },
        )?;

        query_rx.await.map_err(|_| {
            jsonrpsee::types::ErrorObjectOwned::owned(
                ErrorCode::InternalError.code(),
                "Failed to receive task queue length",
                None::<()>,
            )
        })
    }
}
