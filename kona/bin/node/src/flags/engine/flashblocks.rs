use clap::Parser;
use reqwest::Url;

const DEFAULT_FLASHBLOCKS_BUILDER_URL: &str = "ws://localhost:1111";
const DEFAULT_FLASHBLOCKS_HOST: &str = "localhost";
const DEFAULT_FLASHBLOCKS_PORT: u16 = 1112;

const DEFAULT_FLASHBLOCKS_BUILDER_WS_INITIAL_RECONNECT_MS: u64 = 10;
const DEFAULT_FLASHBLOCKS_BUILDER_WS_MAX_RECONNECT_MS: u64 = 5000;
const DEFAULT_FLASHBLOCKS_BUILDER_WS_PING_INTERVAL_MS: u64 = 500;
const DEFAULT_FLASHBLOCKS_BUILDER_WS_PONG_TIMEOUT_MS: u64 = 1500;

/// Flashblocks flags.
#[derive(Clone, Debug, clap::Args)]
pub struct FlashblocksFlags {
    /// Enable Flashblocks client
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks",
        env = "KONA_NODE_FLASHBLOCKS",
        default_value = "false"
    )]
    pub flashblocks: bool,

    /// Flashblocks Builder WebSocket URL
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-builder-url",
        env = "KONA_NODE_FLASHBLOCKS_BUILDER_URL",
        default_value = DEFAULT_FLASHBLOCKS_BUILDER_URL
    )]
    pub flashblocks_builder_url: Url,

    /// Flashblocks WebSocket host for outbound connections
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-host",
        env = "KONA_NODE_FLASHBLOCKS_HOST",
        default_value = DEFAULT_FLASHBLOCKS_HOST
    )]
    pub flashblocks_host: String,

    /// Flashblocks WebSocket port for outbound connections
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-port",
        env = "KONA_NODE_FLASHBLOCKS_PORT",
        default_value_t = DEFAULT_FLASHBLOCKS_PORT
    )]
    pub flashblocks_port: u16,

    /// Websocket connection configuration
    #[command(flatten)]
    pub flashblocks_ws_config: FlashblocksWebsocketFlags,
}

impl Default for FlashblocksFlags {
    fn default() -> Self {
        Self {
            flashblocks: false,
            flashblocks_builder_url: Url::parse(DEFAULT_FLASHBLOCKS_BUILDER_URL).unwrap(),
            flashblocks_host: DEFAULT_FLASHBLOCKS_HOST.to_string(),
            flashblocks_port: DEFAULT_FLASHBLOCKS_PORT,
            flashblocks_ws_config: FlashblocksWebsocketFlags::default(),
        }
    }
}

/// Configuration for the Flashblocks WebSocket connection.
#[derive(Parser, Debug, Clone, Copy)]
pub struct FlashblocksWebsocketFlags {
    /// Minimum time for exponential backoff for timeout if builder disconnected
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-initial-reconnect-ms",
        env = "KONA_NODE_FLASHBLOCKS_BUILDER_WS_INITIAL_RECONNECT_MS",
        default_value_t = DEFAULT_FLASHBLOCKS_BUILDER_WS_INITIAL_RECONNECT_MS
    )]
    pub flashblock_builder_ws_initial_reconnect_ms: u64,

    /// Maximum time for exponential backoff for timeout if builder disconnected
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-max-reconnect-ms",
        env = "KONA_NODE_FLASHBLOCKS_BUILDER_WS_MAX_RECONNECT_MS",
        default_value_t = DEFAULT_FLASHBLOCKS_BUILDER_WS_MAX_RECONNECT_MS
    )]
    pub flashblock_builder_ws_max_reconnect_ms: u64,

    /// Interval in milliseconds between ping messages sent to upstream servers to detect
    /// unresponsive connections
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-ping-interval-ms",
        env = "KONA_NODE_FLASHBLOCKS_BUILDER_WS_PING_INTERVAL_MS",
        default_value_t = DEFAULT_FLASHBLOCKS_BUILDER_WS_PING_INTERVAL_MS
    )]
    pub flashblock_builder_ws_ping_interval_ms: u64,

    /// Timeout in milliseconds to wait for pong responses from upstream servers before considering
    /// the connection dead
    #[arg(
        long,
        visible_alias = "rollup-boost.flashblocks-pong-timeout-ms",
        env = "KONA_NODE_FLASHBLOCKS_BUILDER_WS_PONG_TIMEOUT_MS",
        default_value_t = DEFAULT_FLASHBLOCKS_BUILDER_WS_PONG_TIMEOUT_MS
    )]
    pub flashblock_builder_ws_pong_timeout_ms: u64,
}

impl FlashblocksFlags {
    /// Converts the flashblocks cli arguments to the flashblocks arguments used by the rollup-boost
    /// server.
    pub fn as_rollup_boost_args(self) -> rollup_boost::FlashblocksArgs {
        rollup_boost::FlashblocksArgs {
            flashblocks: self.flashblocks,
            flashblocks_builder_url: self.flashblocks_builder_url,
            flashblocks_host: self.flashblocks_host,
            flashblocks_port: self.flashblocks_port,
            flashblocks_ws_config: rollup_boost::FlashblocksWebsocketConfig {
                flashblock_builder_ws_initial_reconnect_ms: self
                    .flashblocks_ws_config
                    .flashblock_builder_ws_initial_reconnect_ms,
                flashblock_builder_ws_max_reconnect_ms: self
                    .flashblocks_ws_config
                    .flashblock_builder_ws_max_reconnect_ms,
                flashblock_builder_ws_ping_interval_ms: self
                    .flashblocks_ws_config
                    .flashblock_builder_ws_ping_interval_ms,
                flashblock_builder_ws_pong_timeout_ms: self
                    .flashblocks_ws_config
                    .flashblock_builder_ws_pong_timeout_ms,
            },
        }
    }
}

impl Default for FlashblocksWebsocketFlags {
    fn default() -> Self {
        Self {
            flashblock_builder_ws_initial_reconnect_ms: 10,
            flashblock_builder_ws_max_reconnect_ms: 5000,
            flashblock_builder_ws_ping_interval_ms: 500,
            flashblock_builder_ws_pong_timeout_ms: 1500,
        }
    }
}
