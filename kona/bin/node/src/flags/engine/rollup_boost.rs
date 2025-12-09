use crate::flags::engine::flashblocks::FlashblocksFlags;
use rollup_boost::{BlockSelectionPolicy, ExecutionMode};

/// Custom block builder flags.
#[derive(Clone, Debug, clap::Args)]
pub struct RollupBoostFlags {
    /// Execution mode to start rollup boost with
    #[arg(
        long,
        visible_alias = "rollup-boost.execution-mode",
        env = "KONA_NODE_ROLLUP_BOOST_EXECUTION_MODE",
        default_value = "disabled"
    )]
    pub execution_mode: ExecutionMode,

    /// Block selection policy to use for the rollup boost server.
    #[arg(
        long,
        visible_alias = "rollup-boost.block-selection-policy",
        env = "KONA_NODE_ROLLUP_BOOST_BLOCK_SELECTION_POLICY"
    )]
    pub block_selection_policy: Option<BlockSelectionPolicy>,

    /// Should we use the l2 client for computing state root
    #[arg(
        long,
        visible_alias = "rollup-boost.external-state-root",
        env = "KONA_NODE_ROLLUP_BOOST_EXTERNAL_STATE_ROOT",
        default_value = "false"
    )]
    pub external_state_root: bool,

    /// Allow all engine API calls to builder even when marked as unhealthy
    /// This is default true assuming no builder CL set up
    #[arg(
        long,
        visible_alias = "rollup-boost.ignore-unhealthy-builders",
        env = "KONA_NODE_ROLLUP_BOOST_IGNORE_UNHEALTHY_BUILDERS",
        default_value = "false"
    )]
    pub ignore_unhealthy_builders: bool,

    /// Duration in seconds between async health checks on the builder
    #[arg(
        long,
        visible_alias = "rollup-boost.health-check-interval",
        env = "KONA_NODE_ROLLUP_BOOST_HEALTH_CHECK_INTERVAL",
        default_value = "60"
    )]
    pub health_check_interval: u64,

    /// Max duration in seconds between the unsafe head block of the builder and the current time
    #[arg(
        long,
        visible_alias = "rollup-boost.max-unsafe-interval",
        env = "KONA_NODE_ROLLUP_BOOST_MAX_UNSAFE_INTERVAL",
        default_value = "10"
    )]
    pub max_unsafe_interval: u64,

    /// Flashblocks configuration
    #[clap(flatten)]
    pub flashblocks: FlashblocksFlags,
}

impl Default for RollupBoostFlags {
    fn default() -> Self {
        Self {
            execution_mode: ExecutionMode::Disabled,
            block_selection_policy: None,
            external_state_root: false,
            ignore_unhealthy_builders: false,
            flashblocks: FlashblocksFlags::default(),
            health_check_interval: 60,
            max_unsafe_interval: 10,
        }
    }
}

impl RollupBoostFlags {
    /// Converts the rollup boost cli arguments to the rollup boost arguments used by the engine.
    pub fn as_rollup_boost_args(self) -> kona_engine::RollupBoostServerArgs {
        kona_engine::RollupBoostServerArgs {
            initial_execution_mode: self.execution_mode,
            block_selection_policy: self.block_selection_policy,
            external_state_root: self.external_state_root,
            ignore_unhealthy_builders: self.ignore_unhealthy_builders,
            flashblocks: self.flashblocks.flashblocks.then_some(
                kona_engine::FlashblocksClientArgs {
                    flashblocks_builder_url: self.flashblocks.flashblocks_builder_url,
                    flashblocks_host: self.flashblocks.flashblocks_host,
                    flashblocks_port: self.flashblocks.flashblocks_port,
                    flashblocks_ws_config: kona_engine::FlashblocksWebsocketConfig {
                        flashblock_builder_ws_initial_reconnect_ms: self
                            .flashblocks
                            .flashblocks_ws_config
                            .flashblock_builder_ws_initial_reconnect_ms,
                        flashblock_builder_ws_max_reconnect_ms: self
                            .flashblocks
                            .flashblocks_ws_config
                            .flashblock_builder_ws_max_reconnect_ms,
                        flashblock_builder_ws_ping_interval_ms: self
                            .flashblocks
                            .flashblocks_ws_config
                            .flashblock_builder_ws_ping_interval_ms,
                        flashblock_builder_ws_pong_timeout_ms: self
                            .flashblocks
                            .flashblocks_ws_config
                            .flashblock_builder_ws_pong_timeout_ms,
                    },
                },
            ),
        }
    }
}
