//! Node Subcommand.

use crate::{
    cli_metrics::{CliMetrics, init_rollup_config_metrics},
    flags::{
        ChainArgs, DerivationDelegateArgs, L1ClientArgs, L2ClientArgs, P2PArgs, RpcArgs,
        SequencerArgs,
    },
};
use clap::Parser;
use kona_cli::{LogArgs, LogConfig, MetricsArgs};
use kona_node_service::{
    RollupNode, RollupNodeClients, RollupNodeConfig, RollupNodeEndpoints, RollupNodeError,
};
use tracing::{debug, error, info};

/// Command-line interface for running a Kona rollup node.
///
/// The `NodeCommand` struct defines all the configuration options needed to start and run
/// a rollup node in the Kona ecosystem, including its networking and RPC configuration options.
///
/// # Sequencing
///
/// A node given a block signer (`--sequencer.signer.key.path`, or `--sequencer.signer.url`
/// with `--sequencer.signer.address`) can sequence. Its sequencer starts
/// stopped; start it with `admin_startSequencer`. A node
/// without a block signer follows the chain only.
///
/// # Configuration Sources
///
/// Configuration can be provided through:
/// - Command-line arguments
/// - Environment variables (prefixed with `KONA_NODE_`)
/// - Configuration files (for rollup config)
///
/// # Examples
///
/// ```bash
/// # Follow OP Mainnet with default settings
/// kona-node node --chain optimism \
///                --l1-eth-rpc http://localhost:8545 \
///                --l1-beacon http://localhost:5052 \
///                --l2-engine-rpc http://localhost:8551 \
///                --l2-engine-jwt-secret /path/to/jwt.hex
///
/// # Run a node that can sequence a custom chain
/// kona-node node --l2-config-file /path/to/rollup.json \
///                --l1-eth-rpc http://localhost:8545 \
///                --l1-beacon http://localhost:5052 \
///                --l2-engine-rpc http://localhost:8551 \
///                --l2-engine-jwt-secret /path/to/jwt.hex \
///                --sequencer.signer.key.path /path/to/sequencer.key
/// ```
#[derive(Parser, Debug, Clone)]
#[command(about = "Runs the consensus node")]
pub struct NodeCommand {
    /// L1 RPC CLI arguments.
    #[clap(flatten)]
    pub l1_rpc_args: L1ClientArgs,

    /// L2 engine CLI arguments.
    #[clap(flatten)]
    pub l2_client_args: L2ClientArgs,

    /// Optional derivation delegation client.
    #[clap(flatten)]
    pub derivation_delegate_args: DerivationDelegateArgs,

    /// Chain CLI arguments.
    #[command(flatten)]
    pub chain: ChainArgs,
    /// P2P CLI arguments.
    #[command(flatten)]
    pub p2p_flags: P2PArgs,
    /// RPC CLI arguments.
    #[command(flatten)]
    pub rpc_flags: RpcArgs,
    /// SEQUENCER CLI arguments.
    #[command(flatten)]
    pub sequencer_flags: SequencerArgs,
    /// Logging CLI arguments.
    #[command(flatten)]
    pub log_args: LogArgs,
    /// Metrics CLI arguments.
    #[command(flatten)]
    pub metrics: MetricsArgs,
}

impl NodeCommand {
    /// Initializes the logging system based on the logging arguments.
    pub fn init_logs(&self) -> anyhow::Result<()> {
        // Filter out discovery warnings since they're very very noisy.
        let filter = tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("discv5=error".parse()?);

        LogConfig::new(self.log_args.clone()).init_tracing_subscriber(Some(filter))?;
        Ok(())
    }

    /// Initializes CLI metrics for the Node subcommand.
    pub fn init_cli_metrics(&self) -> anyhow::Result<()> {
        if !self.metrics.enabled {
            debug!("CLI metrics are disabled");
            return Ok(());
        }
        metrics::gauge!(
            CliMetrics::IDENTIFIER,
            &[
                (CliMetrics::P2P_PEER_SCORING_LEVEL, self.p2p_flags.scoring.to_string()),
                (CliMetrics::P2P_TOPIC_SCORING_ENABLED, self.p2p_flags.topic_scoring.to_string()),
                (CliMetrics::P2P_BANNING_ENABLED, self.p2p_flags.ban_enabled.to_string()),
                (
                    CliMetrics::P2P_PEER_REDIALING,
                    self.p2p_flags.peer_redial.unwrap_or(0).to_string()
                ),
                (CliMetrics::P2P_FLOOD_PUBLISH, self.p2p_flags.gossip_flood_publish.to_string()),
                (CliMetrics::P2P_DISCOVERY_INTERVAL, self.p2p_flags.discovery_interval.to_string()),
                (
                    CliMetrics::P2P_ADVERTISE_IP,
                    self.p2p_flags
                        .advertise_ip
                        .map(|ip| ip.to_string())
                        .unwrap_or_else(|| String::from("0.0.0.0"))
                ),
                (
                    CliMetrics::P2P_ADVERTISE_TCP_PORT,
                    self.p2p_flags
                        .advertise_tcp_port
                        .map_or_else(|| "auto".to_string(), |p| p.to_string())
                ),
                (
                    CliMetrics::P2P_ADVERTISE_UDP_PORT,
                    self.p2p_flags
                        .advertise_udp_port
                        .map_or_else(|| "auto".to_string(), |p| p.to_string())
                ),
                (CliMetrics::P2P_GOSSIP_MESH_D, self.p2p_flags.gossip_mesh_d.to_string()),
                (CliMetrics::P2P_GOSSIP_MESH_D_LO, self.p2p_flags.gossip_mesh_dlo.to_string()),
                (CliMetrics::P2P_GOSSIP_MESH_D_HI, self.p2p_flags.gossip_mesh_dhi.to_string()),
                (CliMetrics::P2P_GOSSIP_MESH_D_LAZY, self.p2p_flags.gossip_mesh_dlazy.to_string()),
                (CliMetrics::P2P_BAN_DURATION, self.p2p_flags.ban_duration.to_string()),
            ]
        )
        .set(1);
        Ok(())
    }

    /// Resolves the flags into the node's configuration.
    ///
    /// This reads the configuration and key files the flags name, and creates a P2P key file the
    /// flags name if it does not exist yet, but does no network I/O: the node checks the engine
    /// and its P2P ports, and reads its chain state from L1, when it starts.
    pub fn config(&self) -> anyhow::Result<RollupNodeConfig> {
        let chain = self.chain.chain_spec()?;
        Ok(RollupNodeConfig {
            version: crate::version::version().to_owned(),
            p2p_config: self.p2p_flags.clone().config(&chain.rollup_config)?,
            rpc_addr: self.rpc_flags.addr(),
            sequencer: self.sequencer_flags.clone().config()?,
            chain,
        })
    }

    /// Resolves the flags into the endpoints the node's clients connect to, reading the JWT
    /// secret file.
    pub fn endpoints(&self) -> anyhow::Result<RollupNodeEndpoints> {
        Ok(RollupNodeEndpoints {
            l1_rpc: self.l1_rpc_args.l1_eth_rpc.clone(),
            l1_beacon: self.l1_rpc_args.l1_beacon.clone(),
            l2_engine: self.l2_client_args.l2_engine_rpc.clone(),
            l2_jwt_secret: self.l2_client_args.jwt_secret()?,
            derivation_delegate: self.derivation_delegate_args.l2_follow_source.clone(),
        })
    }

    /// Run the Node subcommand.
    pub async fn run(self) -> anyhow::Result<()> {
        let config = self.config()?;
        let endpoints = self.endpoints()?;

        info!(
            target: "rollup_node",
            chain_id = config.chain.rollup_config.l2_chain_id.id(),
            "Starting rollup node services"
        );
        info!(
            target: "rollup_node",
            hardforks = %config.chain.rollup_config.hardforks,
            "Hardfork schedule"
        );

        // If metrics are enabled, initialize the global cli metrics.
        self.metrics.enabled.then(|| init_rollup_config_metrics(&config.chain.rollup_config));

        let clients = RollupNodeClients::connect(&endpoints)?;
        let Err(error) = RollupNode::new(config, clients).start().await else {
            return Ok(());
        };
        if matches!(error, RollupNodeError::JwtRejected) {
            error!(
                target: "rollup_node",
                "Check that --l2-engine-jwt-secret holds the engine's JWT secret, and that the \
                 node's and the engine's clocks agree"
            );
        }
        error!(target: "rollup_node", "Failed to start rollup node service: {error}");
        Err(error.into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_chains::Chain;
    use alloy_rpc_types_engine::JwtSecret;
    use kona_genesis::RollupConfig;
    use kona_peers::BootStoreFile;

    const fn default_flags() -> &'static [&'static str] {
        &[
            "--chain",
            "10",
            "--l1-eth-rpc",
            "http://localhost:8545",
            "--l1-beacon",
            "http://localhost:5052",
            "--l2-engine-rpc",
            "http://localhost:8551",
            "--l2-engine-jwt-secret",
            "/path/to/jwt.hex",
        ]
    }

    #[test]
    fn test_node_cli_defaults() {
        let args = NodeCommand::parse_from(
            std::iter::once(&"node").chain(default_flags().iter()).copied(),
        );
        assert_eq!(args.sequencer_flags, SequencerArgs::default());
    }

    #[test]
    fn test_node_cli_missing_l1_eth_rpc() {
        let err = NodeCommand::try_parse_from(["node"]).unwrap_err();
        assert!(err.to_string().contains("--l1-eth-rpc"));
    }

    #[test]
    fn test_node_cli_missing_l1_beacon() {
        let err = NodeCommand::try_parse_from(["node", "--l1-eth-rpc", "http://localhost:8545"])
            .unwrap_err();
        assert!(err.to_string().contains("--l1-beacon"));
    }

    #[test]
    fn test_node_cli_missing_l2_engine_rpc() {
        let err = NodeCommand::try_parse_from([
            "node",
            "--l1-eth-rpc",
            "http://localhost:8545",
            "--l1-beacon",
            "http://localhost:5052",
        ])
        .unwrap_err();
        assert!(err.to_string().contains("--l2-engine-rpc"));
    }

    /// Required flags whose URLs nothing listens on, so that any network I/O fails.
    const UNREACHABLE_FLAGS: [&str; 6] = [
        "--l1-eth-rpc",
        "http://127.0.0.1:1",
        "--l1-beacon",
        "http://127.0.0.1:2",
        "--l2-engine-rpc",
        "http://127.0.0.1:3",
    ];

    const JWT_HEX: &str = "0xabababababababababababababababababababababababababababababababab";

    /// Parses the node command with [`UNREACHABLE_FLAGS`], a JWT secret file holding [`JWT_HEX`]
    /// and `args`. The returned file must outlive the command's use of it.
    fn parse_node(args: &[&str]) -> (NodeCommand, tempfile::NamedTempFile) {
        let mut jwt = tempfile::NamedTempFile::new().unwrap();
        std::io::Write::write_all(&mut jwt, JWT_HEX.as_bytes()).unwrap();
        let jwt_path = jwt.path().to_str().unwrap().to_owned();
        let command = NodeCommand::try_parse_from(
            std::iter::once("node")
                .chain(UNREACHABLE_FLAGS)
                .chain(["--l2-engine-jwt-secret", &jwt_path])
                .chain(args.iter().copied()),
        )
        .unwrap();
        (command, jwt)
    }

    #[test]
    fn test_config_for_a_registry_chain() {
        let key = crate::flags::signer_test_utils::key_file(crate::flags::signer_test_utils::KEY);
        let (command, _jwt) = parse_node(&[
            "--chain",
            "optimism",
            "--sequencer.signer.key.path",
            key.path().to_str().unwrap(),
            "--sequencer.conductor-url",
            "http://127.0.0.1:4",
        ]);

        let config = command.config().unwrap();
        let endpoints = command.endpoints().unwrap();

        assert_eq!(config.chain.rollup_config.l2_chain_id.id(), 10);
        assert_eq!(config.chain.l1_config.chain_id, 1);
        assert_eq!(endpoints.l1_rpc.as_str(), "http://127.0.0.1:1/");
        assert_eq!(endpoints.l1_beacon.as_str(), "http://127.0.0.1:2/");
        assert_eq!(endpoints.l2_engine.as_str(), "http://127.0.0.1:3/");
        assert_eq!(endpoints.l2_jwt_secret, JwtSecret::from_hex(JWT_HEX).unwrap());
        assert_eq!(endpoints.derivation_delegate, None);
        let sequencer = config.sequencer.unwrap();
        assert_eq!(sequencer.conductor_rpc_url, Some("http://127.0.0.1:4".parse().unwrap()));
        assert_eq!(sequencer.l1_conf_delay, 4);
        assert_eq!(config.p2p_config.bootstore, Some(BootStoreFile::Default { chain_id: 10 }));
    }

    #[test]
    fn test_config_without_a_block_signer_cannot_sequence() {
        let (command, _jwt) = parse_node(&["--chain", "optimism"]);

        assert!(command.config().unwrap().sequencer.is_none());
    }

    #[test]
    fn test_sequencer_flags_require_a_block_signer() {
        let error = NodeCommand::try_parse_from(
            std::iter::once("node")
                .chain(default_flags().iter().copied())
                .chain(["--sequencer.conductor-url", "http://127.0.0.1:4"]),
        )
        .unwrap_err();

        assert_eq!(error.kind(), clap::error::ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn test_endpoints_with_a_follow_source() {
        let (command, _jwt) =
            parse_node(&["--chain", "optimism", "--l2-follow-source", "http://127.0.0.1:5"]);

        let endpoints = command.endpoints().unwrap();

        assert_eq!(endpoints.derivation_delegate, Some("http://127.0.0.1:5".parse().unwrap()));
    }

    #[test]
    fn test_config_from_rollup_and_l1_config_files() {
        let rollup = RollupConfig {
            l2_chain_id: Chain::from_id(901),
            l1_chain_id: 900,
            ..Default::default()
        };
        let mut rollup_file = tempfile::NamedTempFile::new().unwrap();
        serde_json::to_writer(&mut rollup_file, &rollup).unwrap();
        let mut l1_file = tempfile::NamedTempFile::new().unwrap();
        serde_json::to_writer(&mut l1_file, &serde_json::json!({"chainId": 900})).unwrap();

        let (command, _jwt) = parse_node(&[
            "--l2-config-file",
            rollup_file.path().to_str().unwrap(),
            "--l1-config-file",
            l1_file.path().to_str().unwrap(),
        ]);

        let config = command.config().unwrap();

        assert_eq!(config.chain.rollup_config.l2_chain_id.id(), 901);
        assert_eq!(config.chain.l1_config.chain_id, 900);
        assert_eq!(config.p2p_config.bootstore, Some(BootStoreFile::Default { chain_id: 901 }));
        assert_eq!(
            config.chain.dependency_set.dependencies.keys().copied().collect::<Vec<_>>(),
            [901]
        );
    }
}
