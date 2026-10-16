//! P2P CLI Flags
//!
//! These are based on p2p flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/p2p_flags.go

use anyhow::{Context, Result};
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
fn resolve_host(host: &str) -> Result<IpAddr> {
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
            .ok_or_else(|| anyhow::anyhow!("DNS resolution for '{host}' returned no addresses")),
        Err(e) => Err(anyhow::anyhow!("Failed to resolve '{host}': {e}")),
    }
}

/// P2P CLI Flags
///
/// The other P2P flags apply only with `--p2p.enabled`, and are ignored otherwise.
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
#[command(next_help_heading = "P2P")]
pub struct P2PArgs {
    /// Receive and gossip blocks over P2P. When disabled, the node only follows the safe chain
    /// (unless it is a sequencer, when it also produces the ungossiped unsafe chain).
    #[arg(
        id = "p2p_enabled",
        long = "p2p.enabled",
        env = "KONA_NODE_P2P_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub enabled: bool,

    /// Path to a file holding the hex-encoded 32-byte private key of the node's peer ID, created
    /// with a new key if it does not exist. Keep the file across restarts to keep the peer ID: by
    /// default, the node takes a new peer ID each time it starts.
    #[arg(long = "p2p.priv-path", env = "KONA_NODE_P2P_PRIV_PATH", value_name = "PATH")]
    pub priv_path: Option<PathBuf>,

    /// IP address or DNS hostname to listen for peers on. A hostname is resolved when the node
    /// starts.
    #[arg(
        id = "p2p_listen_addr",
        long = "p2p.listen.addr",
        default_value = "0.0.0.0",
        env = "KONA_NODE_P2P_LISTEN_ADDR",
        value_name = "HOST"
    )]
    pub listen_addr: String,
    /// TCP port to listen for peers on. 0 picks any free port.
    #[arg(
        long = "p2p.listen.tcp-port",
        default_value = "9222",
        env = "KONA_NODE_P2P_LISTEN_TCP_PORT",
        value_name = "PORT"
    )]
    pub listen_tcp_port: u16,
    /// UDP port for discovery to listen on. 0 picks any free port.
    #[arg(
        long = "p2p.listen.udp-port",
        default_value = "9223",
        env = "KONA_NODE_P2P_LISTEN_UDP_PORT",
        value_name = "PORT"
    )]
    pub listen_udp_port: u16,

    /// IP address or DNS hostname for discovery to advertise to peers, such as a public address
    /// behind a NAT. Defaults to `--p2p.listen.addr`. A hostname is resolved when the node starts.
    /// Setting it stops discovery from updating the advertised address from what peers report.
    /// Used only with `--p2p.discovery.enabled`.
    #[arg(
        id = "p2p_advertise_addr",
        long = "p2p.advertise.addr",
        env = "KONA_NODE_P2P_ADVERTISE_ADDR",
        value_name = "HOST"
    )]
    pub advertise_addr: Option<String>,
    /// TCP port for discovery to advertise to peers. Defaults to `--p2p.listen.tcp-port`.
    #[arg(
        long = "p2p.advertise.tcp-port",
        env = "KONA_NODE_P2P_ADVERTISE_TCP_PORT",
        value_name = "PORT"
    )]
    pub advertise_tcp_port: Option<u16>,
    /// UDP port for discovery to advertise to peers. Defaults to `--p2p.listen.udp-port`.
    #[arg(
        long = "p2p.advertise.udp-port",
        env = "KONA_NODE_P2P_ADVERTISE_UDP_PORT",
        value_name = "PORT"
    )]
    pub advertise_udp_port: Option<u16>,

    /// `GossipSub` mesh target degree: how many peers to keep in the mesh and send blocks to.
    #[arg(
        long = "p2p.gossip.mesh.d",
        default_value = "8",
        env = "KONA_NODE_P2P_GOSSIP_MESH_D",
        value_name = "N"
    )]
    pub gossip_mesh_d: usize,
    /// `GossipSub` mesh low watermark: with fewer peers in the mesh, the node adds more.
    #[arg(
        long = "p2p.gossip.mesh.lo",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_LO",
        value_name = "N"
    )]
    pub gossip_mesh_dlo: usize,
    /// `GossipSub` mesh high watermark: with more peers in the mesh, the node prunes it.
    #[arg(
        long = "p2p.gossip.mesh.dhi",
        default_value = "12",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DHI",
        value_name = "N"
    )]
    pub gossip_mesh_dhi: usize,
    /// `GossipSub` gossip target: how many peers outside the mesh to announce blocks to, without
    /// sending them the blocks.
    #[arg(
        long = "p2p.gossip.mesh.dlazy",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DLAZY",
        value_name = "N"
    )]
    pub gossip_mesh_dlazy: usize,
    /// Publish the node's own blocks to every known peer on the topic, not only to its mesh.
    /// `--p2p.gossip.mesh.dlazy` is a less aggressive alternative.
    #[arg(
        id = "p2p_flood_publish_enabled",
        long = "p2p.gossip.mesh.flood-publish.enabled",
        env = "KONA_NODE_P2P_GOSSIP_MESH_FLOOD_PUBLISH_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub gossip_flood_publish: bool,

    /// How to score peers by their behavior.
    #[arg(
        long = "p2p.peer-scoring",
        default_value = "light",
        env = "KONA_NODE_P2P_PEER_SCORING",
        value_name = "LEVEL",
        ignore_case = true,
        value_parser = clap::builder::TypedValueParser::try_map(
            clap::builder::PossibleValuesParser::new(["off", "light"]),
            |level| level.parse::<PeerScoreLevel>(),
        )
    )]
    pub scoring: PeerScoreLevel,
    /// Also score peers by their behavior on each gossip topic. op-node only ever did so for the
    /// v1 blocks topic, by mistake, and is phasing topic scoring out
    /// (<https://github.com/ethereum-optimism/optimism/pull/15719>): this switch is kept for
    /// compatibility and debugging.
    #[arg(
        id = "p2p_topic_scoring_enabled",
        long = "p2p.topic-scoring.enabled",
        env = "KONA_NODE_P2P_TOPIC_SCORING_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub topic_scoring: bool,

    /// Ban peers whose score falls below `--p2p.ban.threshold`, for `--p2p.ban.duration`.
    #[arg(
        id = "p2p_ban_enabled",
        long = "p2p.ban.enabled",
        env = "KONA_NODE_P2P_BAN_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub ban_enabled: bool,
    /// The score below which a peer is banned, with `--p2p.ban.enabled`. The default is
    /// op-node's.
    #[arg(
        long = "p2p.ban.threshold",
        default_value = "-100",
        env = "KONA_NODE_P2P_BAN_THRESHOLD",
        value_name = "SCORE",
        allow_negative_numbers = true
    )]
    pub ban_threshold: i64,
    /// How long to ban a peer for, such as `30m` or `1h`, with `--p2p.ban.enabled`. The default
    /// is op-node's.
    #[arg(
        long = "p2p.ban.duration",
        default_value = "1h",
        env = "KONA_NODE_P2P_BAN_DURATION",
        value_name = "DURATION",
        value_parser = humantime::parse_duration
    )]
    pub ban_duration: Duration,

    /// How many times to redial a peer that disconnects, within each `--p2p.redial.period`. 0
    /// redials it indefinitely.
    #[arg(
        long = "p2p.redial.max-attempts",
        env = "KONA_NODE_P2P_REDIAL_MAX_ATTEMPTS",
        default_value = "500",
        value_name = "N"
    )]
    pub peer_redial: Option<u64>,
    /// How long after a peer was last dialed, such as `1h`, its count of redials resets, so that
    /// it can be redialed again.
    #[arg(
        long = "p2p.redial.period",
        env = "KONA_NODE_P2P_REDIAL_PERIOD",
        default_value = "1h",
        value_name = "DURATION",
        value_parser = humantime::parse_duration
    )]
    pub redial_period: Duration,

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
        value_name = "BOOL"
    )]
    pub discovery_enabled: bool,
    /// Comma-separated ENRs or enode URLs of nodes to start discovery from, in addition to the
    /// chain's built-in bootnodes.
    #[arg(
        long = "p2p.discovery.bootnodes",
        value_delimiter = ',',
        env = "KONA_NODE_P2P_DISCOVERY_BOOTNODES",
        value_name = "ENR"
    )]
    pub bootnodes: Vec<String>,
    /// How often discovery looks for new peers, such as `5s`.
    #[arg(
        long = "p2p.discovery.interval",
        default_value = "5s",
        env = "KONA_NODE_P2P_DISCOVERY_INTERVAL",
        value_name = "DURATION",
        value_parser = humantime::parse_duration
    )]
    pub discovery_interval: Duration,
    /// How often, such as `30s`, to drop a random peer from discovery, so that it finds a wider
    /// set of peers. By default, discovery drops none.
    #[arg(
        long = "p2p.discovery.randomize",
        env = "KONA_NODE_P2P_DISCOVERY_RANDOMIZE",
        value_name = "DURATION",
        value_parser = humantime::parse_duration
    )]
    pub discovery_randomize: Option<Duration>,
    /// Keep the peers discovery finds in a bootstore, so that the node finds them again after a
    /// restart. Disabled by default.
    #[arg(
        id = "p2p_bootstore_enabled",
        long = "p2p.discovery.bootstore.enabled",
        env = "KONA_NODE_P2P_DISCOVERY_BOOTSTORE_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true",
        value_name = "BOOL"
    )]
    pub bootstore_enabled: bool,
    /// Path to the bootstore file. By default, `~/.kona/<chain_id>/bootstore.json`.
    #[arg(
        long = "p2p.discovery.bootstore.path",
        env = "KONA_NODE_P2P_DISCOVERY_BOOTSTORE_PATH",
        value_name = "PATH"
    )]
    pub bootstore: Option<PathBuf>,
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

        let listen_ip = resolve_host(&self.listen_addr).context("--p2p.listen.addr")?;

        // Fallback to the listen ip if the advertise ip is not specified. Only discovery
        // advertises the address, so resolve it only with discovery.
        let advertise_ip = match &self.advertise_addr {
            Some(host) if self.discovery_enabled => {
                resolve_host(host).context("--p2p.advertise.addr")?
            }
            _ => listen_ip,
        };

        // If the advertise ip is set, we will disable the dynamic ENR updates.
        let static_ip = self.advertise_addr.is_some();

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

        let discovery_listening_address = SocketAddr::new(listen_ip, self.listen_udp_port);
        let discovery_config =
            NetworkConfig::discv5_config(discovery_listening_address.into(), static_ip);

        let mut gossip_address = libp2p::Multiaddr::from(listen_ip);
        gossip_address.push(libp2p::multiaddr::Protocol::Tcp(self.listen_tcp_port));

        let bootstore = self.bootstore_file(config);
        let bootnodes = self.bootnodes()?.into();

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

    /// Returns the bootnodes given, or none without discovery, which alone dials them.
    fn bootnodes(&self) -> Result<Vec<BootNode>> {
        if !self.discovery_enabled {
            return Ok(Vec::new());
        }
        self.bootnodes
            .iter()
            .map(|raw| raw.parse().with_context(|| format!("invalid bootnode {raw}")))
            .collect()
    }

    /// Returns the bootstore file to use for the rollup's L2 chain, or `None` if the bootstore or
    /// discovery, which alone reads and writes it, is disabled.
    fn bootstore_file(&self, config: &RollupConfig) -> Option<BootStoreFile> {
        if !self.discovery_enabled || !self.bootstore_enabled {
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

        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.enabled",
            "--p2p.discovery.bootstore.enabled",
        ]);
        assert_eq!(
            args.p2p.bootstore_file(&config),
            Some(BootStoreFile::Default { chain_id: 8453 })
        );

        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.enabled",
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

        // Only discovery reads and writes the bootstore.
        let args = MockCommand::parse_from(["test", "--p2p.discovery.bootstore.enabled"]);
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
    fn test_p2p_args_listen_addr() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.addr", "127.0.0.1"]);
        let config = args.p2p.config(&RollupConfig::default()).unwrap();
        let expected: IpAddr = "127.0.0.1".parse().unwrap();
        assert_eq!(config.discovery_address.ip, expected);
        assert!(config.enr_update);
    }

    #[test]
    fn test_p2p_args_listen_tcp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.tcp-port", "1234"]);
        assert_eq!(args.p2p.listen_tcp_port, 1234);
    }

    #[test]
    fn test_p2p_args_peer_scoring() {
        for (level, expected) in [
            ("off", PeerScoreLevel::Off),
            ("light", PeerScoreLevel::Light),
            ("Light", PeerScoreLevel::Light),
        ] {
            let args = MockCommand::parse_from(["test", "--p2p.peer-scoring", level]);
            assert_eq!(args.p2p.scoring, expected, "{level}");
        }
        assert!(MockCommand::try_parse_from(["test", "--p2p.peer-scoring", "none"]).is_err());
    }

    #[test]
    fn test_p2p_args_ban_threshold_takes_a_negative_score() {
        let args = MockCommand::parse_from(["test", "--p2p.ban.threshold", "-50"]);
        assert_eq!(args.p2p.ban_threshold, -50);
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

        // Bootnodes are parsed only with discovery.
        assert_eq!(args.p2p.bootnodes().unwrap(), vec![]);
        let mut p2p = args.p2p;
        p2p.discovery_enabled = true;
        let bootnodes = p2p.bootnodes().unwrap();

        // Otherwise, attempt to use the Node Record format.
        let record = NodeRecord::from_str(
                "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305").unwrap();
        let expected_bootnode = vec![BootNode::from_unsigned(record).unwrap()];

        assert_eq!(bootnodes, expected_bootnode);
    }

    #[test]
    fn a_malformed_bootnode_is_an_error_only_with_discovery() {
        let args =
            MockCommand::parse_from(["test", "--p2p.discovery.bootnodes", "enode://malformed"]);
        assert_eq!(args.p2p.bootnodes().unwrap(), vec![]);

        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.enabled",
            "--p2p.discovery.bootnodes",
            "enode://malformed",
        ]);
        assert!(args.p2p.bootnodes().is_err());
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
    fn test_p2p_args_listen_addr_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses
        // Using localhost which should resolve reliably
        let args = MockCommand::parse_from(["test", "--p2p.listen.addr", "localhost"]);
        let config = args.p2p.config(&RollupConfig::default()).unwrap();
        // localhost typically resolves to 127.0.0.1 or ::1
        assert!(config.discovery_address.ip.is_loopback());
    }

    #[test]
    fn test_p2p_args_advertise_addr_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses for the advertised address
        let args = MockCommand::parse_from([
            "test",
            "--p2p.discovery.enabled",
            "--p2p.advertise.addr",
            "localhost",
        ]);
        let config = args.p2p.config(&RollupConfig::default()).unwrap();
        // localhost typically resolves to 127.0.0.1 or ::1
        assert!(config.discovery_address.ip.is_loopback());
        assert!(!config.enr_update);
    }

    /// Hosts are resolved only when the node builds its P2P config, and the advertised one only
    /// with discovery, so an unresolvable host is harmless while its feature is off.
    #[test]
    fn an_unresolvable_host_is_an_error_only_when_resolved() {
        let host = "this-hostname-definitely-does-not-exist.invalid";
        let config = |args: &[&str]| {
            let args =
                MockCommand::try_parse_from(std::iter::once("test").chain(args.iter().copied()))
                    .unwrap();
            args.p2p.config(&RollupConfig::default())
        };
        assert!(config(&["--p2p.listen.addr", host]).is_err());
        assert!(config(&["--p2p.discovery.enabled", "--p2p.advertise.addr", host]).is_err());
        assert!(config(&["--p2p.advertise.addr", host]).is_ok());
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
