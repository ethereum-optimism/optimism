//! Rpc CLI Arguments
//!
//! Flags for configuring the RPC server.

use clap::Parser;
use std::net::{IpAddr, SocketAddr};

/// RPC CLI Arguments
#[derive(Parser, Debug, Clone, PartialEq, Eq)]
#[command(next_help_heading = "RPC")]
pub struct RpcArgs {
    /// RPC listening address.
    ///
    /// Defaults to `0.0.0.0`: the node is released as a Docker image, where binding `127.0.0.1`
    /// makes the RPC unreachable from outside the container. The server always serves the admin
    /// API, which can stop the sequencer and reset the node, so operators placing the node on an
    /// untrusted network must firewall the port.
    #[arg(long = "rpc.addr", default_value = "0.0.0.0", env = "KONA_NODE_RPC_ADDR")]
    pub listen_addr: IpAddr,
    /// RPC listening port.
    #[arg(long = "rpc.port", default_value = "9545", env = "KONA_NODE_RPC_PORT")]
    pub listen_port: u16,
}

impl Default for RpcArgs {
    fn default() -> Self {
        // Construct default values using the clap parser.
        // This works since none of the cli flags are required.
        Self::parse_from::<[_; 0], &str>([])
    }
}

impl RpcArgs {
    /// Returns the address the RPC server listens on.
    pub const fn addr(&self) -> SocketAddr {
        SocketAddr::new(self.listen_addr, self.listen_port)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;
    use std::net::Ipv4Addr;

    #[rstest]
    #[case::listen_addr(&["--rpc.addr", "1.1.1.1"], |args: &mut RpcArgs| { args.listen_addr = IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)); })]
    #[case::listen_port(&["--rpc.port", "8743"], |args: &mut RpcArgs| { args.listen_port = 8743; })]
    fn test_parse_rpc_args(#[case] args: &[&str], #[case] mutate: impl Fn(&mut RpcArgs)) {
        let args = [&["kona-node"], args].concat();
        let cli = RpcArgs::parse_from(args);
        let mut expected = RpcArgs::default();
        mutate(&mut expected);
        assert_eq!(cli, expected);
    }

    #[test]
    fn rpc_addr_defaults_to_all_interfaces() {
        // 0.0.0.0 is intentional: released as a Docker image, where 127.0.0.1 is unreachable.
        assert_eq!(RpcArgs::default().listen_addr, IpAddr::V4(Ipv4Addr::UNSPECIFIED));
    }
}
