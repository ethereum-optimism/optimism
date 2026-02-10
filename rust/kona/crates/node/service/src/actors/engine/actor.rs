//! The [`EngineActor`].

use crate::{
    EngineActorRequest, EngineError, EngineProcessingRequest, EngineRequestReceiver,
    EngineRpcRequest, EngineRpcRequestReceiver, NodeActor,
};
use async_trait::async_trait;
use std::marker::PhantomData;
use tokio::{sync::mpsc, task::JoinHandle};

/// Builder for the [`EngineActor`].
pub struct EngineActorBuilder<
    EngineRequestReceiver_: EngineRequestReceiver,
    RpcRequestReceiver: EngineRpcRequestReceiver,
> {
    /// The inbound request channel.
    pub inbound_request_rx: mpsc::Receiver<EngineActorRequest>,
    /// The processor for engine requests.
    pub engine_receiver: EngineRequestReceiver_,
    /// The processor for engine RPC requests.
    pub rpc_receiver: RpcRequestReceiver,
}

impl<EngineRequestReceiver_: EngineRequestReceiver, RpcRequestReceiver: EngineRpcRequestReceiver>
    std::fmt::Debug for EngineActorBuilder<EngineRequestReceiver_, RpcRequestReceiver>
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("EngineActorBuilder").finish()
    }
}

/// The [`EngineActor`] is an intermediary that receives [`EngineActorRequest`] and delegates:
/// - Engine RPC queries requests to the configured [`EngineRpcRequestReceiver`]
/// - Node engine requests to the configured [`EngineRequestReceiver`].
///
/// The generic parameters are consumed during [`init`](NodeActor::init) when the sub-tasks are
/// spawned. At runtime, the actor only holds channels and join handles.
pub struct EngineActor<
    EngineRequestReceiver_: EngineRequestReceiver,
    RpcRequestReceiver: EngineRpcRequestReceiver,
> {
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<EngineActorRequest>,
    /// Sender for RPC requests to the rpc sub-task.
    rpc_tx: mpsc::Sender<EngineRpcRequest>,
    /// Sender for engine processing requests to the processing sub-task.
    engine_processing_tx: mpsc::Sender<EngineProcessingRequest>,
    /// Join handle for the RPC sub-task.
    rpc_join_handle: JoinHandle<Result<(), EngineError>>,
    /// Join handle for the engine processing sub-task.
    processing_join_handle: JoinHandle<Result<(), EngineError>>,
    /// Phantom data to hold the generic types consumed during init.
    _phantom: PhantomData<(EngineRequestReceiver_, RpcRequestReceiver)>,
}

impl<EngineRequestReceiver_: EngineRequestReceiver, RpcRequestReceiver: EngineRpcRequestReceiver>
    std::fmt::Debug for EngineActor<EngineRequestReceiver_, RpcRequestReceiver>
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("EngineActor").finish()
    }
}

impl<
    EngineRequestReceiver_: EngineRequestReceiver + 'static,
    RpcRequestReceiver: EngineRpcRequestReceiver + 'static,
> EngineActor<EngineRequestReceiver_, RpcRequestReceiver>
{
    /// Checks if a sub-task has finished and returns its result.
    async fn check_subtask(
        handle: &mut JoinHandle<Result<(), EngineError>>,
        label: &str,
    ) -> Result<(), EngineError> {
        if handle.is_finished() {
            match handle.await {
                Ok(Ok(())) => {
                    info!(target: "engine", "Engine {label} task completed successfully");
                }
                Ok(Err(err)) => {
                    error!(target: "engine", ?err, "Engine {label} task failed");
                    return Err(err);
                }
                Err(e) => {
                    error!(target: "engine", ?e, "Engine {label} task panicked");
                    return Err(EngineError::ChannelClosed);
                }
            }
        }
        Ok(())
    }

    /// Sends a processing request to the engine processing sub-task.
    async fn send_processing(&self, req: EngineProcessingRequest) -> Result<(), EngineError> {
        self.engine_processing_tx.send(req).await.map_err(|_| {
            error!(target: "engine", "Engine processing channel closed unexpectedly");
            EngineError::ChannelClosed
        })
    }
}

#[async_trait]
impl<
    EngineRequestReceiver_: EngineRequestReceiver + 'static,
    RpcRequestReceiver: EngineRpcRequestReceiver + 'static,
> NodeActor for EngineActor<EngineRequestReceiver_, RpcRequestReceiver>
{
    type Error = EngineError;
    type Builder = EngineActorBuilder<EngineRequestReceiver_, RpcRequestReceiver>;
    type InboundData = ();

    async fn init(builder: Self::Builder) -> Result<(Self, Self::InboundData), Self::Error> {
        let (rpc_tx, rpc_rx) = mpsc::channel(1024);
        let (engine_processing_tx, engine_processing_rx) = mpsc::channel(1024);

        let rpc_join_handle = builder.rpc_receiver.start(rpc_rx);
        let processing_join_handle = builder.engine_receiver.start(engine_processing_rx);

        let actor = Self {
            inbound_request_rx: builder.inbound_request_rx,
            rpc_tx,
            engine_processing_tx,
            rpc_join_handle,
            processing_join_handle,
            _phantom: PhantomData,
        };
        Ok((actor, ()))
    }

    async fn step(&mut self) -> Result<(), Self::Error> {
        // Check if either sub-task has failed early.
        Self::check_subtask(&mut self.rpc_join_handle, "query").await?;
        Self::check_subtask(&mut self.processing_join_handle, "processing").await?;

        let req = self.inbound_request_rx.recv().await;
        let Some(request) = req else {
            error!(target: "engine", "Engine inbound request receiver closed unexpectedly");
            return Err(EngineError::ChannelClosed);
        };

        // Route the request to the appropriate channel.
        match request {
            EngineActorRequest::RpcRequest(rpc_req) => {
                self.rpc_tx.send(*rpc_req).await.map_err(|_| {
                    error!(target: "engine", "Engine RPC request handler channel closed unexpectedly");
                    EngineError::ChannelClosed
                })?;
            }
            EngineActorRequest::BuildRequest(req) => {
                self.send_processing(EngineProcessingRequest::Build(req)).await?;
            }
            EngineActorRequest::ProcessSafeL2SignalRequest(signal) => {
                self.send_processing(EngineProcessingRequest::ProcessSafeL2Signal(signal)).await?;
            }
            EngineActorRequest::ProcessFinalizedL2BlockNumberRequest(block_number) => {
                self.send_processing(EngineProcessingRequest::ProcessFinalizedL2BlockNumber(
                    block_number,
                ))
                .await?;
            }
            EngineActorRequest::ProcessUnsafeL2BlockRequest(envelope) => {
                self.send_processing(EngineProcessingRequest::ProcessUnsafeL2Block(envelope))
                    .await?;
            }
            EngineActorRequest::ResetRequest(reset_req) => {
                self.send_processing(EngineProcessingRequest::Reset(reset_req)).await?;
            }
            EngineActorRequest::SealRequest(seal_req) => {
                self.send_processing(EngineProcessingRequest::Seal(seal_req)).await?;
            }
        }
        Ok(())
    }
}
