use crate::{DerivationActorRequest, DerivationClientError, DerivationClientResult};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_derive::Signal;
use kona_protocol::L2BlockInfo;
use std::fmt::Debug;
use tokio::sync::mpsc;

/// Client to use to interact with the [`crate::DerivationActor`].
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait EngineDerivationClient: Debug + Send + Sync {
    /// Notifies the [`crate::DerivationActor`] that engine syncing has completed.
    /// Note: Does not wait for the derivation client to process this message.
    async fn notify_sync_completed(&self, safe_head: L2BlockInfo) -> DerivationClientResult<()>;

    /// Sends the new engine safe_head to the [`crate::DerivationActor`].
    /// Note: Does not wait for the derivation client to process this message.
    async fn send_new_engine_safe_head(&self, safe_head: L2BlockInfo)
    -> DerivationClientResult<()>;

    /// Sends the [`crate::DerivationActor`] the provided [`Signal`].
    /// Note: Does not wait for the derivation client to process this message.
    async fn send_signal(&self, signal: Signal) -> DerivationClientResult<()>;
}

/// Client to use to send messages to the [`crate::DerivationActor`]'s inbound channel.
#[derive(Constructor, Debug)]
pub struct QueuedEngineDerivationClient {
    /// A channel to use to send the [`DerivationActorRequest`]s to the [`crate::DerivationActor`].
    pub derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
}

#[async_trait]
impl EngineDerivationClient for QueuedEngineDerivationClient {
    async fn notify_sync_completed(&self, safe_head: L2BlockInfo) -> DerivationClientResult<()> {
        info!(target: "engine", "Sending sync completed to derivation actor");

        self.derivation_actor_request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(safe_head)))
            .await
            .map_err(|_| {
                DerivationClientError::RequestError("request channel closed.".to_string())
            })?;

        Ok(())
    }

    async fn send_new_engine_safe_head(
        &self,
        safe_head: L2BlockInfo,
    ) -> DerivationClientResult<()> {
        info!(target: "engine", safe_head = ?safe_head, "Sending new safe head to derivation actor");

        self.derivation_actor_request_tx
            .send(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::new(safe_head)))
            .await
            .map_err(|_| {
                DerivationClientError::RequestError("request channel closed.".to_string())
            })?;

        Ok(())
    }

    async fn send_signal(&self, signal: Signal) -> DerivationClientResult<()> {
        info!(target: "engine", signal = ?signal, "Sending signal to derivation actor");

        self.derivation_actor_request_tx
            .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(signal)))
            .await
            .map_err(|_| {
                DerivationClientError::RequestError("request channel closed.".to_string())
            })?;

        Ok(())
    }
}
