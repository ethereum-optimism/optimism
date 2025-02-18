use crate::auth_layer::{AuthClientLayer, AuthClientService};
use alloy_rpc_types_engine::{JwtError, JwtSecret};
use clap::{arg, Parser};
use http::Uri;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use paste::paste;
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Duration;
use thiserror::Error;

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
}

impl ExecutionClient {
    /// Initializes a new [ExecutionClient] with JWT auth for the Engine API and without auth for general execution layer APIs.
    pub fn new(
        auth_rpc: Uri,
        auth_rpc_jwt_secret: JwtSecret,
        timeout: u64,
    ) -> Result<Self, ExecutionClientError> {
        let auth_layer = AuthClientLayer::new(auth_rpc_jwt_secret);
        let auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(auth_rpc.to_string())?;

        Ok(Self {
            auth_client: Arc::new(auth_client),
            auth_rpc,
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
