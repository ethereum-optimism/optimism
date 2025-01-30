use alloy_primitives::{Bytes, B256, U128};
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use clap::{arg, ArgGroup, Parser};
use jsonrpsee::core::client::ClientT;
use jsonrpsee::core::RpcResult;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::proc_macros::rpc;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
use paste::paste;
use reth_optimism_payload_builder::OpPayloadAttributes;
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use std::net::{IpAddr, SocketAddr};
use std::time::Duration;

/// Client interface for interacting with an execution layer node.
///
/// - **Engine API** calls are faciliated via the `auth_client` (requires JWT authentication).
/// -  All other API calls including the **Eth & Miner APIs** are faciliated via the `client` (optional JWT authentication).
///
#[derive(Clone)]
pub struct ExecutionClient<
    C: ClientT = HttpClient<HttpBackend>,
    A: ClientT = HttpClient<AuthClientService<HttpBackend>>,
> {
    /// Handles requests to Eth, Miner, and other Execution Layer APIs (optional JWT authentication)
    pub client: C,
    /// Address of the RPC server for execution layer API calls, excluding the Engine API
    pub http_socket: SocketAddr,
    /// Handles requests to the authenticated Engine API (requires JWT authentication)
    pub auth_client: A,
    /// Address of the RPC server for authenticated Engine API calls
    pub auth_socket: SocketAddr,
}

impl ExecutionClient {
    /// Initializes a new [ExecutionClient] with JWT auth for the Engine API and without auth for general Execution Layer APIs.
    pub fn new(
        http_addr: IpAddr,
        http_port: u16,
        auth_addr: IpAddr,
        auth_port: u16,
        auth_rpc_jwt_secret: JwtSecret,
        timeout: u64,
    ) -> Result<Self, jsonrpsee::core::client::Error> {
        let http_socket = SocketAddr::new(http_addr, http_port);
        let client = HttpClientBuilder::new()
            .request_timeout(Duration::from_millis(timeout))
            .build(format!("http://{}", http_socket))?;

        let auth_layer = AuthClientLayer::new(auth_rpc_jwt_secret);
        let auth_socket = SocketAddr::new(auth_addr, auth_port);
        let auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(format!("http://{}", auth_socket))?;

        Ok(Self {
            client,
            http_socket,
            auth_client,
            auth_socket,
        })
    }
}

impl ExecutionClient<HttpClient<AuthClientService<HttpBackend>>> {
    /// Initializes a new [ExecutionClient] with JWT auth for the Engine API and general Execution Layer APIs.
    pub fn new_with_auth(
        http_addr: IpAddr,
        http_port: u16,
        rpc_jwt_secret: JwtSecret,
        auth_addr: IpAddr,
        auth_port: u16,
        auth_rpc_jwt_secret: JwtSecret,
        timeout: u64,
    ) -> Result<Self, jsonrpsee::core::client::Error> {
        let rpc_auth_layer = AuthClientLayer::new(rpc_jwt_secret);
        let http_socket = SocketAddr::new(http_addr, http_port);
        let client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(rpc_auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(format!("http://{}", http_socket))?;

        let auth_layer = AuthClientLayer::new(auth_rpc_jwt_secret);
        let auth_socket = SocketAddr::new(auth_addr, auth_port);
        let auth_client = HttpClientBuilder::new()
            .set_http_middleware(tower::ServiceBuilder::new().layer(auth_layer))
            .request_timeout(Duration::from_millis(timeout))
            .build(format!("http://{}", auth_socket))?;

        Ok(Self {
            client,
            http_socket,
            auth_client,
            auth_socket,
        })
    }
}

/// Generates Clap argument structs with a prefix to create a unique namespace when specifing RPC client config via the CLI.
macro_rules! define_rpc_args {
    ($(($name:ident, $prefix:ident)),*) => {
        $(
            paste! {
                #[derive(Parser, Debug, Clone, PartialEq, Eq)]
                #[clap(group(ArgGroup::new(concat!(stringify!($prefix), "_jwt"))
                    .required(true)
                    .multiple(false)
                    .args(&[
                        concat!(stringify!($prefix), "_jwtsecret"),
                        concat!(stringify!($prefix), "_jwtsecret_path")
                    ])
                ))]
                pub struct $name {
                    /// Http server address
                    #[arg(long)]
                    pub [<$prefix _http_addr>]: IpAddr,

                    /// Http server port
                    #[arg(long)]
                    pub [<$prefix _http_port>]: u16,

                    /// Auth server address
                    #[arg(long)]
                    pub [<$prefix _auth_addr>]: IpAddr,

                    /// Auth server port
                    #[arg(long)]
                    pub [<$prefix _auth_port>]: u16,

                    /// Path to a JWT secret to use for the authenticated engine-API RPC server.
                    #[arg(long, value_name = "PATH")]
                    pub [<$prefix _auth_rpc_jwt_secret>]: JwtSecret,

                    /// Hex encoded JWT secret to authenticate the regular RPC server(s)
                    ///
                    /// This is __not__ used for the authenticated engine-API RPC server, see
                    /// `authrpc.jwtsecret`.
                    #[arg(long, value_name = "HEX")]
                    pub [<$prefix _rpc_jwt_secret>]: JwtSecret,


                    /// Filename for auth IPC socket/pipe within the datadir
                    #[arg(long)]
                    pub [<$prefix _auth_ipc_path>]: Option<String>,

                    /// Timeout for http calls in milliseconds
                    #[arg(long)]
                    pub [<$prefix _timeout>]: u64,
                }
            }
        )*
    };
}

define_rpc_args!((BuilderArgs, builder), (L2ClientArgs, l2));
