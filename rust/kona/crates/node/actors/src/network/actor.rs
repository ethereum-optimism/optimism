use super::Discovery;
use async_trait::async_trait;
use discv5::Enr;
use kona_gossip::{
    ConnectionGate, ConnectionGater, GossipCommandReceiver, GossipDriver, GossipQueryHandle,
    GossipState, PEER_SCORE_INSPECT_FREQUENCY,
};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{collections::HashSet, sync::Arc};
use thiserror::Error;
use tokio::{
    self, select,
    sync::{
        mpsc::{self, UnboundedReceiver, UnboundedSender},
        watch,
    },
    time::Interval,
};

use crate::{NetworkEngineClient, NodeActor, signer};

/// The network actor handles two core networking components of the rollup node:
/// - *discovery*: Peer discovery over UDP using discv5.
/// - *gossip*: Block gossip over TCP using libp2p.
#[derive(Debug)]
pub struct NetworkActor<NetworkEngineClient_: NetworkEngineClient, Discovery_: Discovery> {
    /// The gossip driver, whose swarm is listening.
    gossip: GossipDriver<ConnectionGater>,
    /// Bans the addresses of disconnected peers in discovery.
    discovery: Discovery_,
    /// Peers found by discovery, to dial.
    enrs: mpsc::Receiver<Enr>,
    /// Paces the peer score checks that ban peers below the threshold.
    peer_score_inspector: Interval,
    /// Gossip state published for readers outside the actor.
    gossip_state_tx: watch::Sender<Arc<GossipState>>,
    /// A channel to receive gossip commands.
    gossip_command_rx: GossipCommandReceiver,
    /// A channel to receive signed unsafe blocks and publish them through the gossip layer.
    publish_rx: mpsc::Receiver<signer::Payload>,
    /// A client to use to interact with the engine actor.
    engine_client: NetworkEngineClient_,
    // Purely-internal channel: loops gossip-swarm events back into this actor's own select. It
    // never crosses an actor boundary, so it lives here rather than being injected.
    unsafe_block_tx: UnboundedSender<OpExecutionPayloadEnvelope>,
    unsafe_block_rx: UnboundedReceiver<OpExecutionPayloadEnvelope>,
}

impl<NetworkEngineClient_: NetworkEngineClient, Discovery_: Discovery>
    NetworkActor<NetworkEngineClient_, Discovery_>
{
    /// Constructs a new [`NetworkActor`].
    ///
    /// The `gossip` swarm must already be listening, as
    /// [`NetworkConfig::start`](super::NetworkConfig::start) leaves it. `enrs` carries the peers
    /// that discovery finds.
    pub fn new(
        engine_client: NetworkEngineClient_,
        gossip: GossipDriver<ConnectionGater>,
        discovery: Discovery_,
        enrs: mpsc::Receiver<Enr>,
        gossip_command_rx: GossipCommandReceiver,
        publish_rx: mpsc::Receiver<signer::Payload>,
    ) -> Self {
        let (gossip_state_tx, _) = watch::channel(Arc::new(gossip.snapshot()));
        let (unsafe_block_tx, unsafe_block_rx) = mpsc::unbounded_channel();
        Self {
            gossip,
            discovery,
            enrs,
            peer_score_inspector: tokio::time::interval(*PEER_SCORE_INSPECT_FREQUENCY),
            gossip_state_tx,
            gossip_command_rx,
            publish_rx,
            engine_client,
            unsafe_block_tx,
            unsafe_block_rx,
        }
    }

    /// Returns a read-only handle to the actor's published gossip state.
    pub fn gossip_query_handle(&self) -> GossipQueryHandle {
        GossipQueryHandle::new(self.gossip_state_tx.subscribe())
    }

    /// Disconnects the peers whose score is below the ban threshold, and asks discovery to ban
    /// their addresses.
    async fn ban_low_scoring_peers(&mut self) {
        let Some(ban_peers) = self.gossip.peer_monitoring.as_ref() else {
            return;
        };

        let peers_to_remove = self
            .gossip
            .swarm
            .connected_peers()
            .filter_map(|peer_id| {
                // A peer without a score counts as zero.
                let score =
                    self.gossip.swarm.behaviour().gossipsub.peer_score(peer_id).unwrap_or_default();
                metrics::histogram!(kona_gossip::Metrics::PEER_SCORES).record(score);
                (score < ban_peers.ban_threshold).then_some(*peer_id)
            })
            .collect::<Vec<_>>();

        let addrs_to_ban = peers_to_remove
            .into_iter()
            .filter_map(|peer_to_remove| {
                if self.gossip.swarm.disconnect_peer_id(peer_to_remove).is_err() {
                    warn!(peer = ?peer_to_remove, "Trying to disconnect a non-existing peer from the gossip driver.");
                }

                if let Some(start_time) = self.gossip.peer_connection_start.remove(&peer_to_remove)
                {
                    metrics::histogram!(
                        kona_gossip::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
                    )
                    .record(start_time.elapsed().as_secs_f64());
                }

                let info = self.gossip.peerstore.remove(&peer_to_remove)?;
                self.gossip.connection_gate.remove_dial(&peer_to_remove);
                let score = self
                    .gossip
                    .swarm
                    .behaviour()
                    .gossipsub
                    .peer_score(&peer_to_remove)
                    .unwrap_or_default();
                metrics::gauge!(kona_gossip::Metrics::BANNED_PEERS, "peer_id" => peer_to_remove.to_string(), "score" => score.to_string()).increment(1);
                Some(info.listen_addrs)
            })
            .flatten()
            .collect::<HashSet<_>>();

        if let Err(err) = self.discovery.ban(addrs_to_ban, ban_peers.ban_duration).await {
            warn!(?err, "Failed to ask discovery to ban the addresses of low-scoring peers");
        }
    }
}

/// An error from the network actor.
#[derive(Debug, Error)]
pub enum NetworkActorError {
    /// Channel closed unexpectedly.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
}

#[async_trait]
impl<NetworkEngineClient_: NetworkEngineClient + 'static, Discovery_: Discovery + 'static> NodeActor
    for NetworkActor<NetworkEngineClient_, Discovery_>
{
    type Error = NetworkActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let mut applied = None;
        let result = select! {
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
                match self.gossip.publish(selector, signed.payload, signed.signature) {
                    Ok(id) => info!("Published unsafe payload | {:?}", id),
                    Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
                }
                Ok(())
            }
            event = self.gossip.next() => {
                let Some(event) = event else {
                    error!(target: "node::p2p", "The gossip swarm stream has ended");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if let Some(payload) = self.gossip.handle_event(event) &&
                    self.unsafe_block_tx.send(payload).is_err()
                {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            enr = self.enrs.recv() => {
                let Some(enr) = enr else {
                    error!(target: "node::p2p", "The enr receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };
                self.gossip.dial(enr);
                Ok(())
            }
            _ = self.peer_score_inspector.tick(), if self.gossip.peer_monitoring.is_some() => {
                self.ban_low_scoring_peers().await;
                Ok(())
            }
            Some((req, applied_tx)) = self.gossip_command_rx.recv(), if !self.gossip_command_rx.is_closed() => {
                req.handle(&mut self.gossip);
                applied = Some(applied_tx);
                Ok(())
            }
        };
        if result.is_ok() && self.gossip_state_tx.receiver_count() > 0 {
            self.gossip_state_tx.send_replace(Arc::new(self.gossip.snapshot()));
        }
        if let Some(applied) = applied {
            let _ = applied.send(());
        }
        result
    }
}
