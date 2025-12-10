//! P2P CLI Flags
//!
//! These are based on p2p flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/p2p_flags.go

use crate::flags::{GlobalArgs, SignerArgs};
use alloy_primitives::{B256, b256};
use alloy_provider::Provider;
use alloy_signer_local::PrivateKeySigner;
use anyhow::Result;
use clap::Parser;
use discv5::enr::k256;
use kona_derive::ChainProvider;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_gossip::GaterConfig;
use kona_node_service::NetworkConfig;
use kona_peers::{BootNode, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use kona_providers_alloy::AlloyChainProvider;
use libp2p::identity::Keypair;
use std::{
    net::{IpAddr, SocketAddr},
    num::ParseIntError,
    path::PathBuf,
    str::FromStr,
};
use tokio::time::Duration;
use url::Url;

/// P2P CLI Flags
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
pub struct P2PArgs {
    /// Disable Discv5 (node discovery).
    #[arg(long = "p2p.no-discovery", default_value = "false", env = "KONA_NODE_P2P_NO_DISCOVERY")]
    pub no_discovery: bool,
    /// Read the hex-encoded 32-byte private key for the peer ID from this txt file.
    /// Created if not already exists. Important to persist to keep the same network identity after
    /// restarting, maintaining the previous advertised identity.
    #[arg(long = "p2p.priv.path", env = "KONA_NODE_P2P_PRIV_PATH")]
    pub priv_path: Option<PathBuf>,
    /// The hex-encoded 32-byte private key for the peer ID.
    #[arg(long = "p2p.priv.raw", env = "KONA_NODE_P2P_PRIV_RAW")]
    pub private_key: Option<B256>,

    /// IP to advertise to external peers from Discv5.
    /// Optional argument. Use the `p2p.listen.ip` if not set.
    ///
    /// Technical note: if this argument is set, the dynamic ENR updates from the discovery layer
    /// will be disabled. This is to allow the advertised IP to be static (to use in a network
    /// behind a NAT for instance).
    #[arg(long = "p2p.advertise.ip", env = "KONA_NODE_P2P_ADVERTISE_IP")]
    pub advertise_ip: Option<IpAddr>,
    /// TCP port to advertise to external peers from the discovery layer. Same as `p2p.listen.tcp`
    /// if set to zero.
    #[arg(long = "p2p.advertise.tcp", env = "KONA_NODE_P2P_ADVERTISE_TCP_PORT")]
    pub advertise_tcp_port: Option<u16>,
    /// UDP port to advertise to external peers from the discovery layer.
    /// Same as `p2p.listen.udp` if set to zero.
    #[arg(long = "p2p.advertise.udp", env = "KONA_NODE_P2P_ADVERTISE_UDP_PORT")]
    pub advertise_udp_port: Option<u16>,

    /// IP to bind LibP2P/Discv5 to.
    #[arg(long = "p2p.listen.ip", default_value = "0.0.0.0", env = "KONA_NODE_P2P_LISTEN_IP")]
    pub listen_ip: IpAddr,
    /// TCP port to bind LibP2P to. Any available system port if set to 0.
    #[arg(long = "p2p.listen.tcp", default_value = "9222", env = "KONA_NODE_P2P_LISTEN_TCP_PORT")]
    pub listen_tcp_port: u16,
    /// UDP port to bind Discv5 to. Same as TCP port if left 0.
    #[arg(long = "p2p.listen.udp", default_value = "9223", env = "KONA_NODE_P2P_LISTEN_UDP_PORT")]
    pub listen_udp_port: u16,
    /// Low-tide peer count. The node actively searches for new peer connections if below this
    /// amount.
    #[arg(long = "p2p.peers.lo", default_value = "20", env = "KONA_NODE_P2P_PEERS_LO")]
    pub peers_lo: u32,
    /// High-tide peer count. The node starts pruning peer connections slowly after reaching this
    /// number.
    #[arg(long = "p2p.peers.hi", default_value = "30", env = "KONA_NODE_P2P_PEERS_HI")]
    pub peers_hi: u32,
    /// Grace period to keep a newly connected peer around, if it is not misbehaving.
    #[arg(
        long = "p2p.peers.grace",
        default_value = "30",
        env = "KONA_NODE_P2P_PEERS_GRACE",
        value_parser = |arg: &str| -> Result<Duration, ParseIntError> {Ok(Duration::from_secs(arg.parse()?))}
    )]
    pub peers_grace: Duration,
    /// Configure GossipSub topic stable mesh target count.
    /// Aka: The desired outbound degree (numbers of peers to gossip to).
    #[arg(long = "p2p.gossip.mesh.d", default_value = "8", env = "KONA_NODE_P2P_GOSSIP_MESH_D")]
    pub gossip_mesh_d: usize,
    /// Configure GossipSub topic stable mesh low watermark.
    /// Aka: The lower bound of outbound degree.
    #[arg(long = "p2p.gossip.mesh.lo", default_value = "6", env = "KONA_NODE_P2P_GOSSIP_MESH_DLO")]
    pub gossip_mesh_dlo: usize,
    /// Configure GossipSub topic stable mesh high watermark.
    /// Aka: The upper bound of outbound degree (additional peers will not receive gossip).
    #[arg(
        long = "p2p.gossip.mesh.dhi",
        default_value = "12",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DHI"
    )]
    pub gossip_mesh_dhi: usize,
    /// Configure GossipSub gossip target.
    /// Aka: The target degree for gossip only (not messaging like p2p.gossip.mesh.d, just
    /// announcements of IHAVE).
    #[arg(
        long = "p2p.gossip.mesh.dlazy",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DLAZY"
    )]
    pub gossip_mesh_dlazy: usize,
    /// Configure GossipSub to publish messages to all known peers on the topic, outside of the
    /// mesh. Also see Dlazy as less aggressive alternative.
    #[arg(
        long = "p2p.gossip.mesh.floodpublish",
        default_value = "false",
        env = "KONA_NODE_P2P_GOSSIP_FLOOD_PUBLISH"
    )]
    pub gossip_flood_publish: bool,
    /// Sets the peer scoring strategy for the P2P stack.
    /// Can be one of: none or light.
    #[arg(long = "p2p.scoring", default_value = "light", env = "KONA_NODE_P2P_SCORING")]
    pub scoring: PeerScoreLevel,

    /// Allows to ban peers based on their score.
    ///
    /// Peers are banned based on a ban threshold (see `p2p.ban.threshold`).
    /// If a peer's score is below the threshold, it gets automatically banned.
    #[arg(long = "p2p.ban.peers", default_value = "false", env = "KONA_NODE_P2P_BAN_PEERS")]
    pub ban_enabled: bool,

    /// The threshold used to ban peers.
    ///
    /// For peers to be banned, the `p2p.ban.peers` flag must be set to `true`.
    /// By default, peers are banned if their score is below -100. This follows the `op-node` default `<https://github.com/ethereum-optimism/optimism/blob/09a8351a72e43647c8a96f98c16bb60e7b25dc6e/op-node/flags/p2p_flags.go#L123-L130>`.
    #[arg(long = "p2p.ban.threshold", default_value = "-100", env = "KONA_NODE_P2P_BAN_THRESHOLD")]
    pub ban_threshold: i64,

    /// The duration in minutes to ban a peer for.
    ///
    /// For peers to be banned, the `p2p.ban.peers` flag must be set to `true`.
    /// By default peers are banned for 1 hour. This follows the `op-node` default `<https://github.com/ethereum-optimism/optimism/blob/09a8351a72e43647c8a96f98c16bb60e7b25dc6e/op-node/flags/p2p_flags.go#L131-L138>`.
    #[arg(long = "p2p.ban.duration", default_value = "60", env = "KONA_NODE_P2P_BAN_DURATION")]
    pub ban_duration: u64,

    /// The interval in seconds to find peers using the discovery service.
    /// Defaults to 5 seconds.
    #[arg(
        long = "p2p.discovery.interval",
        default_value = "5",
        env = "KONA_NODE_P2P_DISCOVERY_INTERVAL"
    )]
    pub discovery_interval: u64,
    /// The directory to store the bootstore.
    #[arg(long = "p2p.bootstore", env = "KONA_NODE_P2P_BOOTSTORE")]
    pub bootstore: Option<PathBuf>,
    /// Disables the bootstore.
    #[arg(long = "p2p.no-bootstore", env = "KONA_NODE_P2P_NO_BOOTSTORE")]
    pub disable_bootstore: bool,
    /// Peer Redialing threshold is the maximum amount of times to attempt to redial a peer that
    /// disconnects. By default, peers are *not* redialed. If set to 0, the peer will be
    /// redialed indefinitely.
    #[arg(long = "p2p.redial", env = "KONA_NODE_P2P_REDIAL", default_value = "500")]
    pub peer_redial: Option<u64>,

    /// The duration in minutes of the peer dial period.
    /// When the last time a peer was dialed is longer than the dial period, the number of peer
    /// dials is reset to 0, allowing the peer to be dialed again.
    #[arg(long = "p2p.redial.period", env = "KONA_NODE_P2P_REDIAL_PERIOD", default_value = "60")]
    pub redial_period: u64,

    /// An optional list of bootnode ENRs or node records to start the node with.
    #[arg(long = "p2p.bootnodes", value_delimiter = ',', env = "KONA_NODE_P2P_BOOTNODES")]
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
        long = "p2p.topic-scoring",
        default_value = "false",
        env = "KONA_NODE_P2P_TOPIC_SCORING"
    )]
    pub topic_scoring: bool,

    /// An optional unsafe block signer address.
    ///
    /// By default, this is fetched from the chain config in the superchain-registry using the
    /// specified L2 chain ID.
    #[arg(long = "p2p.unsafe.block.signer", env = "KONA_NODE_P2P_UNSAFE_BLOCK_SIGNER")]
    pub unsafe_block_signer: Option<alloy_primitives::Address>,

    /// An optional flag to remove random peers from discovery to rotate the peer set.
    ///
    /// This is the number of seconds to wait before removing a peer from the discovery
    /// service. By default, peers are not removed from the discovery service.
    ///
    /// This is useful for discovering a wider set of peers.
    #[arg(long = "p2p.discovery.randomize", env = "KONA_NODE_P2P_DISCOVERY_RANDOMIZE")]
    pub discovery_randomize: Option<u64>,

    /// Specify optional remote signer configuration. Note that this argument is mutually exclusive
    /// with `p2p.sequencer.key` that specifies a local sequencer signer.
    #[command(flatten)]
    pub signer: SignerArgs,
}

impl Default for P2PArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl P2PArgs {
    fn check_ports_inner(ip_addr: IpAddr, tcp_port: u16, udp_port: u16) -> Result<()> {
        if tcp_port == 0 {
            return Ok(());
        }
        if udp_port == 0 {
            return Ok(());
        }
        let tcp_socket = std::net::TcpListener::bind((ip_addr, tcp_port));
        let udp_socket = std::net::UdpSocket::bind((ip_addr, udp_port));
        if let Err(e) = tcp_socket {
            tracing::error!(target: "p2p::flags", tcp_port, "Error binding TCP socket: {e}");
            anyhow::bail!("Error binding TCP socket on port {tcp_port}: {e}");
        }
        if let Err(e) = udp_socket {
            tracing::error!(target: "p2p::flags", udp_port, "Error binding UDP socket: {e}");
            anyhow::bail!("Error binding UDP socket on port {udp_port}: {e}");
        }

        Ok(())
    }

    /// Checks if the listen ports are available on the system.
    ///
    /// If either of the ports are `0`, this check is skipped.
    ///
    /// ## Errors
    ///
    /// - If the TCP port is already in use.
    /// - If the UDP port is already in use.
    pub fn check_ports(&self) -> Result<()> {
        Self::check_ports_inner(self.listen_ip, self.listen_tcp_port, self.listen_udp_port)
    }

    /// Returns the private key as specified in the raw cli flag or via file path.
    pub fn private_key(&self) -> Option<PrivateKeySigner> {
        if let Some(key) = self.private_key {
            match PrivateKeySigner::from_bytes(&key) {
                Ok(signer) => return Some(signer),
                Err(e) => {
                    tracing::error!(target: "p2p::flags", "Failed to parse private key: {}", e);
                    return None;
                }
            }
        }

        if let Some(path) = self.priv_path.as_ref() {
            if path.exists() {
                let contents = std::fs::read_to_string(path).ok()?;
                let decoded = B256::from_str(&contents).ok()?;
                match PrivateKeySigner::from_bytes(&decoded) {
                    Ok(signer) => return Some(signer),
                    Err(e) => {
                        tracing::error!(target: "p2p::flags", "Failed to parse private key from file: {}", e);
                        return None;
                    }
                }
            }
        }

        None
    }

    /// Returns the unsafe block signer from the CLI arguments.
    pub async fn unsafe_block_signer(
        &self,
        args: &GlobalArgs,
        rollup_config: &RollupConfig,
        l1_eth_rpc: Option<Url>,
    ) -> anyhow::Result<alloy_primitives::Address> {
        if let Some(l1_eth_rpc) = l1_eth_rpc {
            /// The storage slot that the unsafe block signer address is stored at.
            /// Computed as: `bytes32(uint256(keccak256("systemconfig.unsafeblocksigner")) - 1)`
            const UNSAFE_BLOCK_SIGNER_ADDRESS_STORAGE_SLOT: B256 =
                b256!("0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");

            let mut provider = AlloyChainProvider::new_http(l1_eth_rpc, 1024);
            let latest_block_num = provider.latest_block_number().await?;
            let block_info = provider.block_info_by_number(latest_block_num).await?;

            // Fetch the unsafe block signer address from the system config.
            let unsafe_block_signer_address = provider
                .inner
                .get_storage_at(
                    rollup_config.l1_system_config_address,
                    UNSAFE_BLOCK_SIGNER_ADDRESS_STORAGE_SLOT.into(),
                )
                .hash(block_info.hash)
                .await?;

            // Convert the unsafe block signer address to the correct type.
            return Ok(alloy_primitives::Address::from_slice(
                &unsafe_block_signer_address.to_be_bytes_vec()[12..],
            ));
        }

        // Otherwise use the genesis signer or the configured unsafe block signer.
        args.genesis_signer().or_else(|_| {
            self.unsafe_block_signer.ok_or(anyhow::anyhow!("Unsafe block signer not provided"))
        })
    }

    /// Constructs kona's P2P network [`NetworkConfig`] from CLI arguments.
    ///
    /// ## Parameters
    ///
    /// - [`GlobalArgs`]: required to fetch the genesis unsafe block signer.
    ///
    /// Errors if the genesis unsafe block signer isn't available for the specified L2 Chain ID.
    pub async fn config(
        self,
        config: &RollupConfig,
        args: &GlobalArgs,
        l1_rpc: Option<Url>,
    ) -> anyhow::Result<NetworkConfig> {
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

        let keypair = self.keypair().unwrap_or_else(|_| Keypair::generate_secp256k1());
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
            ban_duration: Duration::from_secs(60 * self.ban_duration),
            ban_threshold: self.ban_threshold as f64,
        });

        let discovery_listening_address = SocketAddr::new(self.listen_ip, self.listen_udp_port);
        let discovery_config =
            NetworkConfig::discv5_config(discovery_listening_address.into(), static_ip);

        let mut gossip_address = libp2p::Multiaddr::from(self.listen_ip);
        gossip_address.push(libp2p::multiaddr::Protocol::Tcp(self.listen_tcp_port));

        let unsafe_block_signer = self.unsafe_block_signer(args, config, l1_rpc).await?;

        let bootstore = if self.disable_bootstore {
            None
        } else {
            Some(self.bootstore.map_or(
                BootStoreFile::Default { chain_id: args.l2_chain_id.into() },
                BootStoreFile::Custom,
            ))
        };

        let bootnodes = self
            .bootnodes
            .iter()
            .map(|bootnode| BootNode::parse_bootnode(bootnode))
            .collect::<Vec<BootNode>>()
            .into();

        Ok(NetworkConfig {
            discovery_config,
            discovery_interval: Duration::from_secs(self.discovery_interval),
            discovery_address,
            discovery_randomize: self.discovery_randomize.map(Duration::from_secs),
            enr_update: !static_ip,
            gossip_address,
            keypair,
            unsafe_block_signer,
            gossip_config,
            scoring: self.scoring,
            monitor_peers,
            bootstore,
            topic_scoring: self.topic_scoring,
            gater_config: GaterConfig {
                peer_redialing: self.peer_redial,
                dial_period: Duration::from_secs(60 * self.redial_period),
            },
            bootnodes,
            rollup_config: config.clone(),
            gossip_signer: self.signer.config(args)?,
        })
    }

    /// Returns the [`Keypair`] from the cli inputs.
    ///
    /// If the raw private key is empty and the specified file is empty,
    /// this method will generate a new private key and write it out to the file.
    ///
    /// If neither a file is specified, nor a raw private key input, this method
    /// will error.
    pub fn keypair(&self) -> Result<Keypair> {
        // Attempt the parse the private key if specified.
        if let Some(mut private_key) = self.private_key {
            return kona_cli::SecretKeyLoader::parse(&mut private_key.0)
                .map_err(|e| anyhow::anyhow!(e));
        }

        let Some(ref key_path) = self.priv_path else {
            anyhow::bail!("Neither a raw private key nor a private key file path was provided.");
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

    /// A mock command that uses the P2PArgs.
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
    fn test_p2p_args_keypair_raw_private_key() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.priv.raw",
            "1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be",
        ]);
        assert!(args.p2p.keypair().is_ok());
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
            MockCommand::parse_from(["test", "--p2p.priv.path", source_path.to_str().unwrap()]);
        assert!(args.p2p.keypair().is_ok());
    }

    #[test]
    fn test_p2p_args() {
        let args = MockCommand::parse_from(["test"]);
        assert_eq!(args.p2p, P2PArgs::default());
    }

    #[test]
    fn test_p2p_args_randomized() {
        let args = MockCommand::parse_from(["test", "--p2p.discovery.randomize", "10"]);
        assert_eq!(args.p2p.discovery_randomize, Some(10));
        let args = MockCommand::parse_from(["test"]);
        assert_eq!(args.p2p.discovery_randomize, None);
    }

    #[test]
    fn test_p2p_args_no_discovery() {
        let args = MockCommand::parse_from(["test", "--p2p.no-discovery"]);
        assert!(args.p2p.no_discovery);
    }

    #[test]
    fn test_p2p_args_priv_path() {
        let args = MockCommand::parse_from(["test", "--p2p.priv.path", "test.txt"]);
        assert_eq!(args.p2p.priv_path, Some(PathBuf::from("test.txt")));
    }

    #[test]
    fn test_p2p_args_private_key() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.priv.raw",
            "1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be",
        ]);
        let key = b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");
        assert_eq!(args.p2p.private_key, Some(key));
    }

    #[test]
    fn test_p2p_args_sequencer_key() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.sequencer.key",
            "bcc617ea05150ff60490d3c6058630ba94ae9f12a02a87efd291349ca0e54e0a",
        ]);
        let key = b256!("bcc617ea05150ff60490d3c6058630ba94ae9f12a02a87efd291349ca0e54e0a");
        assert_eq!(args.p2p.signer.sequencer_key, Some(key));
    }

    #[test]
    fn test_p2p_args_listen_ip() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.ip", "127.0.0.1"]);
        let expected: IpAddr = "127.0.0.1".parse().unwrap();
        assert_eq!(args.p2p.listen_ip, expected);
    }

    #[test]
    fn test_p2p_args_listen_tcp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.tcp", "1234"]);
        assert_eq!(args.p2p.listen_tcp_port, 1234);
    }

    #[test]
    fn test_p2p_args_listen_udp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.udp", "1234"]);
        assert_eq!(args.p2p.listen_udp_port, 1234);
    }

    #[test]
    fn test_p2p_args_bootnodes() {
        let args = MockCommand::parse_from([
            "test",
            "--p2p.bootnodes",
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
            "--p2p.bootnodes",
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
            "--p2p.bootnodes",
            "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg",
        ]);
        assert_eq!(
            args.p2p.bootnodes,
            vec![
                "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg",
            ]
        );
    }
}
