use crate::{
    EngineClientError, EngineClientResult,
    actors::engine::{
        BuildRequest, CanonicalizeRequest, EngineActorRequest, ResetRequest, SealRequest,
    },
};
use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_engine::EngineState;
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

    /// Fetches a previously started block without changing forkchoice.
    ///
    /// Takes a `PayloadId` from a previous `start_build_block` call and returns
    /// the finalized execution payload envelope.
    async fn seal_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope>;

    /// Imports a sealed payload and awaits its canonicalization before publication.
    async fn canonicalize_block(
        &self,
        payload: OpExecutionPayloadEnvelope,
        parent: L2BlockInfo,
    ) -> EngineClientResult<()>;

    /// Returns the current unsafe head [`L2BlockInfo`].
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo>;

    /// Returns the current safe head after derivation and any cross-chain verification.
    async fn get_safe_head(&self) -> EngineClientResult<L2BlockInfo>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_engine::EngineSyncStateUpdate;
    use kona_protocol::BlockInfo;

    #[tokio::test]
    async fn safe_head_watch_tracks_advances_and_rewinds_without_using_local_safe() {
        let (requests, _) = mpsc::channel(1);
        let (_, unsafe_head_rx) = watch::channel(L2BlockInfo::default());
        let (state_tx, engine_state_rx) = watch::channel(EngineState::default());
        let client = QueuedSequencerEngineClient {
            engine_actor_request_tx: requests,
            unsafe_head_rx,
            engine_state_rx,
        };
        for number in [8, 12, 7] {
            let safe = L2BlockInfo {
                block_info: BlockInfo { number, ..Default::default() },
                ..Default::default()
            };
            state_tx.send_modify(|state| {
                state.sync_state = state.sync_state.apply_update(EngineSyncStateUpdate {
                    safe_head: Some(safe),
                    local_safe_head: Some(L2BlockInfo {
                        block_info: BlockInfo { number: 1000, ..Default::default() },
                        ..Default::default()
                    }),
                    ..Default::default()
                })
            });
            assert_eq!(client.get_safe_head().await.unwrap(), safe);
        }
    }
}

/// Queue-based implementation of the [`SequencerEngineClient`] trait. This handles all
/// channel-based communication.
#[derive(Constructor, Debug)]
pub struct QueuedSequencerEngineClient {
    /// A channel to use to send the `EngineActor` requests.
    pub engine_actor_request_tx: mpsc::Sender<EngineActorRequest>,
    /// A channel to receive the latest unsafe head [`L2BlockInfo`].
    pub unsafe_head_rx: watch::Receiver<L2BlockInfo>,
    /// The engine's published state, including safe-head advances and rewinds.
    pub engine_state_rx: watch::Receiver<EngineState>,
}

#[async_trait]
impl SequencerEngineClient for QueuedSequencerEngineClient {
    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(*self.unsafe_head_rx.borrow())
    }

    async fn get_safe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(self.engine_state_rx.borrow().sync_state.safe_head())
    }

    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        let (result_tx, mut result_rx) = mpsc::channel(1);

        self.engine_actor_request_tx
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        result_rx.recv().await.ok_or_else(|| {
            error!(target: "block_engine", "Failed to receive built payload");
            EngineClientError::ResponseError("response channel closed.".to_string())
        })?
    }

    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        let (payload_id_tx, mut payload_id_rx) = mpsc::channel(1);

        if self
            .engine_actor_request_tx
            .send(EngineActorRequest::Build(Box::new(BuildRequest {
                attributes,
                result_tx: payload_id_tx,
            })))
            .await
            .is_err()
        {
            return Err(EngineClientError::RequestError("request channel closed.".to_string()));
        }

        payload_id_rx.recv().await.ok_or_else(|| {
            error!(target: "block_engine", "Failed to receive payload for initiated block build");
            EngineClientError::ResponseError("response channel closed.".to_string())
        })?.map_err(EngineClientError::StartBuildError)
    }

    async fn seal_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        let (result_tx, mut result_rx) = mpsc::channel(1);

        self.engine_actor_request_tx
            .send(EngineActorRequest::Seal(Box::new(SealRequest {
                payload_id,
                attributes,
                result_tx,
            })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;

        match result_rx.recv().await {
            Some(Ok(payload)) => Ok(payload),
            Some(Err(err)) => Err(EngineClientError::SealError(err)),
            None => {
                error!(target: "block_engine", "Failed to receive built payload");
                Err(EngineClientError::ResponseError("response channel closed.".to_string()))
            }
        }
    }
    async fn canonicalize_block(
        &self,
        payload: OpExecutionPayloadEnvelope,
        parent: L2BlockInfo,
    ) -> EngineClientResult<()> {
        let (result_tx, mut result_rx) = mpsc::channel(1);
        self.engine_actor_request_tx
            .send(EngineActorRequest::Canonicalize(Box::new(CanonicalizeRequest {
                payload,
                parent,
                result_tx,
            })))
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))?;
        result_rx
            .recv()
            .await
            .ok_or_else(|| {
                EngineClientError::ResponseError("response channel closed.".to_string())
            })?
            .map(|_| ())
            .map_err(EngineClientError::CanonicalizeError)
    }
}
