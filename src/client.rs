use crate::auth_layer::{AuthClientLayer, AuthClientService};
use crate::server::{EngineApiClient, PayloadSource};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, JwtError, JwtSecret,
    PayloadId, PayloadStatus,
};
use clap::{arg, Parser};
use http::Uri;
use jsonrpsee::core::{ClientError, RpcResult};
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::types::ErrorCode;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpPayloadAttributes};
use opentelemetry::trace::SpanKind;
use paste::paste;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
use thiserror::Error;
use tracing::{error, instrument};

#[derive(Error, Debug)]
pub enum ExecutionClientError {
    #[error(transparent)]
    HttpClient(#[from] jsonrpsee::core::client::Error),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Jwt(#[from] JwtError),
}

/// Client interface for interacting with execution layer node's Engine API.
///
/// - **Engine API** calls are faciliated via the `auth_client` (requires JWT authentication).
///
#[derive(Clone)]
pub struct ExecutionClient {
    /// Handles requests to the authenticated Engine API (requires JWT authentication)
    pub auth_client: Arc<HttpClient<AuthClientService<HttpBackend>>>,
    /// Uri of the RPC server for authenticated Engine API calls
    pub auth_rpc: Uri,
    /// The source of the payload
    pub payload_source: PayloadSource,
}

impl ExecutionClient {
    /// Initializes a new [ExecutionClient] with JWT auth for the Engine API and without auth for general execution layer APIs.
    pub fn new(
        auth_rpc: Uri,
        auth_rpc_jwt_secret: JwtSecret,
        timeout: u64,
        payload_source: PayloadSource,
    ) -> Result<Self, ExecutionClientError> {
        let auth_layer = AuthClientLayer::new(auth_rpc_jwt_secret);
        let auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(auth_rpc.to_string())?;

        Ok(Self {
            auth_client: Arc::new(auth_client),
            auth_rpc,
            payload_source,
        })
    }

    #[instrument(skip_all, 
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
        )
    )]
    pub async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        self.auth_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes.clone())
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err,
                other_error => {
                    error!(
                        message = "error calling fork_choice_updated_v3 for l2 client",
                        "url" = ?self.auth_rpc,
                        "error" = %other_error,
                        "head_block_hash" = %fork_choice_state.head_block_hash,
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    #[instrument(skip_all, 
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
        )
    )]
    pub async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<(OpExecutionPayloadEnvelopeV3, PayloadSource)> {
        self.auth_client
            .get_payload_v3(payload_id)
            .await
            .map(|payload| (payload, self.payload_source.clone()))
            .map_err(|e| match e {
                ClientError::Call(err) => err,
                other_error => {
                    error!(
                        message = "error calling get_payload_v3",
                        "error" = %other_error,
                        "payload_id" = %payload_id
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }

    #[instrument(skip_all, 
        err
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
        )
    )]
    pub async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus> {
        let execution_payload = ExecutionPayload::from(payload.clone());
        let block_hash = execution_payload.block_hash();
        self.auth_client
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await
            .map_err(|e| match e {
                ClientError::Call(err) => err,
                other_error => {
                    error!(
                        message = "error calling new_payload_v3",
                        "url" = ?self.auth_rpc,
                        "error" = %other_error,
                        "block_hash" = %block_hash
                    );
                    ErrorCode::InternalError.into()
                }
            })
    }
}

/// Generates Clap argument structs with a prefix to create a unique namespace when specifing RPC client config via the CLI.
macro_rules! define_rpc_args {
    ($(($name:ident, $prefix:ident)),*) => {
        $(
            paste! {
                #[derive(Parser, Debug, Clone, PartialEq, Eq)]
                pub struct $name {
                    /// Auth server address
                    #[arg(long, env, default_value = "127.0.0.1:8551")]
                    pub [<$prefix _url>]: Uri,

                    /// Hex encoded JWT secret to use for the authenticated engine-API RPC server.
                    #[arg(long, env, value_name = "HEX")]
                    pub [<$prefix _jwt_token>]: Option<JwtSecret>,

                    /// Path to a JWT secret to use for the authenticated engine-API RPC server.
                    #[arg(long, env, value_name = "PATH")]
                    pub [<$prefix _jwt_path>]: Option<PathBuf>,

                    /// Timeout for http calls in milliseconds
                    #[arg(long, env, default_value_t = 1000)]
                    pub [<$prefix _timeout>]: u64,
                }
            }
        )*
    };
}

define_rpc_args!((BuilderArgs, builder), (L2ClientArgs, l2));
