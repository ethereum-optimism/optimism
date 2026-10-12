use super::{BuildTaskError, EngineState, FinalizeBlockId, SealTaskError};
use crate::{EngineClientError, EngineClientResult};
use alloy_rpc_types_engine::PayloadId;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{mpsc, oneshot, watch};

/// Work executed by the engine actor.
#[derive(Debug)]
pub(super) enum Message {
    /// Start building a block.
    Build(Box<BuildRequest>),
    /// Consolidate the safe head with attributes derived from L1.
    Consolidate(Box<OpAttributesWithParent>),
    /// Follow a safe head supplied by the derivation delegate.
    FollowSafeHead(Box<L2BlockInfo>),
    /// Finalize an L2 block.
    Finalize(Box<FinalizeBlockId>),
    /// Insert an unsafe L2 block.
    InsertUnsafeBlock(Box<OpExecutionPayloadEnvelope>),
    /// Reset the forkchoice and signal derivation to reset, replying once the reset succeeds.
    Reset(oneshot::Sender<()>),
    /// Seal a block.
    Seal(Box<SealRequest>),
}

/// A request to start building a block.
#[derive(Debug)]
pub(super) struct BuildRequest {
    /// The attributes to build the block from.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives the payload ID once the build starts, or the reason it could not start.
    pub(super) result_tx: oneshot::Sender<Result<PayloadId, BuildTaskError>>,
}

/// A request to seal and canonicalize a block.
#[derive(Debug)]
pub(super) struct SealRequest {
    /// The payload ID returned when the build started.
    pub(super) payload_id: PayloadId,
    /// The attributes the block was built from.
    pub(super) attributes: OpAttributesWithParent,
    /// Receives the sealed payload or the reason sealing failed.
    pub(super) result_tx: oneshot::Sender<Result<OpExecutionPayloadEnvelope, SealTaskError>>,
}

/// Published engine state and the engine's request queue.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
    state: watch::Receiver<EngineState>,
}

impl Handle {
    /// Construct a handle from the engine's request sender and published state.
    pub(super) const fn new(
        messages: mpsc::Sender<Message>,
        state: watch::Receiver<EngineState>,
    ) -> Self {
        Self { messages, state }
    }

    /// Subscribes to the engine state, which the actor publishes as it executes tasks.
    pub fn state(&self) -> watch::Receiver<EngineState> {
        self.state.clone()
    }

    /// Resets the engine forkchoice and signals derivation to reset.
    ///
    /// Waits until the reset succeeds. The engine retains the reset across temporary failures.
    pub async fn reset(&self) -> EngineClientResult<()> {
        let (result_tx, result_rx) = oneshot::channel();
        self.send(Message::Reset(result_tx)).await?;
        result_rx.await.map_err(|_| response_closed())
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

    /// Seals and canonicalizes a block started with [`Self::start_build`].
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

    /// Queues attributes derived from L1 to consolidate the safe head with.
    ///
    /// Success means the attributes were queued, not that they were processed.
    pub async fn consolidate(&self, attributes: OpAttributesWithParent) -> EngineClientResult<()> {
        self.send(Message::Consolidate(Box::new(attributes))).await
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
        (Handle::new(messages, state), receiver)
    }

    #[tokio::test]
    async fn requests_return_actor_replies() {
        let (handle, mut receiver) = handle();
        let reset = handle.reset();
        tokio::pin!(reset);
        let reply = tokio::select! {
            result = &mut reset => panic!("reset returned before it was received: {result:?}"),
            message = receiver.recv() => {
                let Some(Message::Reset(reply)) = message else { panic!("expected reset") };
                reply
            }
        };
        // Receiving the request is insufficient: the actor must complete the reset.
        tokio::select! {
            biased;
            result = &mut reset => panic!("reset returned before the actor replied: {result:?}"),
            _ = ready(()) => {}
        }
        reply.send(()).unwrap();
        reset.await.unwrap();

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
        handle.consolidate(attributes()).await.unwrap();
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::Consolidate(consolidated) if *consolidated == attributes()
        ));
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
