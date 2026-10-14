use super::{BuildTaskError, EngineState, FinalizeBlockId, SealTaskError};
use crate::{EngineClientError, EngineClientResult};
use alloy_rpc_types_engine::PayloadId;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{
    mpsc::{self, error::TrySendError},
    oneshot, watch,
};

/// Work executed by the engine actor.
#[derive(Debug)]
pub(super) enum Message {
    /// Start building a block.
    Build(Box<BuildRequest>),
    /// Consolidate the safe head with attributes derived from L1.
    Consolidate(Box<ConsolidateRequest>),
    /// Follow a safe head supplied by the derivation delegate.
    FollowSafeHead(Box<L2BlockInfo>),
    /// Finalize an L2 block.
    Finalize(Box<FinalizeBlockId>),
    /// Insert an unsafe L2 block.
    InsertUnsafeBlock(Box<OpExecutionPayloadEnvelope>),
    /// Reset the forkchoice, replying with the new engine state once the reset succeeds.
    Reset(ResetRequest),
    /// Seal a block, without importing it.
    Seal(Box<SealRequest>),
    /// Import a sealed block and make it the unsafe head.
    Canonicalize(Box<CanonicalizeRequest>),
}

/// A request to start building a block.
#[derive(Debug)]
pub(super) struct BuildRequest {
    /// The attributes to build the block from.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives the payload ID once the build starts, or the reason it could not start.
    pub(super) result_tx: oneshot::Sender<Result<PayloadId, BuildTaskError>>,
}

/// A request to consolidate the safe head with derived attributes.
#[derive(Debug)]
pub(super) struct ConsolidateRequest {
    /// The attributes derived from L1.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives how the engine consolidated the attributes.
    pub(super) result_tx: oneshot::Sender<Consolidation>,
}

/// A request to reset the forkchoice.
#[derive(Debug)]
pub(super) struct ResetRequest {
    /// Whether to refuse the reset unless the execution layer validated the latest forkchoice.
    pub(super) only_if_el_synced: bool,
    /// Receives the engine state after the reset, or why the engine refused it.
    pub(super) result_tx: oneshot::Sender<EngineClientResult<EngineState>>,
}

/// The engine's reply to a consolidation.
///
/// The engine publishes its state before replying, so the state matches the published one.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Consolidation {
    /// The engine state after the consolidation. If the engine reset before or during the
    /// consolidation, it published the reset before replying.
    pub state: EngineState,
    /// Whether the execution layer rejected the attributes' block, and a deposits-only block built
    /// from the same attributes replaced it, so derivation must flush its channel.
    pub flush: bool,
}

/// A request to seal a block, without importing it.
#[derive(Debug)]
pub(super) struct SealRequest {
    /// The payload ID returned when the build started.
    pub(super) payload_id: PayloadId,
    /// The attributes the block was built from.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives the sealed payload or the reason sealing failed.
    pub(super) result_tx: oneshot::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
}

/// A request to import a sealed block and make it the unsafe head.
#[derive(Debug)]
pub(super) struct CanonicalizeRequest {
    /// The sealed block.
    pub(super) payload: OpExecutionPayloadEnvelope,
    /// The attributes the block was built from.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives the reason the block did not become the unsafe head, if it did not.
    pub(super) result_tx: oneshot::Sender<Result<(), SealTaskError>>,
}

/// Published engine state and the engine's request queue.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
    state: watch::Receiver<EngineState>,
    resets: watch::Receiver<EngineState>,
}

impl Handle {
    /// Construct a handle from the engine's request sender, published state, and published
    /// resets.
    pub(super) const fn new(
        messages: mpsc::Sender<Message>,
        state: watch::Receiver<EngineState>,
        resets: watch::Receiver<EngineState>,
    ) -> Self {
        Self { messages, state, resets }
    }

    /// Subscribes to the engine state, which the actor publishes as it executes tasks.
    pub fn state(&self) -> watch::Receiver<EngineState> {
        self.state.clone()
    }

    /// Subscribes to the engine's resets from now on. The receiver sees the engine state after
    /// each later reset, before the reply to the request that caused it.
    pub fn resets(&self) -> watch::Receiver<EngineState> {
        let mut resets = self.resets.clone();
        resets.mark_unchanged();
        resets
    }

    /// Resets the engine forkchoice to a starting point found from the execution layer's heads,
    /// and returns the engine state afterwards.
    ///
    /// Waits until the reset succeeds. The engine retains the reset across temporary failures.
    pub async fn reset(&self) -> EngineClientResult<EngineState> {
        self.request_reset(false).await
    }

    /// Resets the engine forkchoice like [`Self::reset`], but only if the execution layer is
    /// synced: the engine sends it the current forkchoice again, and refuses the reset with
    /// [`EngineClientError::ElSyncing`] unless the execution layer validates it. The caller may
    /// try again later.
    ///
    /// Resetting while the execution layer syncs would point it at its own stale heads, away from
    /// the chain it is syncing.
    pub async fn reset_if_el_synced(&self) -> EngineClientResult<EngineState> {
        self.request_reset(true).await
    }

    async fn request_reset(&self, only_if_el_synced: bool) -> EngineClientResult<EngineState> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Reset(ResetRequest { only_if_el_synced, result_tx })).await?;
        result_rx.await.map_err(|_| response_closed())?
    }

    /// Starts building a block and returns the payload ID to seal it with.
    ///
    /// The engine does not retry a build that fails to start; the error is returned instead.
    pub async fn start_build(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Build(Box::new(BuildRequest { attributes, result_tx }))).await?;
        Ok(result_rx.await.map_err(|_| response_closed())??)
    }

    /// Seals a block started with [`Self::start_build`], without importing it. The caller makes it
    /// canonical with [`Self::canonicalize`], once it may: a sequencer behind op-conductor first
    /// commits it to the conductor.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head moved since the
    /// build started.
    pub async fn seal(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Seal(Box::new(SealRequest { payload_id, attributes, result_tx })))
            .await?;
        Ok(result_rx.await.map_err(|_| response_closed())??)
    }

    /// Imports a block sealed with [`Self::seal`] and makes it the unsafe head.
    ///
    /// Fails with [`SealTaskError::UnsafeHeadChangedSinceBuild`] if the unsafe head is no longer
    /// the block's parent, and with [`SealTaskError::PayloadInsertionFailed`] if importing it
    /// fails. Unlike a derived block, one the execution layer rejects is not replaced with a
    /// deposits-only block: the conductor has not committed the replacement. The sequencer drops
    /// the block and builds another at its height.
    pub async fn canonicalize(
        &self,
        payload: OpExecutionPayloadEnvelope,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<()> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Canonicalize(Box::new(CanonicalizeRequest {
            payload,
            attributes,
            result_tx,
        })))
        .await?;
        Ok(result_rx.await.map_err(|_| response_closed())??)
    }

    /// Consolidates the safe head with attributes derived from L1, and returns the engine state
    /// afterwards.
    ///
    /// The engine retries temporary failures, so this waits until the consolidation completes.
    /// Attributes whose parent is not the engine's safe head are stale, typically because the
    /// engine reset since they were derived; the engine skips them and replies with its state.
    pub async fn consolidate(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidation> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Consolidate(Box::new(ConsolidateRequest { attributes, result_tx })))
            .await?;
        result_rx.await.map_err(|_| response_closed())
    }

    /// Queues a safe head supplied by the derivation delegate, for the engine to follow.
    ///
    /// Success means the safe head was queued, not that it was processed.
    pub async fn follow_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()> {
        self.send(Message::FollowSafeHead(Box::new(safe_head))).await
    }

    /// Queues finalization of an L2 block.
    ///
    /// Success means the block was queued, not that it was finalized.
    pub async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
        self.send(Message::Finalize(Box::new(block_id))).await
    }

    /// Queues an unsafe block for insertion.
    ///
    /// Success means the block was queued, not that it was inserted.
    pub async fn insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> EngineClientResult<()> {
        self.send(Message::InsertUnsafeBlock(Box::new(block))).await
    }

    /// Queues an unsafe block for insertion without waiting for room in the queue.
    ///
    /// Success means the block was queued, not that it was inserted.
    pub fn try_insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> Result<(), TrySendError<()>> {
        self.messages
            .try_send(Message::InsertUnsafeBlock(Box::new(block)))
            .map_err(crate::without_value)
    }

    async fn send(&self, message: Message) -> EngineClientResult<()> {
        self.messages
            .send(message)
            .await
            .map_err(|_| EngineClientError::RequestError("request channel closed.".to_string()))
    }
}

fn response_closed() -> EngineClientError {
    EngineClientError::ResponseError("response channel closed.".to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::engine::EngineBuildError;
    use kona_protocol::L2BlockInfo;
    use std::future::ready;

    fn attributes() -> OpAttributesWithParent {
        OpAttributesWithParent::new(Default::default(), Default::default(), None, false)
    }

    fn handle() -> (Handle, mpsc::Receiver<Message>) {
        let (messages, receiver) = mpsc::channel(1);
        let (_, state) = watch::channel(EngineState::default());
        let (_, resets) = watch::channel(EngineState::default());
        (Handle::new(messages, state, resets), receiver)
    }

    #[tokio::test]
    async fn requests_return_actor_replies() {
        let (handle, mut receiver) = handle();
        let reset = handle.reset();
        tokio::pin!(reset);
        let reply = tokio::select! {
            result = &mut reset => panic!("reset returned before it was received: {result:?}"),
            message = receiver.recv() => {
                let Some(Message::Reset(request)) = message else { panic!("expected reset") };
                assert!(!request.only_if_el_synced);
                request.result_tx
            }
        };
        // Receiving the request is insufficient: the actor must complete the reset.
        tokio::select! {
            biased;
            result = &mut reset => panic!("reset returned before the actor replied: {result:?}"),
            _ = ready(()) => {}
        }
        let state = EngineState::default();
        reply.send(Ok(state)).unwrap();
        assert_eq!(reset.await.unwrap(), state);

        // A reset that asks for a synced execution layer can be refused.
        let (result, ()) = tokio::join!(handle.reset_if_el_synced(), async {
            let Message::Reset(request) = receiver.recv().await.unwrap() else {
                panic!("expected reset");
            };
            assert!(request.only_if_el_synced);
            request.result_tx.send(Err(EngineClientError::ElSyncing)).unwrap();
        });
        assert!(matches!(result, Err(EngineClientError::ElSyncing)));

        let (result, ()) = tokio::join!(handle.consolidate(attributes()), async {
            let Message::Consolidate(request) = receiver.recv().await.unwrap() else {
                panic!("expected consolidate");
            };
            assert_eq!(request.attributes, attributes());
            request.result_tx.send(Consolidation { state, flush: true }).unwrap();
        });
        assert_eq!(result.unwrap(), Consolidation { state, flush: true });

        let payload_id = PayloadId::new([1; 8]);
        let (result, ()) = tokio::join!(handle.start_build(attributes()), async {
            let Message::Build(request) = receiver.recv().await.unwrap() else {
                panic!("expected build");
            };
            request.result_tx.send(Ok(payload_id)).unwrap();
        });
        assert_eq!(result.unwrap(), payload_id);

        let (result, ()) = tokio::join!(handle.start_build(attributes()), async {
            let Message::Build(request) = receiver.recv().await.unwrap() else {
                panic!("expected build");
            };
            request
                .result_tx
                .send(Err(BuildTaskError::EngineBuildError(EngineBuildError::EngineSyncing)))
                .unwrap();
        });
        assert!(matches!(
            result,
            Err(EngineClientError::StartBuildError(BuildTaskError::EngineBuildError(
                EngineBuildError::EngineSyncing
            )))
        ));

        let (result, ()) = tokio::join!(handle.seal(payload_id, attributes()), async {
            let Message::Seal(request) = receiver.recv().await.unwrap() else {
                panic!("expected seal");
            };
            assert_eq!(request.payload_id, payload_id);
            request.result_tx.send(Err(SealTaskError::DepositOnlyPayloadFailed)).unwrap();
        });
        assert!(matches!(
            result,
            Err(EngineClientError::SealError(SealTaskError::DepositOnlyPayloadFailed))
        ));
    }

    #[tokio::test]
    async fn queued_requests_send_their_message() {
        let (handle, mut receiver) = handle();
        let head = L2BlockInfo::default();
        handle.follow_safe_head(head).await.unwrap();
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::FollowSafeHead(block) if *block == head
        ));
        let block_id = FinalizeBlockId::ByNumber(1);
        handle.finalize(block_id).await.unwrap();
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::Finalize(id) if *id == block_id
        ));
        let block = OpExecutionPayloadEnvelope::V1(
            alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&alloy_consensus::Block::<
                op_alloy_consensus::OpTxEnvelope,
            >::default()),
        );
        handle.insert_unsafe_block(block.clone()).await.unwrap();
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::InsertUnsafeBlock(inserted) if *inserted == block
        ));
    }

    #[tokio::test]
    async fn requests_report_closed_channels() {
        let (handle, mut receiver) = handle();
        // The actor dropped the request without replying.
        let (result, ()) = tokio::join!(handle.reset(), async {
            drop(receiver.recv().await.unwrap());
        });
        assert!(matches!(result, Err(EngineClientError::ResponseError(_))));
        let (result, ()) = tokio::join!(handle.start_build(attributes()), async {
            drop(receiver.recv().await.unwrap());
        });
        assert!(matches!(result, Err(EngineClientError::ResponseError(_))));

        drop(receiver);
        assert!(matches!(handle.reset().await, Err(EngineClientError::RequestError(_))));
        assert!(matches!(
            handle.finalize(FinalizeBlockId::ByNumber(1)).await,
            Err(EngineClientError::RequestError(_))
        ));
    }
}
