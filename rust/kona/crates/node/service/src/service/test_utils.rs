//! Configs for the service's tests.

use crate::{ChainSpec, NetworkConfig, RollupNodeConfig, RollupNodeEndpoints};
use alloy_rpc_types_engine::JwtSecret;
use kona_disc::LocalNode;
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::{ChainDependency, DependencySet};
use libp2p::{Multiaddr, multiaddr::Protocol};
use std::{
    collections::BTreeMap,
    net::{IpAddr, Ipv4Addr},
    sync::Arc,
};
use url::Url;

/// A JWT secret for tests.
pub(crate) const JWT_HEX: &str =
    "0xabababababababababababababababababababababababababababababababab";

/// A P2P config listening on localhost, on `tcp_port` for gossip and `udp_port` for discovery.
pub(crate) fn network_config(tcp_port: u16, udp_port: u16) -> NetworkConfig {
    let local_node = LocalNode::new(
        enr::k256::ecdsa::SigningKey::from_bytes(&[1u8; 32].into()).unwrap(),
        IpAddr::V4(Ipv4Addr::LOCALHOST),
        tcp_port,
        udp_port,
    );
    let mut gossip_address = Multiaddr::from(Ipv4Addr::LOCALHOST);
    gossip_address.push(Protocol::Tcp(tcp_port));
    let mut config = NetworkConfig::new(RollupConfig::default(), local_node, gossip_address);
    config.discovery_config = NetworkConfig::discv5_config(
        discv5::ListenConfig::Ipv4 { ip: Ipv4Addr::LOCALHOST, port: udp_port },
        false,
    );
    config
}

/// A config for a node that does not sequence, with the given P2P config.
#[allow(clippy::zero_sized_map_values)] // `DependencySet` is a map to `ChainDependency {}`.
pub(crate) fn node_config(p2p_config: NetworkConfig) -> RollupNodeConfig {
    RollupNodeConfig {
        version: "test".to_owned(),
        chain: ChainSpec {
            rollup_config: Arc::new(RollupConfig::default()),
            l1_config: Arc::new(L1ChainConfig::default()),
            dependency_set: Arc::new(DependencySet {
                dependencies: BTreeMap::from([(0, ChainDependency {})]),
                override_message_expiry_window: None,
            }),
        },
        p2p_config: Some(p2p_config),
        rpc_addr: Some((Ipv4Addr::LOCALHOST, 0).into()),
        sequencer: None,
    }
}

/// Endpoints nothing listens on, with the given derivation delegate.
pub(crate) fn endpoints(derivation_delegate: Option<Url>) -> RollupNodeEndpoints {
    RollupNodeEndpoints {
        l1_rpc: "http://127.0.0.1:1".parse().unwrap(),
        l1_beacon: "http://127.0.0.1:2".parse().unwrap(),
        l2_engine: "http://127.0.0.1:3".parse().unwrap(),
        l2_jwt_secret: JwtSecret::from_hex(JWT_HEX).unwrap(),
        derivation_delegate,
    }
}
