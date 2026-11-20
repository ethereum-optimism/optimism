use crate::{EngineActorRequest, EngineClientError, EngineClientResult, ResetRequest};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_engine::FinalizeBlockId;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use std::fmt::Debug;
use tokio::sync::mpsc;

/// Client to use to interact with the engine.
#[cfg_attr(test, mockall::automock(type SafeL2Signal = OpAttributesWithParent;))]
#[async_trait]
pub trait DerivationEngineClient: Debug + Send + Sync {
    /// Resets the engine's forkchoice.
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()>;

    /// Sends a request to finalize the L2 block identified by the provided [`FinalizeBlockId`].
    /// Note: This does not wait for the engine to process it.
    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()>;

    /// Sends attributes from the derivation pipeline to the engine for consolidation.
    ///
    /// `reset_epoch` is the number of engine resets derivation has handled, so the engine can
    /// drop attributes derived before its latest reset.
    ///
    /// Note: This does not wait for the engine to process it.
    async fn send_derived_attributes(
        &self,
        attributes: OpAttributesWithParent,
        reset_epoch: u64,
    ) -> EngineClientResult<()>;

    /// Asks the engine to finalize the L2 block at `number`, which derivation recorded as safe.
    ///
    /// `reset_epoch` is the number of engine resets derivation has handled, so the engine can
    /// drop finalizations recorded before its latest reset.
    ///
    /// Note: This does not wait for the engine to process it.
    async fn send_derived_finalization(
        &self,
        number: u64,
        reset_epoch: u64,
    ) -> EngineClientResult<()>;

    /// Sends the engine a safe block reported by a derivation delegate.
    ///
    /// Note: This does not wait for the engine to process it.
    async fn send_delegated_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()>;
}

/// Client to use to send messages to the Engine Actor's inbound channel.
#[derive(Constructor, Debug)]
pub struct QueuedDerivationEngineClient {
    /// A channel to use to send the [`EngineActorRequest`]s to the `EngineActor`.
    pub engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
}

impl QueuedDerivationEngineClient {
    async fn send(&self, request: EngineActorRequest) -> EngineClientResult<()> {
        self.engine_actor_request_tx
            .send(request)
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))
    }
}

#[async_trait]
impl DerivationEngineClient for QueuedDerivationEngineClient {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        let (result_tx, mut result_rx) = mpsc::channel(1);

        self.engine_actor_request_tx
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        result_rx.recv().await.ok_or_else(|| {
            error!(target: "derivation_engine_client", "Failed to receive built payload");
            EngineClientError::ResponseError("response channel closed.".to_string())
        })?
    }

    async fn send_finalized_l2_block(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
        self.engine_actor_request_tx
            .send(EngineActorRequest::ProcessFinalizedL2Block(Box::new(block_id)))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        Ok(())
    }

    async fn send_derived_attributes(
        &self,
        attributes: OpAttributesWithParent,
        reset_epoch: u64,
    ) -> EngineClientResult<()> {
        self.send(EngineActorRequest::ProcessDerivedAttributes {
            attributes: Box::new(attributes),
            reset_epoch,
        })
        .await
    }

    async fn send_derived_finalization(
        &self,
        number: u64,
        reset_epoch: u64,
    ) -> EngineClientResult<()> {
        self.send(EngineActorRequest::ProcessDerivedFinalization { number, reset_epoch }).await
    }

    async fn send_delegated_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()> {
        self.send(EngineActorRequest::ProcessDelegatedSafeHead(Box::new(safe_head))).await
    }
}
