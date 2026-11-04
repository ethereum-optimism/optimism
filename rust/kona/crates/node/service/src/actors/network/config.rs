//! Configuration for the `Network`.

use alloy_primitives::Address;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_gossip::{GaterConfig, PeerAddr};
use kona_peers::{BootNodes, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use kona_sources::BlockSigner;
use libp2p::identity::Keypair;
use std::{fmt, str::FromStr};
use tokio::time::Duration;

/// Configuration for kona's P2P stack.
#[derive(Debug, Clone)]
pub struct NetworkConfig {
    /// Discovery Config.
    pub discovery_config: discv5::Config,
    /// The local node's advertised address to external peers.
    /// Note: This may be different from the node's discovery listen address.
    pub discovery_address: LocalNode,
    /// The interval to find peers.
    pub discovery_interval: Duration,
    /// The interval to remove peers from the discovery service.
    pub discovery_randomize: Option<Duration>,
    /// Whether to run discovery. Without it, the node binds no UDP socket, and connects only to
    /// its static peers, the peers that dial it, and the peers added with `opp2p_connectPeer`.
    pub discovery_enabled: bool,
    /// The gossip address.
    pub gossip_address: libp2p::Multiaddr,
    /// The unsafe block signer.
    pub unsafe_block_signer: Address,
    /// The keypair.
    pub keypair: Keypair,
    /// The gossip config.
    pub gossip_config: libp2p::gossipsub::Config,
    /// The peer score level.
    pub scoring: PeerScoreLevel,
    /// Whether to enable topic scoring.
    pub topic_scoring: bool,
    /// Peer score monitoring config.
    pub monitor_peers: Option<PeerMonitoring>,
    /// An optional path to the bootstore.
    pub bootstore: Option<BootStoreFile>,
    /// The configuration for the connection gater.
    pub gater_config: GaterConfig,
    /// An optional list of bootnode ENRs to start the node with.
    pub bootnodes: BootNodes,
    /// The peers to dial at startup and keep connected.
    pub static_peers: Vec<PeerAddr>,
    /// The [`RollupConfig`].
    pub rollup_config: RollupConfig,
    /// Signs the sequencer's blocks for gossip. Required in sequencer mode. It is used by the
    /// [`SignerActor`](crate::SignerActor), not by the network itself.
    pub gossip_signer: Option<BlockSigner>,
}

/// How many peers must report the same external address before discovery advertises it. At least
/// 2, below which discv5 panics.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct IpVotes(usize);

impl IpVotes {
    /// The default: discv5's, and op-node's, which is a constant.
    pub const DEFAULT: Self = Self(10);

    /// Returns `votes`, if it is at least 2.
    pub const fn new(votes: usize) -> Option<Self> {
        if votes < 2 { None } else { Some(Self(votes)) }
    }

    /// Returns the number of votes.
    pub const fn get(self) -> usize {
        self.0
    }
}

impl FromStr for IpVotes {
    type Err = String;

    fn from_str(raw: &str) -> Result<Self, Self::Err> {
        let votes: usize = raw.parse().map_err(|e| format!("invalid number of votes: {e}"))?;
        Self::new(votes).ok_or_else(|| format!("needs at least 2 votes, got {votes}"))
    }
}

impl fmt::Display for IpVotes {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(f)
    }
}

impl NetworkConfig {
    /// Returns the [`discv5::Config`] from the CLI arguments.
    ///
    /// With `ip_votes`, discovery updates the ENR's address once that many peers report the same
    /// external address. Without, the ENR's address stays as it was built, for a static IP.
    pub fn discv5_config(
        listen_config: discv5::ListenConfig,
        ip_votes: Option<IpVotes>,
    ) -> discv5::Config {
        // Discovery binds `listen_config` as is, except that a port of 0 is bound to a free port
        // by `Discv5Builder::build`, so that the ENR can advertise it.
        let mut builder = discv5::ConfigBuilder::new(listen_config);

        // When a node sees too few new inbound sessions, discv5's auto-NAT strips `ip`/`udp` from
        // the ENR, leaving `tcp` without an address, and ignores IP votes for 6h, which takes
        // reachable nodes off the network. op-node has no equivalent, and discv5 before 0.12
        // never ran it.
        builder.auto_nat_listen_duration(None);

        match ip_votes {
            Some(votes) => {
                builder.enr_peer_update_min(votes.get());
            }
            None => {
                builder.disable_enr_update();
            }
        }

        builder.build()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::{IpAddr, Ipv4Addr};

    fn listen() -> discv5::ListenConfig {
        discv5::ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::LOCALHOST), 0)
    }

    /// discv5 panics below 2 votes, so they can't be made, nor parsed.
    #[test]
    fn ip_votes_are_at_least_2() {
        assert_eq!(IpVotes::new(0), None);
        assert_eq!(IpVotes::new(1), None);
        assert_eq!(IpVotes::new(2).map(IpVotes::get), Some(2));
        assert_eq!("10".parse(), Ok(IpVotes::DEFAULT));
        assert!("1".parse::<IpVotes>().is_err());
        assert!("two".parse::<IpVotes>().is_err());
    }

    #[test]
    fn discv5_config_disables_auto_nat() {
        for ip_votes in [Some(IpVotes::DEFAULT), None] {
            let config = NetworkConfig::discv5_config(listen(), ip_votes);
            assert_eq!(config.auto_nat_listen_duration, None, "ip_votes = {ip_votes:?}");
        }
    }

    /// With votes, the ENR's address is updated once that many peers agree on it. Without, it is
    /// never updated.
    #[test]
    fn discv5_config_sets_ip_votes() {
        let config = NetworkConfig::discv5_config(listen(), IpVotes::new(3));
        assert!(config.enr_update);
        assert_eq!(config.enr_peer_update_min, 3);

        let config = NetworkConfig::discv5_config(listen(), None);
        assert!(!config.enr_update);
    }
}
