//! Node Subcommand.

use crate::{
    cli_metrics::{CliMetrics, init_rollup_config_metrics},
    flags::{
        ChainArgs, DerivationDelegateArgs, L1ClientArgs, L2ClientArgs, P2PArgs, RpcArgs,
        SequencerArgs,
    },
};
use backon::{ExponentialBuilder, Retryable};
use clap::Parser;
use kona_cli::{LogArgs, LogConfig, MetricsArgs};
use kona_engine::EngineClient;
use kona_node_service::{EngineConfig, L1ClientConfig, NodeMode, RollupNodeConfig};
use op_alloy_provider::ext::engine::OpEngineApi;
use std::sync::Arc;
use strum::IntoEnumIterator;
use tracing::{debug, error, info};

/// A JWT token validation error.
#[derive(Debug, thiserror::Error)]
pub(super) enum JwtValidationError {
    #[error("JWT signature is invalid")]
    InvalidSignature,
    #[error("Failed to exchange capabilities with engine: {0}")]
    CapabilityExchange(String),
}

/// Command-line interface for running a Kona rollup node.
///
/// The `NodeCommand` struct defines all the configuration options needed to start and run
/// a rollup node in the Kona ecosystem. It supports multiple node modes including validator
/// and sequencer modes, and provides comprehensive networking and RPC configuration options.
///
/// # Node Modes
///
/// The node can operate in different modes:
/// - **Validator**: Validates L2 blocks and participates in consensus
/// - **Sequencer**: Sequences transactions and produces L2 blocks
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
/// # Run as validator on OP Mainnet with default settings
/// kona-node node --chain optimism \
///                --l1-eth-rpc http://localhost:8545 \
///                --l1-beacon http://localhost:5052 \
///                --l2-engine-rpc http://localhost:8551 \
///                --l2-engine-jwt-secret /path/to/jwt.hex
///
/// # Run as sequencer for a custom chain
/// kona-node node --mode sequencer \
///                --l2-config-file /path/to/rollup.json \
///                --l1-eth-rpc http://localhost:8545 \
///                --l1-beacon http://localhost:5052 \
///                --l2-engine-rpc http://localhost:8551 \
///                --l2-engine-jwt-secret /path/to/jwt.hex \
///                --p2p.sequencer.key.path /path/to/p2p-sequencer.key
/// ```
#[derive(Parser, Debug, Clone)]
#[command(about = "Runs the consensus node")]
pub struct NodeCommand {
    /// The mode to run the node in.
    #[arg(
        long = "mode",
        default_value_t = NodeMode::Validator,
        env = "KONA_NODE_MODE",
        help = format!(
            "The mode to run the node in. Supported modes are: {}",
            NodeMode::iter()
                .map(|mode| format!("\"{}\"", mode.to_string()))
                .collect::<Vec<_>>()
                .join(", ")
        )
    )]
    pub node_mode: NodeMode,

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

    /// Check if the error is related to JWT signature validation
    fn is_jwt_signature_error(error: &dyn std::error::Error) -> bool {
        let mut source = Some(error);
        while let Some(err) = source {
            let err_str = err.to_string().to_lowercase();
            if err_str.contains("signature invalid") ||
                (err_str.contains("jwt") && err_str.contains("invalid")) ||
                err_str.contains("unauthorized") ||
                err_str.contains("authentication failed")
            {
                return true;
            }
            source = err.source();
        }
        false
    }

    /// Helper to check JWT signature error from `anyhow::Error` (for retry condition)
    fn is_jwt_signature_error_from_anyhow(error: &anyhow::Error) -> bool {
        Self::is_jwt_signature_error(error.as_ref() as &dyn std::error::Error)
    }

    /// Validates the engine's JWT secret by exchanging capabilities with the engine, which fails
    /// if the engine was given a different secret. Retries other failures, such as an engine
    /// that is not up yet.
    async fn validate_jwt(engine_config: &EngineConfig) -> anyhow::Result<()> {
        let engine =
            EngineClient::rpc_client(engine_config.l2_url.clone(), engine_config.l2_jwt_secret);

        let exchange = || async {
            match engine.exchange_capabilities(vec![]).await {
                Ok(_) => {
                    debug!("Successfully exchanged capabilities with engine");
                    Ok(())
                }
                Err(e) => {
                    if Self::is_jwt_signature_error(&e) {
                        error!(
                            "Engine API JWT secret differs from the one in --l2-engine-jwt-secret"
                        );
                        return Err(JwtValidationError::InvalidSignature.into());
                    }
                    Err(JwtValidationError::CapabilityExchange(e.to_string()).into())
                }
            }
        };

        exchange
            .retry(ExponentialBuilder::default())
            .when(|e| !Self::is_jwt_signature_error_from_anyhow(e))
            .notify(|_, duration| {
                debug!("Retrying engine capability handshake after {duration:?}");
            })
            .await
    }

    /// Resolves the flags into the node's configuration.
    ///
    /// This reads the configuration and key files the flags name, and creates a P2P key file the
    /// flags name if it does not exist yet, but does no network I/O: [`Self::run`] checks the
    /// engine and the P2P ports, and the node reads its chain state from L1 when it starts.
    pub fn config(&self) -> anyhow::Result<RollupNodeConfig> {
        let cfg = self.chain.rollup_config()?;

        let l1_client = L1ClientConfig {
            chain_config: self.chain.l1_config(cfg.l1_chain_id)?,
            trust_rpc: self.l1_rpc_args.l1_trust_rpc,
            beacon: self.l1_rpc_args.l1_beacon.clone(),
            rpc_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            slot_duration_override: self.l1_rpc_args.l1_slot_duration_override,
        };

        let engine_config = EngineConfig {
            config: Arc::new(cfg.clone()),
            l2_url: self.l2_client_args.l2_engine_rpc.clone(),
            l2_jwt_secret: self.l2_client_args.jwt_secret()?,
            l1_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            mode: self.node_mode,
        };

        Ok(RollupNodeConfig {
            version: crate::version::version().to_owned(),
            l1_client,
            l2_trust_rpc: self.l2_client_args.l2_trust_rpc,
            engine_config,
            p2p_config: self.p2p_flags.clone().config(&cfg)?,
            rpc_config: self.rpc_flags.clone().into(),
            sequencer_config: self.sequencer_flags.config(),
            derivation_delegate_config: self.derivation_delegate_args.clone().config(),
            dependency_set: self.chain.dependency_set(&cfg)?,
            config: cfg,
        })
    }

    /// Run the Node subcommand.
    pub async fn run(self) -> anyhow::Result<()> {
        let config = self.config()?;

        info!(
            target: "rollup_node",
            chain_id = config.config.l2_chain_id.id(),
            "Starting rollup node services"
        );
        info!(
            target: "rollup_node",
            hardforks = %config.config.hardforks,
            "Hardfork schedule"
        );

        // If metrics are enabled, initialize the global cli metrics.
        self.metrics.enabled.then(|| init_rollup_config_metrics(&config.config));

        // Check the host and the engine before starting the node.
        self.p2p_flags.check_ports()?;
        Self::validate_jwt(&config.engine_config).await?;

        config.build().start().await.map_err(|e| {
            error!(target: "rollup_node", "Failed to start rollup node service: {e}");
            anyhow::anyhow!("{e}")
        })?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_chains::Chain;
    use alloy_rpc_types_engine::JwtSecret;
    use anyhow::anyhow;
    use kona_genesis::RollupConfig;
    use kona_peers::BootStoreFile;

    #[derive(Debug)]
    struct MockError {
        message: String,
    }

    impl std::fmt::Display for MockError {
        fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            write!(f, "{}", self.message)
        }
    }

    impl std::error::Error for MockError {}

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
        assert_eq!(args.node_mode, NodeMode::Validator);
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
        let (command, _jwt) = parse_node(&[
            "--chain",
            "optimism",
            "--mode",
            "sequencer",
            "--conductor.rpc",
            "http://127.0.0.1:4",
        ]);

        let config = command.config().unwrap();

        assert_eq!(config.config.l2_chain_id.id(), 10);
        assert_eq!(config.l1_client.chain_config.chain_id, 1);
        assert_eq!(config.l1_client.rpc_url.as_str(), "http://127.0.0.1:1/");
        assert_eq!(config.l1_client.beacon.as_str(), "http://127.0.0.1:2/");
        assert_eq!(config.engine_config.l2_url.as_str(), "http://127.0.0.1:3/");
        assert_eq!(config.engine_config.l2_jwt_secret, JwtSecret::from_hex(JWT_HEX).unwrap());
        assert_eq!(config.engine_config.mode, NodeMode::Sequencer);
        assert_eq!(
            config.sequencer_config.conductor_rpc_url,
            Some("http://127.0.0.1:4".parse().unwrap())
        );
        assert_eq!(config.p2p_config.bootstore, Some(BootStoreFile::Default { chain_id: 10 }));
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

        assert_eq!(config.config.l2_chain_id.id(), 901);
        assert_eq!(config.engine_config.config.l2_chain_id.id(), 901);
        assert_eq!(config.l1_client.chain_config.chain_id, 900);
        assert_eq!(config.p2p_config.bootstore, Some(BootStoreFile::Default { chain_id: 901 }));
        assert!(config.dependency_set.is_none());
    }

    #[test]
    fn test_is_jwt_signature_error() {
        let jwt_error = MockError { message: "signature invalid".to_string() };
        assert!(NodeCommand::is_jwt_signature_error(&jwt_error));

        let other_error = MockError { message: "network timeout".to_string() };
        assert!(!NodeCommand::is_jwt_signature_error(&other_error));
    }

    #[test]
    fn test_is_jwt_signature_error_from_anyhow() {
        let jwt_anyhow_error = anyhow!("signature invalid");
        assert!(NodeCommand::is_jwt_signature_error_from_anyhow(&jwt_anyhow_error));

        let other_anyhow_error = anyhow!("network timeout");
        assert!(!NodeCommand::is_jwt_signature_error_from_anyhow(&other_anyhow_error));
    }
}
