//! Custom RPC subscriptions to the engine owner's published heads.

use crate::jsonrpsee::WsServer;
use jsonrpsee::{
    PendingSubscriptionSink,
    core::{SubscriptionResult, to_json_raw_value},
};
use kona_engine::EngineState;
use kona_protocol::L2BlockInfo;
use tokio::sync::watch;

/// An RPC server that handles subscriptions to the node's state.
#[derive(Debug)]
pub struct WsRPC {
    state: watch::Receiver<EngineState>,
}

impl WsRPC {
    /// Constructs head subscriptions from the engine owner's publication.
    pub const fn new(state: watch::Receiver<EngineState>) -> Self {
        Self { state }
    }

    async fn subscribe(
        &self,
        sink: PendingSubscriptionSink,
        head: fn(&EngineState) -> L2BlockInfo,
    ) -> SubscriptionResult {
        let mut state = self.state.clone();
        state.has_changed()?;
        let mut current = head(&state.borrow());
        let sink = sink.accept().await?;
        loop {
            let update = tokio::select! {
                _ = sink.closed() => return Ok(()),
                update = state.wait_for(|state| head(state) != current) => update.map(|state| head(&state)),
            };
            let Ok(next) = update else { break };
            sink.send(to_json_raw_value(&next)?).await?;
            current = next;
        }
        warn!(target: "rpc::ws", "Engine state publisher closed");
        Ok(())
    }
}

#[async_trait::async_trait]
impl WsServer for WsRPC {
    async fn ws_safe_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        self.subscribe(sink, |state| state.sync_state.safe_head()).await
    }

    async fn ws_finalized_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        self.subscribe(sink, |state| state.sync_state.finalized_head()).await
    }

    async fn ws_unsafe_head_updates(&self, sink: PendingSubscriptionSink) -> SubscriptionResult {
        self.subscribe(sink, |state| state.sync_state.unsafe_head()).await
    }
}
