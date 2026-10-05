use async_trait::async_trait;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use libp2p::TransportError;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::{
    self, select,
    sync::mpsc::{self, UnboundedReceiver, UnboundedSender},
};

use crate::{
    NetworkEngineClient, NodeActor, SignedPayload,
    actors::network::{
        driver::NetworkDriverError, error::NetworkBuilderError, handler::NetworkHandler,
    },
};

/// The network actor handles two core networking components of the rollup node:
/// - *discovery*: Peer discovery over UDP using discv5.
/// - *gossip*: Block gossip over TCP using libp2p.
#[derive(Debug)]
pub struct NetworkActor<NetworkEngineClient_: NetworkEngineClient> {
    /// The live libp2p [`NetworkHandler`].
    handler: NetworkHandler,
    /// A channel to receive p2p RPC requests.
    p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
    /// A channel to receive admin RPC queries.
    admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
    /// A channel to receive signed unsafe blocks and publish them through the gossip layer.
    publish_rx: mpsc::Receiver<SignedPayload>,
    /// A client to use to interact with the engine actor.
    engine_client: NetworkEngineClient_,
    // Purely-internal channel: loops gossip-swarm events back into this actor's own select. It
    // never crosses an actor boundary, so it lives here rather than being injected.
    unsafe_block_tx: UnboundedSender<OpExecutionPayloadEnvelope>,
    unsafe_block_rx: UnboundedReceiver<OpExecutionPayloadEnvelope>,
}

impl<NetworkEngineClient_: NetworkEngineClient> NetworkActor<NetworkEngineClient_> {
    /// Constructs a new [`NetworkActor`].
    ///
    /// `handler` must already be live — i.e. the libp2p swarm it wraps must already have been
    /// built and started — before being passed in. Passing an unstarted handler will cause
    /// `step()` to hang or fail on its first poll of the gossip swarm. Keeping the constructor
    /// sync and treating the "is this live?" invariant as the caller's responsibility is the
    /// deliberate trade-off over an `init()`-style trait method.
    pub fn new(
        engine_client: NetworkEngineClient_,
        handler: NetworkHandler,
        p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
        admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
        publish_rx: mpsc::Receiver<SignedPayload>,
    ) -> Self {
        let (unsafe_block_tx, unsafe_block_rx) = mpsc::unbounded_channel();
        Self {
            handler,
            p2p_rpc_rx,
            admin_query_rx,
            publish_rx,
            engine_client,
            unsafe_block_tx,
            unsafe_block_rx,
        }
    }
}

/// An error from the network actor.
#[derive(Debug, Error)]
pub enum NetworkActorError {
    /// Network builder error.
    #[error(transparent)]
    NetworkBuilder(#[from] NetworkBuilderError),
    /// Network driver error.
    #[error(transparent)]
    NetworkDriver(#[from] NetworkDriverError),
    /// Driver startup failed.
    #[error(transparent)]
    DriverStartup(#[from] TransportError<std::io::Error>),
    /// The network driver was missing its unsafe block receiver.
    #[error("Missing unsafe block receiver in network driver")]
    MissingUnsafeBlockReceiver,
    /// Channel closed unexpectedly.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
}

#[async_trait]
impl<NetworkEngineClient_: NetworkEngineClient + 'static> NodeActor
    for NetworkActor<NetworkEngineClient_>
{
    type Error = NetworkActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
            block = self.unsafe_block_rx.recv() => {
                let Some(block) = block else {
                    error!(target: "node::p2p", "The unsafe block receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if self.engine_client.send_unsafe_block(block).await.is_err() {
                    warn!(target: "network", "Failed to forward unsafe block to engine");
                    return Err(NetworkActorError::ChannelClosed);
                }
                Ok(())
            }
            Some(signed) = self.publish_rx.recv(), if !self.publish_rx.is_closed() => {
                // Published even if the signer rotated since signing: peers then reject the block,
                // which is harmless.
                let timestamp = signed.payload.timestamp();
                let selector = |handler: &kona_gossip::BlockHandler| handler.topic(timestamp);
                match self.handler.gossip.publish(selector, signed.payload, signed.signature) {
                    Ok(id) => info!("Published unsafe payload | {:?}", id),
                    Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
                }
                Ok(())
            }
            event = self.handler.gossip.next() => {
                let Some(event) = event else {
                    error!(target: "node::p2p", "The gossip swarm stream has ended");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if let Some(payload) = self.handler.gossip.handle_event(event) &&
                    self.unsafe_block_tx.send(payload).is_err()
                {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            enr = self.handler.enr_receiver.recv() => {
                let Some(enr) = enr else {
                    error!(target: "node::p2p", "The enr receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };
                self.handler.gossip.dial(enr);
                Ok(())
            }
            _ = self.handler.peer_score_inspector.tick(), if self.handler.gossip.peer_monitoring.as_ref().is_some() => {
                self.handler.handle_peer_monitoring().await;
                Ok(())
            }
            Some(NetworkAdminQuery::PostUnsafePayload { payload }) = self.admin_query_rx.recv(), if !self.admin_query_rx.is_closed() => {
                debug!(target: "node::p2p", "Broadcasting unsafe payload from admin api");
                if self.unsafe_block_tx.send(payload).is_err() {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            Some(req) = self.p2p_rpc_rx.recv(), if !self.p2p_rpc_rx.is_closed() => {
                req.handle(&mut self.handler.gossip, &self.handler.discovery);
                Ok(())
            }
        }
    }
}
