use crate::metrics::ServerMetrics;
use alloy_primitives::{Bytes, B256, U128, U64};
use alloy_rpc_types_engine::{
    ExecutionPayload, ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId,
    PayloadStatus,
};
use jsonrpsee::core::{async_trait, ClientError, RegisterMethodError, RpcResult};
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::proc_macros::rpc;
use jsonrpsee::types::error::INVALID_REQUEST_CODE;
use jsonrpsee::types::{ErrorCode, ErrorObject};
use jsonrpsee::RpcModule;
use lru::LruCache;
use op_alloy_rpc_jsonrpsee::traits::{MinerApiExtClient, MinerApiExtServer};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
use opentelemetry::global::{self, BoxedSpan, BoxedTracer};
use opentelemetry::trace::{Span, TraceContextExt, Tracer};
use opentelemetry::{Context, KeyValue};
use paste::paste;
use reth_optimism_payload_builder::{OpPayloadAttributes, OpPayloadBuilderAttributes};
use reth_payload_primitives::PayloadBuilderAttributes;
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use std::net::{IpAddr, SocketAddr};
use std::num::NonZero;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::Mutex;
use tracing::{debug, error, info};

use clap::{arg, ArgGroup, Parser};
use clap::{
    builder::{PossibleValue, RangedU64ValueParser, TypedValueParser},
    Arg, Args, Command,
};
use std::path::PathBuf;

/// Parameters for configuring the rpc more granularity via CLI
#[derive(Debug, Clone, Args, PartialEq, Eq)]
#[command(next_help_heading = "RPC")]
#[command(group(
    ArgGroup::new("jwt")
        .required(true)
        .args(&["auth_jwtsecret", "rpc_jwtsecret"]),
))]
pub struct RpcClientArgs {
    /// Http server address to listen on
    #[arg(long = "http.addr")]
    pub http_addr: IpAddr,

    /// Http server port to listen on
    #[arg(long = "http.port")]
    pub http_port: u16,

    /// Http Corsdomain to allow request from
    #[arg(long = "http.corsdomain")]
    pub http_corsdomain: Option<String>,

    /// Auth server address to listen on
    #[arg(long = "authrpc.addr")]
    pub auth_addr: IpAddr,

    /// Auth server port to listen on
    #[arg(long = "authrpc.port")]
    pub auth_port: u16,

    /// Path to a JWT secret to use for the authenticated engine-API RPC server.
    ///
    /// This will enforce JWT authentication for all requests coming from the consensus layer.
    ///
    /// If no path is provided, a secret will be generated and stored in the datadir under
    /// `<DIR>/<CHAIN_ID>/jwt.hex`. For mainnet this would be `~/.reth/mainnet/jwt.hex` by default.
    #[arg(
        long = "authrpc.jwtsecret",
        value_name = "PATH",
        global = true,
        required = false
    )]
    pub auth_jwtsecret: Option<PathBuf>,

    /// Filename for auth IPC socket/pipe within the datadir
    #[arg(long = "auth-ipc.path")]
    pub auth_ipc_path: Option<String>,

    /// Hex encoded JWT secret to authenticate the regular RPC server(s), see `--http.api` and
    /// `--ws.api`.
    ///
    /// This is __not__ used for the authenticated engine-API RPC server, see
    /// `--authrpc.jwtsecret`.
    #[arg(
        long = "rpc.jwtsecret",
        value_name = "HEX",
        global = true,
        required = false
    )]
    pub rpc_jwtsecret: Option<JwtSecret>,

    /// Timeout for http calls in milliseconds
    #[arg(long, env)]
    pub timeout: u64,
}

pub struct ExecutionClient {
    pub client: HttpClient<HttpBackend>,
    pub http_socket: SocketAddr,
    pub auth_client: HttpClient<AuthClientService<HttpBackend>>,
    pub auth_socket: SocketAddr,
}

impl ExecutionClient {
    pub fn new(
        http_addr: IpAddr,
        http_port: u16,
        auth_addr: IpAddr,
        auth_port: u16,
        jwt_secret: JwtSecret,
        timeout: u64,
    ) -> Result<Self, jsonrpsee::core::client::Error> {
        let http_socket = SocketAddr::new(http_addr, http_port);
        let client = HttpClientBuilder::new()
            .request_timeout(Duration::from_millis(timeout))
            .build(format!("http://{}", http_socket))?;

        let auth_layer = AuthClientLayer::new(jwt_secret);
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

#[rpc(server, client, namespace = "engine")]
pub trait EngineApi {
    #[method(name = "forkchoiceUpdatedV3")]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    #[method(name = "getPayloadV3")]
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV3>;

    #[method(name = "newPayloadV3")]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus>;
}

#[rpc(server, client, namespace = "eth")]
pub trait EthApi {
    #[method(name = "sendRawTransaction")]
    async fn send_raw_transaction(&self, bytes: Bytes) -> RpcResult<B256>;
}

/*TODO: Remove this in favor of the `MinerApi` from Reth once the
       trait methods are updated to be async
*/
/// Miner namespace rpc interface that can control miner/builder settings
#[rpc(server, client, namespace = "miner")]
pub trait MinerApi {
    /// Sets the extra data string that is included when this miner mines a block.
    ///
    /// Returns an error if the extra data is too long.
    #[method(name = "setExtra")]
    async fn set_extra(&self, record: Bytes) -> RpcResult<bool>;

    /// Sets the minimum accepted gas price for the miner.
    #[method(name = "setGasPrice")]
    async fn set_gas_price(&self, gas_price: U128) -> RpcResult<bool>;

    /// Sets the gaslimit to target towards during mining.
    #[method(name = "setGasLimit")]
    async fn set_gas_limit(&self, gas_price: U128) -> RpcResult<bool>;
}

macro_rules! define_rpc_args {
    ($(($name:ident, $prefix:ident)),*) => {
        $(
            paste! {
                #[derive(Debug, Clone, Args, PartialEq, Eq)]
                pub struct $name {
                    #[arg(long)]
                    pub [<$prefix _http_addr>]: IpAddr,

                    #[arg(long)]
                    pub [<$prefix _http_port>]: u16,

                    #[arg(long)]
                    pub [<$prefix _http_corsdomain>]: Option<String>,

                    #[arg(long)]
                    pub [<$prefix _auth_addr>]: IpAddr,

                    #[arg(long)]
                    pub [<$prefix _auth_port>]: u16,

                    #[arg(long, value_name = "PATH", global = true)]
                    pub [<$prefix _auth_jwtsecret>]: Option<PathBuf>,

                    #[arg(long)]
                    pub [<$prefix _auth_ipc_path>]: Option<String>,

                    #[arg(long, value_name = "HEX", global = true)]
                    pub [<$prefix _rpc_jwtsecret>]: Option<JwtSecret>,

                    #[arg(long)]
                    pub [<$prefix _timeout>]: u64,
                }
            }
        )*
    };
}

define_rpc_args!((BuilderArgs, builder), (L2ClientArgs, l2));
