//! Address scopes, for deciding which addresses discovery uses.
//!
//! The scope classes are ported from geth's `p2p/netutil/net.go`, with its special-purpose lists
//! brought up to date with the IANA registries.

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

use discv5::Enr;
use ipnet::{Ipv4Net, Ipv6Net};

use crate::AddrError;

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
/// LAN instead, and loopback has its own scope. The IPv4-IPv6 translation blocks, `64:ff9b::/96`
/// and `64:ff9b:1::/48`, are left public, although the second isn't globally reachable: an address
/// in them stands for an IPv4 address behind a translator, whose scope isn't worked out here.
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

/// Which addresses discovery uses in other nodes' records: in its table and lookups, and when
/// dialing saved records and bootnodes. It never uses unspecified or special-purpose addresses.
///
/// A node on a public network can't reach other hosts' loopback or LAN addresses, so it uses only
/// public ones. A node on a devnet allows private ones too.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AddrFilter {
    /// Only public addresses.
    PublicOnly,
    /// Public, loopback and LAN addresses.
    AllowPrivate,
}

impl AddrFilter {
    /// Returns the filter for a node that advertises `ip`. Private addresses are allowed if `ip` is
    /// a loopback or LAN address, as on a devnet. They aren't if it is public, special-purpose, or
    /// unspecified, as when listening on all interfaces, which says nothing about the network the
    /// node is on.
    pub fn for_advertised_ip(ip: IpAddr) -> Self {
        match AddrScope::of(ip) {
            AddrScope::Loopback | AddrScope::Lan => Self::AllowPrivate,
            AddrScope::Unspecified | AddrScope::Special | AddrScope::Public => Self::PublicOnly,
        }
    }

    /// Checks whether discovery may use `addr`.
    pub fn check_addr(self, addr: IpAddr) -> Result<(), AddrError> {
        match (AddrScope::of(addr), self) {
            (AddrScope::Unspecified, _) => Err(AddrError::Unspecified),
            (AddrScope::Special, _) => Err(AddrError::Special),
            (AddrScope::Loopback, Self::PublicOnly) => Err(AddrError::Loopback),
            (AddrScope::Lan, Self::PublicOnly) => Err(AddrError::Lan),
            _ => Ok(()),
        }
    }

    /// Checks every IP address in `enr` with [`Self::check_addr`]. A record with no IP address
    /// fails, as it has none to contact.
    pub fn check_enr(self, enr: &Enr) -> Result<(), AddrError> {
        let ips = [enr.ip4().map(IpAddr::V4), enr.ip6().map(IpAddr::V6)];
        if ips.iter().all(Option::is_none) {
            return Err(AddrError::NoAddress);
        }
        ips.into_iter().flatten().try_for_each(|ip| self.check_addr(ip))
    }

    /// Returns discv5's table filter for this filter. discv5 applies it to the records in other
    /// nodes' answers to our lookups, and to those added with `add_enr`, such as bootnodes. It
    /// keeps those [`Self::check_enr`] rejects out of the table, so that they aren't relayed to
    /// others, and out of lookups, so that discovery doesn't contact them. A node we have a session
    /// with gets into the table without it, and is dialed at the address it reached us from.
    ///
    /// discv5 takes a plain `fn`, so there is one for each filter.
    pub fn table_filter(self) -> fn(&Enr) -> bool {
        match self {
            Self::PublicOnly => |enr| Self::PublicOnly.check_enr(enr).is_ok(),
            Self::AllowPrivate => |enr| Self::AllowPrivate.check_enr(enr).is_ok(),
        }
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

    /// Private addresses are allowed for a node that advertises a loopback or LAN address, and
    /// not for one that advertises a public, special-purpose or no address.
    #[rstest]
    #[case::loopback("127.0.0.1", AddrFilter::AllowPrivate)]
    #[case::lan("10.0.0.1", AddrFilter::AllowPrivate)]
    #[case::shared("100.64.0.1", AddrFilter::AllowPrivate)]
    #[case::loopback_v6("::1", AddrFilter::AllowPrivate)]
    #[case::public("1.2.3.4", AddrFilter::PublicOnly)]
    #[case::special("192.0.2.1", AddrFilter::PublicOnly)]
    #[case::unspecified("0.0.0.0", AddrFilter::PublicOnly)]
    #[case::unspecified_v6("::", AddrFilter::PublicOnly)]
    fn test_for_advertised_ip(#[case] ip: IpAddr, #[case] expected: AddrFilter) {
        assert_eq!(AddrFilter::for_advertised_ip(ip), expected);
    }

    #[rstest]
    #[case::public("23.55.1.242", Ok(()), Ok(()))]
    #[case::lan("192.168.0.1", Err(AddrError::Lan), Ok(()))]
    #[case::loopback("127.0.0.1", Err(AddrError::Loopback), Ok(()))]
    #[case::mapped_loopback("::ffff:127.0.0.1", Err(AddrError::Loopback), Ok(()))]
    #[case::unique_local_v6("fc00::4", Err(AddrError::Lan), Ok(()))]
    #[case::special("192.0.2.1", Err(AddrError::Special), Err(AddrError::Special))]
    #[case::broadcast("255.255.255.255", Err(AddrError::Special), Err(AddrError::Special))]
    #[case::unspecified("0.0.0.0", Err(AddrError::Unspecified), Err(AddrError::Unspecified))]
    fn test_check_addr(
        #[case] addr: IpAddr,
        #[case] public_only: Result<(), AddrError>,
        #[case] allow_private: Result<(), AddrError>,
    ) {
        assert_eq!(AddrFilter::PublicOnly.check_addr(addr), public_only);
        assert_eq!(AddrFilter::AllowPrivate.check_addr(addr), allow_private);
    }

    /// A record with the given addresses.
    fn enr(ip4: Option<&str>, ip6: Option<&str>) -> Enr {
        let key = enr::CombinedKey::generate_secp256k1();
        let mut builder = Enr::builder();
        if let Some(ip4) = ip4 {
            builder.ip4(ip4.parse().expect("a valid IPv4 address"));
        }
        if let Some(ip6) = ip6 {
            builder.ip6(ip6.parse().expect("a valid IPv6 address"));
        }
        builder.build(&key).expect("the record builds")
    }

    /// Every address in a record must pass, and a record without one fails.
    #[rstest]
    #[case::ip4_only(Some("23.55.1.242"), None, Ok(()))]
    #[case::ip6_only(None, Some("2a00:1450:4001::1"), Ok(()))]
    #[case::both(Some("23.55.1.242"), Some("2a00:1450:4001::1"), Ok(()))]
    #[case::private_ip4(Some("192.168.0.1"), Some("2a00:1450:4001::1"), Err(AddrError::Lan))]
    #[case::private_ip6(Some("23.55.1.242"), Some("fe80::1"), Err(AddrError::Lan))]
    #[case::none(None, None, Err(AddrError::NoAddress))]
    fn test_check_enr(
        #[case] ip4: Option<&str>,
        #[case] ip6: Option<&str>,
        #[case] expected: Result<(), AddrError>,
    ) {
        assert_eq!(AddrFilter::PublicOnly.check_enr(&enr(ip4, ip6)), expected);
    }

    /// Each filter's table filter keeps out the records it rejects.
    #[rstest]
    #[case::public_only(AddrFilter::PublicOnly, [false, false, true])]
    #[case::allow_private(AddrFilter::AllowPrivate, [true, true, true])]
    fn test_table_filter(#[case] filter: AddrFilter, #[case] expected: [bool; 3]) {
        let table_filter = filter.table_filter();
        let records = ["127.0.0.1", "10.0.0.1", "23.55.1.242"].map(|ip| enr(Some(ip), None));
        assert_eq!(records.each_ref().map(table_filter), expected);
        assert!(
            !table_filter(&enr(Some("192.0.2.1"), None)),
            "a special-purpose record never passes"
        );
        assert!(!table_filter(&enr(None, None)), "a record without an address never passes");
    }
}
