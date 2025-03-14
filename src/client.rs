use crate::auth_layer::{AuthClientLayer, AuthClientService};
use crate::server::{EngineApiClient, PayloadSource};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, JwtError, JwtSecret,
    PayloadId, PayloadStatus,
};
use clap::{arg, Parser};
use http::Uri;
// use jsonrpsee::core::ClientError;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::types::ErrorObjectOwned;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpPayloadAttributes};
use opentelemetry::trace::SpanKind;
use paste::paste;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
use thiserror::Error;
use tracing::{error, instrument};

pub(crate) type ClientResult<T> = Result<T, ClientError>;

#[derive(Error, Debug)]
pub(crate) enum ClientError {
    #[error(transparent)]
    Jsonrpsee(#[from] jsonrpsee::core::client::Error),
    #[error("Invalid payload: {0}")]
    InvalidPayload(String)
}

impl From<ClientError> for ErrorObjectOwned {
    fn from(err: ClientError) -> Self {
        match err {
            ClientError::Jsonrpsee(err) => {
                match err {
                    jsonrpsee::core::ClientError::Call(error_object) => error_object,
                    e => ErrorObjectOwned::owned(0, e.to_string(), Option::<()>::None) 
,
                }
            },
            e => {
                ErrorObjectOwned::owned(0, e.to_string(), Option::<()>::None) 
            }
        }
    }
}

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
        err
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
            head_block_hash = %fork_choice_state.head_block_hash,
            url = %self.auth_rpc,
            payload_id
        )
    )]
    pub async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> ClientResult<ForkchoiceUpdated> {
        let res = self.auth_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes.clone())
            .await?;

        if let Some(payload_id) = res.payload_id {
            tracing::Span::current().record("payload_id", payload_id.to_string());
        }
        
        if res.is_invalid() {
            return Err(ClientError::InvalidPayload(res.payload_status.status.to_string()))
        }

        Ok(res)
    }

    #[instrument(skip(self), 
        err,
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
            url = %self.auth_rpc,
            %payload_id,
        )
    )]
    pub async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> ClientResult<OpExecutionPayloadEnvelopeV3> {
        Ok(self.auth_client
            .get_payload_v3(payload_id)
            .await?)
    }

    #[instrument(skip_all, 
        err
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
            url = %self.auth_rpc,
            block_hash,
        )
    )]
    pub async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> ClientResult<PayloadStatus> {
        let execution_payload = ExecutionPayload::from(payload.clone());
        let block_hash = execution_payload.block_hash();
        tracing::Span::current().record("block_hash", block_hash.to_string());
        Ok(self.auth_client
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await?)
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
