//! Custom RPC subscription endpoints to for the kona node to stream internal state/data.

use jsonrpsee::{
    PendingSubscriptionSink, SubscriptionSink, core::SubscriptionResult, tracing::warn,
};
use kona_engine::EngineState;
use kona_protocol::L2BlockInfo;

use crate::{EngineRpcClient, jsonrpsee::WsServer};
use jsonrpsee::core::to_json_raw_value;

/// An RPC server that handles subscriptions to the node's state.
#[derive(Debug)]
pub struct WsRPC<EngineRpcClient_> {
    /// The engine query sender.
    engine_client: EngineRpcClient_,
}

impl<EngineRpcClient_: EngineRpcClient> WsRPC<EngineRpcClient_> {
    /// Constructs a new [`WsRPC`] instance.
    pub const fn new(engine_client: EngineRpcClient_) -> Self {
        Self { engine_client }
    }

    async fn engine_state_watcher(
        &self,
    ) -> Result<tokio::sync::watch::Receiver<EngineState>, jsonrpsee::core::SubscriptionError> {
        self.engine_client.dev_subscribe_to_engine_state().await.map_err(|_| jsonrpsee::core::SubscriptionError::from("Internal error. Failed to subscribe to engine state updates. The engine query handler is likely closed."))
    }

    async fn send_state_update(
        sink: &SubscriptionSink,
        state: L2BlockInfo,
    ) -> Result<(), jsonrpsee::core::SubscriptionError> {
        sink.send(to_json_raw_value(&state).map_err(|_| {
            jsonrpsee::core::SubscriptionError::from(
                "Internal error. Impossible to convert l2 block info to json",
            )
        })?)
        .await
        .map_err(|_| {
            jsonrpsee::core::SubscriptionError::from(
                "Failed to send head update. Subscription likely dropped.",
            )
        })
    }
}

#[async_trait::async_trait]
impl<EngineRpcClient_: EngineRpcClient + 'static> WsServer for WsRPC<EngineRpcClient_> {
    async fn ws_safe_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        let sink = sink.accept().await?;

        let mut subscription = self.engine_state_watcher().await?;

        let mut current_safe_head = subscription.borrow().sync_state.safe_head();

        while let Ok(new_state) = subscription
            .wait_for(|state| state.sync_state.safe_head() != current_safe_head)
            .await
            .map(|state| *state)
        {
            info!(target: "rpc::ws", "Sending safe head update: {:?}", new_state.sync_state.safe_head());
            current_safe_head = new_state.sync_state.safe_head();
            Self::send_state_update(&sink, current_safe_head).await?;
        }

        warn!(target: "rpc::ws", "Subscription to safe head updates has been closed.");
        Ok(())
    }

    async fn ws_finalized_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        let sink = sink.accept().await?;

        let mut subscription = self.engine_state_watcher().await?;

        let mut current_finalized_head = subscription.borrow().sync_state.finalized_head();

        while let Ok(new_state) = subscription
            .wait_for(|state| state.sync_state.finalized_head() != current_finalized_head)
            .await
            .map(|state| *state)
        {
            current_finalized_head = new_state.sync_state.finalized_head();
            Self::send_state_update(&sink, current_finalized_head).await?;
        }

        warn!(target: "rpc::ws", "Subscription to finalized head updates has been closed.");
        Ok(())
    }

    async fn ws_unsafe_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        let sink = sink.accept().await?;

        let mut subscription = self.engine_state_watcher().await?;

        let mut current_unsafe_head = subscription.borrow().sync_state.unsafe_head();

        while let Ok(new_state) = subscription
            .wait_for(|state| state.sync_state.unsafe_head() != current_unsafe_head)
            .await
            .map(|state| *state)
        {
            current_unsafe_head = new_state.sync_state.unsafe_head();
            Self::send_state_update(&sink, current_unsafe_head).await?;
        }

        warn!(target: "rpc::ws", "Subscription to unsafe head updates has been closed.");
        Ok(())
    }
}
