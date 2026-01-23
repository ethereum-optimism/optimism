//! Additional Node command arguments.

//! clap [Args](clap::Args) for optimism rollup configuration

use clap::builder::ArgPredicate;
use op_alloy_consensus::interop::SafetyLevel;
use reth_optimism_txpool::supervisor::DEFAULT_SUPERVISOR_URL;
use std::{path::PathBuf, time::Duration};
use url::Url;

/// Parameters for rollup configuration
#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
#[command(next_help_heading = "Rollup")]
pub struct RollupArgs {
    /// Endpoint for the sequencer mempool (can be both HTTP and WS)
    #[arg(long = "rollup.sequencer", visible_aliases = ["rollup.sequencer-http", "rollup.sequencer-ws"])]
    pub sequencer: Option<String>,

    /// Disable transaction pool gossip
    #[arg(long = "rollup.disable-tx-pool-gossip")]
    pub disable_txpool_gossip: bool,

    /// By default the pending block equals the latest block
    /// to save resources and not leak txs from the tx-pool,
    /// this flag enables computing of the pending block
    /// from the tx-pool instead.
    ///
    /// If `compute_pending_block` is not enabled, the payload builder
    /// will use the payload attributes from the latest block. Note
    /// that this flag is not yet functional.
    #[arg(long = "rollup.compute-pending-block")]
    pub compute_pending_block: bool,

    /// enables discovery v4 if provided
    #[arg(long = "rollup.discovery.v4", default_value = "false")]
    pub discovery_v4: bool,

    /// Enable transaction conditional support on sequencer
    #[arg(long = "rollup.enable-tx-conditional", default_value = "false")]
    pub enable_tx_conditional: bool,

    /// HTTP endpoint for the supervisor
    #[arg(
        long = "rollup.supervisor-http",
        value_name = "SUPERVISOR_HTTP_URL",
        default_value = DEFAULT_SUPERVISOR_URL
    )]
    pub supervisor_http: String,

    /// Safety level for the supervisor
    #[arg(
        long = "rollup.supervisor-safety-level",
        default_value_t = SafetyLevel::CrossUnsafe,
    )]
    pub supervisor_safety_level: SafetyLevel,

    /// Optional headers to use when connecting to the sequencer.
    #[arg(long = "rollup.sequencer-headers", requires = "sequencer")]
    pub sequencer_headers: Vec<String>,

    /// RPC endpoint for historical data.
    #[arg(
        long = "rollup.historicalrpc",
        alias = "rollup.historical-rpc",
        value_name = "HISTORICAL_HTTP_URL"
    )]
    pub historical_rpc: Option<String>,

    /// Minimum suggested priority fee (tip) in wei, default `1_000_000`
    #[arg(long, default_value_t = 1_000_000)]
    pub min_suggested_priority_fee: u64,

    /// A URL pointing to a secure websocket subscription that streams out flashblocks.
    ///
    /// If given, the flashblocks are received to build pending block. All request with "pending"
    /// block tag will use the pending state based on flashblocks.
    #[arg(long, alias = "websocket-url")]
    pub flashblocks_url: Option<Url>,

    /// Enable flashblock consensus client to drive the chain forward
    ///
    /// When enabled, the flashblock consensus client will process flashblock sequences and submit
    /// them to the engine API to advance the chain.
    /// Requires `flashblocks_url` to be set.
    #[arg(long, default_value_t = false, requires = "flashblocks_url")]
    pub flashblock_consensus: bool,

    /// If true, initialize external-proofs exex to save and serve trie nodes to provide proofs
    /// faster.
    #[arg(
        long = "proofs-history",
        value_name = "PROOFS_HISTORY",
        default_value_ifs([
            ("proofs-history.storage-path", ArgPredicate::IsPresent, "true")
        ])
    )]
    pub proofs_history: bool,

    /// The path to the storage DB for proofs history.
    #[arg(long = "proofs-history.storage-path", value_name = "PROOFS_HISTORY_STORAGE_PATH")]
    pub proofs_history_storage_path: Option<PathBuf>,

    /// The window to span blocks for proofs history. Value is the number of blocks.
    /// Default is 1 month of blocks based on 2 seconds block time.
    /// 30 * 24 * 60 * 60 / 2 = `1_296_000`
    #[arg(
        long = "proofs-history.window",
        default_value_t = 1_296_000,
        value_name = "PROOFS_HISTORY_WINDOW"
    )]
    pub proofs_history_window: u64,

    /// Interval between proof-storage prune runs. Accepts human-friendly durations
    /// like "100s", "5m", "1h". Defaults to 15s.
    ///
    /// - Shorter intervals prune smaller batches more often, so each prune run tends to be faster
    ///   and the blocking pause for writes is shorter, at the cost of more frequent pauses.
    /// - Longer intervals prune larger batches less often, which reduces how often pruning runs,
    ///   but each run can take longer and block writes for longer.
    ///
    /// A shorter interval is preferred so that prune
    /// runs stay small and don’t stall writes for too long.
    ///
    /// CLI: `--proofs-history.prune-interval 10m`
    #[arg(
        long = "proofs-history.prune-interval",
        value_name = "PROOFS_HISTORY_PRUNE_INTERVAL",
        default_value = "15s",
        value_parser = humantime::parse_duration
    )]
    pub proofs_history_prune_interval: Duration,
    /// Verification interval: perform full block execution every N blocks for data integrity.
    /// - 0: Disabled (Default) (always use fast path with pre-computed data from notifications)
    /// - 1: Always verify (always execute blocks, slowest)
    /// - N: Verify every Nth block (e.g., 100 = every 100 blocks)
    ///
    /// Periodic verification helps catch data corruption or consensus bugs while maintaining
    /// good performance.
    ///
    /// CLI: `--proofs-history.verification-interval 100`
    #[arg(
        long = "proofs-history.verification-interval",
        value_name = "PROOFS_HISTORY_VERIFICATION_INTERVAL",
        default_value_t = 0
    )]
    pub proofs_history_verification_interval: u64,
}

impl Default for RollupArgs {
    fn default() -> Self {
        Self {
            sequencer: None,
            disable_txpool_gossip: false,
            compute_pending_block: false,
            discovery_v4: false,
            enable_tx_conditional: false,
            supervisor_http: DEFAULT_SUPERVISOR_URL.to_string(),
            supervisor_safety_level: SafetyLevel::CrossUnsafe,
            sequencer_headers: Vec::new(),
            historical_rpc: None,
            min_suggested_priority_fee: 1_000_000,
            flashblocks_url: None,
            flashblock_consensus: false,
            proofs_history: false,
            proofs_history_storage_path: None,
            proofs_history_window: 1_296_000,
            proofs_history_prune_interval: Duration::from_secs(15),
            proofs_history_verification_interval: 0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use clap::{Args, Parser};

    /// A helper type to parse Args more easily
    #[derive(Parser)]
    struct CommandParser<T: Args> {
        #[command(flatten)]
        args: T,
    }

    #[test]
    fn test_parse_optimism_default_args() {
        let default_args = RollupArgs::default();
        let args = CommandParser::<RollupArgs>::parse_from(["reth"]).args;
        assert_eq!(args, default_args);
    }

    #[test]
    fn test_parse_optimism_compute_pending_block_args() {
        let expected_args = RollupArgs { compute_pending_block: true, ..Default::default() };
        let args =
            CommandParser::<RollupArgs>::parse_from(["reth", "--rollup.compute-pending-block"])
                .args;
        assert_eq!(args, expected_args);
    }

    #[test]
    fn test_parse_optimism_discovery_v4_args() {
        let expected_args = RollupArgs { discovery_v4: true, ..Default::default() };
        let args = CommandParser::<RollupArgs>::parse_from(["reth", "--rollup.discovery.v4"]).args;
        assert_eq!(args, expected_args);
    }

    #[test]
    fn test_parse_optimism_sequencer_http_args() {
        let expected_args =
            RollupArgs { sequencer: Some("http://host:port".into()), ..Default::default() };
        let args = CommandParser::<RollupArgs>::parse_from([
            "reth",
            "--rollup.sequencer-http",
            "http://host:port",
        ])
        .args;
        assert_eq!(args, expected_args);
    }

    #[test]
    fn test_parse_optimism_disable_txpool_args() {
        let expected_args = RollupArgs { disable_txpool_gossip: true, ..Default::default() };
        let args =
            CommandParser::<RollupArgs>::parse_from(["reth", "--rollup.disable-tx-pool-gossip"])
                .args;
        assert_eq!(args, expected_args);
    }

    #[test]
    fn test_parse_optimism_enable_tx_conditional() {
        let expected_args = RollupArgs { enable_tx_conditional: true, ..Default::default() };
        let args =
            CommandParser::<RollupArgs>::parse_from(["reth", "--rollup.enable-tx-conditional"])
                .args;
        assert_eq!(args, expected_args);
    }

    #[test]
    fn test_parse_optimism_many_args() {
        let expected_args = RollupArgs {
            disable_txpool_gossip: true,
            compute_pending_block: true,
            enable_tx_conditional: true,
            sequencer: Some("http://host:port".into()),
            ..Default::default()
        };
        let args = CommandParser::<RollupArgs>::parse_from([
            "reth",
            "--rollup.disable-tx-pool-gossip",
            "--rollup.compute-pending-block",
            "--rollup.enable-tx-conditional",
            "--rollup.sequencer-http",
            "http://host:port",
        ])
        .args;
        assert_eq!(args, expected_args);
    }
}
