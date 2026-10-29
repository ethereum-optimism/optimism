//! P2P CLI Flags
//!
//! These are based on p2p flags from the [`op-node`][op-node] CLI.
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/develop/op-node/flags/p2p_flags.go

use crate::flags::{GlobalArgs, SignerArgs};
use alloy_primitives::B256;
use alloy_signer_local::PrivateKeySigner;
use anyhow::Result;
use clap::Parser;
use enr::k256;
use kona_derive::ChainProvider;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_gossip::{GaterConfig, PeerAddr};
use kona_node_service::NetworkConfig;
use kona_peers::{BootNode, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use kona_providers_alloy::AlloyChainProvider;
use libp2p::{identity::Keypair, multiaddr::Protocol};
use std::{
    net::{IpAddr, SocketAddr, ToSocketAddrs},
    num::ParseIntError,
    path::PathBuf,
    str::FromStr,
};
use tokio::time::Duration;
use url::Url;

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

/// Returns the port to advertise: `advertise` if set to a non-zero port, `listen` otherwise.
fn advertised_port(advertise: Option<u16>, listen: u16) -> u16 {
    advertise.filter(|&port| port != 0).unwrap_or(listen)
}

/// Parses one `--p2p.bootnodes` entry: an ENR (`enr:...`) or an enode URL (`enode://...`).
fn parse_bootnode(raw: &str) -> Result<BootNode, String> {
    let raw = raw.trim();
    if raw.is_empty() {
        return Err("Bootnode must not be empty".to_string());
    }
    BootNode::parse_bootnode(raw).map_err(|e| format!("{:#}", anyhow::Error::new(e)))
}

/// Parses one static peer: a TCP address that ends in the peer's ID, such as
/// `/dns4/node.example.com/tcp/9222/p2p/16Uiu2...`.
///
/// The node only gossips over TCP, so any other transport could never be dialed. DNS names are
/// resolved when dialing, not here, so a peer whose name does not resolve yet does not stop the
/// node from starting.
fn parse_static_peer(addr: &str) -> Result<PeerAddr, String> {
    let addr = addr.trim();
    if addr.is_empty() {
        return Err("Static peer must not be empty".to_string());
    }
    let peer: PeerAddr = addr.parse().map_err(|e| format!("{:#}", anyhow::Error::new(e)))?;
    let protocols: Vec<_> = peer.addr().iter().collect();
    let shape = match protocols.as_slice() {
        [
            Protocol::Ip4(_) |
            Protocol::Ip6(_) |
            Protocol::Dns(_) |
            Protocol::Dns4(_) |
            Protocol::Dns6(_),
            rest @ ..,
        ] => match rest {
            [Protocol::Tcp(0), ..] => Err("TCP port must not be 0"),
            [Protocol::Tcp(_), Protocol::P2p(_)] => Ok(()),
            [Protocol::Tcp(_), ..] => Err("Address must end in /tcp/<port>/p2p/<peer id>"),
            _ => Err("Address must have a /tcp/<port> after the host"),
        },
        _ => Err("Address must start with /ip4, /ip6, /dns, /dns4 or /dns6"),
    };
    shape.map_err(str::to_string)?;
    Ok(peer)
}

/// P2P CLI Flags
#[derive(Parser, Clone, Debug, PartialEq, Eq)]
pub struct P2PArgs {
    /// Disable Discv5 (node discovery). The node then binds no UDP port, ignores its bootnodes,
    /// and connects only to its static peers, the peers that dial it, and the peers added with
    /// `opp2p_connectPeer`.
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

    /// IP address or DNS hostname to advertise to external peers from Discv5.
    /// Optional argument. Use the `p2p.listen.ip` if not set.
    /// Accepts either an IP address (e.g., "1.2.3.4") or a DNS hostname (e.g.,
    /// "node1.example.com"). DNS hostnames are resolved to IP addresses at startup.
    ///
    /// Technical note: if this argument is set, the dynamic ENR updates from the discovery layer
    /// will be disabled. This is to allow the advertised IP to be static (to use in a network
    /// behind a NAT for instance).
    #[arg(long = "p2p.advertise.ip", env = "KONA_NODE_P2P_ADVERTISE_IP", value_parser = resolve_host)]
    pub advertise_ip: Option<IpAddr>,
    /// TCP port to advertise to external peers from the discovery layer. Same as `p2p.listen.tcp`
    /// if set to zero.
    #[arg(long = "p2p.advertise.tcp", env = "KONA_NODE_P2P_ADVERTISE_TCP_PORT")]
    pub advertise_tcp_port: Option<u16>,
    /// UDP port to advertise to external peers from the discovery layer.
    /// Same as `p2p.listen.udp` if set to zero.
    #[arg(long = "p2p.advertise.udp", env = "KONA_NODE_P2P_ADVERTISE_UDP_PORT")]
    pub advertise_udp_port: Option<u16>,

    /// IP address or DNS hostname to bind LibP2P/Discv5 to.
    /// Accepts either an IP address (e.g., "0.0.0.0") or a DNS hostname (e.g.,
    /// "node1.example.com"). DNS hostnames are resolved to IP addresses at startup.
    #[arg(long = "p2p.listen.ip", default_value = "0.0.0.0", env = "KONA_NODE_P2P_LISTEN_IP", value_parser = resolve_host)]
    pub listen_ip: IpAddr,
    /// TCP port to bind `LibP2P` to. Any available system port if set to 0.
    #[arg(long = "p2p.listen.tcp", default_value = "9222", env = "KONA_NODE_P2P_LISTEN_TCP_PORT")]
    pub listen_tcp_port: u16,
    /// UDP port to bind Discv5 to. Any available system port if set to 0, which then cannot be
    /// combined with `p2p.advertise.udp`.
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
    /// Configure `GossipSub` topic stable mesh target count.
    /// Aka: The desired outbound degree (numbers of peers to gossip to).
    #[arg(long = "p2p.gossip.mesh.d", default_value = "8", env = "KONA_NODE_P2P_GOSSIP_MESH_D")]
    pub gossip_mesh_d: usize,
    /// Configure `GossipSub` topic stable mesh low watermark.
    /// Aka: The lower bound of outbound degree.
    #[arg(long = "p2p.gossip.mesh.lo", default_value = "6", env = "KONA_NODE_P2P_GOSSIP_MESH_DLO")]
    pub gossip_mesh_dlo: usize,
    /// Configure `GossipSub` topic stable mesh high watermark.
    /// Aka: The upper bound of outbound degree (additional peers will not receive gossip).
    #[arg(
        long = "p2p.gossip.mesh.dhi",
        default_value = "12",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DHI"
    )]
    pub gossip_mesh_dhi: usize,
    /// Configure `GossipSub` gossip target.
    /// Aka: The target degree for gossip only (not messaging like p2p.gossip.mesh.d, just
    /// announcements of IHAVE).
    #[arg(
        long = "p2p.gossip.mesh.dlazy",
        default_value = "6",
        env = "KONA_NODE_P2P_GOSSIP_MESH_DLAZY"
    )]
    pub gossip_mesh_dlazy: usize,
    /// Configure `GossipSub` to publish messages to all known peers on the topic, outside of the
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
    /// disconnects. The count resets once the peer has not been dialed for `p2p.redial.period`.
    /// If set to 0, the peer will be redialed indefinitely.
    #[arg(long = "p2p.redial", env = "KONA_NODE_P2P_REDIAL", default_value = "500")]
    pub peer_redial: Option<u64>,

    /// The duration in minutes of the peer dial period.
    /// When the last time a peer was dialed is longer than the dial period, the number of peer
    /// dials is reset to 0, allowing the peer to be dialed again.
    #[arg(long = "p2p.redial.period", env = "KONA_NODE_P2P_REDIAL_PERIOD", default_value = "60")]
    pub redial_period: u64,

    /// An optional list of bootnode ENRs or node records to start the node with.
    ///
    /// Each entry is an ENR (`enr:...`) or an enode URL (`enode://...`). The flag can be repeated,
    /// or given a comma-separated list.
    #[arg(
        long = "p2p.bootnodes",
        value_delimiter = ',',
        env = "KONA_NODE_P2P_BOOTNODES",
        value_parser = parse_bootnode
    )]
    pub bootnodes: Vec<BootNode>,

    /// Multiaddrs of peers to dial at startup and keep connected, such as
    /// `/dns4/node.example.com/tcp/9222/p2p/16Uiu2...`.
    ///
    /// Each address must be TCP and end in the peer's ID: an `opp2p_self` RPC call lists a node's
    /// addresses in this form. The flag can be repeated, or given a comma-separated list.
    ///
    /// Several addresses with the same peer ID make one peer, dialed at whichever address connects
    /// first. Addresses with the node's own ID are ignored, so a cluster can share one list.
    #[arg(
        long = "p2p.static-peers",
        // op-node's name for the flag, for configs migrated from op-node.
        visible_alias = "p2p.static",
        value_delimiter = ',',
        env = "KONA_NODE_P2P_STATIC_PEERS",
        value_parser = parse_static_peer
    )]
    pub static_peers: Vec<PeerAddr>,

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
    /// Checks if the listen ports are available on the system.
    ///
    /// A port of `0` is not checked, and neither is the UDP port if discovery is disabled, since
    /// the node then doesn't bind it.
    ///
    /// ## Errors
    ///
    /// - If the TCP port is already in use.
    /// - If the UDP port is already in use.
    pub fn check_ports(&self) -> Result<()> {
        let (ip, tcp_port, udp_port) = (self.listen_ip, self.listen_tcp_port, self.listen_udp_port);
        if tcp_port != 0 &&
            let Err(e) = std::net::TcpListener::bind((ip, tcp_port))
        {
            tracing::error!(target: "p2p::flags", tcp_port, "Error binding TCP socket: {e}");
            anyhow::bail!("Error binding TCP socket on port {tcp_port}: {e}");
        }
        if !self.no_discovery &&
            udp_port != 0 &&
            let Err(e) = std::net::UdpSocket::bind((ip, udp_port))
        {
            tracing::error!(target: "p2p::flags", udp_port, "Error binding UDP socket: {e}");
            anyhow::bail!("Error binding UDP socket on port {udp_port}: {e}");
        }

        Ok(())
    }

    /// Returns what an operator should know about how the node can find peers with these flags.
    fn peering_warnings(&self) -> Vec<&'static str> {
        let mut warnings = Vec::new();
        if self.no_discovery {
            if self.static_peers.is_empty() {
                warnings.push(
                    "Discovery is disabled and no static peers are set: the node only connects \
                     to peers that dial it, or that are added with opp2p_connectPeer",
                );
            }
            if !self.bootnodes.is_empty() {
                warnings.push("Discovery is disabled, so the bootnodes are ignored");
            }
        }
        warnings
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

        if let Some(path) = self.priv_path.as_ref() &&
            path.exists()
        {
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
            let mut provider = AlloyChainProvider::new_http(l1_eth_rpc, 1024);
            let latest_block_num = provider.latest_block_number().await?;
            let block_info = provider.block_info_by_number(latest_block_num).await?;
            return Ok(kona_providers_alloy::unsafe_block_signer(
                &provider.inner,
                rollup_config.l1_system_config_address,
                block_info.hash,
            )
            .await?);
        }

        // Otherwise use the genesis signer or the configured unsafe block signer.
        args.genesis_signer().or_else(|_| {
            self.unsafe_block_signer
                .ok_or_else(|| anyhow::anyhow!("Unsafe block signer not provided"))
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
        for warning in self.peering_warnings() {
            tracing::warn!(target: "p2p::flags", "{warning}");
        }

        // Note: the advertised address is contained in the ENR for external peers from the
        // discovery layer to use.

        // Fallback to the listen ip if the advertise ip is not specified
        let advertise_ip = self.advertise_ip.unwrap_or(self.listen_ip);

        // If the advertise ip is set, we will disable the dynamic ENR updates.
        let static_ip = self.advertise_ip.is_some();

        let advertise_tcp_port = advertised_port(self.advertise_tcp_port, self.listen_tcp_port);
        // Discovery advertises the port it listens on for 0, see `Discv5Builder::build`.
        let advertise_udp_port = self.advertise_udp_port.unwrap_or_default();

        let keypair = self.keypair().unwrap_or_else(|e| {
            let generated = Keypair::generate_secp256k1();
            tracing::warn!(
                target: "p2p::config",
                error = %e,
                peer_id = %generated.public().to_peer_id(),
                "Failed to load P2P keypair from configuration, generated ephemeral keypair. \
                 Set --p2p.priv.path or --p2p.priv.raw for a persistent peer ID."
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
            Some(self.bootstore.map_or_else(
                || BootStoreFile::Default { chain_id: args.l2_chain_id.into() },
                BootStoreFile::Custom,
            ))
        };

        let bootnodes = self.bootnodes.into();

        Ok(NetworkConfig {
            discovery_config,
            discovery_interval: Duration::from_secs(self.discovery_interval),
            discovery_address,
            discovery_randomize: self.discovery_randomize.map(Duration::from_secs),
            discovery_enabled: !self.no_discovery,
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
            static_peers: self.static_peers,
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
            let keypair = kona_cli::SecretKeyLoader::parse(&mut private_key.0)
                .map_err(|e| anyhow::anyhow!(e))?;
            tracing::info!(
                target: "p2p::config",
                peer_id = %keypair.public().to_peer_id(),
                "Successfully loaded P2P keypair from raw private key"
            );
            return Ok(keypair);
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
    use libp2p::PeerId;

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

    /// The TCP port is always checked, and the UDP port only if discovery binds it.
    #[test]
    fn test_check_ports_skips_udp_without_discovery() {
        let ip = std::net::Ipv4Addr::LOCALHOST;
        let port = |addr: std::io::Result<SocketAddr>| addr.unwrap().port().to_string();
        let udp = std::net::UdpSocket::bind((ip, 0)).unwrap();
        let tcp = std::net::TcpListener::bind((ip, 0)).unwrap();
        let (taken_udp, taken_tcp) = (port(udp.local_addr()), port(tcp.local_addr()));
        let free_tcp = port(std::net::TcpListener::bind((ip, 0)).unwrap().local_addr());
        let check = |tcp_port: &str, no_discovery: bool| {
            let mut args = vec!["test", "--p2p.listen.ip", "127.0.0.1"];
            args.extend(["--p2p.listen.tcp", tcp_port, "--p2p.listen.udp", &taken_udp]);
            if no_discovery {
                args.push("--p2p.no-discovery");
            }
            MockCommand::parse_from(args).p2p.check_ports()
        };

        assert!(check(&free_tcp, false).is_err());
        assert!(check(&free_tcp, true).is_ok());
        assert!(check(&taken_tcp, true).is_err());
        // A TCP port of 0 no longer skips the UDP check.
        assert!(check("0", false).is_err());
    }

    /// Without discovery, the node warns if nothing gives it peers, and that bootnodes go unused.
    #[test]
    fn test_peering_warnings() {
        let static_peer = format!("/ip4/10.0.0.1/tcp/9222/p2p/{}", PeerId::random());
        let warnings = |args: &[&str]| {
            let args = MockCommand::parse_from(std::iter::once("test").chain(args.iter().copied()));
            args.p2p.peering_warnings()
        };

        assert!(warnings(&[]).is_empty());
        assert!(warnings(&["--p2p.bootnodes", ENODE_1]).is_empty());
        assert!(warnings(&["--p2p.no-discovery", "--p2p.static-peers", &static_peer]).is_empty());

        let no_peers = warnings(&["--p2p.no-discovery"]);
        assert!(
            matches!(no_peers.as_slice(), [w] if w.contains("no static peers")),
            "{no_peers:?}"
        );

        let bootnodes = warnings(&[
            "--p2p.no-discovery",
            "--p2p.static-peers",
            &static_peer,
            "--p2p.bootnodes",
            ENODE_1,
        ]);
        assert!(
            matches!(bootnodes.as_slice(), [w] if w.contains("bootnodes are ignored")),
            "{bootnodes:?}"
        );
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
    fn test_advertised_port_falls_back_to_listen_port() {
        assert_eq!(advertised_port(None, 9223), 9223);
        assert_eq!(advertised_port(Some(0), 9223), 9223);
        assert_eq!(advertised_port(Some(30303), 9223), 30303);
    }

    #[test]
    fn test_p2p_args_listen_udp_port() {
        let args = MockCommand::parse_from(["test", "--p2p.listen.udp", "1234"]);
        assert_eq!(args.p2p.listen_udp_port, 1234);
    }

    const ENODE_1: &str = "enode://ca2774c3c401325850b2477fd7d0f27911efbf79b1e8b335066516e2bd8c4c9e0ba9696a94b1cb030a88eac582305ff55e905e64fb77fe0edcd70a4e5296d3ec@34.65.175.185:30305";
    const ENODE_2: &str = "enode://dd751a9ef8912be1bfa7a5e34e2c3785cc5253110bd929f385e07ba7ac19929fb0e0c5d93f77827291f4da02b2232240fbc47ea7ce04c46e333e452f8656b667@34.65.107.0:30305";

    /// The bootnodes the CLI arguments parse to.
    fn bootnodes(args: &[&str]) -> Vec<BootNode> {
        MockCommand::parse_from(std::iter::once("test").chain(args.iter().copied())).p2p.bootnodes
    }

    #[test]
    fn test_p2p_args_bootnodes() {
        let record = NodeRecord::from_str(ENODE_1).unwrap();
        let expected = BootNode::from_unsigned(record).unwrap();
        assert_eq!(bootnodes(&["--p2p.bootnodes", ENODE_1]), vec![expected]);
    }

    #[test]
    fn test_p2p_args_bootnodes_multiple() {
        let expected = [ENODE_1, ENODE_2].map(|raw| BootNode::parse_bootnode(raw).unwrap());
        // A comma-separated list, as the environment variable needs, with spaces trimmed.
        assert_eq!(bootnodes(&["--p2p.bootnodes", &format!(" {ENODE_1}, {ENODE_2}")]), expected);
        // A repeated flag.
        assert_eq!(bootnodes(&["--p2p.bootnodes", ENODE_1, "--p2p.bootnodes", ENODE_2]), expected);
    }

    #[test]
    fn test_p2p_args_bootnode_enr() {
        let enr = "enr:-J64QBbwPjPLZ6IOOToOLsSjtFUjjzN66qmBZdUexpO32Klrc458Q24kbty2PdRaLacHM5z-cZQr8mjeQu3pik6jPSOGAYYFIqBfgmlkgnY0gmlwhDaRWFWHb3BzdGFja4SzlAUAiXNlY3AyNTZrMaECmeSnJh7zjKrDSPoNMGXoopeDF4hhpj5I0OsQUUt4u8uDdGNwgiQGg3VkcIIkBg";
        let expected = BootNode::parse_bootnode(enr).unwrap();
        assert!(matches!(expected, BootNode::Enr(_)));
        assert_eq!(bootnodes(&["--p2p.bootnodes", enr]), vec![expected]);
    }

    #[test]
    fn test_p2p_args_bootnodes_default_empty() {
        assert!(bootnodes(&[]).is_empty());
    }

    #[test]
    fn test_p2p_args_bootnodes_rejects_empty_entries() {
        for list in ["", ",", &format!("{ENODE_1},"), &format!("{ENODE_1}, ,{ENODE_2}")] {
            let err = MockCommand::try_parse_from(["test", "--p2p.bootnodes", list])
                .unwrap_err()
                .to_string();
            assert!(err.contains("Bootnode must not be empty"), "{list:?}: {err}");
        }
    }

    /// A malformed bootnode is a CLI error, not a panic when the config is built.
    #[test]
    fn test_p2p_args_bootnodes_rejects_invalid_entries() {
        let err =
            MockCommand::try_parse_from(["test", "--p2p.bootnodes", &format!("{ENODE_1},nope")])
                .unwrap_err()
                .to_string();
        assert!(err.contains("invalid value 'nope'"), "{err}");
        assert!(err.contains("Invalid enode URL"), "{err}");
    }

    #[test]
    fn test_p2p_args_static_peers_default_empty() {
        let args = MockCommand::parse_from(["test"]);
        assert!(args.p2p.static_peers.is_empty());
    }

    #[test]
    fn test_p2p_args_static_peers() {
        let (id1, id2) = (PeerId::random(), PeerId::random());
        let addr1 = format!("/ip4/10.0.0.1/tcp/9222/p2p/{id1}");
        let addr2 = format!("/dns4/node.example.com/tcp/9222/p2p/{id2}");
        let expected = vec![(id1, addr1.clone()), (id2, addr2.clone())];
        let static_peers = |args: &[&str]| {
            let args = MockCommand::parse_from(std::iter::once("test").chain(args.iter().copied()));
            args.p2p.static_peers.iter().map(|p| (p.peer_id(), p.to_string())).collect::<Vec<_>>()
        };
        for flag in ["--p2p.static-peers", "--p2p.static"] {
            // A comma-separated list, as the environment variable needs, with spaces trimmed.
            assert_eq!(static_peers(&[flag, &format!(" {addr1}, {addr2}")]), expected, "{flag}");
            // A repeated flag.
            assert_eq!(static_peers(&[flag, &addr1, flag, &addr2]), expected, "{flag}");
        }
    }

    #[test]
    fn test_p2p_args_static_peers_rejects_empty_entries() {
        let addr = format!("/ip4/10.0.0.1/tcp/9222/p2p/{}", PeerId::random());
        for list in ["", ",", &format!("{addr},"), &format!("{addr}, ,{addr}")] {
            let err = MockCommand::try_parse_from(["test", "--p2p.static-peers", list])
                .unwrap_err()
                .to_string();
            assert!(err.contains("Static peer must not be empty"), "{list:?}: {err}");
        }
    }

    #[test]
    fn test_p2p_args_static_peers_rejects_invalid_addresses() {
        let id = PeerId::random();
        let cases = [
            ("127.0.0.1:9222".to_string(), "Invalid multiaddr"),
            ("/ip4/127.0.0.1/tcp/9222".to_string(), "does not end in /p2p/<peer id>"),
            (format!("/p2p/{id}"), "nothing to dial"),
            (format!("/ip4/127.0.0.1/udp/9222/p2p/{id}"), "must have a /tcp/<port>"),
            (format!("/ip4/127.0.0.1/tcp/0/p2p/{id}"), "must not be 0"),
            (format!("/memory/1/p2p/{id}"), "must start with /ip4"),
            (format!("/ip4/127.0.0.1/tcp/9222/ws/p2p/{id}"), "must end in /tcp/<port>/p2p"),
            // Anything after the peer ID.
            (
                format!("/ip4/127.0.0.1/tcp/9222/p2p/{id}/p2p/{}", PeerId::random()),
                "must end in /tcp/<port>/p2p",
            ),
            // A UDP port next to the TCP one, in either order.
            (format!("/ip4/127.0.0.1/tcp/9222/udp/9223/p2p/{id}"), "must end in /tcp/<port>/p2p"),
            (format!("/ip4/127.0.0.1/udp/9223/tcp/9222/p2p/{id}"), "must have a /tcp/<port>"),
        ];
        for (addr, expected) in cases {
            let valid = format!("/ip4/127.0.0.1/tcp/9222/p2p/{id}");
            // One bad entry rejects the whole list.
            let err = MockCommand::try_parse_from([
                "test",
                "--p2p.static-peers",
                &format!("{valid},{addr}"),
            ])
            .unwrap_err()
            .to_string();
            assert!(err.contains(expected), "{addr}: {err}");
        }
    }

    #[test]
    fn test_p2p_args_listen_ip_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses
        // Using localhost which should resolve reliably
        let args = MockCommand::parse_from(["test", "--p2p.listen.ip", "localhost"]);
        // localhost typically resolves to 127.0.0.1 or ::1
        assert!(
            args.p2p.listen_ip == "127.0.0.1".parse::<IpAddr>().unwrap() ||
                args.p2p.listen_ip == "::1".parse::<IpAddr>().unwrap()
        );
    }

    #[test]
    fn test_p2p_args_advertise_ip_dns_resolution() {
        // Test that DNS hostnames are resolved to IP addresses for advertise_ip
        let args = MockCommand::parse_from(["test", "--p2p.advertise.ip", "localhost"]);
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
