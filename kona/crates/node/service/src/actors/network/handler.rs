use std::collections::HashSet;

use alloy_primitives::Address;
use discv5::Enr;
use kona_disc::{Discv5Handler, HandlerRequest};
use kona_gossip::{ConnectionGater, GossipDriver};
use kona_sources::BlockSignerHandler;
use tokio::sync::{mpsc, watch};

/// A network handler used to communicate with the network once it is started.
#[derive(Debug)]
pub struct NetworkHandler {
    /// The gossip driver.
    pub gossip: GossipDriver<ConnectionGater>,
    /// The discovery handler.
    pub discovery: Discv5Handler,
    /// The receiver for the ENRs.
    pub enr_receiver: mpsc::Receiver<Enr>,
    /// The sender for the unsafe block signer.
    pub unsafe_block_signer_sender: watch::Sender<Address>,
    /// The peer score inspector. Is used to ban peers that are below a given threshold.
    pub peer_score_inspector: tokio::time::Interval,
    /// A handler for the block signer.
    pub signer: Option<BlockSignerHandler>,
}

impl NetworkHandler {
    pub(super) async fn handle_peer_monitoring(&mut self) {
        // Inspect peer scores and ban peers that are below the threshold.
        let Some(ban_peers) = self.gossip.peer_monitoring.as_ref() else {
            return;
        };

        // We iterate over all connected peers and check their scores.
        // We collect a list of peers to remove
        let peers_to_remove = self
            .gossip
            .swarm
            .connected_peers()
            .filter_map(|peer_id| {
                // If the score is not available, we use a default value of 0.
                let score =
                    self.gossip.swarm.behaviour().gossipsub.peer_score(peer_id).unwrap_or_default();

                // Record the peer score in the metrics.
                kona_macros::record!(
                    histogram,
                    kona_gossip::Metrics::PEER_SCORES,
                    "peer",
                    peer_id.to_string(),
                    score
                );

                if score < ban_peers.ban_threshold {
                    return Some(*peer_id);
                }

                None
            })
            .collect::<Vec<_>>();

        // We remove the addresses from the gossip layer.
        let addrs_to_ban = peers_to_remove.into_iter().filter_map(|peer_to_remove| {
                        // In that case, we ban the peer. This means...
                        // 1. We remove the peer from the network gossip.
                        // 2. We ban the peer from the discv5 service.
                        if self.gossip.swarm.disconnect_peer_id(peer_to_remove).is_err() {
                            warn!(peer = ?peer_to_remove, "Trying to disconnect a non-existing peer from the gossip driver.");
                        }

                        // Record the duration of the peer connection.
                        if let Some(start_time) = self.gossip.peer_connection_start.remove(&peer_to_remove) {
                            let peer_duration = start_time.elapsed();
                            kona_macros::record!(
                                histogram,
                                kona_gossip::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS,
                                peer_duration.as_secs_f64()
                            );
                        }

                        if let Some(info) = self.gossip.peerstore.remove(&peer_to_remove){
                            use kona_gossip::ConnectionGate;
                            self.gossip.connection_gate.remove_dial(&peer_to_remove);
                            let score = self.gossip.swarm.behaviour().gossipsub.peer_score(&peer_to_remove).unwrap_or_default();
                            kona_macros::inc!(gauge, kona_gossip::Metrics::BANNED_PEERS, "peer_id" => peer_to_remove.to_string(), "score" => score.to_string());
                            return Some(info.listen_addrs);
                        }

                        None
                    }).collect::<Vec<_>>().into_iter().flatten().collect::<HashSet<_>>();

        // We send a request to the discovery handler to ban the set of addresses.
        if let Err(send_err) = self
            .discovery
            .sender
            .send(HandlerRequest::BanAddrs {
                addrs_to_ban: addrs_to_ban.into(),
                ban_duration: ban_peers.ban_duration,
            })
            .await
        {
            warn!(err = ?send_err, "Impossible to send a request to the discovery handler. The channel connection is dropped.");
        }
    }
}
