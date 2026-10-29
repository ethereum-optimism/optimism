use std::collections::HashSet;

use discv5::Enr;
use kona_disc::{Discv5Handler, HandlerRequest};
use kona_gossip::{ConnectionGater, GossipDriver};
use tokio::sync::mpsc;

/// A network handler used to communicate with the network once it is started.
#[derive(Debug)]
pub struct NetworkHandler {
    /// The gossip driver.
    pub gossip: GossipDriver<ConnectionGater>,
    /// The discovery service, or `None` if discovery is disabled.
    pub discovery: Option<RunningDiscovery>,
    /// The peer score inspector. Is used to ban peers that are below a given threshold.
    pub peer_score_inspector: tokio::time::Interval,
}

/// A discovery service that was started.
#[derive(Debug)]
pub struct RunningDiscovery {
    /// The handler to send requests to the discovery service.
    pub handler: Discv5Handler,
    /// The receiver of the ENRs it discovers.
    pub enr_receiver: mpsc::Receiver<Enr>,
}

impl NetworkHandler {
    pub(super) async fn handle_peer_monitoring(&mut self) {
        // Inspect peer scores and ban peers that are below the threshold.
        let Some(ban_duration) = self.gossip.peer_monitoring.as_ref().map(|m| m.ban_duration)
        else {
            return;
        };

        // Record the score of every connected peer in the metrics.
        for peer_id in self.gossip.connected_peers() {
            metrics::histogram!(kona_gossip::Metrics::PEER_SCORES)
                .record(self.gossip.peer_score(peer_id));
        }

        // Disconnect the peers, and ban them in discovery too, so it doesn't find them again.
        let peer_ids: HashSet<_> = self.gossip.peers_to_ban().into_iter().collect();
        if peer_ids.is_empty() {
            return;
        }
        for peer_id in &peer_ids {
            self.gossip.ban_peer(*peer_id);
        }

        if let Some(discovery) = &self.discovery &&
            let Err(send_err) = discovery
                .handler
                .sender
                .send(HandlerRequest::BanPeers { peer_ids, ban_duration })
                .await
        {
            warn!(err = ?send_err, "Impossible to send a request to the discovery handler. The channel connection is dropped.");
        }
    }
}
