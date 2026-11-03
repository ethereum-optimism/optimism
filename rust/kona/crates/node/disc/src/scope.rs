//! Address scopes, for deciding whether an address another node relayed is worth dialing.
//!
//! Ported from geth's `p2p/netutil/net.go`, with its special-purpose lists brought up to date with
//! the IANA registries.

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

use ipnet::{Ipv4Net, Ipv6Net};

use crate::RelayAddrError;

/// The IPv4 blocks of the [IANA registry] that no peer can listen on, as of its 2025-10-09 update.
/// geth's list is a subset of it.
///
/// This includes anycast blocks, which are globally reachable but name no single host. The
/// private-use, link-local and shared blocks are LAN instead, and loopback has its own scope.
///
/// [IANA registry]: https://www.iana.org/assignments/iana-ipv4-special-registry/
const SPECIAL_V4: [Ipv4Net; 11] = [
    // "This" network.
    Ipv4Net::new_assert(Ipv4Addr::new(0, 0, 0, 0), 8),
    // IETF Protocol Assignments: Service Continuity, dummy address, PCP and TURN anycast,
    // NAT64/DNS64 discovery, and the unassigned rest.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 0, 0, 0), 24),
    // TEST-NET-1.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 0, 2, 0), 24),
    // AS112.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 31, 196, 0), 24),
    // AMT.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 52, 193, 0), 24),
    // Deprecated 6to4 Relay Anycast, and 6a44 relay anycast.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 88, 99, 0), 24),
    // Direct Delegation AS112.
    Ipv4Net::new_assert(Ipv4Addr::new(192, 175, 48, 0), 24),
    // Benchmarking.
    Ipv4Net::new_assert(Ipv4Addr::new(198, 18, 0, 0), 15),
    // TEST-NET-2.
    Ipv4Net::new_assert(Ipv4Addr::new(198, 51, 100, 0), 24),
    // TEST-NET-3.
    Ipv4Net::new_assert(Ipv4Addr::new(203, 0, 113, 0), 24),
    // Reserved, including Limited Broadcast.
    Ipv4Net::new_assert(Ipv4Addr::new(240, 0, 0, 0), 4),
];

/// Shared Address Space, for carrier-grade NAT (RFC 6598), also used by overlay networks such as
/// Tailscale. Nodes behind the same NAT or on the same overlay can reach each other, so it counts
/// as LAN.
const SHARED_V4: Ipv4Net = Ipv4Net::new_assert(Ipv4Addr::new(100, 64, 0, 0), 10);

/// The IPv6 blocks of the [IANA registry] that no peer can listen on, as of its 2025-10-09 update.
/// geth's list is a subset of it.
///
/// As in [`SPECIAL_V4`], this includes anycast blocks. The unique-local and link-local blocks are
/// LAN instead, and loopback has its own scope. The IPv4-IPv6 translation blocks are left public:
/// their scope is that of the IPv4 address in them, which they don't map to as IPv4-mapped
/// addresses do.
///
/// [IANA registry]: https://www.iana.org/assignments/iana-ipv6-special-registry/
const SPECIAL_V6: [Ipv6Net; 8] = [
    // Discard-Only.
    Ipv6Net::new_assert(Ipv6Addr::new(0x100, 0, 0, 0, 0, 0, 0, 0), 64),
    // Dummy prefix.
    Ipv6Net::new_assert(Ipv6Addr::new(0x100, 0, 0, 1, 0, 0, 0, 0), 64),
    // IETF Protocol Assignments: TEREDO, PCP, TURN and DNS-SD SRP anycast, benchmarking, AMT,
    // AS112, ORCHID, ORCHIDv2 and drone remote IDs.
    Ipv6Net::new_assert(Ipv6Addr::new(0x2001, 0, 0, 0, 0, 0, 0, 0), 23),
    // Documentation.
    Ipv6Net::new_assert(Ipv6Addr::new(0x2001, 0xdb8, 0, 0, 0, 0, 0, 0), 32),
    // 6to4.
    Ipv6Net::new_assert(Ipv6Addr::new(0x2002, 0, 0, 0, 0, 0, 0, 0), 16),
    // Direct Delegation AS112.
    Ipv6Net::new_assert(Ipv6Addr::new(0x2620, 0x4f, 0x8000, 0, 0, 0, 0, 0), 48),
    // Documentation.
    Ipv6Net::new_assert(Ipv6Addr::new(0x3fff, 0, 0, 0, 0, 0, 0, 0), 20),
    // Segment Routing (SRv6) SIDs.
    Ipv6Net::new_assert(Ipv6Addr::new(0x5f00, 0, 0, 0, 0, 0, 0, 0), 16),
];

/// Where an IP address can be reached from.
///
/// An IPv4-mapped IPv6 address, such as `::ffff:10.0.0.1`, has the scope of the IPv4 address in
/// it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AddrScope {
    /// `0.0.0.0` or `::`, which names no host.
    Unspecified,
    /// A multicast, broadcast, documentation or other special-purpose address, which names no
    /// host to dial.
    Special,
    /// A loopback address, reachable only from the same host.
    Loopback,
    /// A private, link-local or shared (carrier-grade NAT) address, reachable only from the same
    /// local network.
    Lan,
    /// Any other address.
    Public,
}

impl AddrScope {
    /// Returns the scope of `ip`.
    pub fn of(ip: IpAddr) -> Self {
        match ip.to_canonical() {
            IpAddr::V4(ip) => Self::of_v4(ip),
            IpAddr::V6(ip) => Self::of_v6(ip),
        }
    }

    fn of_v4(ip: Ipv4Addr) -> Self {
        if ip.is_unspecified() {
            Self::Unspecified
        } else if ip.is_multicast() || SPECIAL_V4.iter().any(|net| net.contains(&ip)) {
            Self::Special
        } else if ip.is_loopback() {
            Self::Loopback
        } else if ip.is_private() || ip.is_link_local() || SHARED_V4.contains(&ip) {
            Self::Lan
        } else {
            Self::Public
        }
    }

    fn of_v6(ip: Ipv6Addr) -> Self {
        if ip.is_unspecified() {
            Self::Unspecified
        } else if ip.is_multicast() || SPECIAL_V6.iter().any(|net| net.contains(&ip)) {
            Self::Special
        } else if ip.is_loopback() {
            Self::Loopback
        } else if ip.is_unique_local() || ip.is_unicast_link_local() {
            Self::Lan
        } else {
            Self::Public
        }
    }
}

/// Checks whether `addr`, relayed by a node of scope `sender`, is worth dialing, as geth's
/// `CheckRelayAddr` does:
///
/// - an unspecified or special-purpose address never is;
/// - a loopback address is only from a loopback sender;
/// - a LAN address is only from a loopback or LAN sender;
/// - a public address always is.
///
/// An unspecified or special-purpose sender counts as public.
pub fn check_relay_addr(sender: AddrScope, addr: IpAddr) -> Result<(), RelayAddrError> {
    match AddrScope::of(addr) {
        AddrScope::Unspecified => Err(RelayAddrError::Unspecified),
        AddrScope::Special => Err(RelayAddrError::Special),
        AddrScope::Loopback if sender != AddrScope::Loopback => Err(RelayAddrError::Loopback),
        AddrScope::Lan if !matches!(sender, AddrScope::Loopback | AddrScope::Lan) => {
            Err(RelayAddrError::Lan)
        }
        _ => Ok(()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    /// geth's `TestIsLAN` and `TestIsSpecialNetwork` inputs, plus IPv6 cases and the IANA blocks
    /// geth's lists miss.
    #[rstest]
    // LAN in geth.
    #[case::loopback_v4("127.0.0.1", AddrScope::Loopback)]
    #[case::private_10_a("10.0.1.1", AddrScope::Lan)]
    #[case::private_10_b("10.22.0.3", AddrScope::Lan)]
    #[case::private_172("172.31.252.251", AddrScope::Lan)]
    #[case::private_192("192.168.1.4", AddrScope::Lan)]
    #[case::link_local_v6("fe80::f4a1:8eff:fec5:9d9d", AddrScope::Lan)]
    #[case::link_local_v6_upper("febf::ab32:2233", AddrScope::Lan)]
    #[case::unique_local_v6("fc00::4", AddrScope::Lan)]
    #[case::mapped_loopback("::ffff:127.0.0.1", AddrScope::Loopback)]
    #[case::mapped_private("::ffff:10.10.0.2", AddrScope::Lan)]
    // Not LAN in geth.
    #[case::test_net_1("192.0.2.1", AddrScope::Special)]
    #[case::public_v4("1.0.0.0", AddrScope::Public)]
    #[case::past_private_172("172.32.0.1", AddrScope::Public)]
    #[case::site_local_v6("fec0::2233", AddrScope::Public)]
    #[case::mapped_public("::ffff:88.99.100.2", AddrScope::Public)]
    // Special in geth.
    #[case::unspecified_v4("0.0.0.0", AddrScope::Unspecified)]
    #[case::this_network("0.2.0.8", AddrScope::Special)]
    #[case::test_net_1_other("192.0.2.44", AddrScope::Special)]
    #[case::documentation_v6("2001:db8:85a3:8d3:1319:8a2e:370:7348", AddrScope::Special)]
    #[case::broadcast("255.255.255.255", AddrScope::Special)]
    #[case::multicast_v4("224.0.0.22", AddrScope::Special)]
    #[case::multicast_v6("ff05::1:3", AddrScope::Special)]
    #[case::mapped_broadcast("::ffff:255.255.255.255", AddrScope::Special)]
    #[case::mapped_test_net_1("::ffff:192.0.2.1", AddrScope::Special)]
    // Not special in geth.
    #[case::past_test_net_1("192.0.3.1", AddrScope::Public)]
    // Beyond geth's tests.
    #[case::unspecified_v6("::", AddrScope::Unspecified)]
    #[case::mapped_unspecified("::ffff:0.0.0.0", AddrScope::Unspecified)]
    #[case::loopback_v6("::1", AddrScope::Loopback)]
    #[case::loopback_v4_other("127.0.2.19", AddrScope::Loopback)]
    #[case::link_local_v4("169.254.1.1", AddrScope::Lan)]
    #[case::six_to_four("2002::1", AddrScope::Special)]
    #[case::public_v6("2a00:1450:4001::1", AddrScope::Public)]
    // Beyond geth's lists.
    #[case::shared_first("100.64.0.0", AddrScope::Lan)]
    #[case::shared_last("100.127.255.255", AddrScope::Lan)]
    #[case::past_shared("100.128.0.0", AddrScope::Public)]
    #[case::dummy_v4("192.0.0.8", AddrScope::Special)]
    #[case::turn_anycast_v4("192.0.0.10", AddrScope::Special)]
    #[case::ietf_unassigned_v4("192.0.0.200", AddrScope::Special)]
    #[case::reserved("240.0.0.1", AddrScope::Special)]
    #[case::dummy_v6("100:0:0:1::1", AddrScope::Special)]
    #[case::turn_anycast_v6("2001:1::2", AddrScope::Special)]
    #[case::drone_ids("2001:30::1", AddrScope::Special)]
    #[case::past_ietf_v6("2001:200::1", AddrScope::Public)]
    #[case::as112_direct_v6("2620:4f:8000::1", AddrScope::Special)]
    #[case::documentation_v6_new("3fff::1", AddrScope::Special)]
    #[case::srv6("5f00::1", AddrScope::Special)]
    #[case::nat64("64:ff9b::808:808", AddrScope::Public)]
    fn test_addr_scope(#[case] ip: IpAddr, #[case] expected: AddrScope) {
        assert_eq!(AddrScope::of(ip), expected);
    }

    /// geth's `TestCheckRelayIP` table, plus IPv6 cases.
    #[rstest]
    #[case::unspecified_from_loopback("127.0.0.1", "0.0.0.0", Err(RelayAddrError::Unspecified))]
    #[case::unspecified_from_lan("192.168.0.1", "0.0.0.0", Err(RelayAddrError::Unspecified))]
    #[case::unspecified_from_public("23.55.1.242", "0.0.0.0", Err(RelayAddrError::Unspecified))]
    #[case::broadcast_from_loopback("127.0.0.1", "255.255.255.255", Err(RelayAddrError::Special))]
    #[case::broadcast_from_lan("192.168.0.1", "255.255.255.255", Err(RelayAddrError::Special))]
    #[case::broadcast_from_public("23.55.1.242", "255.255.255.255", Err(RelayAddrError::Special))]
    #[case::loopback_from_lan("192.168.0.1", "127.0.2.19", Err(RelayAddrError::Loopback))]
    #[case::lan_from_public("23.55.1.242", "192.168.0.1", Err(RelayAddrError::Lan))]
    #[case::loopback_from_loopback("127.0.0.1", "127.0.2.19", Ok(()))]
    #[case::lan_from_loopback("127.0.0.1", "192.168.0.1", Ok(()))]
    #[case::public_from_loopback("127.0.0.1", "23.55.1.242", Ok(()))]
    #[case::lan_from_lan("192.168.0.1", "192.168.0.1", Ok(()))]
    #[case::public_from_lan("192.168.0.1", "23.55.1.242", Ok(()))]
    #[case::public_from_public("23.55.1.242", "23.55.1.242", Ok(()))]
    // Beyond geth's tests.
    #[case::loopback_from_public("23.55.1.242", "127.0.0.1", Err(RelayAddrError::Loopback))]
    #[case::loopback_v6_from_lan_v6("fc00::4", "::1", Err(RelayAddrError::Loopback))]
    #[case::loopback_v6_from_loopback_v4("127.0.0.1", "::1", Ok(()))]
    #[case::mapped_loopback_from_lan(
        "192.168.0.1",
        "::ffff:127.0.0.1",
        Err(RelayAddrError::Loopback)
    )]
    #[case::loopback_from_mapped_loopback("::ffff:127.0.0.1", "127.0.0.1", Ok(()))]
    #[case::lan_v6_from_public_v6("2a00:1450:4001::1", "fe80::1", Err(RelayAddrError::Lan))]
    #[case::lan_v4_from_lan_v6("fc00::4", "10.0.0.1", Ok(()))]
    #[case::shared_from_public("23.55.1.242", "100.64.0.1", Err(RelayAddrError::Lan))]
    #[case::lan_from_shared("100.64.0.1", "192.168.0.1", Ok(()))]
    #[case::lan_from_unspecified("0.0.0.0", "192.168.0.1", Err(RelayAddrError::Lan))]
    #[case::public_from_special("192.0.2.1", "23.55.1.242", Ok(()))]
    fn test_check_relay_addr(
        #[case] sender: IpAddr,
        #[case] addr: IpAddr,
        #[case] expected: Result<(), RelayAddrError>,
    ) {
        assert_eq!(check_relay_addr(AddrScope::of(sender), addr), expected);
    }
}
