use backoff::{ExponentialBackoff, ExponentialBackoffBuilder};
use clap::Parser;
use std::time::Duration;
use url::Url;

#[derive(Parser, Clone, Debug)]
pub struct FlashblocksArgs {
    /// Enable Flashblocks client
    #[arg(long, env, default_value = "false")]
    pub flashblocks: bool,

    /// Flashblocks Builder WebSocket URL
    #[arg(long, env, default_value = "ws://127.0.0.1:1111")]
    pub flashblocks_builder_url: Url,

    /// Flashblocks WebSocket host for outbound connections
    #[arg(long, env, default_value = "127.0.0.1")]
    pub flashblocks_host: String,

    /// Flashblocks WebSocket port for outbound connections
    #[arg(long, env, default_value = "1112")]
    pub flashblocks_port: u16,

    /// Websocket connection configuration
    #[command(flatten)]
    pub flashblocks_ws_config: FlashblocksWebsocketConfig,
}

#[derive(Parser, Debug, Clone, Copy)]
pub struct FlashblocksWebsocketConfig {
    /// Minimum time for exponential backoff for timeout if builder disconnected
    #[arg(long, env, default_value = "10")]
    pub flashblock_builder_ws_initial_reconnect_ms: u64,

    /// Maximum time for exponential backoff for timeout if builder disconnected
    #[arg(long, env, default_value = "5000")]
    pub flashblock_builder_ws_max_reconnect_ms: u64,

    /// Timeout for connection attempt
    #[arg(long, env, default_value = "5000")]
    pub flashblock_builder_ws_connect_timeout_ms: u64,

    /// Interval in milliseconds between ping messages sent to upstream servers to detect unresponsive connections
    #[arg(long, env, default_value = "500")]
    pub flashblock_builder_ws_ping_interval_ms: u64,

    /// Timeout in milliseconds to wait for pong responses from upstream servers before considering the connection dead
    #[arg(long, env, default_value = "1500")]
    pub flashblock_builder_ws_pong_timeout_ms: u64,
}

impl FlashblocksWebsocketConfig {
    /// Creates `ExponentialBackoff` use to control builder websocket reconnection time
    pub fn backoff(&self) -> ExponentialBackoff {
        ExponentialBackoffBuilder::default()
            .with_initial_interval(self.initial_interval())
            .with_max_interval(self.max_interval())
            .with_randomization_factor(0 as f64)
            .with_max_elapsed_time(None)
            .with_multiplier(2.0)
            .build()
    }

    /// Returns initial time for exponential backoff
    pub fn initial_interval(&self) -> Duration {
        Duration::from_millis(self.flashblock_builder_ws_initial_reconnect_ms)
    }

    /// Returns maximal time for exponential backoff
    pub fn max_interval(&self) -> Duration {
        Duration::from_millis(self.flashblock_builder_ws_max_reconnect_ms)
    }

    /// Returns ping interval
    pub fn ping_interval(&self) -> Duration {
        Duration::from_millis(self.flashblock_builder_ws_ping_interval_ms)
    }

    /// Returns pong interval
    pub fn pong_interval(&self) -> Duration {
        Duration::from_millis(self.flashblock_builder_ws_pong_timeout_ms)
    }
}
