//! Rpc CLI Arguments
//!
//! Flags for configuring the RPC server.

use clap::Parser;
use kona_rpc::RpcBuilder;
use std::{
    net::{IpAddr, SocketAddr},
    path::PathBuf,
};

/// RPC CLI Arguments
#[derive(Parser, Debug, Clone, PartialEq, Eq)]
pub struct RpcArgs {
    /// Whether to disable the rpc server.
    #[arg(long = "rpc.disabled", default_value = "false", env = "KONA_NODE_RPC_DISABLED")]
    pub rpc_disabled: bool,
    /// Prevent the RPC server from attempting to restart.
    #[arg(long = "rpc.no-restart", default_value = "false", env = "KONA_NODE_RPC_NO_RESTART")]
    pub no_restart: bool,
    /// RPC listening address.
    #[arg(long = "rpc.addr", default_value = "0.0.0.0", env = "KONA_NODE_RPC_ADDR")]
    pub listen_addr: IpAddr,
    /// RPC listening port.
    #[arg(long = "port", alias = "rpc.port", default_value = "9545", env = "KONA_NODE_RPC_PORT")]
    pub listen_port: u16,
    /// Enable the admin API.
    #[arg(long = "rpc.enable-admin", env = "KONA_NODE_RPC_ENABLE_ADMIN")]
    pub enable_admin: bool,
    /// File path used to persist state changes made via the admin API so they persist across
    /// restarts. Disabled if not set.
    #[arg(long = "rpc.admin-state", env = "KONA_NODE_RPC_ADMIN_STATE")]
    pub admin_persistence: Option<PathBuf>,
    /// Enables websocket rpc server to track block production
    #[arg(long = "rpc.ws-enabled", default_value = "false", env = "KONA_NODE_RPC_WS_ENABLED")]
    pub ws_enabled: bool,
    /// Enables development RPC endpoints for engine state introspection
    #[arg(long = "rpc.dev-enabled", default_value = "false", env = "KONA_NODE_RPC_DEV_ENABLED")]
    pub dev_enabled: bool,
}

impl Default for RpcArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl From<RpcArgs> for Option<RpcBuilder> {
    fn from(args: RpcArgs) -> Self {
        if args.rpc_disabled {
            return None;
        }
        Some(RpcBuilder {
            no_restart: args.no_restart,
            socket: SocketAddr::new(args.listen_addr, args.listen_port),
            enable_admin: args.enable_admin,
            admin_persistence: args.admin_persistence,
            ws_enabled: args.ws_enabled,
            dev_enabled: args.dev_enabled,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;
    use std::net::Ipv4Addr;

    #[rstest]
    #[case::disable_rpc(&["--rpc.disabled"], |args: &mut RpcArgs| { args.rpc_disabled = true; })]
    #[case::no_restart(&["--rpc.no-restart"], |args: &mut RpcArgs| { args.no_restart = true; })]
    #[case::disable_rpc(&["--rpc.addr", "1.1.1.1"], |args: &mut RpcArgs| { args.listen_addr = IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)); })]
    #[case::disable_rpc(&["--port", "8743"], |args: &mut RpcArgs| { args.listen_port = 8743; })]
    #[case::disable_rpc_alias(&["--rpc.port", "8743"], |args: &mut RpcArgs| { args.listen_port = 8743; })]
    #[case::disable_rpc(&["--rpc.enable-admin"], |args: &mut RpcArgs| { args.enable_admin = true; })]
    #[case::disable_rpc(&["--rpc.admin-state", "/"], |args: &mut RpcArgs| { args.admin_persistence = Some(PathBuf::from("/")); })]
    fn test_parse_rpc_args(#[case] args: &[&str], #[case] mutate: impl Fn(&mut RpcArgs)) {
        let args = [&["kona-node"], args].concat();
        let cli = RpcArgs::parse_from(args);
        let mut expected = RpcArgs::default();
        mutate(&mut expected);
        assert_eq!(cli, expected);
    }
}
