//! Tests for the local record the driver advertises.

use super::*;
use rstest::rstest;

/// Only an unset TCP port of 0 is set to the gossip TCP port, and only once: another port is
/// kept, and so is the first gossip port. It fails if the gossip address has no TCP port to set
/// it to, as with the memory transport. An IPv6 node's port is `tcp6`.
#[rstest]
#[case::unset(Ipv4Addr::LOCALHOST.into(), 0, "/ip4/127.0.0.1/tcp/9222", Some(9222))]
#[case::unset_ipv6(Ipv6Addr::LOCALHOST.into(), 0, "/ip6/::1/tcp/9222", Some(9222))]
#[case::set(Ipv4Addr::LOCALHOST.into(), 30303, "/ip4/127.0.0.1/tcp/9222", Some(30303))]
#[case::set_without_gossip_port(Ipv4Addr::LOCALHOST.into(), 30303, "/memory/0", Some(30303))]
#[case::unset_without_gossip_port(Ipv4Addr::LOCALHOST.into(), 0, "/memory/0", None)]
#[tokio::test]
async fn test_advertise_gossip_tcp_port(
    #[case] ip: IpAddr,
    #[case] tcp_port: u16,
    #[case] gossip_addr: Multiaddr,
    #[case] advertised: Option<u16>,
) {
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    // A fixed listen port, so that building binds no socket, which IPv6 would need the host to
    // support.
    let driver = Discv5Driver::builder(
        LocalNode::new(secret_key, ip, tcp_port, 0),
        TEST_CHAIN_ID,
        ConfigBuilder::new(ListenConfig::from_ip(ip, 30303)).build(),
    )
    .build()
    .unwrap();
    let enr_tcp_port = || {
        let enr = driver.disc.local_enr();
        if ip.is_ipv4() { enr.tcp4() } else { enr.tcp6() }
    };

    let result = driver.advertise_gossip_tcp_port(&gossip_addr);

    match advertised {
        Some(port) => {
            result.unwrap();
            assert_eq!(enr_tcp_port(), Some(port));
            // A second address changes nothing.
            driver.advertise_gossip_tcp_port(&"/ip4/127.0.0.1/tcp/1".parse().unwrap()).unwrap();
            assert_eq!(enr_tcp_port(), Some(port));
        }
        None => {
            assert!(matches!(result, Err(AdvertiseTcpPortError::NoGossipTcpPort(_))));
        }
    }
}
