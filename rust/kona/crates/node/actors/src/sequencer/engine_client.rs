use crate::{EngineClientError, EngineClientResult, engine};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::fmt::Debug;
use tokio::sync::watch;

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
    /// The engine actor's handle.
    pub engine: engine::Handle,
    /// A channel to receive the latest unsafe head [`L2BlockInfo`].
    pub unsafe_head_rx: watch::Receiver<L2BlockInfo>,
}

#[async_trait]
impl SequencerEngineClient for QueuedSequencerEngineClient {
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(*self.unsafe_head_rx.borrow())
    }

    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        info!(target: "sequencer", "Sending reset request to engine.");
        self.engine
            .reset()
            .await
            .inspect(|_| info!(target: "sequencer", "Engine reset successfully."))
            .inspect_err(|err| {
                if matches!(err, EngineClientError::ResponseError(_)) {
                    error!(target: "block_engine", "Failed to receive built payload");
                }
            })
    }

    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        trace!(target: "sequencer", "Sending start build request to engine.");
        self.engine
            .start_build(attributes)
            .await
            .inspect(|payload_id| trace!(target: "sequencer", ?payload_id, "Start build request successfully."))
            .inspect_err(|err| {
                if matches!(err, EngineClientError::ResponseError(_)) {
                    error!(target: "block_engine", "Failed to receive payload for initiated block build");
                }
            })
    }

    async fn seal_and_canonicalize_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        trace!(target: "sequencer", ?attributes, "Sending seal request to engine.");
        match self.engine.seal(payload_id, attributes).await {
            Ok(payload) => {
                trace!(target: "sequencer", ?payload, "Seal succeeded.");
                Ok(payload)
            }
            Err(err @ EngineClientError::SealError(_)) => {
                info!(target: "sequencer", ?err, "Seal failed.");
                Err(err)
            }
            Err(err) => {
                if matches!(err, EngineClientError::ResponseError(_)) {
                    error!(target: "block_engine", "Failed to receive built payload");
                }
                Err(err)
            }
        }
    }
}
