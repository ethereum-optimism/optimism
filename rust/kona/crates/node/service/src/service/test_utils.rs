//! Configs for the service's tests.

use crate::{
    DerivationDelegateConfig, EngineConfig, L1ClientConfig, NetworkConfig, NodeMode,
    RollupNodeConfig, SequencerConfig,
};
use alloy_rpc_types_engine::JwtSecret;
use kona_disc::LocalNode;
use kona_genesis::{L1ChainConfig, RollupConfig};
use libp2p::{Multiaddr, multiaddr::Protocol};
use std::{
    net::{IpAddr, Ipv4Addr},
    sync::Arc,
};

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

/// A node config whose L1 and L2 endpoints nothing listens on, with the given P2P config.
pub(crate) fn node_config(
    p2p_config: NetworkConfig,
    derivation_delegate_config: Option<DerivationDelegateConfig>,
) -> RollupNodeConfig {
    RollupNodeConfig {
        version: "test".to_owned(),
        rollup_config: Arc::new(RollupConfig::default()),
        l1_client: L1ClientConfig {
            chain_config: Arc::new(L1ChainConfig::default()),
            beacon: "http://127.0.0.1:1".parse().unwrap(),
            rpc_url: "http://127.0.0.1:2".parse().unwrap(),
        },
        engine_config: EngineConfig {
            l2_url: "http://127.0.0.1:3".parse().unwrap(),
            l2_jwt_secret: JwtSecret::from_hex(JWT_HEX).unwrap(),
            mode: NodeMode::Validator,
        },
        p2p_config,
        rpc_config: None,
        sequencer_config: SequencerConfig::default(),
        derivation_delegate_config,
        dependency_set: None,
    }
}
