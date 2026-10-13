//! Configs for the service's tests.

use crate::{NetworkConfig, NodeMode, RollupNodeConfig, RollupNodeEndpoints, SequencerConfig};
use alloy_rpc_types_engine::JwtSecret;
use kona_disc::LocalNode;
use kona_genesis::{L1ChainConfig, RollupConfig};
use libp2p::{Multiaddr, multiaddr::Protocol};
use std::{
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

/// A validator's node config with the given P2P config.
pub(crate) fn node_config(p2p_config: NetworkConfig) -> RollupNodeConfig {
    RollupNodeConfig {
        version: "test".to_owned(),
        rollup_config: Arc::new(RollupConfig::default()),
        l1_chain_config: Arc::new(L1ChainConfig::default()),
        mode: NodeMode::Validator,
        p2p_config,
        rpc_config: None,
        sequencer_config: SequencerConfig::default(),
        dependency_set: None,
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
