//! Contains a builder for the discovery service.

use discv5::{Config, Discv5, Enr, ListenConfig};
use enr::k256;
use kona_peers::{BootNodes, BootStoreFile, OpStackEnr};
use socket2::{Domain, Protocol, Socket, Type};
use std::{
    net::{IpAddr, SocketAddr},
    sync::Arc,
};
use tokio::time::Duration;

use crate::{Discv5BuilderError, Discv5Driver};

#[derive(Debug, Clone, PartialEq, Eq)]
/// The local node information exposed by the discovery service to the network.
pub struct LocalNode {
    /// The keypair to use for the local node. Should match the keypair used by the
    /// gossip service.
    pub signing_key: k256::ecdsa::SigningKey,
    /// The IP address to advertise.
    pub ip: IpAddr,
    /// The TCP port to advertise.
    pub tcp_port: u16,
    /// Fallback UDP port.
    pub udp_port: u16,
}

impl From<&LocalNode> for discv5::ListenConfig {
    fn from(local_node: &LocalNode) -> Self {
        match local_node.ip {
            IpAddr::V4(ip) => Self::Ipv4 { ip, port: local_node.tcp_port },
            IpAddr::V6(ip) => Self::Ipv6 { ip, port: local_node.tcp_port },
        }
    }
}

impl LocalNode {
    /// Creates a new [`LocalNode`] instance.
    pub const fn new(
        signing_key: k256::ecdsa::SigningKey,
        ip: IpAddr,
        tcp_port: u16,
        udp_port: u16,
    ) -> Self {
        Self { signing_key, ip, tcp_port, udp_port }
    }
}

impl LocalNode {
    /// Build the local node ENR. This should contain the information we wish to
    /// broadcast to the other nodes in the network. See
    /// [the op-node implementation](https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-node/p2p/discovery.go#L61-L97)
    /// for the go equivalent
    fn build_enr(self, chain_id: u64) -> Result<Enr, enr::Error> {
        let opstack = OpStackEnr::from_chain_id(chain_id);
        let mut opstack_data = Vec::new();
        use alloy_rlp::Encodable;
        opstack.encode(&mut opstack_data);

        let mut enr_builder = Enr::builder();
        enr_builder.add_value_rlp(OpStackEnr::OP_CL_KEY, opstack_data.into());
        match self.ip {
            IpAddr::V4(addr) => {
                enr_builder.ip4(addr).tcp4(self.tcp_port).udp4(self.udp_port);
            }
            IpAddr::V6(addr) => {
                enr_builder.ip6(addr).tcp6(self.tcp_port).udp6(self.udp_port);
            }
        }

        enr_builder.build(&self.signing_key.into())
    }
}

/// Binds a free UDP port for discovery to listen on, if its listen port is 0.
///
/// discv5 binds its socket itself and never reports the port it gets, so the port could not be
/// advertised. The socket bound here is passed to discv5 in [`ListenConfig::FromSockets`]
/// instead. A port picked that way cannot match an advertised port set in advance.
///
/// Binding a socket needs a Tokio runtime, so a listen port of 0 can only be built inside one.
fn bind_free_listen_port(
    config: &mut Config,
    local_node: &LocalNode,
) -> Result<(), Discv5BuilderError> {
    let ip = match config.listen_config {
        ListenConfig::Ipv4 { ip, port: 0 } => IpAddr::V4(ip),
        ListenConfig::Ipv6 { ip, port: 0 } => IpAddr::V6(ip),
        _ => return Ok(()),
    };
    if local_node.udp_port != 0 {
        return Err(Discv5BuilderError::AdvertisedUdpPortWithoutFixedListenPort);
    }
    config.listen_config = bind_free_port(ip)?;
    Ok(())
}

/// Binds a UDP socket to a free port at `ip` the way discv5 binds its own, and returns the listen
/// config that passes it to discv5.
fn bind_free_port(ip: IpAddr) -> Result<ListenConfig, Discv5BuilderError> {
    // `from_std` registers the socket with the runtime's reactor, and panics outside a runtime.
    tokio::runtime::Handle::try_current()
        .map_err(std::io::Error::other)
        .map_err(Discv5BuilderError::BindFailed)?;
    let socket = match ip {
        IpAddr::V4(_) => std::net::UdpSocket::bind((ip, 0)),
        // Like discv5, keep an IPv6 socket from also receiving IPv4.
        IpAddr::V6(_) => {
            Socket::new(Domain::IPV6, Type::DGRAM, Some(Protocol::UDP)).and_then(|socket| {
                socket.set_only_v6(true)?;
                socket.bind(&SocketAddr::new(ip, 0).into())?;
                Ok(socket.into())
            })
        }
    }
    .and_then(|socket| socket.set_nonblocking(true).map(|()| socket))
    .and_then(tokio::net::UdpSocket::from_std)
    .map_err(Discv5BuilderError::BindFailed)?;
    let socket = Some(Arc::new(socket));
    Ok(if ip.is_ipv4() {
        ListenConfig::FromSockets { ipv4: socket, ipv6: None }
    } else {
        ListenConfig::FromSockets { ipv4: None, ipv6: socket }
    })
}

/// Returns the UDP port to advertise in the ENR: the one `local_node` has, if it is not 0, and
/// otherwise the port discovery listens on.
fn advertised_udp_port(config: &Config, local_node: &LocalNode) -> Result<u16, Discv5BuilderError> {
    if local_node.udp_port != 0 {
        return Ok(local_node.udp_port);
    }
    let listen_port = match &config.listen_config {
        ListenConfig::Ipv4 { port, .. } | ListenConfig::Ipv6 { port, .. } => *port,
        ListenConfig::DualStack { ipv4_port, ipv6_port, .. } => {
            if local_node.ip.is_ipv4() {
                *ipv4_port
            } else {
                *ipv6_port
            }
        }
        ListenConfig::FromSockets { ipv4, ipv6 } => {
            let socket = if local_node.ip.is_ipv4() {
                ipv4.as_ref().or(ipv6.as_ref())
            } else {
                ipv6.as_ref().or(ipv4.as_ref())
            };
            socket
                .map(|socket| socket.local_addr().map(|addr| addr.port()))
                .transpose()
                .map_err(Discv5BuilderError::BindFailed)?
                .unwrap_or(0)
        }
    };
    // Only a dual-stack listen port of 0 is left unbound.
    if listen_port == 0 {
        return Err(Discv5BuilderError::UdpPortZero);
    }
    Ok(listen_port)
}

/// Discovery service builder.
#[derive(Debug, Clone)]
pub struct Discv5Builder {
    /// The node information advertised by the discovery service.
    local_node: LocalNode,
    /// The chain ID of the network.
    chain_id: u64,
    /// The discovery config for the discovery service.
    discovery_config: Config,
    /// The interval to find peers.
    interval: Option<Duration>,
    /// The interval to randomize discovery peers.
    randomize: Option<Duration>,
    /// An optional path to the bootstore.
    bootstore: Option<BootStoreFile>,
    /// Additional bootnodes to manually add to the initial bootstore
    bootnodes: BootNodes,
    /// The interval to store the bootnodes to disk.
    store_interval: Option<Duration>,
    /// Whether or not to forward the initial set of valid ENRs to the gossip layer.
    forward: bool,
}

impl Discv5Builder {
    /// Creates a new [`Discv5Builder`] instance.
    pub fn new(local_node: LocalNode, chain_id: u64, discovery_config: Config) -> Self {
        Self {
            local_node,
            chain_id,
            discovery_config,
            interval: None,
            randomize: None,
            bootstore: None,
            bootnodes: BootNodes::default(),
            store_interval: None,
            forward: true,
        }
    }

    /// Sets a bootstore file.
    pub fn with_bootstore_file(mut self, bootstore: Option<BootStoreFile>) -> Self {
        self.bootstore = bootstore;
        self
    }

    /// Sets the initial bootnodes to add to the bootstore.
    pub fn with_bootnodes(mut self, bootnodes: BootNodes) -> Self {
        self.bootnodes = bootnodes;
        self
    }

    /// Sets the interval to store the bootnodes to disk.
    pub const fn with_store_interval(mut self, store_interval: Duration) -> Self {
        self.store_interval = Some(store_interval);
        self
    }

    /// Sets the discovery service advertised local node information.
    pub fn with_local_node(mut self, local_node: LocalNode) -> Self {
        self.local_node = local_node;
        self
    }

    /// Sets the chain ID of the network.
    pub const fn with_chain_id(mut self, chain_id: u64) -> Self {
        self.chain_id = chain_id;
        self
    }

    /// Sets the interval to find peers.
    pub const fn with_interval(mut self, interval: Duration) -> Self {
        self.interval = Some(interval);
        self
    }

    /// Sets the discovery config for the discovery service.
    pub fn with_discovery_config(mut self, config: Config) -> Self {
        self.discovery_config = config;
        self
    }

    /// Sets the interval to randomize discovery peers.
    pub const fn with_discovery_randomize(mut self, interval: Option<Duration>) -> Self {
        self.randomize = interval;
        self
    }

    /// Disables forwarding of the initial set of valid ENRs to the gossip layer.
    pub const fn disable_forward(mut self) -> Self {
        self.forward = false;
        self
    }

    /// Builds a [`Discv5Driver`].
    pub fn build(self) -> Result<Discv5Driver, Discv5BuilderError> {
        let chain_id = self.chain_id;

        let mut config = self.discovery_config;

        let mut local_node = self.local_node;
        let key = local_node.signing_key.clone();

        bind_free_listen_port(&mut config, &local_node)?;
        local_node.udp_port = advertised_udp_port(&config, &local_node)?;
        let enr = local_node.build_enr(chain_id).map_err(Discv5BuilderError::EnrBuildFailed)?;

        let interval = self.interval.unwrap_or(Duration::from_secs(5));
        let disc = Discv5::new(enr, key.into(), config)
            .map_err(|e| Discv5BuilderError::Discv5CreationFailed(e.to_string()))?;
        let mut driver =
            Discv5Driver::new(disc, interval, chain_id, self.bootstore, self.bootnodes)
                .map_err(|e| Discv5BuilderError::Discv5CreationFailed(e.to_string()))?;
        driver.store_interval = self.store_interval.unwrap_or(Duration::from_secs(60));
        driver.forward = self.forward;
        driver.remove_interval = self.randomize;
        Ok(driver)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use discv5::ConfigBuilder;
    use enr::CombinedKey;
    use kona_peers::EnrValidation;
    use std::net::{IpAddr, Ipv4Addr};

    #[test]
    fn test_builds_valid_enr() {
        let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let addr = LocalNode::new(k256_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099, 9099);
        let mut builder = Discv5Builder::new(
            addr,
            10,
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::UNSPECIFIED, port: 9099 })
                .build(),
        );
        builder = builder.with_discovery_config(
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::UNSPECIFIED, port: 9099 })
                .build(),
        );
        let driver = builder.build().unwrap();
        let enr = driver.disc.local_enr();
        assert!(EnrValidation::validate(&enr, 10).is_valid());
    }

    #[test]
    fn test_advertised_udp_port_falls_back_to_listen_port() {
        let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let addr = LocalNode::new(k256_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099, 0);
        let builder = Discv5Builder::new(
            addr,
            10,
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::UNSPECIFIED, port: 9099 })
                .build(),
        );
        let driver = builder.build().unwrap();
        assert_eq!(driver.disc.local_enr().udp4(), Some(9099));
    }

    #[tokio::test]
    async fn test_listen_udp_port_zero_binds_and_advertises_a_free_port() {
        let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let addr = LocalNode::new(k256_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 9099, 0);
        let builder = Discv5Builder::new(
            addr,
            10,
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::LOCALHOST, port: 0 }).build(),
        );
        let driver = builder.build().unwrap();

        let port = driver.disc.local_enr().udp4().expect("ENR has a UDP port");
        assert_ne!(port, 0);
        // The advertised port is the one bound for discovery, so it cannot be bound again.
        assert!(std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, port)).is_err());
    }

    #[test]
    fn test_listen_udp_port_zero_needs_a_runtime() {
        let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let addr = LocalNode::new(k256_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 9099, 0);
        let builder = Discv5Builder::new(
            addr,
            10,
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::LOCALHOST, port: 0 }).build(),
        );
        assert!(matches!(builder.build(), Err(Discv5BuilderError::BindFailed(_))));
    }

    /// A free IPv6 port is bound the way discv5 binds its own IPv6 socket, which receives only
    /// IPv6. Skipped where IPv6 is not available.
    #[tokio::test]
    async fn test_bind_free_port_ipv6_only() {
        let config = match bind_free_port(IpAddr::V6(std::net::Ipv6Addr::LOCALHOST)) {
            Ok(config) => config,
            Err(err) => {
                eprintln!("skipping, cannot bind IPv6: {err}");
                return;
            }
        };
        let ListenConfig::FromSockets { ipv4: None, ipv6: Some(socket) } = config else {
            panic!("expected a bound IPv6 socket");
        };
        assert_ne!(socket.local_addr().unwrap().port(), 0);
        assert!(socket2::SockRef::from(&*socket).only_v6().unwrap());
    }

    #[test]
    fn test_rejects_advertised_udp_port_without_fixed_listen_port() {
        let CombinedKey::Secp256k1(k256_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        // The port bound for a listen port of 0 cannot match an advertised port set in advance.
        let addr = LocalNode::new(k256_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099, 9099);
        let builder = Discv5Builder::new(
            addr,
            10,
            ConfigBuilder::new(ListenConfig::Ipv4 { ip: Ipv4Addr::UNSPECIFIED, port: 0 }).build(),
        );
        assert!(matches!(
            builder.build(),
            Err(Discv5BuilderError::AdvertisedUdpPortWithoutFixedListenPort)
        ));
    }
}
