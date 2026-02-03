//! Rollup-boost abstraction used by the engine client.

use alloy_json_rpc::{ErrorPayload, RpcError};
use alloy_transport::TransportErrorKind;
use rollup_boost::ExecutionMode;
use std::fmt::Debug;

use rollup_boost::BlockSelectionPolicy;
pub use rollup_boost::RollupBoostServer;
use url::Url;

/// Configuration for the rollup-boost server.
#[derive(Clone, Debug)]
pub struct RollupBoostServerArgs {
    /// The initial execution mode of the rollup-boost server.
    pub initial_execution_mode: ExecutionMode,
    /// The block selection policy of the rollup-boost server.
    pub block_selection_policy: Option<BlockSelectionPolicy>,
    /// Whether to use the l2 client for computing state root.
    pub external_state_root: bool,
    /// Allow all engine API calls to builder even when marked as unhealthy
    /// This is default true assuming no builder CL set up
    pub ignore_unhealthy_builders: bool,
    /// Flashblocks configuration
    pub flashblocks: Option<FlashblocksClientArgs>,
}

/// Configuration for the Flashblocks client.
#[derive(Clone, Debug)]
pub struct FlashblocksClientArgs {
    /// Flashblocks Builder WebSocket URL
    pub flashblocks_builder_url: Url,

    /// Flashblocks WebSocket host for outbound connections
    pub flashblocks_host: String,

    /// Flashblocks WebSocket port for outbound connections
    pub flashblocks_port: u16,

    /// Websocket connection configuration
    pub flashblocks_ws_config: FlashblocksWebsocketConfig,
}

/// Configuration for the Flashblocks WebSocket connection.
#[derive(Debug, Clone, Copy)]
pub struct FlashblocksWebsocketConfig {
    /// Minimum time for exponential backoff for timeout if builder disconnected
    pub flashblock_builder_ws_initial_reconnect_ms: u64,

    /// Maximum time for exponential backoff for timeout if builder disconnected
    pub flashblock_builder_ws_max_reconnect_ms: u64,

    /// Timeout for connection attempt
    pub flashblock_builder_ws_connect_timeout_ms: u64,

    /// Interval in milliseconds between ping messages sent to upstream servers to detect
    /// unresponsive connections
    pub flashblock_builder_ws_ping_interval_ms: u64,

    /// Timeout in milliseconds to wait for pong responses from upstream servers before considering
    /// the connection dead
    pub flashblock_builder_ws_pong_timeout_ms: u64,
}

/// An error that occurred in the rollup-boost server.
#[derive(Debug, thiserror::Error)]
pub enum RollupBoostServerError {
    /// JSON-RPC error.
    #[error("Rollup boost server error: {0}")]
    Jsonrpsee(#[from] jsonrpsee_types::ErrorObjectOwned),
}

impl From<RollupBoostServerError> for RpcError<TransportErrorKind> {
    fn from(error: RollupBoostServerError) -> Self {
        match error {
            RollupBoostServerError::Jsonrpsee(error) => Self::ErrorResp(ErrorPayload {
                code: error.code().into(),
                message: error.message().to_string().into(),
                data: None,
            }),
        }
    }
}
