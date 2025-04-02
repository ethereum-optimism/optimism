use crate::client::auth::{AuthClientLayer, AuthClientService};
use crate::server::{EngineApiClient, PayloadSource};
use alloy_primitives::B256;
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, JwtError, JwtSecret,
    PayloadId, PayloadStatus,
};
use clap::{Parser, arg};
use http::Uri;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::types::ErrorObjectOwned;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelopeV3, OpPayloadAttributes};
use opentelemetry::trace::SpanKind;
use paste::paste;
use std::path::PathBuf;
use std::time::Duration;
use thiserror::Error;
use tracing::{error, info, instrument};

const INTERNAL_ERROR: i32 = 13;

pub(crate) type ClientResult<T> = Result<T, RpcClientError>;

#[derive(Error, Debug)]
pub enum RpcClientError {
    #[error(transparent)]
    Jsonrpsee(#[from] jsonrpsee::core::client::Error),
    #[error("Invalid payload: {0}")]
    InvalidPayload(String),
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Jwt(#[from] JwtError),
}

trait Code: Sized {
    fn code(&self) -> i32;

    fn set_code(self) -> Self {
        tracing::Span::current().record("code", self.code());
        self
    }
}

impl<T, E: Code> Code for Result<T, E> {
    fn code(&self) -> i32 {
        match self {
            Ok(_) => 0,
            Err(e) => e.code(),
        }
    }
}

/// TODO: Add more robust error code system
impl Code for RpcClientError {
    fn code(&self) -> i32 {
        match self {
            RpcClientError::Jsonrpsee(e) => e.code(),
            // Status code 13 == internal error
            _ => INTERNAL_ERROR,
        }
    }
}

impl Code for jsonrpsee::core::client::Error {
    fn code(&self) -> i32 {
        match self {
            jsonrpsee::core::client::Error::Call(call) => call.code(),
            _ => INTERNAL_ERROR,
        }
    }
}

impl From<RpcClientError> for ErrorObjectOwned {
    fn from(err: RpcClientError) -> Self {
        match err {
            RpcClientError::Jsonrpsee(jsonrpsee::core::ClientError::Call(error_object)) => {
                error_object
            }
            // Status code 13 == internal error
            e => ErrorObjectOwned::owned(INTERNAL_ERROR, e.to_string(), Option::<()>::None),
        }
    }
}

/// Client interface for interacting with execution layer node's Engine API.
///
/// - **Engine API** calls are faciliated via the `auth_client` (requires JWT authentication).
///
#[derive(Clone)]
pub struct RpcClient {
    /// Handles requests to the authenticated Engine API (requires JWT authentication)
    auth_client: HttpClient<AuthClientService<HttpBackend>>,
    /// Uri of the RPC server for authenticated Engine API calls
    auth_rpc: Uri,
    /// The source of the payload
    payload_source: PayloadSource,
}

impl RpcClient {
    /// Initializes a new [ExecutionClient] with JWT auth for the Engine API and without auth for general execution layer APIs.
    pub fn new(
        auth_rpc: Uri,
        auth_rpc_jwt_secret: JwtSecret,
        timeout: u64,
        payload_source: PayloadSource,
    ) -> Result<Self, RpcClientError> {
        let auth_layer = AuthClientLayer::new(auth_rpc_jwt_secret);
        let auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(auth_rpc.to_string())?;

        Ok(Self {
            auth_client,
            auth_rpc,
            payload_source,
        })
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
            head_block_hash = %fork_choice_state.head_block_hash,
            url = %self.auth_rpc,
            code,
            payload_id
        )
    )]
    pub async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> ClientResult<ForkchoiceUpdated> {
        info!("Sending fork_choice_updated_v3 to {}", self.payload_source);
        let res = self
            .auth_client
            .fork_choice_updated_v3(fork_choice_state, payload_attributes.clone())
            .await
            .set_code()?;

        if let Some(payload_id) = res.payload_id {
            tracing::Span::current().record("payload_id", payload_id.to_string());
        }

        if res.is_invalid() {
            return Err(RpcClientError::InvalidPayload(
                res.payload_status.status.to_string(),
            ))
            .set_code();
        }

        Ok(res)
    }

    #[instrument(
        skip(self),
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
        info!("Sending get_payload_v3 to {}", self.payload_source);
        Ok(self
            .auth_client
            .get_payload_v3(payload_id)
            .await
            .set_code()?)
    }

    #[instrument(
        skip_all,
        err,
        fields(
            otel.kind = ?SpanKind::Client,
            target = self.payload_source.to_string(),
            url = %self.auth_rpc,
            block_hash,
            code,
        )
    )]
    pub async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> ClientResult<PayloadStatus> {
        info!("Sending new_payload_v3 to {}", self.payload_source);
        let execution_payload = ExecutionPayload::from(payload.clone());
        let block_hash = execution_payload.block_hash();
        tracing::Span::current().record("block_hash", block_hash.to_string());

        let res = self
            .auth_client
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await
            .set_code()?;

        if res.is_invalid() {
            return Err(RpcClientError::InvalidPayload(res.status.to_string()).set_code());
        }

        Ok(res)
    }
}

/// Generates Clap argument structs with a prefix to create a unique namespace when specifying RPC client config via the CLI.
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

#[cfg(test)]
mod tests {
    use assert_cmd::Command;
    use http::Uri;
    use jsonrpsee::core::client::ClientT;

    use crate::client::auth::AuthClientService;
    use crate::server::PayloadSource;
    use alloy_rpc_types_engine::JwtSecret;
    use jsonrpsee::RpcModule;
    use jsonrpsee::http_client::HttpClient;
    use jsonrpsee::http_client::transport::Error as TransportError;
    use jsonrpsee::http_client::transport::HttpBackend;
    use jsonrpsee::{
        core::ClientError,
        rpc_params,
        server::{ServerBuilder, ServerHandle},
    };
    use predicates::prelude::*;
    use reth_rpc_layer::{AuthLayer, JwtAuthValidator};
    use std::net::SocketAddr;
    use std::result::Result;
    use std::str::FromStr;

    use super::*;

    const AUTH_PORT: u32 = 8550;
    const AUTH_ADDR: &str = "0.0.0.0";
    const SECRET: &str = "f79ae8046bc11c9927afe911db7143c51a806c4a537cc08e0d37140b0192f430";

    #[test]
    fn test_invalid_args() {
        let mut cmd = Command::cargo_bin("rollup-boost").unwrap();
        cmd.arg("--invalid-arg");

        cmd.assert().failure().stderr(predicate::str::contains(
            "error: unexpected argument '--invalid-arg' found",
        ));
    }

    #[tokio::test]
    async fn valid_jwt() {
        let secret = JwtSecret::from_hex(SECRET).unwrap();

        let auth_rpc = Uri::from_str(&format!("http://{}:{}", AUTH_ADDR, AUTH_PORT)).unwrap();
        let client = RpcClient::new(auth_rpc, secret, 1000, PayloadSource::L2).unwrap();
        let response = send_request(client.auth_client).await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "You are the dark lord");
    }

    #[tokio::test]
    async fn invalid_jwt() {
        let secret = JwtSecret::random();
        let auth_rpc = Uri::from_str(&format!("http://{}:{}", AUTH_ADDR, AUTH_PORT)).unwrap();
        let client = RpcClient::new(auth_rpc, secret, 1000, PayloadSource::L2).unwrap();
        let response = send_request(client.auth_client).await;
        assert!(response.is_err());
        assert!(matches!(
            response.unwrap_err(),
            ClientError::Transport(e)
                if matches!(e.downcast_ref::<TransportError>(), Some(TransportError::Rejected { status_code: 401 }))
        ));
    }

    async fn send_request(
        client: HttpClient<AuthClientService<HttpBackend>>,
    ) -> Result<String, ClientError> {
        let server = spawn_server().await;

        let response = client
            .request::<String, _>("greet_melkor", rpc_params![])
            .await;

        server.stop().unwrap();
        server.stopped().await;

        response
    }

    /// Spawn a new RPC server equipped with a `JwtLayer` auth middleware.
    async fn spawn_server() -> ServerHandle {
        let secret = JwtSecret::from_hex(SECRET).unwrap();
        let addr = format!("{AUTH_ADDR}:{AUTH_PORT}");
        let validator = JwtAuthValidator::new(secret);
        let layer = AuthLayer::new(validator);
        let middleware = tower::ServiceBuilder::default().layer(layer);

        // Create a layered server
        let server = ServerBuilder::default()
            .set_http_middleware(middleware)
            .build(addr.parse::<SocketAddr>().unwrap())
            .await
            .unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("greet_melkor", |_, _, _| "You are the dark lord")
            .unwrap();

        server.start(module)
    }
}
