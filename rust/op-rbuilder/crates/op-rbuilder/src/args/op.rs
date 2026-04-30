//! Additional Node command arguments.
//!
//! Copied from OptimismNode to allow easy extension.

//! clap [Args](clap::Args) for optimism rollup configuration

use crate::{
    flashtestations::args::FlashtestationsArgs, gas_limiter::args::GasLimiterArgs,
    tx_signer::Signer,
};
use alloy_primitives::Address;
use anyhow::{Result, anyhow};
use clap::Parser;
use reth_optimism_cli::commands::Commands;
use reth_optimism_node::args::RollupArgs;
use std::path::PathBuf;

/// Parameters for rollup configuration
#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
#[command(next_help_heading = "Rollup")]
pub struct OpRbuilderArgs {
    /// Rollup configuration
    #[command(flatten)]
    pub rollup_args: RollupArgs,
    /// Builder secret key for signing last transaction in block
    #[arg(long = "rollup.builder-secret-key", env = "BUILDER_SECRET_KEY")]
    pub builder_signer: Option<Signer>,

    /// chain block time in milliseconds
    #[arg(
        long = "rollup.chain-block-time",
        default_value = "1000",
        env = "CHAIN_BLOCK_TIME"
    )]
    pub chain_block_time: u64,

    /// max gas a transaction can use
    #[arg(long = "builder.max_gas_per_txn")]
    pub max_gas_per_txn: Option<u64>,

    /// Signals whether to log pool transaction events
    #[arg(long = "builder.log-pool-transactions", default_value = "false")]
    pub log_pool_transactions: bool,

    /// How much time extra to wait for the block building job to complete and not get garbage collected
    #[arg(long = "builder.extra-block-deadline-secs", default_value = "20")]
    pub extra_block_deadline_secs: u64,
    /// Whether to enable revert protection by default
    #[arg(long = "builder.enable-revert-protection", default_value = "false")]
    pub enable_revert_protection: bool,

    /// Path to builder playgorund to automatically start up the node connected to it
    #[arg(
        long = "builder.playground",
        num_args = 0..=1,
        default_missing_value = "$HOME/.playground/devnet/",
        value_parser = expand_path,
        env = "PLAYGROUND_DIR",
    )]
    pub playground: Option<PathBuf>,
    #[command(flatten)]
    pub flashblocks: FlashblocksArgs,
    #[command(flatten)]
    pub telemetry: TelemetryArgs,
    #[command(flatten)]
    pub flashtestations: FlashtestationsArgs,
    #[command(flatten)]
    pub gas_limiter: GasLimiterArgs,
}

impl Default for OpRbuilderArgs {
    fn default() -> Self {
        let args = crate::args::Cli::parse_from(["dummy", "node"]);
        let Commands::Node(node_command) = args.command else {
            unreachable!()
        };
        node_command.ext
    }
}

fn expand_path(s: &str) -> Result<PathBuf> {
    shellexpand::full(s)
        .map_err(|e| anyhow!("expansion error for `{s}`: {e}"))?
        .into_owned()
        .parse()
        .map_err(|e| anyhow!("invalid path after expansion: {e}"))
}

/// Parameters for Flashblocks configuration
/// The names in the struct are prefixed with `flashblocks` to avoid conflicts
/// with the standard block building configuration since these args are flattened
/// into the main `OpRbuilderArgs` struct with the other rollup/node args.
#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
pub struct FlashblocksArgs {
    /// When set to true, the builder will build flashblocks
    /// and will build standard blocks at the chain block time.
    ///
    /// The default value will change in the future once the flashblocks
    /// feature is stable.
    #[arg(
        long = "flashblocks.enabled",
        default_value = "false",
        env = "ENABLE_FLASHBLOCKS"
    )]
    pub enabled: bool,

    /// The port that we bind to for the websocket server that provides flashblocks
    #[arg(
        long = "flashblocks.port",
        env = "FLASHBLOCKS_WS_PORT",
        default_value = "1111"
    )]
    pub flashblocks_port: u16,

    /// The address that we bind to for the websocket server that provides flashblocks
    #[arg(
        long = "flashblocks.addr",
        env = "FLASHBLOCKS_WS_ADDR",
        default_value = "127.0.0.1"
    )]
    pub flashblocks_addr: String,

    /// flashblock block time in milliseconds
    #[arg(
        long = "flashblocks.block-time",
        default_value = "250",
        env = "FLASHBLOCK_BLOCK_TIME"
    )]
    pub flashblocks_block_time: u64,

    /// Builder would always thry to produce fixed number of flashblocks without regard to time of
    /// FCU arrival.
    /// In cases of late FCU it could lead to partially filled blocks.
    #[arg(
        long = "flashblocks.fixed",
        default_value = "false",
        env = "FLASHBLOCK_FIXED"
    )]
    pub flashblocks_fixed: bool,

    /// Time by which blocks would be completed earlier in milliseconds.
    ///
    /// This time used to account for latencies, this time would be deducted from total block
    /// building time before calculating number of fbs.
    #[arg(
        long = "flashblocks.leeway-time",
        default_value = "75",
        env = "FLASHBLOCK_LEEWAY_TIME"
    )]
    pub flashblocks_leeway_time: u64,

    /// Whether to disable state root calculation for each flashblock
    #[arg(
        long = "flashblocks.disable-state-root",
        default_value = "false",
        env = "FLASHBLOCKS_DISABLE_STATE_ROOT"
    )]
    pub flashblocks_disable_state_root: bool,

    /// Flashblocks number contract address
    ///
    /// This is the address of the contract that will be used to increment the flashblock number.
    /// If set a builder tx will be added to the start of every flashblock instead of the regular builder tx.
    #[arg(
        long = "flashblocks.number-contract-address",
        env = "FLASHBLOCK_NUMBER_CONTRACT_ADDRESS"
    )]
    pub flashblocks_number_contract_address: Option<Address>,

    /// Use permit signatures if flashtestations is enabled with the flashtestation key
    /// to increment the flashblocks number
    #[arg(
        long = "flashblocks.number-contract-use-permit",
        env = "FLASHBLOCK_NUMBER_CONTRACT_USE_PERMIT",
        default_value = "false"
    )]
    pub flashblocks_number_contract_use_permit: bool,

    /// Flashblocks p2p configuration
    #[command(flatten)]
    pub p2p: FlashblocksP2pArgs,
}

impl Default for FlashblocksArgs {
    fn default() -> Self {
        let args = crate::args::Cli::parse_from(["dummy", "node"]);
        let Commands::Node(node_command) = args.command else {
            unreachable!()
        };
        node_command.ext.flashblocks
    }
}

#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
pub struct FlashblocksP2pArgs {
    /// Enable libp2p networking for flashblock propagation
    #[arg(
        long = "flashblocks.p2p_enabled",
        env = "FLASHBLOCK_P2P_ENABLED",
        default_value = "false"
    )]
    pub p2p_enabled: bool,

    /// Port for the flashblocks p2p node
    #[arg(
        long = "flashblocks.p2p_port",
        env = "FLASHBLOCK_P2P_PORT",
        default_value = "9009"
    )]
    pub p2p_port: u16,

    /// Path to the file containing a hex-encoded libp2p private key.
    /// If the file does not exist, a new key will be generated.
    #[arg(
        long = "flashblocks.p2p_private_key_file",
        env = "FLASHBLOCK_P2P_PRIVATE_KEY_FILE"
    )]
    pub p2p_private_key_file: Option<String>,

    /// Comma-separated list of multiaddrs of known Flashblocks peers
    /// Example: "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ,/ip4/104.131.131.82/udp/4001/quic-v1/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ"
    #[arg(
        long = "flashblocks.p2p_known_peers",
        env = "FLASHBLOCK_P2P_KNOWN_PEERS"
    )]
    pub p2p_known_peers: Option<String>,

    /// Maximum number of peers for the flashblocks p2p node
    #[arg(
        long = "flashblocks.p2p_max_peer_count",
        env = "FLASHBLOCK_P2P_MAX_PEER_COUNT",
        default_value = "50"
    )]
    pub p2p_max_peer_count: u32,
}

/// Parameters for telemetry configuration
#[derive(Debug, Clone, Default, PartialEq, Eq, clap::Args)]
pub struct TelemetryArgs {
    /// OpenTelemetry endpoint for traces
    #[arg(long = "telemetry.otlp-endpoint", env = "OTEL_EXPORTER_OTLP_ENDPOINT")]
    pub otlp_endpoint: Option<String>,

    /// OpenTelemetry headers for authentication
    #[arg(long = "telemetry.otlp-headers", env = "OTEL_EXPORTER_OTLP_HEADERS")]
    pub otlp_headers: Option<String>,

    /// Inverted sampling frequency in blocks. 1 - each block, 100 - every 100th block.
    #[arg(
        long = "telemetry.sampling-ratio",
        env = "SAMPLING_RATIO",
        default_value = "100"
    )]
    pub sampling_ratio: u64,
}
