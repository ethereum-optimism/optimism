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

use crate::DevEngineApiServer;
use jsonrpsee::core::to_json_raw_value;
use tokio::sync::watch;

/// Implementation of the development RPC API.
#[derive(Debug)]
pub struct DevEngineRpc {
    queue_length: watch::Receiver<usize>,
}

impl DevEngineRpc {
    /// Reads the engine owner's actual pending-work publication.
    pub const fn new(queue_length: watch::Receiver<usize>) -> Self {
        Self { queue_length }
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
    async fn dev_task_queue_length(&self) -> RpcResult<usize> {
        self.queue_length.has_changed().map_err(|_| ErrorCode::InternalError)?;
        Ok(*self.queue_length.borrow())
    }

    async fn dev_subscribe_engine_queue_length(
        &self,
        sink: PendingSubscriptionSink,
    ) -> SubscriptionResult {
        let sink = sink.accept().await?;

        let mut subscription = self.queue_length.clone();
        subscription.has_changed()?;

        let mut current_queue_length = *subscription.borrow();

        Self::send_queue_length_update(&sink, &current_queue_length).await?;

        loop {
            let update = tokio::select! {
                _ = sink.closed() => return Ok(()),
                update = subscription.wait_for(|length| *length != current_queue_length) => update.map(|length| *length),
            };
            let Ok(new_queue_length) = update else { break };
            Self::send_queue_length_update(&sink, &new_queue_length).await?;
            current_queue_length = new_queue_length;
        }

        tracing::warn!(target: "rpc::dev::engine_queue_size", "Subscription to engine queue size has been closed.");
        Ok(())
    }
}
