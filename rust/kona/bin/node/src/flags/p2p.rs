//! P2P CLI Flags
//!
//! These are based on p2p flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/p2p_flags.go

use anyhow::Result;
use clap::{ArgAction, Parser};
use enr::k256;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_gossip::GaterConfig;
use kona_node_service::NetworkConfig;
use kona_peers::{BootNode, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use libp2p::identity::Keypair;
use std::{
    net::{IpAddr, SocketAddr, ToSocketAddrs},
    path::PathBuf,
};
use tokio::time::Duration;

/// Resolves a hostname or IP address string to an [`IpAddr`].
///
/// Accepts either:
/// - A valid IP address string (e.g., "127.0.0.1", "`::1`")
/// - A DNS hostname (e.g., "node1.example.com")
///
/// For DNS hostnames, this performs synchronous DNS resolution and returns the first
/// resolved IP address.
fn resolve_host(host: &str) -> Result<IpAddr, String> {
    // First, try to parse as a direct IP address
    if let Ok(ip) = host.parse::<IpAddr>() {
        return Ok(ip);
    }

    // If that fails, try DNS resolution
    // We append a port to make it a valid socket address for resolution
    let socket_addr = format!("{host}:0");
    match socket_addr.to_socket_addrs() {
        Ok(mut addrs) => addrs
            .next()
            .map(|addr| addr.ip())
            .ok_or_else(|| format!("DNS resolution for '{host}' returned no addresses")),
        Err(e) => Err(format!("Failed to resolve '{host}': {e}")),
    }
}

/// P2P CLI Flags
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "P2P")]
pub struct P2PArgs {
    /// Read the hex-encoded 32-byte private key for the peer ID from this txt file.
    /// Created if not already exists. Important to persist to keep the same network identity after
    /// restarting, maintaining the previous advertised identity.
    #[arg(long = "p2p.priv-path", env = "KONA_NODE_P2P_PRIV_PATH", hide = true)]
    pub priv_path: Option<PathBuf>,

    /// IP address or DNS hostname to advertise to external peers from Discv5.
    /// Optional argument. Use the `p2p.listen.addr` if not set.
    /// Accepts either an IP address (e.g., "1.2.3.4") or a DNS hostname (e.g.,
    /// "node1.example.com"). DNS hostnames are resolved to IP addresses at startup.
    ///
    /// Technical note: if this argument is set, the dynamic ENR updates from the discovery layer
    /// will be disabled. This is to allow the advertised IP to be static (to use in a network
    /// behind a NAT for instance).
    #[arg(long = "p2p.advertise.addr", env = "KONA_NODE_P2P_ADVERTISE_ADDR", value_parser = resolve_host, hide = true)]
    pub advertise_ip: Option<IpAddr>,
    /// TCP port to advertise to external peers from the discovery layer. Same as
    /// `p2p.listen.tcp-port` if set to zero.
    #[arg(long = "p2p.advertise.tcp-port", env = "KONA_NODE_P2P_ADVERTISE_TCP_PORT", hide = true)]
    pub advertise_tcp_port: Option<u16>,
    /// UDP port to advertise to external peers from the discovery layer.
    /// Same as `p2p.listen.udp-port` if set to zero.
    #[arg(long = "p2p.advertise.udp-port", env = "KONA_NODE_P2P_ADVERTISE_UDP_PORT", hide = true)]
    pub advertise_udp_port: Option<u16>,

    /// IP address or DNS hostname to bind LibP2P/Discv5 to.
    /// Accepts either an IP address (e.g., "0.0.0.0") or a DNS hostname (e.g.,
    /// "node1.example.com"). DNS hostnames are resolved to IP addresses at startup.
    #[arg(long = "p2p.listen.addr", default_value = "0.0.0.0", env = "KONA_NODE_P2P_LISTEN_ADDR", value_parser = resolve_host, hide = true)]
    pub listen_ip: IpAddr,
    /// TCP port to bind `LibP2P` to. Any available system port if set to 0.
    #[arg(
        long = "p2p.listen.tcp-port",
        default_value = "9222",
        env = "KONA_NODE_P2P_LISTEN_TCP_PORT",
        hide = true
    )]
    pub listen_tcp_port: u16,
    /// UDP port to bind Discv5 to. Same as TCP port if left 0.
    #[arg(
        long = "p2p.listen.udp-port",
        default_value = "9223",
        env = "KONA_NODE_P2P_LISTEN_UDP_PORT",
        hide = true
    )]
    pub listen_udp_port: u16,
    /// Configure `GossipSub` topic stable mesh target count.
    /// Aka: The desired outbound degree (numbers of peers to gossip to).
    #[arg(
        long = "p2p.gossip.mesh.d",
        default_value = "8",
        env = "KONA_NODE_P2P_GOSSIP_MESH_D",
        hide = true
    )]
    pub gossip_mesh_d: usize,
    /// Configure `GossipSub` topic stable mesh low watermark.
    /// Aka: The lower bound of outbound degree.
    #[arg(
        long = "p2p.gossip.mesh.lo",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_LO",
        hide = true
    )]
    pub gossip_mesh_dlo: usize,
    /// Configure `GossipSub` topic stable mesh high watermark.
    /// Aka: The upper bound of outbound degree (additional peers will not receive gossip).
    #[arg(
        long = "p2p.gossip.mesh.dhi",
        default_value = "12",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DHI",
        hide = true
    )]
    pub gossip_mesh_dhi: usize,
    /// Configure `GossipSub` gossip target.
    /// Aka: The target degree for gossip only (not messaging like p2p.gossip.mesh.d, just
    /// announcements of IHAVE).
    #[arg(
        long = "p2p.gossip.mesh.dlazy",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DLAZY",
        hide = true
    )]
    pub gossip_mesh_dlazy: usize,
    /// Configure `GossipSub` to publish messages to all known peers on the topic, outside of the
    /// mesh. Also see Dlazy as less aggressive alternative.
    #[arg(
        id = "p2p_flood_publish_enabled",
        long = "p2p.gossip.mesh.flood-publish.enabled",
        env = "KONA_NODE_P2P_GOSSIP_MESH_FLOOD_PUBLISH_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        hide = true
    )]
    pub gossip_flood_publish: bool,
    /// Sets the peer scoring strategy for the P2P stack.
    /// Can be one of: none or light.
    #[arg(
        long = "p2p.peer-scoring",
        default_value = "light",
        env = "KONA_NODE_P2P_PEER_SCORING",
        hide = true
    )]
    pub scoring: PeerScoreLevel,

    /// Allows to ban peers based on their score.
    ///
    /// Peers are banned based on a ban threshold (see `p2p.ban.threshold`).
    /// If a peer's score is below the threshold, it gets automatically banned.
    #[arg(
        id = "p2p_ban_enabled",
        long = "p2p.ban.enabled",
        env = "KONA_NODE_P2P_BAN_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        hide = true
    )]
    pub ban_enabled: bool,

    /// The threshold used to ban peers.
    ///
    /// For peers to be banned, the `p2p.ban.enabled` flag must be set.
    /// By default, peers are banned if their score is below -100. This follows the `op-node` default `<https://github.com/ethereum-optimism/optimism/blob/09a8351a72e43647c8a96f98c16bb60e7b25dc6e/op-node/flags/p2p_flags.go#L123-L130>`.
    #[arg(
        long = "p2p.ban.threshold",
        default_value = "-100",
        env = "KONA_NODE_P2P_BAN_THRESHOLD",
        hide = true
    )]
    pub ban_threshold: i64,

    /// How long to ban a peer for, such as `30m` or `1h`.
    ///
    /// For peers to be banned, the `p2p.ban.enabled` flag must be set.
    /// By default peers are banned for 1 hour. This follows the `op-node` default `<https://github.com/ethereum-optimism/optimism/blob/09a8351a72e43647c8a96f98c16bb60e7b25dc6e/op-node/flags/p2p_flags.go#L131-L138>`.
    #[arg(
        long = "p2p.ban.duration",
        default_value = "1h",
        env = "KONA_NODE_P2P_BAN_DURATION",
        value_parser = humantime::parse_duration,
        hide = true
    )]
    pub ban_duration: Duration,

    /// Find peers with discv5. Disabled by default: the node then keeps only the peers it is told
    /// to connect to, and those that connect to it, and the other discovery flags are ignored.
    #[arg(
        id = "p2p_discovery_enabled",
        long = "p2p.discovery.enabled",
        env = "KONA_NODE_P2P_DISCOVERY_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        hide = true
    )]
    pub discovery_enabled: bool,

    /// How often to find peers using the discovery service, such as `5s`.
    #[arg(
        long = "p2p.discovery.interval",
        default_value = "5s",
        env = "KONA_NODE_P2P_DISCOVERY_INTERVAL",
        value_parser = humantime::parse_duration,
        hide = true
    )]
    pub discovery_interval: Duration,
    /// Keep the peers discovery finds in a bootstore, so that the node finds them again after a
    /// restart. Disabled by default, and ignored without discovery.
    #[arg(
        id = "p2p_bootstore_enabled",
        long = "p2p.discovery.bootstore.enabled",
        env = "KONA_NODE_P2P_DISCOVERY_BOOTSTORE_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        hide = true
    )]
    pub bootstore_enabled: bool,
    /// The bootstore's file. By default, `~/.kona/<chain_id>/bootstore.json`.
    #[arg(
        long = "p2p.discovery.bootstore.path",
        env = "KONA_NODE_P2P_DISCOVERY_BOOTSTORE_PATH",
        hide = true
    )]
    pub bootstore: Option<PathBuf>,
    /// Peer Redialing threshold is the maximum amount of times to attempt to redial a peer that
    /// disconnects. By default, peers are *not* redialed. If set to 0, the peer will be
    /// redialed indefinitely.
    #[arg(
        long = "p2p.redial.max-attempts",
        env = "KONA_NODE_P2P_REDIAL_MAX_ATTEMPTS",
        default_value = "500",
        hide = true
    )]
    pub peer_redial: Option<u64>,

    /// The peer dial period, such as `1h`.
    /// When the last time a peer was dialed is longer than the dial period, the number of peer
    /// dials is reset to 0, allowing the peer to be dialed again.
    #[arg(
        long = "p2p.redial.period",
        env = "KONA_NODE_P2P_REDIAL_PERIOD",
        default_value = "1h",
        value_parser = humantime::parse_duration,
        hide = true
    )]
    pub redial_period: Duration,

    /// An optional list of bootnode ENRs or node records to start discovery with.
    #[arg(
        long = "p2p.discovery.bootnodes",
        value_delimiter = ',',
        env = "KONA_NODE_P2P_DISCOVERY_BOOTNODES",
        hide = true
    )]
    pub bootnodes: Vec<String>,

    /// Optionally enable topic scoring.
    ///
    /// Topic scoring is a mechanism to score peers based on their behavior in the gossip network.
    /// Historically, topic scoring was only enabled for the v1 topic on the OP Stack p2p network
    /// in the `op-node`. This was a silent bug, and topic scoring is actively being
    /// [phased out of the `op-node`][out].
    ///
    /// This flag is only presented for backwards compatibility and debugging purposes.
    ///
    /// [out]: https://github.com/ethereum-optimism/optimism/pull/15719
    #[arg(
        id = "p2p_topic_scoring_enabled",
        long = "p2p.topic-scoring.enabled",
        env = "KONA_NODE_P2P_TOPIC_SCORING_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        hide = true
    )]
    pub topic_scoring: bool,

    /// An optional flag to remove random peers from discovery to rotate the peer set.
    ///
    /// This is how long to wait before removing a peer from the discovery service, such as `30s`.
    /// By default, peers are not removed from the discovery service.
    ///
    /// This is useful for discovering a wider set of peers.
    #[arg(
        long = "p2p.discovery.randomize",
        env = "KONA_NODE_P2P_DISCOVERY_RANDOMIZE",
        value_parser = humantime::parse_duration,
        hide = true
    )]
    pub discovery_randomize: Option<Duration>,
}

impl Default for P2PArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl P2PArgs {
    /// Constructs kona's P2P network [`NetworkConfig`] from CLI arguments.
    pub fn config(self, config: &RollupConfig) -> Result<NetworkConfig> {
        // Note: the advertised address is contained in the ENR for external peers from the
        // discovery layer to use.

        // Fallback to the listen ip if the advertise ip is not specified
        let advertise_ip = self.advertise_ip.unwrap_or(self.listen_ip);

        // If the advertise ip is set, we will disable the dynamic ENR updates.
        let static_ip = self.advertise_ip.is_some();

        // If the advertise tcp port is null, use the listen tcp port
        let advertise_tcp_port = match self.advertise_tcp_port {
            None => self.listen_tcp_port,
            Some(port) => port,
        };

        let advertise_udp_port = match self.advertise_udp_port {
            None => self.listen_udp_port,
            Some(port) => port,
        };

        let keypair = self.keypair().unwrap_or_else(|e| {
            let generated = Keypair::generate_secp256k1();
            tracing::warn!(
                target: "p2p::config",
                error = %e,
                peer_id = %generated.public().to_peer_id(),
                "Failed to load P2P keypair from configuration, generated ephemeral keypair. \
                 Set --p2p.priv-path for a persistent peer ID."
            );
            generated
        });
        let secp256k1_key = keypair.clone().try_into_secp256k1()
            .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to secp256k1. This is a bug since we only support secp256k1 keys: {e}"))?
            .secret().to_bytes();
        let local_node_key = k256::ecdsa::SigningKey::from_bytes(&secp256k1_key.into())
            .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to k256 signing key. This is a bug since we only support secp256k1 keys: {e}"))?;

        let discovery_address =
            LocalNode::new(local_node_key, advertise_ip, advertise_tcp_port, advertise_udp_port);
        let gossip_config = kona_gossip::default_config_builder()
            .mesh_n(self.gossip_mesh_d)
            .mesh_n_low(self.gossip_mesh_dlo)
            .mesh_n_high(self.gossip_mesh_dhi)
            .gossip_lazy(self.gossip_mesh_dlazy)
            .flood_publish(self.gossip_flood_publish)
            .build()?;

        let monitor_peers = self.ban_enabled.then_some(PeerMonitoring {
            ban_duration: self.ban_duration,
            ban_threshold: self.ban_threshold as f64,
        });

        let discovery_listening_address = SocketAddr::new(self.listen_ip, self.listen_udp_port);
        let discovery_config =
            NetworkConfig::discv5_config(discovery_listening_address.into(), static_ip);

        let mut gossip_address = libp2p::Multiaddr::from(self.listen_ip);
        gossip_address.push(libp2p::multiaddr::Protocol::Tcp(self.listen_tcp_port));

        let bootstore = self.bootstore_file(config);

        let bootnodes = self
            .bootnodes
            .iter()
            .map(|bootnode| BootNode::parse_bootnode(bootnode))
            .collect::<Vec<BootNode>>()
            .into();

        Ok(NetworkConfig {
            discovery_config,
            discovery_interval: self.discovery_interval,
            discovery_address,
            discovery_randomize: self.discovery_randomize,
            enr_update: !static_ip,
            gossip_address,
            keypair,
            gossip_config,
            scoring: self.scoring,
            monitor_peers,
            bootstore,
            discovery_enabled: self.discovery_enabled,
            topic_scoring: self.topic_scoring,
            gater_config: GaterConfig {
                peer_redialing: self.peer_redial,
                dial_period: self.redial_period,
            },
            bootnodes,
            rollup_config: config.clone(),
        })
    }

    /// Returns the bootstore file to use for the rollup's L2 chain, or `None` if the bootstore is
    /// disabled.
    fn bootstore_file(&self, config: &RollupConfig) -> Option<BootStoreFile> {
        if !self.bootstore_enabled {
            return None;
        }
        let chain_id = config.l2_chain_id.id();
        Some(
            self.bootstore
                .clone()
                .map_or(BootStoreFile::Default { chain_id }, BootStoreFile::Custom),
        )
    }

    /// Returns the [`Keypair`] read from `--p2p.priv-path`, which is created with a new key if it
    /// does not exist yet.
    ///
    /// Fails if no key file is given.
    pub fn keypair(&self) -> Result<Keypair> {
        let Some(ref key_path) = self.priv_path else {
            anyhow::bail!("No private key file path was provided.");
        };

        kona_cli::SecretKeyLoader::load(key_path).map_err(|e| anyhow::anyhow!(e))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::b256;
    use clap::Parser;
    use kona_peers::NodeRecord;
    use std::str::FromStr;

    /// A mock command that uses the `P2PArgs`.
    #[derive(Parser, Debug, Clone)]
    #[command(about = "Mock command")]
    struct MockCommand {
        /// P2P CLI Flags
        #[clap(flatten)]
        pub p2p: P2PArgs,
    }

    #[test]
    fn test_p2p_args_keypair_missing_both() {
        let args = MockCommand::parse_from(["test"]);
        assert!(args.p2p.keypair().is_err());
    }

    #[test]
    fn test_p2p_args_keypair_from_path() {
        // Create a temporary directory.
        let dir = std::env::temp_dir();
        let mut source_path = dir.clone();
        assert!(std::env::set_current_dir(dir).is_ok());

        // Write a private key to a file.
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        let hex = alloy_primitives::hex::encode(key.0);
        source_path.push("test.txt");
        std::fs::write(&source_path, &hex).unwrap();

        // Parse the keypair from the file.
        let args =
            MockCommand::parse_from(["test", "--p2p.priv-path", source_path.to_str().unwrap()]);
        assert!(args.p2p.keypair().is_ok());
    }

    #[test]
    fn test_p2p_args() {
        let args = MockCommand::parse_from(["test"]);
        assert_eq!(args.p2p, P2PArgs::default());
    }

    #[test]
    fn test_p2p_args_bootstore_file() {
        let config =
            RollupConfig { l2_chain_id: alloy_chains::Chain::from_id(8453), ..Default::default() };

        let args = MockCommand::parse_from(["test"]);
        assert_eq!(args.p2p.bootstore_file(&config), None);

        let args = MockCommand::parse_from(["test", "--p2p.discovery.bootstore.enabled"]);
        assert_eq!(
            args.p2p.bootstore_file(&config),
            Some(BootStoreFile::Default { chain_id: 8453 })
        );

        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.bootstore.enabled",
            "--p2p.discovery.bootstore.path",
            "/tmp/bootstore.json",
        ]);
        assert_eq!(
            args.p2p.bootstore_file(&config),
            Some(BootStoreFile::Custom(PathBuf::from("/tmp/bootstore.json")))
        );

        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.bootstore.path",
            "/tmp/bootstore.json",
        ]);
        assert_eq!(args.p2p.bootstore_file(&config), None);
    }

    #[test]
    fn test_p2p_args_randomized() {
        let args = MockCommand::parse_from(["test", "--p2p.discovery.randomize", "10s"]);
        assert_eq!(args.p2p.discovery_randomize, Some(Duration::from_secs(10)));
        let args = MockCommand::parse_from(["test"]);
        assert_eq!(args.p2p.discovery_randomize, None);
    }

    #[test]
    fn test_p2p_args_priv_path() {
        let args = MockCommand::parse_from(["test", "--p2p.priv-path", "test.txt"]);
        assert_eq!(args.p2p.priv_path, Some(PathBuf::from("test.txt")));
    }

    #[test]
    fn test_p2p_args_listen_ip() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.addr", "127.0.0.1"]);
        let expected: IpAddr = "127.0.0.1".parse().unwrap();
        assert_eq!(args.p2p.listen_ip, expected);
    }

    #[test]
    fn test_p2p_args_listen_tcp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.tcp-port", "1234"]);
        assert_eq!(args.p2p.listen_tcp_port, 1234);
    }

    #[test]
    fn test_p2p_args_listen_udp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.udp-port", "1234"]);
        assert_eq!(args.p2p.listen_udp_port, 1234);
    }

    #[test]
    fn test_p2p_args_bootnodes() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.bootnodes",
            "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305",
        ]);
        assert_eq!(
            args.p2p.bootnodes,
            vec![
                "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305",
            ]
        );

        // Parse the bootnodes.
        let bootnodes = args
            .p2p
            .bootnodes
            .iter()
            .map(|bootnode| BootNode::parse_bootnode(bootnode))
            .collect::<Vec<BootNode>>();

        // Otherwise, attempt to use the Node Record format.
        let record = NodeRecord::from_str(
                "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305").unwrap();
        let expected_bootnode = vec![BootNode::from_unsigned(record).unwrap()];

        assert_eq!(bootnodes, expected_bootnode);
    }

    #[test]
    fn test_p2p_args_bootnodes_multiple() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.bootnodes",
            "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305,enode://dd751a9ef8912be1bfa7a5e34e2c3785cc5253110bd929f385e07ba7ac19929fb0e0c5d93f77827291f4da02b2232240fbc47ea7ce04c46e333e452f8656b667@34.65.107.0:30305",
        ]);
        assert_eq!(
            args.p2p.bootnodes,
            vec![
                "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305",
                "enode://dd751a9ef8912be1bfa7a5e34e2c3785cc5253110bd929f385e07ba7ac19929fb0e0c5d93f77827291f4da02b2232240fbc47ea7ce04c46e333e452f8656b667@34.65.107.0:30305",
            ]
        );
    }

    #[test]
    fn test_p2p_args_bootnode_enr() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.bootnodes",
            "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg",
        ]);
        assert_eq!(
            args.p2p.bootnodes,
            vec![
                "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg",
            ]
        );
    }

    #[test]
    fn test_p2p_args_listen_ip_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses
        // Using localhost which should resolve reliably
        let args = MockCommand::parse_from(["test", "--p2p.listen.addr", "localhost"]);
        // localhost typically resolves to 127.0.0.1 or ::1
        assert!(
            args.p2p.listen_ip == "127.0.0.1".parse::<IpAddr>().unwrap() ||
                args.p2p.listen_ip == "::1".parse::<IpAddr>().unwrap()
        );
    }

    #[test]
    fn test_p2p_args_advertise_ip_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses for advertise_ip
        let args = MockCommand::parse_from(["test", "--p2p.advertise.addr", "localhost"]);
        // localhost typically resolves to 127.0.0.1 or ::1
        let ip = args.p2p.advertise_ip.unwrap();
        assert!(
            ip == "127.0.0.1".parse::<IpAddr>().unwrap() || ip == "::1".parse::<IpAddr>().unwrap()
        );
    }

    #[test]
    fn test_resolve_host_with_ip() {
        // Test that IP addresses are passed through directly
        let ip = resolve_host("192.168.1.1").unwrap();
        assert_eq!(ip, "192.168.1.1".parse::<IpAddr>().unwrap());

        let ipv6 = resolve_host("::1").unwrap();
        assert_eq!(ipv6, "::1".parse::<IpAddr>().unwrap());
    }

    #[test]
    fn test_resolve_host_with_dns() {
        // Test DNS resolution with localhost
        let ip = resolve_host("localhost").unwrap();
        assert!(
            ip == "127.0.0.1".parse::<IpAddr>().unwrap() || ip == "::1".parse::<IpAddr>().unwrap()
        );
    }

    #[test]
    fn test_resolve_host_invalid() {
        // Test that invalid hostnames return an error
        let result = resolve_host("this-hostname-definitely-does-not-exist.invalid");
        assert!(result.is_err());
    }
}
