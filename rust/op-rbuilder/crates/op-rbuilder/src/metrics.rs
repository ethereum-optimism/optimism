use alloy_primitives::{Address, hex};
use metrics::IntoF64;
use reth_metrics::{
    Metrics,
    metrics::{Counter, Gauge, Histogram, gauge},
};

use crate::{
    args::OpRbuilderArgs,
    flashtestations::attestation::{compute_workload_id_from_parsed, parse_report_body},
};

/// The latest version from Cargo.toml.
pub const CARGO_PKG_VERSION: &str = env!("CARGO_PKG_VERSION");

/// The 8 character short SHA of the latest commit.
pub const VERGEN_GIT_SHA: &str = env!("VERGEN_GIT_SHA_SHORT");

/// The build timestamp.
pub const VERGEN_BUILD_TIMESTAMP: &str = env!("VERGEN_BUILD_TIMESTAMP");

/// The target triple.
pub const VERGEN_CARGO_TARGET_TRIPLE: &str = env!("VERGEN_CARGO_TARGET_TRIPLE");

/// The build features.
pub const VERGEN_CARGO_FEATURES: &str = env!("VERGEN_CARGO_FEATURES");

/// The latest commit message and author name and email.
pub const VERGEN_GIT_AUTHOR: &str = env!("VERGEN_GIT_COMMIT_AUTHOR");
pub const VERGEN_GIT_COMMIT_MESSAGE: &str = env!("VERGEN_GIT_COMMIT_MESSAGE");

/// The build profile name.
pub const BUILD_PROFILE_NAME: &str = env!("OP_RBUILDER_BUILD_PROFILE");

/// The short version information for op-rbuilder.
pub const SHORT_VERSION: &str = env!("OP_RBUILDER_SHORT_VERSION");

/// The long version information for op-rbuilder.
pub const LONG_VERSION: &str = concat!(
    env!("OP_RBUILDER_LONG_VERSION_0"),
    "\n",
    env!("OP_RBUILDER_LONG_VERSION_1"),
    "\n",
    env!("OP_RBUILDER_LONG_VERSION_2"),
    "\n",
    env!("OP_RBUILDER_LONG_VERSION_3"),
    "\n",
    env!("OP_RBUILDER_LONG_VERSION_4"),
    "\n",
    env!("OP_RBUILDER_LONG_VERSION_5"),
);

pub const VERSION: VersionInfo = VersionInfo {
    version: CARGO_PKG_VERSION,
    build_timestamp: VERGEN_BUILD_TIMESTAMP,
    cargo_features: VERGEN_CARGO_FEATURES,
    git_sha: VERGEN_GIT_SHA,
    target_triple: VERGEN_CARGO_TARGET_TRIPLE,
    build_profile: BUILD_PROFILE_NAME,
    commit_author: VERGEN_GIT_AUTHOR,
    commit_message: VERGEN_GIT_COMMIT_MESSAGE,
};

/// op-rbuilder metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "op_rbuilder")]
pub struct OpRBuilderMetrics {
    /// Block built success
    pub block_built_success: Counter,
    /// Block synced success
    pub block_synced_success: Counter,
    /// Number of flashblocks added to block (Total per block)
    pub flashblock_count: Histogram,
    /// Number of messages sent
    pub messages_sent_count: Counter,
    /// Histogram of the time taken to build a block
    pub total_block_built_duration: Histogram,
    /// Latest time taken to build a block
    pub total_block_built_gauge: Gauge,
    /// Histogram of the time taken to build a Flashblock
    pub flashblock_build_duration: Histogram,
    /// Histogram of the time taken to sync a Flashblock
    pub flashblock_sync_duration: Histogram,
    /// Flashblock UTF8 payload byte size histogram
    pub flashblock_byte_size_histogram: Histogram,
    /// Histogram of transactions in a Flashblock
    pub flashblock_num_tx_histogram: Histogram,
    /// Number of invalid blocks
    pub invalid_built_blocks_count: Counter,
    /// Number of invalid synced blocks
    pub invalid_synced_blocks_count: Counter,
    /// Histogram of fetching transactions from the pool duration
    pub transaction_pool_fetch_duration: Histogram,
    /// Latest time taken to fetch tx from the pool
    pub transaction_pool_fetch_gauge: Gauge,
    /// Histogram of state root calculation duration
    pub state_root_calculation_duration: Histogram,
    /// Latest state root calculation duration
    pub state_root_calculation_gauge: Gauge,
    /// Histogram of sequencer transaction execution duration
    pub sequencer_tx_duration: Histogram,
    /// Latest sequencer transaction execution duration
    pub sequencer_tx_gauge: Gauge,
    /// Histogram of state merge transitions duration
    pub state_transition_merge_duration: Histogram,
    /// Latest state merge transitions duration
    pub state_transition_merge_gauge: Gauge,
    /// Histogram of the duration of payload simulation of all transactions
    pub payload_transaction_simulation_duration: Histogram,
    /// Latest payload simulation of all transactions duration
    pub payload_transaction_simulation_gauge: Gauge,
    /// Number of transaction considered for inclusion in the block
    pub payload_num_tx_considered: Histogram,
    /// Latest number of transactions considered for inclusion in the block
    pub payload_num_tx_considered_gauge: Gauge,
    /// Payload byte size histogram
    pub payload_byte_size: Histogram,
    /// Latest Payload byte size
    pub payload_byte_size_gauge: Gauge,
    /// Histogram of transactions in the payload
    pub payload_num_tx: Histogram,
    /// Latest number of transactions in the payload
    pub payload_num_tx_gauge: Gauge,
    /// Histogram of transactions in the payload that were successfully simulated
    pub payload_num_tx_simulated: Histogram,
    /// Latest number of transactions in the payload that were successfully simulated
    pub payload_num_tx_simulated_gauge: Gauge,
    /// Histogram of transactions in the payload that were successfully simulated
    pub payload_num_tx_simulated_success: Histogram,
    /// Latest number of transactions in the payload that were successfully simulated
    pub payload_num_tx_simulated_success_gauge: Gauge,
    /// Histogram of transactions in the payload that failed simulation
    pub payload_num_tx_simulated_fail: Histogram,
    /// Latest number of transactions in the payload that failed simulation
    pub payload_num_tx_simulated_fail_gauge: Gauge,
    /// Histogram of gas used by successful transactions
    pub successful_tx_gas_used: Histogram,
    /// Histogram of gas used by reverted transactions
    pub reverted_tx_gas_used: Histogram,
    /// Gas used by reverted transactions in the latest block
    pub payload_reverted_tx_gas_used: Gauge,
    /// Histogram of tx simulation duration
    pub tx_simulation_duration: Histogram,
    /// Byte size of transactions
    pub tx_byte_size: Histogram,
    /// How much less flashblocks we issue to be on time with block construction
    pub reduced_flashblocks_number: Histogram,
    /// How much less flashblocks we issued in reality, comparing to calculated number for block
    pub missing_flashblocks_count: Histogram,
    /// How much time we have deducted from block building time
    pub flashblocks_time_drift: Histogram,
    /// Time offset we used for first flashblock
    pub first_flashblock_time_offset: Histogram,
    /// Number of requests sent to the eth_sendBundle endpoint
    pub bundle_requests: Counter,
    /// Number of valid bundles received at the eth_sendBundle endpoint
    pub valid_bundles: Counter,
    /// Number of bundles that failed to execute
    pub failed_bundles: Counter,
    /// Number of reverted bundles
    pub bundles_reverted: Histogram,
    /// Histogram of eth_sendBundle request duration
    pub bundle_receive_duration: Histogram,
}

impl OpRBuilderMetrics {
    #[expect(clippy::too_many_arguments)]
    pub fn set_payload_builder_metrics(
        &self,
        payload_transaction_simulation_time: impl IntoF64 + Copy,
        num_txs_considered: impl IntoF64 + Copy,
        num_txs_simulated: impl IntoF64 + Copy,
        num_txs_simulated_success: impl IntoF64 + Copy,
        num_txs_simulated_fail: impl IntoF64 + Copy,
        num_bundles_reverted: impl IntoF64,
        reverted_gas_used: impl IntoF64,
    ) {
        self.payload_transaction_simulation_duration
            .record(payload_transaction_simulation_time);
        self.payload_transaction_simulation_gauge
            .set(payload_transaction_simulation_time);
        self.payload_num_tx_considered.record(num_txs_considered);
        self.payload_num_tx_considered_gauge.set(num_txs_considered);
        self.payload_num_tx_simulated.record(num_txs_simulated);
        self.payload_num_tx_simulated_gauge.set(num_txs_simulated);
        self.payload_num_tx_simulated_success
            .record(num_txs_simulated_success);
        self.payload_num_tx_simulated_success_gauge
            .set(num_txs_simulated_success);
        self.payload_num_tx_simulated_fail
            .record(num_txs_simulated_fail);
        self.payload_num_tx_simulated_fail_gauge
            .set(num_txs_simulated_fail);
        self.bundles_reverted.record(num_bundles_reverted);
        self.payload_reverted_tx_gas_used.set(reverted_gas_used);
    }
}

/// Set gauge metrics for some flags so we can inspect which ones are set
/// and which ones aren't.
pub fn record_flag_gauge_metrics(builder_args: &OpRbuilderArgs) {
    gauge!("op_rbuilder_flags_flashblocks_enabled").set(builder_args.flashblocks.enabled as i32);
    gauge!("op_rbuilder_flags_flashtestations_enabled")
        .set(builder_args.flashtestations.flashtestations_enabled as i32);
    gauge!("op_rbuilder_flags_enable_revert_protection")
        .set(builder_args.enable_revert_protection as i32);
}

/// Record TEE workload ID and measurement metrics
/// Parses the quote, computes workload ID, and records workload_id, mr_td (TEE measurement), and rt_mr0 (runtime measurement register 0)
/// These identify the trusted execution environment configuration provided by GCP
pub fn record_tee_metrics(raw_quote: &[u8], tee_address: &Address) -> eyre::Result<()> {
    let parsed_quote = parse_report_body(raw_quote)?;
    let workload_id = compute_workload_id_from_parsed(&parsed_quote);

    let workload_id_hex = hex::encode(workload_id);
    let mr_td_hex = hex::encode(parsed_quote.mr_td);
    let rt_mr0_hex = hex::encode(parsed_quote.rt_mr0);

    let tee_address_static: &'static str = Box::leak(tee_address.to_string().into_boxed_str());
    let workload_id_static: &'static str = Box::leak(workload_id_hex.into_boxed_str());
    let mr_td_static: &'static str = Box::leak(mr_td_hex.into_boxed_str());
    let rt_mr0_static: &'static str = Box::leak(rt_mr0_hex.into_boxed_str());

    // Record TEE address
    let tee_address_labels: [(&str, &str); 1] = [("tee_address", tee_address_static)];
    gauge!("op_rbuilder_tee_address", &tee_address_labels).set(1);

    // Record workload ID
    let workload_labels: [(&str, &str); 1] = [("workload_id", workload_id_static)];
    gauge!("op_rbuilder_tee_workload_id", &workload_labels).set(1);

    // Record MRTD (TEE measurement)
    let mr_td_labels: [(&str, &str); 1] = [("mr_td", mr_td_static)];
    gauge!("op_rbuilder_tee_mr_td", &mr_td_labels).set(1);

    // Record RTMR0 (runtime measurement register 0)
    let rt_mr0_labels: [(&str, &str); 1] = [("rt_mr0", rt_mr0_static)];
    gauge!("op_rbuilder_tee_rt_mr0", &rt_mr0_labels).set(1);

    Ok(())
}

/// Contains version information for the application.
#[derive(Debug, Clone)]
pub struct VersionInfo {
    /// The version of the application.
    pub version: &'static str,
    /// The build timestamp of the application.
    pub build_timestamp: &'static str,
    /// The cargo features enabled for the build.
    pub cargo_features: &'static str,
    /// The Git SHA of the build.
    pub git_sha: &'static str,
    /// The target triple for the build.
    pub target_triple: &'static str,
    /// The build profile (e.g., debug or release).
    pub build_profile: &'static str,
    /// The author of the latest commit.
    pub commit_author: &'static str,
    /// The message of the latest commit.
    pub commit_message: &'static str,
}

impl VersionInfo {
    /// This exposes op-rbuilder's version information over prometheus.
    pub fn register_version_metrics(&self) {
        let labels: [(&str, &str); 8] = [
            ("version", self.version),
            ("build_timestamp", self.build_timestamp),
            ("cargo_features", self.cargo_features),
            ("git_sha", self.git_sha),
            ("target_triple", self.target_triple),
            ("build_profile", self.build_profile),
            ("commit_author", self.commit_author),
            ("commit_message", self.commit_message),
        ];

        let gauge = gauge!("builder_info", &labels);
        gauge.set(1);
    }
}
