//! Checks [`RollupNode::start`](crate::RollupNode::start) makes before it starts the node's
//! actors.

use crate::NetworkConfig;
use backon::{ExponentialBuilder, Retryable};
use kona_engine::EngineClient;
use libp2p::multiaddr::Protocol;
use op_alloy_provider::ext::engine::OpEngineApi;
use std::net::{IpAddr, TcpListener, UdpSocket};

/// An error from the engine's JWT check.
#[derive(Debug, thiserror::Error)]
pub(super) enum JwtValidationError {
    /// The engine rejected the node's JWT secret.
    #[error("the engine rejected the JWT secret: it was given a different one")]
    InvalidSignature,
    /// Exchanging capabilities with the engine failed for another reason.
    #[error("failed to exchange capabilities with the engine: {0}")]
    CapabilityExchange(String),
}

/// Returns whether `error`, or an error it was caused by, is the engine rejecting the JWT.
fn is_jwt_signature_error(error: &(dyn std::error::Error + 'static)) -> bool {
    let mut source = Some(error);
    while let Some(err) = source {
        let err_str = err.to_string().to_lowercase();
        if err_str.contains("signature invalid") ||
            (err_str.contains("jwt") && err_str.contains("invalid")) ||
            err_str.contains("unauthorized") ||
            err_str.contains("authentication failed")
        {
            return true;
        }
        source = err.source();
    }
    false
}

/// Checks the engine's JWT secret by exchanging capabilities with the engine, which fails if the
/// engine was given a different secret. Other failures, such as an engine that is not up yet, are
/// retried with backon's default exponential backoff, up to three times.
pub(super) async fn validate_engine_jwt(engine: &EngineClient) -> Result<(), JwtValidationError> {
    let exchange = || async {
        match engine.exchange_capabilities(vec![]).await {
            Ok(_) => {
                debug!("Successfully exchanged capabilities with engine");
                Ok(())
            }
            Err(e) if is_jwt_signature_error(&e) => Err(JwtValidationError::InvalidSignature),
            Err(e) => Err(JwtValidationError::CapabilityExchange(e.to_string())),
        }
    };

    exchange
        .retry(ExponentialBuilder::default())
        .when(|e| !matches!(e, JwtValidationError::InvalidSignature))
        .notify(|_, duration| {
            debug!("Retrying engine capability handshake after {duration:?}");
        })
        .await
}

/// Checks that the P2P stack's gossip (TCP) and discovery (UDP) listen ports are free, so that a
/// port in use is reported before the node waits on its other dependencies.
///
/// Each port is checked on its own, and ports of `0`, which the OS chooses, are not checked.
pub(super) fn check_listen_ports(config: &NetworkConfig) -> Result<(), String> {
    let gossip_ip = config.gossip_address.iter().find_map(|protocol| match protocol {
        Protocol::Ip4(ip) => Some(IpAddr::V4(ip)),
        Protocol::Ip6(ip) => Some(IpAddr::V6(ip)),
        _ => None,
    });
    let gossip_port = config.gossip_address.iter().find_map(|protocol| match protocol {
        Protocol::Tcp(port) => Some(port),
        _ => None,
    });
    if let (Some(ip), Some(port)) = (gossip_ip, gossip_port) &&
        port != 0
    {
        TcpListener::bind((ip, port))
            .map_err(|e| format!("Error binding TCP socket on port {port}: {e}"))?;
    }

    let discovery: Vec<(IpAddr, u16)> = match &config.discovery_config.listen_config {
        discv5::ListenConfig::Ipv4 { ip, port } => vec![((*ip).into(), *port)],
        discv5::ListenConfig::Ipv6 { ip, port } => vec![((*ip).into(), *port)],
        discv5::ListenConfig::DualStack { ipv4, ipv4_port, ipv6, ipv6_port } => {
            vec![((*ipv4).into(), *ipv4_port), ((*ipv6).into(), *ipv6_port)]
        }
        // The sockets are already bound.
        discv5::ListenConfig::FromSockets { .. } => vec![],
    };
    for (ip, port) in discovery.into_iter().filter(|(_, port)| *port != 0) {
        UdpSocket::bind((ip, port))
            .map_err(|e| format!("Error binding UDP socket on port {port}: {e}"))?;
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        RollupNode, RollupNodeClients,
        service::test_utils::{JWT_HEX, network_config, node_config},
    };
    use alloy_provider::RootProvider;
    use alloy_rpc_types_engine::JwtSecret;
    use kona_genesis::RollupConfig;
    use std::{net::Ipv4Addr, sync::Arc};

    /// Returns a port that was free a moment ago.
    fn free_tcp_port() -> u16 {
        TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).unwrap().local_addr().unwrap().port()
    }

    #[test]
    fn free_ports_pass() {
        let udp_port =
            UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).unwrap().local_addr().unwrap().port();
        assert!(check_listen_ports(&network_config(free_tcp_port(), udp_port)).is_ok());
    }

    #[test]
    fn a_tcp_port_in_use_fails() {
        let taken = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).unwrap();
        let port = taken.local_addr().unwrap().port();
        let err = check_listen_ports(&network_config(port, 0)).unwrap_err();
        assert!(err.contains("TCP"), "{err}");
    }

    #[test]
    fn a_udp_port_in_use_fails() {
        let taken = UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).unwrap();
        let port = taken.local_addr().unwrap().port();
        let err = check_listen_ports(&network_config(0, port)).unwrap_err();
        assert!(err.contains("UDP"), "{err}");
    }

    #[tokio::test]
    async fn start_checks_the_listen_ports_first() {
        // The L1 and L2 endpoints are unreachable, so the port check must fail the start before
        // anything waits on them.
        let taken = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).unwrap();
        let port = taken.local_addr().unwrap().port();
        let config = node_config(network_config(port, 0), None);
        let clients = RollupNodeClients::connect(&config).unwrap();

        let err = RollupNode::new(config, clients).start().await.unwrap_err();

        assert!(err.contains("TCP"), "{err}");
    }

    #[tokio::test]
    async fn a_rejected_jwt_is_not_retried() {
        let server = httpmock::MockServer::start();
        let rejection = server.mock(|when, then| {
            when.method("POST");
            then.status(401).body("signature invalid");
        });
        let engine = EngineClient::new(
            RootProvider::new_http(server.url("/l1").parse().unwrap()),
            EngineClient::rpc_client(
                server.url("/").parse().unwrap(),
                JwtSecret::from_hex(JWT_HEX).unwrap(),
            ),
            Arc::new(RollupConfig::default()),
        );

        let err = validate_engine_jwt(&engine).await.unwrap_err();

        assert!(matches!(err, JwtValidationError::InvalidSignature), "{err}");
        rejection.assert_calls(1);
    }

    #[test]
    fn jwt_signature_errors_are_recognized() {
        let error = std::io::Error::other("signature invalid");
        assert!(is_jwt_signature_error(&error));
        let error = std::io::Error::other("network timeout");
        assert!(!is_jwt_signature_error(&error));
    }
}
