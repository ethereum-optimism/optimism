//! Contains a builder for the discovery service.

use discv5::{Config, Discv5, Enr, enr::k256};
use kona_peers::{BootNodes, BootStoreFile, OpStackEnr};
use std::net::IpAddr;
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
    fn build_enr(self, chain_id: u64) -> Result<Enr, discv5::enr::Error> {
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

        let config = self.discovery_config;

        let local_node = self.local_node;
        let key = local_node.signing_key.clone();

        let enr = local_node.build_enr(chain_id).map_err(|_| Discv5BuilderError::EnrBuildFailed)?;

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
    use discv5::{ConfigBuilder, ListenConfig, enr::CombinedKey};
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
}
