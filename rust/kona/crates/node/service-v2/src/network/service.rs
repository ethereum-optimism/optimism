//! Long-running P2P network service.

use crate::network::handler::NetworkHandler;
use alloy_primitives::Address;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot};
use tokio_util::sync::CancellationToken;

const REQUEST_CAPACITY: usize = 256;

/// Cloneable network capabilities exposed to other node services.
#[derive(Debug, Clone)]
pub struct NetworkClient {
    publish_tx: mpsc::Sender<PublishRequest>,
    p2p_tx: mpsc::Sender<P2pRpcRequest>,
    admin_tx: mpsc::Sender<NetworkAdminQuery>,
}

impl NetworkClient {
    /// Publishes an authorized locally built payload and waits for the gossip attempt to finish.
    pub async fn publish_unsafe(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), NetworkClientError> {
        let (response, result) = oneshot::channel();
        self.publish_tx
            .send(PublishRequest { payload: Box::new(payload), response })
            .await
            .map_err(|_| NetworkClientError::Unavailable)?;
        result.await.map_err(|_| NetworkClientError::ResponseDropped)?
    }

    /// Returns the request sender used by the P2P RPC namespace.
    pub fn p2p_sender(&self) -> mpsc::Sender<P2pRpcRequest> {
        self.p2p_tx.clone()
    }

    /// Returns the request sender used by the admin payload-injection method.
    pub fn admin_sender(&self) -> mpsc::Sender<NetworkAdminQuery> {
        self.admin_tx.clone()
    }

    #[cfg(test)]
    pub(crate) fn test_pair(capacity: usize) -> (Self, mpsc::Receiver<PublishRequest>) {
        let (publish_tx, publish_rx) = mpsc::channel(capacity);
        let (p2p_tx, _) = mpsc::channel(1);
        let (admin_tx, _) = mpsc::channel(1);
        (Self { publish_tx, p2p_tx, admin_tx }, publish_rx)
    }
}

#[derive(Debug)]
pub(crate) struct PublishRequest {
    pub(crate) payload: Box<OpExecutionPayloadEnvelope>,
    pub(crate) response: oneshot::Sender<Result<(), NetworkClientError>>,
}

/// Owns discovery, gossip, peer monitoring, signing, and network RPC requests.
#[derive(Debug)]
pub struct NetworkService {
    handler: NetworkHandler,
    signer_rx: mpsc::Receiver<Address>,
    inbound_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    publish_rx: mpsc::Receiver<PublishRequest>,
    p2p_rx: mpsc::Receiver<P2pRpcRequest>,
    admin_rx: mpsc::Receiver<NetworkAdminQuery>,
}

impl NetworkService {
    /// Creates a network service around an already-started network stack.
    pub fn new(
        handler: NetworkHandler,
        signer_rx: mpsc::Receiver<Address>,
        inbound_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    ) -> (Self, NetworkClient) {
        let (publish_tx, publish_rx) = mpsc::channel(REQUEST_CAPACITY);
        let (p2p_tx, p2p_rx) = mpsc::channel(REQUEST_CAPACITY);
        let (admin_tx, admin_rx) = mpsc::channel(REQUEST_CAPACITY);
        (
            Self { handler, signer_rx, inbound_payload_tx, publish_rx, p2p_rx, admin_rx },
            NetworkClient { publish_tx, p2p_tx, admin_tx },
        )
    }

    /// Runs the network until shutdown or a terminal transport failure.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), NetworkServiceError> {
        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                signer = self.signer_rx.recv() => {
                    let signer = signer.ok_or(NetworkServiceError::SignerChannelClosed)?;
                    if self.handler.unsafe_block_signer_sender.send(signer).is_err() {
                        warn!(target: "network", "No gossip signer receiver accepted the update");
                    }
                }
                request = self.publish_rx.recv() => {
                    let request = request.ok_or(NetworkServiceError::PublishChannelClosed)?;
                    let result = self.publish(*request.payload).await;
                    let _ = request.response.send(result);
                }
                event = self.handler.gossip.next() => {
                    let event = event.ok_or(NetworkServiceError::GossipEnded)?;
                    if let Some(payload) = self.handler.gossip.handle_event(event) {
                        self.inbound_payload_tx
                            .send(payload)
                            .await
                            .map_err(|_| NetworkServiceError::UnsafeChainStopped)?;
                    }
                }
                enr = self.handler.enr_receiver.recv() => {
                    let enr = enr.ok_or(NetworkServiceError::DiscoveryEnded)?;
                    self.handler.gossip.dial(enr);
                }
                _ = self.handler.peer_score_inspector.tick(),
                    if self.handler.gossip.peer_monitoring.is_some() =>
                {
                    self.handler.handle_peer_monitoring().await;
                }
                request = self.admin_rx.recv() => {
                    let request = request.ok_or(NetworkServiceError::AdminChannelClosed)?;
                    match request {
                        NetworkAdminQuery::PostUnsafePayload { payload } => {
                            self.inbound_payload_tx
                                .send(payload)
                                .await
                                .map_err(|_| NetworkServiceError::UnsafeChainStopped)?;
                        }
                    }
                }
                request = self.p2p_rx.recv() => {
                    let request = request.ok_or(NetworkServiceError::P2pChannelClosed)?;
                    request.handle(&mut self.handler.gossip, &self.handler.discovery);
                }
            }
        }
    }

    async fn publish(
        &mut self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), NetworkClientError> {
        let signer = self.handler.signer.as_ref().ok_or(NetworkClientError::SignerUnavailable)?;
        let sender = *self.handler.unsafe_block_signer_sender.borrow();
        let signature = signer
            .sign_block(payload.payload_hash(), self.handler.discovery.chain_id, sender)
            .await
            .map_err(|error| NetworkClientError::Signing(error.to_string()))?;
        let timestamp = payload.timestamp();
        match self.handler.gossip.publish(|handler| handler.topic(timestamp), payload, signature) {
            Ok(_) |
            Err(kona_gossip::PublishError::PublishError(
                libp2p::gossipsub::PublishError::Duplicate |
                libp2p::gossipsub::PublishError::NoPeersSubscribedToTopic,
            )) => Ok(()),
            Err(error) => Err(NetworkClientError::Publish(error.to_string())),
        }
    }
}

/// A publication error visible to unsafe-chain workflows.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum NetworkClientError {
    /// Network service is unavailable.
    #[error("network service is unavailable")]
    Unavailable,
    /// Publication may have completed, but its acknowledgement was dropped.
    #[error("network publication response was dropped")]
    ResponseDropped,
    /// No local payload signer is configured.
    #[error("local payload signer is unavailable")]
    SignerUnavailable,
    /// Payload signing failed.
    #[error("payload signing failed: {0}")]
    Signing(String),
    /// Gossip publication failed.
    #[error("gossip publication failed: {0}")]
    Publish(String),
}

/// Terminal network service failure.
#[derive(Debug, Error)]
pub enum NetworkServiceError {
    /// Gossip stream ended unexpectedly.
    #[error("gossip stream ended")]
    GossipEnded,
    /// Discovery stream ended unexpectedly.
    #[error("discovery stream ended")]
    DiscoveryEnded,
    /// Unsafe-chain service stopped receiving payloads.
    #[error("unsafe-chain payload channel closed")]
    UnsafeChainStopped,
    /// L1 signer-update producer stopped unexpectedly.
    #[error("unsafe signer update channel closed")]
    SignerChannelClosed,
    /// Every publication client was dropped.
    #[error("network publication channel closed")]
    PublishChannelClosed,
    /// P2P RPC channel closed.
    #[error("P2P RPC channel closed")]
    P2pChannelClosed,
    /// Admin RPC channel closed.
    #[error("network admin channel closed")]
    AdminChannelClosed,
}
