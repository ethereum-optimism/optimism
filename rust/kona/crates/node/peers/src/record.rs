//! Commonly used `NodeRecord` type for peers.
//!
//! This is a simplified version of the `NodeRecord` type in reth.
//!
//! Adapted from <https://github.com/paradigmxyz/reth/blob/0e087ae1c35502f0b8d128c64e4c57269af20c0e/crates/net/peers/src/node_record.rs>.

use crate::PeerId;
use core::{
    fmt,
    fmt::Write,
    net::{IpAddr, SocketAddr},
    num::ParseIntError,
    str::FromStr,
};
use std::net::ToSocketAddrs;

/// Represents an ENR in discovery.
///
/// Note: this is only an excerpt of the [`NodeRecord`] data structure.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Hash)]
pub struct NodeRecord {
    /// The Address of a node.
    pub address: IpAddr,
    /// UDP discovery port.
    pub udp_port: u16,
    /// TCP port of the port that accepts connections.
    pub tcp_port: u16,
    /// Public key of the discovery service
    pub id: PeerId,
}

impl NodeRecord {
    /// Converts the `address` into an [`core::net::Ipv4Addr`] if the `address` is a mapped
    /// [`Ipv6Addr`](std::net::Ipv6Addr).
    ///
    /// Returns `true` if the address was converted.
    ///
    /// See also [`std::net::Ipv6Addr::to_ipv4_mapped`]
    pub fn convert_ipv4_mapped(&mut self) -> bool {
        // convert IPv4 mapped IPv6 address
        if let IpAddr::V6(v6) = self.address {
            if let Some(v4) = v6.to_ipv4_mapped() {
                self.address = v4.into();
                return true;
            }
        }
        false
    }

    /// Same as [`Self::convert_ipv4_mapped`] but consumes the type
    pub fn into_ipv4_mapped(mut self) -> Self {
        self.convert_ipv4_mapped();
        self
    }

    /// Sets the tcp port
    pub const fn with_tcp_port(mut self, port: u16) -> Self {
        self.tcp_port = port;
        self
    }

    /// Sets the udp port
    pub const fn with_udp_port(mut self, port: u16) -> Self {
        self.udp_port = port;
        self
    }

    /// Creates a new record from a socket addr and peer id.
    pub const fn new(addr: SocketAddr, id: PeerId) -> Self {
        Self { address: addr.ip(), tcp_port: addr.port(), udp_port: addr.port(), id }
    }

    /// Creates a new record from an ip address and ports.
    pub fn new_with_ports(
        ip_addr: IpAddr,
        tcp_port: u16,
        udp_port: Option<u16>,
        id: PeerId,
    ) -> Self {
        let udp_port = udp_port.unwrap_or(tcp_port);
        Self { address: ip_addr, tcp_port, udp_port, id }
    }

    /// The TCP socket address of this node
    #[must_use]
    pub const fn tcp_addr(&self) -> SocketAddr {
        SocketAddr::new(self.address, self.tcp_port)
    }

    /// The UDP socket address of this node
    #[must_use]
    pub const fn udp_addr(&self) -> SocketAddr {
        SocketAddr::new(self.address, self.udp_port)
    }
}

impl fmt::Display for NodeRecord {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("enode://")?;
        alloy_primitives::hex::encode(self.id.as_slice()).fmt(f)?;
        f.write_char('@')?;
        match self.address {
            IpAddr::V4(ip) => {
                ip.fmt(f)?;
            }
            IpAddr::V6(ip) => {
                // encapsulate with brackets
                f.write_char('[')?;
                ip.fmt(f)?;
                f.write_char(']')?;
            }
        }
        f.write_char(':')?;
        self.tcp_port.fmt(f)?;
        if self.tcp_port != self.udp_port {
            f.write_str("?discport=")?;
            self.udp_port.fmt(f)?;
        }

        Ok(())
    }
}

/// Possible error types when parsing a [`NodeRecord`]
#[derive(Debug, thiserror::Error)]
pub enum NodeRecordParseError {
    /// Invalid url
    #[error("Failed to parse url: {0}")]
    InvalidUrl(String),
    /// Invalid id
    #[error("Failed to parse id")]
    InvalidId(String),
    /// Invalid discport
    #[error("Failed to discport query: {0}")]
    Discport(ParseIntError),
}

impl FromStr for NodeRecord {
    type Err = NodeRecordParseError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        use url::{Host, Url};

        let url = Url::parse(s).map_err(|e| NodeRecordParseError::InvalidUrl(e.to_string()))?;

        let port = url
            .port()
            .ok_or_else(|| NodeRecordParseError::InvalidUrl("no port specified".to_string()))?;

        let address = match url.host() {
            Some(Host::Ipv4(ip)) => IpAddr::V4(ip),
            Some(Host::Ipv6(ip)) => IpAddr::V6(ip),
            Some(Host::Domain(dns)) => format!("{dns}:{port}")
                .to_socket_addrs()
                .map_err(|e| NodeRecordParseError::InvalidUrl(e.to_string()))?
                .next()
                .map(|addr| addr.ip())
                .ok_or_else(|| {
                    NodeRecordParseError::InvalidUrl(format!("no IP found for host: {url:?}"))
                })?,

            _ => return Err(NodeRecordParseError::InvalidUrl(format!("invalid host: {url:?}"))),
        };

        let udp_port = if let Some(discovery_port) = url
            .query_pairs()
            .find_map(|(maybe_disc, port)| (maybe_disc.as_ref() == "discport").then_some(port))
        {
            discovery_port.parse::<u16>().map_err(NodeRecordParseError::Discport)?
        } else {
            port
        };

        let id = url
            .username()
            .parse::<PeerId>()
            .map_err(|e| NodeRecordParseError::InvalidId(e.to_string()))?;

        Ok(Self { address, id, tcp_port: port, udp_port })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_url_parse_domain() {
        let url = "enode://6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0@localhost:30303?discport=30301";
        let node: NodeRecord = url.parse().unwrap();
        let localhost_socket_addr = "localhost:30303".to_socket_addrs().unwrap().next().unwrap();
        assert_eq!(node, NodeRecord {
            address: localhost_socket_addr.ip(),
            tcp_port: 30303,
            udp_port: 30301,
            id: "6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0".parse().unwrap(),
        })
    }

    #[test]
    fn test_url_parse() {
        let url = "enode://6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0@10.3.58.6:30303?discport=30301";
        let node: NodeRecord = url.parse().unwrap();
        assert_eq!(node, NodeRecord {
            address: IpAddr::V4([10,3,58,6].into()),
            tcp_port: 30303,
            udp_port: 30301,
            id: "6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0".parse().unwrap(),
        })
    }

    #[test]
    fn test_node_display() {
        let url = "enode://6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0@10.3.58.6:30303";
        let node: NodeRecord = url.parse().unwrap();
        assert_eq!(url, &format!("{node}"));
    }

    #[test]
    fn test_node_display_discport() {
        let url = "enode://6f8a80d14311c39f35f516fa664deaaaa13e85b2f7493f37f6144d86991ec012937307647bd3b9a82abe2974e1407241d54947bbb39763a4cac9f77166ad92a0@10.3.58.6:30303?discport=30301";
        let node: NodeRecord = url.parse().unwrap();
        assert_eq!(url, &format!("{node}"));
    }
}
