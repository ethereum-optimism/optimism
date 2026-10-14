//! Checks [`RollupNode::start`](crate::RollupNode::start) makes before it starts the node's
//! actors.

use super::error::{ListenProtocol, RollupNodeError};
use crate::NetworkConfig;
use backon::{ExponentialBuilder, Retryable};
use kona_sources::EngineApi;
use libp2p::multiaddr::Protocol;
use std::net::{IpAddr, SocketAddr, TcpListener, UdpSocket};

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
/// engine rejects the node's JWT. Other failures, such as an engine that is not up yet, are retried
/// with `backoff`; [`RollupNode::start`](crate::RollupNode::start) uses backon's default
/// exponential backoff, which retries up to three times.
pub(super) async fn validate_engine_jwt(
    engine: &EngineApi,
    backoff: ExponentialBuilder,
) -> Result<(), RollupNodeError> {
    let exchange = || async {
        match engine.exchange_capabilities(vec![]).await {
            Ok(_) => {
                debug!("Successfully exchanged capabilities with engine");
                Ok(())
            }
            Err(e) if is_jwt_signature_error(&e) => Err(RollupNodeError::JwtRejected),
            Err(e) => Err(RollupNodeError::EngineCapabilityExchange(e)),
        }
    };

    exchange
        .retry(backoff)
        .when(|e| !matches!(e, RollupNodeError::JwtRejected))
        .notify(|_, duration| {
            debug!("Retrying engine capability handshake after {duration:?}");
        })
        .await
}

/// Checks that the P2P stack's gossip (TCP) and discovery (UDP) listen addresses can be bound, so
/// that an unusable address is reported before the node waits on its other dependencies.
///
/// Each address is checked on its own, and ports of `0`, which the OS chooses, are not checked.
pub(super) fn check_listen_ports(config: &NetworkConfig) -> Result<(), RollupNodeError> {
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
        let addr = SocketAddr::new(ip, port);
        TcpListener::bind(addr).map_err(|source| RollupNodeError::ListenAddressUnavailable {
            protocol: ListenProtocol::Tcp,
            addr,
            source,
        })?;
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
        let addr = SocketAddr::new(ip, port);
        UdpSocket::bind(addr).map_err(|source| RollupNodeError::ListenAddressUnavailable {
            protocol: ListenProtocol::Udp,
            addr,
            source,
        })?;
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        RollupNode, RollupNodeClients,
        service::test_utils::{JWT_HEX, endpoints, network_config, node_config},
    };
    use alloy_rpc_types_engine::JwtSecret;
    use kona_genesis::RollupConfig;
    use std::{net::Ipv4Addr, sync::Arc, time::Duration};

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
        assert!(
            matches!(
                err,
                RollupNodeError::ListenAddressUnavailable { protocol: ListenProtocol::Tcp, addr, .. }
                    if addr.port() == port
            ),
            "{err}"
        );
    }

    #[test]
    fn a_udp_port_in_use_fails() {
        let taken = UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).unwrap();
        let port = taken.local_addr().unwrap().port();
        let err = check_listen_ports(&network_config(0, port)).unwrap_err();
        assert!(
            matches!(
                err,
                RollupNodeError::ListenAddressUnavailable { protocol: ListenProtocol::Udp, addr, .. }
                    if addr.port() == port
            ),
            "{err}"
        );
    }

    #[tokio::test]
    async fn start_checks_the_listen_ports_first() {
        // The L1 and L2 endpoints are unreachable, so the port check must fail the start before
        // anything waits on them.
        let taken = TcpListener::bind((Ipv4Addr::LOCALHOST, 0)).unwrap();
        let port = taken.local_addr().unwrap().port();
        let config = node_config(network_config(port, 0));
        let clients = RollupNodeClients::connect(&endpoints(None)).unwrap();

        let err = RollupNode::new(config, clients).start().await.unwrap_err();

        assert!(
            matches!(
                err,
                RollupNodeError::ListenAddressUnavailable { protocol: ListenProtocol::Tcp, .. }
            ),
            "{err}"
        );
    }

    /// An Engine API client whose Engine API is `server`.
    fn engine(server: &httpmock::MockServer) -> EngineApi {
        EngineApi::new(
            EngineApi::rpc_client(
                server.url("/").parse().unwrap(),
                JwtSecret::from_hex(JWT_HEX).unwrap(),
            ),
            Arc::new(RollupConfig::default()),
        )
    }

    /// A backoff that retries twice without waiting long.
    fn fast_backoff() -> ExponentialBuilder {
        ExponentialBuilder::default().with_min_delay(Duration::from_millis(1)).with_max_times(2)
    }

    #[tokio::test]
    async fn a_rejected_jwt_is_not_retried() {
        let server = httpmock::MockServer::start();
        let rejection = server.mock(|when, then| {
            when.method("POST");
            then.status(401).body("signature invalid");
        });

        let err = validate_engine_jwt(&engine(&server), fast_backoff()).await.unwrap_err();

        assert!(matches!(err, RollupNodeError::JwtRejected), "{err}");
        rejection.assert_calls(1);
    }

    #[tokio::test]
    async fn other_engine_failures_are_retried() {
        let server = httpmock::MockServer::start();
        let failure = server.mock(|when, then| {
            when.method("POST");
            then.status(500).body("internal error");
        });

        let err = validate_engine_jwt(&engine(&server), fast_backoff()).await.unwrap_err();

        assert!(matches!(err, RollupNodeError::EngineCapabilityExchange(_)), "{err}");
        failure.assert_calls(3);
    }

    #[test]
    fn messages_leave_the_cause_to_the_source() {
        let source = std::io::Error::other("address in use");
        let err = RollupNodeError::ListenAddressUnavailable {
            protocol: ListenProtocol::Tcp,
            addr: SocketAddr::new(Ipv4Addr::LOCALHOST.into(), 9222),
            source,
        };

        assert_eq!(err.to_string(), "failed to bind the P2P TCP listen address 127.0.0.1:9222");
        assert_eq!(std::error::Error::source(&err).unwrap().to_string(), "address in use");
    }

    #[test]
    fn jwt_signature_errors_are_recognized() {
        let error = std::io::Error::other("signature invalid");
        assert!(is_jwt_signature_error(&error));
        let error = std::io::Error::other("network timeout");
        assert!(!is_jwt_signature_error(&error));
    }
}
