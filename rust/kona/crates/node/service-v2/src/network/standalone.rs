//! Standalone networking driver used only by the `kona-node-v2 net` diagnostic command.

use crate::network::NetworkHandler;
use kona_gossip::P2pRpcRequest;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::{mpsc, oneshot};

const REQUEST_CAPACITY: usize = 256;

/// Diagnostic network driver; rollup-node networking is owned privately by Engine instead.
#[derive(Debug)]
pub struct StandaloneNetwork {
    handler: NetworkHandler,
    payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    p2p_rx: mpsc::Receiver<P2pRpcRequest>,
}

impl StandaloneNetwork {
    /// Creates a standalone driver and its P2P RPC request sender.
    pub fn new(
        handler: NetworkHandler,
        payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    ) -> (Self, mpsc::Sender<P2pRpcRequest>) {
        let (p2p_tx, p2p_rx) = mpsc::channel(REQUEST_CAPACITY);
        (Self { handler, payload_tx, p2p_rx }, p2p_tx)
    }

    /// Runs until explicit shutdown or terminal transport failure.
    pub async fn run(
        mut self,
        mut shutdown: oneshot::Receiver<()>,
    ) -> Result<(), StandaloneNetworkError> {
        loop {
            tokio::select! {
                biased;
                _ = &mut shutdown => return Ok(()),
                event = self.handler.gossip.next() => {
                    let event = event.ok_or(StandaloneNetworkError::GossipEnded)?;
                    if let Some(payload) = self.handler.gossip.handle_event(event) {
                        self.payload_tx
                            .send(payload)
                            .await
                            .map_err(|_| StandaloneNetworkError::PayloadReceiverStopped)?;
                    }
                }
                enr = self.handler.enr_receiver.recv() => {
                    let enr = enr.ok_or(StandaloneNetworkError::DiscoveryEnded)?;
                    self.handler.gossip.dial(enr);
                }
                _ = self.handler.peer_score_inspector.tick(),
                    if self.handler.gossip.peer_monitoring.is_some() =>
                {
                    self.handler.handle_peer_monitoring().await;
                }
                request = self.p2p_rx.recv() => {
                    let request = request.ok_or(StandaloneNetworkError::RpcStopped)?;
                    request.handle(&mut self.handler.gossip, &self.handler.discovery);
                }
            }
        }
    }
}

/// Terminal standalone networking failure.
#[derive(Debug, thiserror::Error)]
pub enum StandaloneNetworkError {
    /// Gossip stream ended unexpectedly.
    #[error("gossip stream ended")]
    GossipEnded,
    /// Discovery stream ended unexpectedly.
    #[error("discovery stream ended")]
    DiscoveryEnded,
    /// Diagnostic payload consumer stopped.
    #[error("unsafe payload receiver stopped")]
    PayloadReceiverStopped,
    /// P2P RPC request senders were dropped.
    #[error("P2P RPC channel closed")]
    RpcStopped,
}
