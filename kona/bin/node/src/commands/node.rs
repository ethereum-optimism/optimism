//! Node Subcommand.

use crate::{
    flags::{
        BuilderClientArgs, GlobalArgs, L1ClientArgs, L2ClientArgs, P2PArgs, RollupBoostFlags,
        RpcArgs, SequencerArgs,
    },
    metrics::{CliMetrics, init_rollup_config_metrics},
};
use alloy_provider::RootProvider;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport_http::Http;
use anyhow::{Result, bail};
use backon::{ExponentialBuilder, Retryable};
use clap::Parser;
use kona_cli::{LogConfig, MetricsArgs};
use kona_engine::{HyperAuthClient, OpEngineClient};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_node_service::{EngineConfig, L1ConfigBuilder, NodeMode, RollupNodeBuilder};
use kona_registry::{L1Config, scr_rollup_config_by_alloy_ident};
use op_alloy_network::Optimism;
use op_alloy_provider::ext::engine::OpEngineApi;
use serde_json::from_reader;
use std::{fs::File, io::Write, path::PathBuf, sync::Arc, time::Duration};
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
/// # Run as validator with default settings
/// kona node --l1-eth-rpc http://localhost:8545 \
///           --l1-beacon http://localhost:5052 \
///           --l2-engine-rpc http://localhost:8551
///
/// # Run as sequencer with custom JWT secret
/// kona node --mode sequencer \
///           --l1-eth-rpc http://localhost:8545 \
///           --l1-beacon http://localhost:5052 \
///           --l2-engine-rpc http://localhost:8551 \
///           --l2-engine-jwt-secret /path/to/jwt.hex
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

    /// Optional block builder client.
    #[clap(flatten)]
    pub builder_client_args: BuilderClientArgs,

    /// Path to a custom L2 rollup configuration file
    /// (overrides the default rollup configuration from the registry)
    #[arg(long, visible_alias = "rollup-cfg", env = "KONA_NODE_ROLLUP_CONFIG")]
    pub l2_config_file: Option<PathBuf>,
    /// Path to a custom L1 rollup configuration file
    /// (overrides the default rollup configuration from the registry)
    #[arg(long, visible_alias = "rollup-l1-cfg", env = "KONA_NODE_L1_CHAIN_CONFIG")]
    pub l1_config_file: Option<PathBuf>,
    /// P2P CLI arguments.
    #[command(flatten)]
    pub p2p_flags: P2PArgs,
    /// RPC CLI arguments.
    #[command(flatten)]
    pub rpc_flags: RpcArgs,
    /// SEQUENCER CLI arguments.
    #[command(flatten)]
    pub sequencer_flags: SequencerArgs,

    /// Rollup boost CLI arguments - contains the builder and l2 engine arguments.
    #[command(flatten)]
    pub rollup_boost_flags: RollupBoostFlags,
}

impl Default for NodeCommand {
    fn default() -> Self {
        Self {
            l1_rpc_args: L1ClientArgs::default(),
            l2_client_args: L2ClientArgs::default(),
            builder_client_args: BuilderClientArgs::default(),
            l2_config_file: None,
            l1_config_file: None,
            node_mode: NodeMode::Validator,
            p2p_flags: P2PArgs::default(),
            rpc_flags: RpcArgs::default(),
            sequencer_flags: SequencerArgs::default(),
            rollup_boost_flags: RollupBoostFlags::default(),
        }
    }
}

impl NodeCommand {
    /// Initializes the logging system based on global arguments.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        // Filter out discovery warnings since they're very very noisy.
        let filter = tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("discv5=error".parse()?);

        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(Some(filter))?;
        Ok(())
    }

    /// Initializes CLI metrics for the Node subcommand.
    pub fn init_cli_metrics(&self, args: &MetricsArgs) -> anyhow::Result<()> {
        if !args.enabled {
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
                        .unwrap_or(String::from("0.0.0.0"))
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
                (CliMetrics::P2P_PEERS_LO, self.p2p_flags.peers_lo.to_string()),
                (CliMetrics::P2P_PEERS_HI, self.p2p_flags.peers_hi.to_string()),
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

    /// Helper to check JWT signature error from anyhow::Error (for retry condition)
    fn is_jwt_signature_error_from_anyhow(error: &anyhow::Error) -> bool {
        Self::is_jwt_signature_error(error.as_ref() as &dyn std::error::Error)
    }

    /// Validate the jwt secret if specified by exchanging capabilities with the engine.
    /// Since the engine client will fail if the jwt token is invalid, this allows to ensure
    /// that the jwt token passed as a cli arg is correct.
    pub async fn validate_jwt(&self) -> anyhow::Result<JwtSecret> {
        let jwt_secret = self.l2_jwt_secret()?;

        let engine = OpEngineClient::<RootProvider, RootProvider<Optimism>>::rpc_client::<Optimism>(
            self.l2_client_args.l2_engine_rpc.clone(),
            jwt_secret,
        );

        let exchange = || async {
            match <RootProvider<Optimism> as OpEngineApi<
                Optimism,
                Http<HyperAuthClient>,
            >>::exchange_capabilities(&engine, vec![])
            .await
            {
                Ok(_) => {
                    debug!("Successfully exchanged capabilities with engine");
                    Ok(jwt_secret)
                }
                Err(e) => {
                    if Self::is_jwt_signature_error(&e) {
                        error!(
                            "Engine API JWT secret differs from the one specified by --l2.jwt-secret"
                        );
                        error!(
                            "Ensure that the JWT secret file specified is correct (by default it is `jwt.hex` in the current directory)"
                        );
                        return Err(JwtValidationError::InvalidSignature.into())
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

    /// Run the Node subcommand.
    pub async fn run(self, args: &GlobalArgs) -> anyhow::Result<()> {
        let cfg = self.get_l2_config(args)?;

        info!(
            target: "rollup_node",
            chain_id = cfg.l2_chain_id.id(),
            "Starting rollup node services"
        );
        for hf in cfg.hardforks.to_string().lines() {
            info!(target: "rollup_node", "{hf}");
        }

        let l1_config = L1ConfigBuilder {
            chain_config: self.get_l1_config(cfg.l1_chain_id)?,
            trust_rpc: self.l1_rpc_args.l1_trust_rpc,
            beacon: self.l1_rpc_args.l1_beacon.clone(),
            rpc_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            slot_duration_override: self.l1_rpc_args.l1_slot_duration_override,
        };

        // If metrics are enabled, initialize the global cli metrics.
        args.metrics.enabled.then(|| init_rollup_config_metrics(&cfg));

        let jwt_secret = self.validate_jwt().await?;

        self.p2p_flags.check_ports()?;
        let p2p_config = self
            .p2p_flags
            .clone()
            .config(&cfg, args, Some(self.l1_rpc_args.l1_eth_rpc.clone()))
            .await?;
        let rpc_config = self.rpc_flags.clone().into();

        let engine_config = EngineConfig {
            config: Arc::new(cfg.clone()),
            builder_url: self.builder_client_args.l2_builder_rpc.clone(),
            builder_jwt_secret: self.builder_jwt_secret()?,
            builder_timeout: Duration::from_millis(self.builder_client_args.builder_timeout),
            l2_url: self.l2_client_args.l2_engine_rpc.clone(),
            l2_jwt_secret: jwt_secret,
            l2_timeout: Duration::from_millis(self.l2_client_args.l2_engine_timeout),
            l1_url: self.l1_rpc_args.l1_eth_rpc.clone(),
            mode: self.node_mode,
            rollup_boost: self.rollup_boost_flags.as_rollup_boost_args(),
        };

        RollupNodeBuilder::new(
            cfg,
            l1_config,
            self.l2_client_args.l2_trust_rpc,
            engine_config,
            p2p_config,
            rpc_config,
        )
        .with_sequencer_config(self.sequencer_flags.config())
        .build()
        .start()
        .await
        .map_err(|e| {
            error!(target: "rollup_node", "Failed to start rollup node service: {e}");
            anyhow::anyhow!("{e}")
        })?;

        Ok(())
    }

    /// Get the L1 config, either from a file or the known chains.
    pub fn get_l1_config(&self, l1_chain_id: u64) -> Result<L1ChainConfig> {
        match &self.l1_config_file {
            Some(path) => {
                debug!("Loading l1 config from file: {:?}", path);
                let file = File::open(path)
                    .map_err(|e| anyhow::anyhow!("Failed to open l1 config file: {e}"))?;
                from_reader(file).map_err(|e| anyhow::anyhow!("Failed to parse l1 config: {e}"))
            }
            None => {
                debug!("Loading l1 config from known chains");
                let cfg = L1Config::get_l1_genesis(l1_chain_id).map_err(|e| {
                    anyhow::anyhow!("Failed to find l1 config for chain ID {l1_chain_id}: {e}")
                })?;
                Ok(cfg.into())
            }
        }
    }

    /// Get the L2 rollup config, either from a file or the superchain registry.
    pub fn get_l2_config(&self, args: &GlobalArgs) -> Result<RollupConfig> {
        match &self.l2_config_file {
            Some(path) => {
                debug!("Loading l2 config from file: {:?}", path);
                let file = File::open(path)
                    .map_err(|e| anyhow::anyhow!("Failed to open l2 config file: {e}"))?;
                from_reader(file).map_err(|e| anyhow::anyhow!("Failed to parse l2 config: {e}"))
            }
            None => {
                debug!("Loading l2 config from superchain registry");
                let Some(cfg) = scr_rollup_config_by_alloy_ident(&args.l2_chain_id) else {
                    bail!("Failed to find l2 config for chain ID {}", args.l2_chain_id);
                };
                Ok(cfg.clone())
            }
        }
    }

    /// Returns the L2 JWT secret for the engine API
    /// using the provided [PathBuf]. If the file is not found,
    /// it will return the default JWT secret.
    pub fn l2_jwt_secret(&self) -> anyhow::Result<JwtSecret> {
        if let Some(path) = &self.l2_client_args.l2_engine_jwt_secret &&
            let Ok(secret) = std::fs::read_to_string(path)
        {
            return JwtSecret::from_hex(secret)
                .map_err(|e| anyhow::anyhow!("Failed to parse JWT secret: {e}"));
        }

        if let Some(secret) = &self.l2_client_args.l2_engine_jwt_encoded {
            return Ok(*secret);
        }

        Self::default_jwt_secret("l2_jwt.hex")
    }

    /// Returns the builder JWT secret for the engine API
    /// using the provided [PathBuf]. If the file is not found,
    /// it will return the default JWT secret.
    pub fn builder_jwt_secret(&self) -> anyhow::Result<JwtSecret> {
        if let Some(path) = &self.builder_client_args.builder_jwt_path &&
            let Ok(secret) = std::fs::read_to_string(path)
        {
            return JwtSecret::from_hex(secret)
                .map_err(|e| anyhow::anyhow!("Failed to parse JWT secret: {e}"));
        }

        if let Some(secret) = &self.builder_client_args.builder_jwt_secret {
            return Ok(*secret);
        }

        Self::default_jwt_secret("builder_jwt.hex")
    }

    /// Uses the current directory to attempt to read
    /// the JWT secret from a file named `file_name`.
    /// If the file is not found, it will create a new random JWT secret and write it to the file.
    pub fn default_jwt_secret(file_name: &str) -> anyhow::Result<JwtSecret> {
        let cur_dir = std::env::current_dir()
            .map_err(|e| anyhow::anyhow!("Failed to get current directory: {e}"))?;
        std::fs::read_to_string(cur_dir.join(file_name)).map_or_else(
            |_| {
                let secret = JwtSecret::random();

                if let Ok(mut file) = File::create(file_name) &&
                    let Err(e) = file
                        .write_all(alloy_primitives::hex::encode(secret.as_bytes()).as_bytes())
                {
                    return Err(anyhow::anyhow!("Failed to write JWT secret to file: {e}"));
                }

                Ok(secret)
            },
            |content| Ok(JwtSecret::from_hex(content)?),
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use anyhow::anyhow;

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
            "--l1-eth-rpc",
            "http://localhost:8545",
            "--l1-beacon",
            "http://localhost:5052",
            "--l2-engine-rpc",
            "http://localhost:8551",
        ]
    }

    #[test]
    fn test_node_cli_defaults() {
        let args = NodeCommand::parse_from(["node"].iter().chain(default_flags().iter()).copied());
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
