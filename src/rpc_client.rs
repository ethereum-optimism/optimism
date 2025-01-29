use clap::{arg, ArgGroup, Parser};
use clap::{
    builder::{PossibleValue, RangedU64ValueParser, TypedValueParser},
    Arg, Args, Command,
};
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use std::net::{IpAddr, Ipv4Addr, SocketAddr};
use std::path::PathBuf;
use std::time::Duration;

/// Parameters for configuring the rpc more granularity via CLI
#[derive(Debug, Clone, Args, PartialEq, Eq)]
#[command(next_help_heading = "RPC")]
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
}

pub struct RpcClient {
    pub client: HttpClient<HttpBackend>,
    pub auth_client: HttpClient<AuthClientService<HttpBackend>>,
}

impl RpcClient {
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
            auth_client,
        })
    }
}
