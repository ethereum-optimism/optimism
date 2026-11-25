//! Rpc CLI Arguments
//!
//! Flags for configuring the RPC server.

use clap::Parser;
use kona_rpc::RpcBuilder;
use std::net::{IpAddr, SocketAddr};

/// RPC CLI Arguments
#[derive(Parser, Debug, Clone, PartialEq, Eq)]
pub struct RpcArgs {
    /// Whether to disable the rpc server.
    #[arg(long = "rpc.disabled", default_value = "false", env = "KONA_NODE_RPC_DISABLED")]
    pub rpc_disabled: bool,
    /// RPC listening address.
    ///
    /// The admin API (`admin_*`, including `admin_startSequencer`, `admin_stopSequencer` and
    /// `admin_postUnsafePayload`) is always served on this port, without authentication. Binding
    /// a non-loopback address, such as `0.0.0.0` to publish the port from a Docker container,
    /// exposes it: firewall the port so that only trusted callers reach it.
    #[arg(long = "rpc.addr", default_value = "127.0.0.1", env = "KONA_NODE_RPC_ADDR")]
    pub listen_addr: IpAddr,
    /// RPC listening port.
    #[arg(long = "port", alias = "rpc.port", default_value = "9545", env = "KONA_NODE_RPC_PORT")]
    pub listen_port: u16,
    /// Serves the experimental `opstack` block-building namespace on the RPC port, through which
    /// op-test-sequencer builds, commits and publishes blocks on the node. Like the admin API, it
    /// is unauthenticated.
    #[arg(
        long = "experimental.sequencer-api",
        default_value = "false",
        env = "KONA_NODE_EXPERIMENTAL_SEQUENCER_API"
    )]
    pub experimental_sequencer_api: bool,
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
            socket: SocketAddr::new(args.listen_addr, args.listen_port),
            experimental_sequencer_api: args.experimental_sequencer_api,
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
    #[case::disable_rpc(&["--rpc.addr", "1.1.1.1"], |args: &mut RpcArgs| { args.listen_addr = IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)); })]
    #[case::disable_rpc(&["--port", "8743"], |args: &mut RpcArgs| { args.listen_port = 8743; })]
    #[case::disable_rpc_alias(&["--rpc.port", "8743"], |args: &mut RpcArgs| { args.listen_port = 8743; })]
    #[case::experimental_sequencer_api(&["--experimental.sequencer-api"], |args: &mut RpcArgs| { args.experimental_sequencer_api = true; })]
    fn test_parse_rpc_args(#[case] args: &[&str], #[case] mutate: impl Fn(&mut RpcArgs)) {
        let args = [&["kona-node"], args].concat();
        let cli = RpcArgs::parse_from(args);
        let mut expected = RpcArgs::default();
        mutate(&mut expected);
        assert_eq!(cli, expected);
    }

    #[test]
    fn rpc_addr_defaults_to_loopback() {
        // The admin API is always served, so the RPC must not be reachable remotely by default.
        assert_eq!(RpcArgs::default().listen_addr, IpAddr::V4(Ipv4Addr::LOCALHOST));
    }
}
