//! Rpc CLI Arguments
//!
//! Flags for configuring the RPC server.

use clap::{ArgAction, Parser};
use std::net::{IpAddr, SocketAddr};

/// RPC CLI Arguments
///
/// The address and port apply only with `--rpc.enabled`, and are ignored otherwise.
#[derive(Parser, Debug, Clone, PartialEq, Eq)]
#[command(next_help_heading = "RPC")]
pub struct RpcArgs {
    /// Serve the RPC API, including the admin API. Disabled by default. A sequencer requires it,
    /// since it starts stopped and is started through the admin API.
    #[arg(
        id = "rpc_enabled",
        long = "rpc.enabled",
        env = "KONA_NODE_RPC_ENABLED",
        action = ArgAction::Set,
        num_args = 0..=1,
        require_equals = true,
        default_value_t = false,
        default_missing_value = "true"
    )]
    pub enabled: bool,
    /// RPC listening address.
    ///
    /// Defaults to `127.0.0.1`, because the server always serves the admin API, which can stop
    /// the sequencer and reset the node. Set it to `0.0.0.0` to reach the RPC from outside a
    /// container, and keep the port off untrusted networks.
    #[arg(long = "rpc.addr", default_value = "127.0.0.1", env = "KONA_NODE_RPC_ADDR")]
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
    /// Returns the address the RPC server listens on, or `None` if the node serves no RPC.
    pub const fn addr(&self) -> Option<SocketAddr> {
        if self.enabled { Some(SocketAddr::new(self.listen_addr, self.listen_port)) } else { None }
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
    fn serves_no_rpc_by_default() {
        assert_eq!(RpcArgs::default().addr(), None);
    }

    #[test]
    fn serves_on_the_address_and_port_given_while_enabled() {
        let args = RpcArgs::parse_from([
            "kona-node",
            "--rpc.enabled",
            "--rpc.addr",
            "1.1.1.1",
            "--rpc.port",
            "1",
        ]);
        assert_eq!(args.addr(), Some(SocketAddr::new(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), 1)));
        let args = RpcArgs::parse_from(["kona-node", "--rpc.enabled=false", "--rpc.port", "1"]);
        assert_eq!(args.addr(), None);
    }

    #[test]
    fn rpc_addr_defaults_to_loopback() {
        // The RPC serves the admin API, so it is reachable only locally unless asked otherwise.
        assert_eq!(RpcArgs::default().listen_addr, IpAddr::V4(Ipv4Addr::LOCALHOST));
    }
}
