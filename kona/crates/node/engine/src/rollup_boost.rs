//! Rollup-boost abstraction used by the engine client.

use alloy_json_rpc::{ErrorPayload, RpcError};
use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use alloy_transport::TransportErrorKind;
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes,
};
use rollup_boost::{EngineApiExt, EngineApiServer, ExecutionMode, Health, Probes};
use std::{fmt::Debug, sync::Arc};

use rollup_boost::BlockSelectionPolicy;
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

/// Trait object used to erase the concrete rollup-boost server type.
#[async_trait::async_trait]
pub trait RollupBoostServerLike: Debug + Send + Sync {
    /// Sets the execution mode.
    fn set_execution_mode(&self, execution_mode: ExecutionMode);

    /// Gets the execution mode.
    fn get_execution_mode(&self) -> ExecutionMode;

    /// Creates a new payload v3.
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> Result<PayloadStatus, RollupBoostServerError>;

    /// Creates a new payload v4.
    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Vec<Bytes>,
    ) -> Result<PayloadStatus, RollupBoostServerError>;

    /// Performs a fork choice updated v3.
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated, RollupBoostServerError>;

    /// Gets a payload v3.
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> Result<OpExecutionPayloadEnvelopeV3, RollupBoostServerError>;

    /// Gets a payload v4.
    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> Result<OpExecutionPayloadEnvelopeV4, RollupBoostServerError>;
}

#[async_trait::async_trait]
impl<T: EngineApiExt + Send + Sync + 'static + Debug> RollupBoostServerLike
    for rollup_boost::RollupBoostServer<T>
{
    fn set_execution_mode(&self, execution_mode: ExecutionMode) {
        *self.execution_mode.lock() = execution_mode;
    }

    fn get_execution_mode(&self) -> ExecutionMode {
        *self.execution_mode.lock()
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> Result<PayloadStatus, RollupBoostServerError> {
        EngineApiServer::new_payload_v3(self, payload, versioned_hashes, parent_beacon_block_root)
            .await
            .map_err(RollupBoostServerError::from)
    }

    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Vec<Bytes>,
    ) -> Result<PayloadStatus, RollupBoostServerError> {
        EngineApiServer::new_payload_v4(
            self,
            payload,
            versioned_hashes,
            parent_beacon_block_root,
            execution_requests,
        )
        .await
        .map_err(RollupBoostServerError::from)
    }

    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> Result<ForkchoiceUpdated, RollupBoostServerError> {
        EngineApiServer::fork_choice_updated_v3(self, fork_choice_state, payload_attributes)
            .await
            .map_err(RollupBoostServerError::from)
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> Result<OpExecutionPayloadEnvelopeV3, RollupBoostServerError> {
        EngineApiServer::get_payload_v3(self, payload_id)
            .await
            .map_err(RollupBoostServerError::from)
    }

    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> Result<OpExecutionPayloadEnvelopeV4, RollupBoostServerError> {
        EngineApiServer::get_payload_v4(self, payload_id)
            .await
            .map_err(RollupBoostServerError::from)
    }
}

/// Structure that wraps a rollup boost server and its probes.
///
/// TODO(op-rs/kona#3053): remove this wrapper and use the RollupBoostServer directly
/// Remove the dynamic dispatch and use generics instead.
#[derive(Debug)]
pub struct RollupBoostServer {
    /// The rollup boost server implementation
    pub server: Box<dyn RollupBoostServerLike + Send + Sync + 'static>,
    /// Rollup boost probes
    pub probes: Arc<Probes>,
}

impl RollupBoostServer {
    /// Gets the health of the rollup boost server.
    pub fn get_health(&self) -> Health {
        self.probes.health()
    }
}
