use crate::{
    EngineClientError, EngineClientResult,
    actors::engine::{BuildRequest, EngineActorRequest, ResetRequest, SealRequest},
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use tokio::sync::{mpsc, watch};

/// Trait to be used by the Sequencer to interact with the engine, abstracting communication
/// mechanism.
#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait SequencerEngineClient: Debug + Send + Sync {
    /// Resets the engine's forkchoice, awaiting confirmation that it succeeded or returning the
    /// error in performing the reset.
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()>;

    /// Starts building a block with the provided attributes.
    ///
    /// Returns a `PayloadId` that can be used to seal the block later.
    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId>;

    /// Seals and canonicalizes a previously started block.
    ///
    /// Takes a `PayloadId` from a previous `start_build_block` call and returns
    /// the finalized execution payload envelope.
    async fn seal_and_canonicalize_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope>;

    /// Returns the current unsafe head [`L2BlockInfo`].
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo>;
}

/// Queue-based implementation of the [`SequencerEngineClient`] trait. This handles all
/// channel-based communication.
#[derive(Constructor, Debug)]
pub struct QueuedSequencerEngineClient {
    /// A channel to use to send the EngineActor requests.
    pub engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
    /// A channel to receive the latest unsafe head [`L2BlockInfo`].
    pub unsafe_head_rx: watch::Receiver<L2BlockInfo>,
}

#[async_trait]
impl SequencerEngineClient for QueuedSequencerEngineClient {
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(*self.unsafe_head_rx.borrow())
    }

    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        let (result_tx, mut result_rx) = mpsc::channel(1);

        info!(target: "sequencer", "Sending reset request to engine.");
        self.engine_actor_request_tx
            .send(EngineActorRequest::ResetRequest(Box::new(ResetRequest { result_tx })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        result_rx
            .recv()
            .await
            .inspect(|_| info!(target: "sequencer", "Engine reset successfully."))
            .ok_or_else(|| {
                error!(target: "block_engine", "Failed to receive built payload");
                EngineClientError::ResponseError("response channel closed.".to_string())
            })?
    }

    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        let (payload_id_tx, mut payload_id_rx) = mpsc::channel(1);

        trace!(target: "sequencer", "Sending start build request to engine.");
        if self
            .engine_actor_request_tx
            .send(EngineActorRequest::BuildRequest(Box::new(BuildRequest {
                attributes,
                result_tx: payload_id_tx,
            })))
            .await
            .is_err()
        {
            return Err(EngineClientError::RequestError("request channel closed.".to_string()));
        }

        payload_id_rx.recv()
            .await
            .inspect(|payload_id| trace!(target: "sequencer", ?payload_id, "Start build request successfully."))
            .ok_or_else(|| {
            error!(target: "block_engine", "Failed to receive payload for initiated block build");
            EngineClientError::ResponseError("response channel closed.".to_string())
        })
    }

    async fn seal_and_canonicalize_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        let (result_tx, mut result_rx) = mpsc::channel(1);

        trace!(target: "sequencer", ?attributes, "Sending seal request to engine.");
        self.engine_actor_request_tx
            .send(EngineActorRequest::SealRequest(Box::new(SealRequest {
                payload_id,
                attributes,
                result_tx,
            })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        match result_rx.recv().await {
            Some(Ok(payload)) => {
                trace!(target: "sequencer", ?payload, "Seal succeeded.");
                Ok(payload)
            }
            Some(Err(err)) => {
                info!(target: "sequencer", ?err, "Seal failed.");
                Err(EngineClientError::SealError(err))
            }
            None => {
                error!(target: "block_engine", "Failed to receive built payload");
                Err(EngineClientError::ResponseError("response channel closed.".to_string()))
            }
        }
    }
}
