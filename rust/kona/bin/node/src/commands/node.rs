//! Node Subcommand.

use crate::{
    cli_metrics::{CliMetrics, init_rollup_config_metrics},
    flags::{
        ChainArgs, DerivationDelegateArgs, L1ClientArgs, L2ClientArgs, LogArgs, MetricsArgs,
        P2PArgs, RpcArgs, SequencerArgs, SignerArgs,
    },
};
use clap::{CommandFactory, Parser, error::ErrorKind};
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
/// A node run with `--sequencer.enabled` can sequence. It needs `--rpc.enabled`: its sequencer
/// starts stopped; start it with `admin_startSequencer`. With `--p2p.enabled`, it gossips the
/// blocks it builds, and needs a block signer to sign them: `--signer.key-path`, or `--signer.url`
/// with `--signer.address`. A node without `--sequencer.enabled` follows the chain only.
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
///                --l1.eth-url http://localhost:8545 \
///                --l1.beacon-url http://localhost:5052 \
///                --l2.engine.url http://localhost:8551 \
///                --l2.engine.jwt-secret-path /path/to/jwt.hex
///
/// # Run a node that can sequence a custom chain
/// kona-node node --sequencer.enabled \
///                --rpc.enabled \
///                --chain.rollup-config-path /path/to/rollup.json \
///                --l1.eth-url http://localhost:8545 \
///                --l1.beacon-url http://localhost:5052 \
///                --l2.engine.url http://localhost:8551 \
///                --l2.engine.jwt-secret-path /path/to/jwt.hex \
///                --p2p.enabled \
///                --signer.key-path /path/to/sequencer.key
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
    /// Block signer CLI arguments.
    #[command(flatten)]
    pub signer_flags: SignerArgs,
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

        self.log_args.config().init_tracing_subscriber(Some(filter))?;
        Ok(())
    }

    /// Initializes CLI metrics for the Node subcommand. They record the P2P settings, so only a
    /// node with P2P on publishes them.
    pub fn init_cli_metrics(&self) -> anyhow::Result<()> {
        if !self.metrics.enabled || !self.p2p_flags.enabled {
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
                (
                    CliMetrics::P2P_DISCOVERY_INTERVAL,
                    humantime::format_duration(self.p2p_flags.discovery_interval).to_string()
                ),
                (
                    CliMetrics::P2P_ADVERTISE_IP,
                    self.p2p_flags
                        .advertise_addr
                        .clone()
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
                (
                    CliMetrics::P2P_BAN_DURATION,
                    humantime::format_duration(self.p2p_flags.ban_duration).to_string()
                ),
            ]
        )
        .set(1);
        Ok(())
    }

    /// Checks how the flags combine where clap cannot, because the rule depends on whether
    /// features are enabled. The errors are clap's, so they read and exit like clap's own.
    ///
    /// - `--sequencer.enabled` requires `--rpc.enabled`: a sequencer starts stopped, and is started
    ///   through the admin API.
    /// - With `--sequencer.enabled` and `--p2p.enabled`, the signer flags must choose exactly one
    ///   complete block signer. Otherwise they are ignored.
    ///
    /// This reads no files: [`Self::config`] reads them.
    pub fn check(&self) -> Result<(), clap::Error> {
        if self.sequencer_flags.enabled && !self.rpc_flags.enabled {
            return Err(Self::error(
                ErrorKind::MissingRequiredArgument,
                "--sequencer.enabled requires --rpc.enabled",
            ));
        }
        if self.signs_blocks() {
            self.signer_flags.check().map_err(|error| {
                let kind = error.kind().expect("checking the signer flags reads no key file");
                Self::error(kind, error)
            })?;
        }
        Ok(())
    }

    /// Returns a clap error for the node subcommand, which carries its usage.
    fn error(kind: ErrorKind, message: impl std::fmt::Display) -> clap::Error {
        let mut cli = crate::cli::Cli::command();
        cli.build();
        cli.find_subcommand_mut("node")
            .expect("the CLI has the node subcommand")
            .error(kind, message)
    }

    /// Whether the node signs the blocks it builds, which it does only to gossip them: when it
    /// can sequence and runs P2P.
    const fn signs_blocks(&self) -> bool {
        self.sequencer_flags.enabled && self.p2p_flags.enabled
    }

    /// Resolves the flags into the node's configuration, after [checking](Self::check) them.
    ///
    /// This reads the configuration and key files the flags name, and creates a P2P key file the
    /// flags name if it does not exist yet, but does no network I/O: the node checks the engine
    /// and its P2P ports, and reads its chain state from L1, when it starts.
    pub fn config(&self) -> anyhow::Result<RollupNodeConfig> {
        self.check()?;
        let chain = self.chain.chain_spec()?;
        Ok(RollupNodeConfig {
            version: crate::version::version().to_owned(),
            p2p_config: self
                .p2p_flags
                .enabled
                .then(|| self.p2p_flags.clone().config(&chain.rollup_config))
                .transpose()?,
            rpc_addr: self.rpc_flags.addr(),
            sequencer: self.sequencer_flags.clone().config(),
            signer: self.signs_blocks().then(|| self.signer_flags.clone().config()).transpose()?,
            chain,
        })
    }

    /// Resolves the flags into the endpoints the node's clients connect to, reading the JWT
    /// secret file.
    pub fn endpoints(&self) -> anyhow::Result<RollupNodeEndpoints> {
        Ok(RollupNodeEndpoints {
            l1_rpc: self.l1_rpc_args.eth_url.clone(),
            l1_beacon: self.l1_rpc_args.beacon_url.clone(),
            l2_engine: self.l2_client_args.engine_url.clone(),
            l2_jwt_secret: self.l2_client_args.jwt_secret()?,
            derivation_delegate: self.derivation_delegate_args.cl_url.clone(),
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
                "Check that --l2.engine.jwt-secret-path holds the engine's JWT secret, and that the \
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
            "--l1.eth-url",
            "http://localhost:8545",
            "--l1.beacon-url",
            "http://localhost:5052",
            "--l2.engine.url",
            "http://localhost:8551",
            "--l2.engine.jwt-secret-path",
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
    fn test_node_cli_missing_l1_eth_url() {
        let err = NodeCommand::try_parse_from(["node"]).unwrap_err();
        assert!(err.to_string().contains("--l1.eth-url"));
    }

    #[test]
    fn test_node_cli_missing_l1_beacon_url() {
        let err = NodeCommand::try_parse_from(["node", "--l1.eth-url", "http://localhost:8545"])
            .unwrap_err();
        assert!(err.to_string().contains("--l1.beacon-url"));
    }

    #[test]
    fn test_node_cli_missing_l2_engine_url() {
        let err = NodeCommand::try_parse_from([
            "node",
            "--l1.eth-url",
            "http://localhost:8545",
            "--l1.beacon-url",
            "http://localhost:5052",
        ])
        .unwrap_err();
        assert!(err.to_string().contains("--l2.engine.url"));
    }

    /// Required flags whose URLs nothing listens on, so that any network I/O fails.
    const UNREACHABLE_FLAGS: [&str; 6] = [
        "--l1.eth-url",
        "http://127.0.0.1:1",
        "--l1.beacon-url",
        "http://127.0.0.1:2",
        "--l2.engine.url",
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
                .chain(["--l2.engine.jwt-secret-path", &jwt_path])
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
            "--rpc.enabled",
            "--sequencer.enabled",
            "--sequencer.conductor.url",
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
        // P2P is off by default.
        assert!(config.p2p_config.is_none());
    }

    /// The P2P flags apply only with `--p2p.enabled`: without it, none is resolved or parsed
    /// beyond its syntax.
    #[test]
    fn test_p2p_flags_are_ignored_without_p2p() {
        let (command, _jwt) = parse_node(&[
            "--chain",
            "optimism",
            "--p2p.enabled=false",
            "--p2p.listen.addr",
            "this-hostname-definitely-does-not-exist.invalid",
            "--p2p.advertise.addr",
            "this-hostname-definitely-does-not-exist.invalid",
            "--p2p.discovery.enabled",
            "--p2p.discovery.bootnodes",
            "enode://malformed",
            "--p2p.discovery.bootstore.enabled",
        ]);

        assert!(command.config().unwrap().p2p_config.is_none());
    }

    #[test]
    fn test_config_without_a_block_signer_cannot_sequence() {
        let (command, _jwt) = parse_node(&["--chain", "optimism"]);

        assert!(command.config().unwrap().sequencer.is_none());
    }

    #[test]
    fn test_sequencer_flags_are_ignored_without_the_sequencer() {
        let (command, _jwt) =
            parse_node(&["--chain", "optimism", "--sequencer.conductor.url", "http://127.0.0.1:4"]);

        assert!(command.config().unwrap().sequencer.is_none());
    }

    /// A node signs blocks only to gossip the blocks it builds, so it needs a signer only with
    /// both the sequencer and P2P, and ignores the signer flags, unchecked and unread, otherwise.
    #[test]
    fn test_a_signer_is_used_and_required_only_with_the_sequencer_and_p2p() {
        let key = crate::flags::signer_test_utils::key_file(crate::flags::signer_test_utils::KEY);
        const ADDRESS: &str = "0x0000000000000000000000000000000000000001";
        // Each choice of signer flags, with `KEY` for the key file, and the clap error it is with
        // the sequencer and P2P.
        let signers: [(&[&str], Option<ErrorKind>); 4] = [
            (&[], Some(ErrorKind::MissingRequiredArgument)),
            (&["--signer.key-path", "KEY"], None),
            (
                &["--signer.key-path", "KEY", "--signer.url", "http://a"],
                Some(ErrorKind::MissingRequiredArgument),
            ),
            (
                &[
                    "--signer.key-path",
                    "KEY",
                    "--signer.url",
                    "http://a",
                    "--signer.address",
                    ADDRESS,
                ],
                Some(ErrorKind::ArgumentConflict),
            ),
        ];
        for (sequencer, p2p) in [(false, false), (false, true), (true, false), (true, true)] {
            let used = sequencer && p2p;
            // A node that ignores the signer must not read its key file, so give it none.
            let key_path = if used { key.path().to_str().unwrap() } else { "/nonexistent/key" };
            for (signer, error) in &signers {
                let mut args = vec![
                    "--chain",
                    "optimism",
                    "--rpc.enabled",
                    if sequencer { "--sequencer.enabled" } else { "--sequencer.enabled=false" },
                    if p2p { "--p2p.enabled" } else { "--p2p.enabled=false" },
                ];
                args.extend(signer.iter().map(|arg| if *arg == "KEY" { key_path } else { arg }));
                let (command, _jwt) = parse_node(&args);
                let case = format!("sequencer {sequencer}, p2p {p2p}, {signer:?}");

                match (used, error) {
                    (false, _) => {
                        command.check().unwrap_or_else(|e| panic!("{case}: {e}"));
                        assert!(command.config().unwrap().signer.is_none(), "{case}");
                    }
                    (true, Some(kind)) => {
                        assert_eq!(command.check().unwrap_err().kind(), *kind, "{case}")
                    }
                    (true, None) => assert!(command.config().unwrap().signer.is_some(), "{case}"),
                }
            }
        }
    }

    /// The signer's key file is read when the node is configured, not when its flags are checked.
    #[test]
    fn test_a_missing_signer_key_file_fails_the_config() {
        let (command, _jwt) = parse_node(&[
            "--chain",
            "optimism",
            "--rpc.enabled",
            "--sequencer.enabled",
            "--p2p.enabled",
            "--signer.key-path",
            "/nonexistent/sequencer.key",
        ]);

        command.check().unwrap();
        assert!(command.config().is_err());
    }

    /// A sequencer starts stopped, and is started through the admin API.
    #[test]
    fn test_sequencer_requires_the_rpc() {
        let (command, _jwt) = parse_node(&["--chain", "optimism", "--sequencer.enabled"]);

        let error = command.check().unwrap_err();
        assert_eq!(error.kind(), ErrorKind::MissingRequiredArgument);
        assert!(error.to_string().contains("--sequencer.enabled requires --rpc.enabled"));
        assert!(error.to_string().contains("Usage: kona-node node"), "{error}");
        assert!(command.config().is_err());
    }

    #[test]
    fn test_endpoints_with_a_follow_source() {
        let (command, _jwt) =
            parse_node(&["--chain", "optimism", "--l2.cl-url", "http://127.0.0.1:5"]);

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
            "--chain.rollup-config-path",
            rollup_file.path().to_str().unwrap(),
            "--chain.l1-config-path",
            l1_file.path().to_str().unwrap(),
            "--p2p.enabled",
            "--p2p.discovery.enabled",
            "--p2p.discovery.bootstore.enabled",
        ]);

        let config = command.config().unwrap();

        assert_eq!(config.chain.rollup_config.l2_chain_id.id(), 901);
        assert_eq!(config.chain.l1_config.chain_id, 900);
        let p2p_config = config.p2p_config.unwrap();
        assert!(p2p_config.discovery_enabled);
        assert_eq!(p2p_config.bootstore, Some(BootStoreFile::Default { chain_id: 901 }));
        assert_eq!(
            config.chain.dependency_set.dependencies.keys().copied().collect::<Vec<_>>(),
            [901]
        );
    }
}
