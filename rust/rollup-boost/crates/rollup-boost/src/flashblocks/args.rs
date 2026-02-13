use backoff::{ExponentialBackoff, ExponentialBackoffBuilder};
use clap::{Args, Parser};
use ed25519_dalek::{SigningKey, VerifyingKey};
use std::time::Duration;
use url::Url;

use hex::FromHex;

#[derive(Args, Clone, Debug)]
#[group(requires = "flashblocks_ws")]
pub struct FlashblocksWsArgs {
    /// Enable Flashblocks Websocket client
    #[arg(
        // Keep the flag as "flashblocks" for backward compatibility
        long = "flashblocks",
        id = "flashblocks_ws",
        conflicts_with = "flashblocks_p2p",
        env,
        default_value = "false"
    )]
    pub flashblocks_ws: bool,

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

#[derive(Args, Clone, Debug)]
#[group(requires = "flashblocks_p2p")]
pub struct FlashblocksP2PArgs {
    /// Enable Flashblocks P2P Authorization
    #[arg(
        long,
        id = "flashblocks_p2p",
        conflicts_with = "flashblocks_ws",
        env,
        required = false
    )]
    pub flashblocks_p2p: bool,

    #[arg(
        long = "flashblocks-authorizer-sk",
        env = "FLASHBLOCKS_AUTHORIZER_SK",
        value_parser = parse_sk,
        required = false,
    )]
    pub authorizer_sk: SigningKey,

    #[arg(
        long = "flashblocks-builder-vk",
        env = "FLASHBLOCKS_BUILDER_VK",
        value_parser = parse_vk,
        required = false,
    )]
    pub builder_vk: VerifyingKey,
}

pub fn parse_sk(s: &str) -> eyre::Result<SigningKey> {
    let bytes = <[u8; 32]>::from_hex(s.trim())?;
    Ok(SigningKey::from_bytes(&bytes))
}

pub fn parse_vk(s: &str) -> eyre::Result<VerifyingKey> {
    let bytes = <[u8; 32]>::from_hex(s.trim())?;
    Ok(VerifyingKey::from_bytes(&bytes)?)
}
