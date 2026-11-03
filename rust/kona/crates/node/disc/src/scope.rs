//! Address scopes, for deciding whether an address another node relayed is worth dialing.
//!
//! Ported from geth's `p2p/netutil/net.go`, with its special-purpose lists brought up to date with
//! the IANA registries.

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

use discv5::Enr;
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

    /// Returns our own scope: that of `ip`, the IP our record advertises, or, while it advertises
    /// none (`ip` is unspecified), the most local scope among `bootnode_ips`, as records reach us
    /// through our bootnodes. Public if none of them is loopback or LAN.
    ///
    /// Our scope is never unspecified or special-purpose: such an address names no host we are
    /// on, so it says nothing about which addresses we can reach. When nothing else tells us, we
    /// take the most conservative scope, which is, perhaps counter-intuitively, public: a public
    /// node can reach the fewest addresses, only public ones, so it accepts the fewest relayed
    /// records. A loopback node would accept them all. A special-purpose `ip`, such as a
    /// documentation address, is public for the same reason: it is a mistake or a test, and
    /// bootnodes can't tell us better than the address we were told to advertise.
    ///
    /// Being conservative costs a node without bootnodes, such as the first node of a network,
    /// little: it still learns of the nodes that start sessions with it, as discv5 adds those to
    /// its table without the table filter, and the driver forwards their records unchecked. It only
    /// misses the loopback and LAN records those nodes relay.
    ///
    /// It doesn't follow an IP that votes set later, as discv5's table filter is chosen once.
    pub fn own(ip: IpAddr, bootnode_ips: impl IntoIterator<Item = IpAddr>) -> Self {
        match Self::of(ip) {
            Self::Unspecified => {}
            Self::Special => return Self::Public,
            scope => return scope,
        }
        bootnode_ips.into_iter().map(Self::of).fold(Self::Public, |most_local, scope| {
            match (most_local, scope) {
                (Self::Loopback, _) | (_, Self::Loopback) => Self::Loopback,
                (Self::Lan, _) | (_, Self::Lan) => Self::Lan,
                _ => Self::Public,
            }
        })
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
    pub fn check_relayed_addr(sender: Self, addr: IpAddr) -> Result<(), RelayAddrError> {
        match Self::of(addr) {
            Self::Unspecified => Err(RelayAddrError::Unspecified),
            Self::Special => Err(RelayAddrError::Special),
            Self::Loopback if sender != Self::Loopback => Err(RelayAddrError::Loopback),
            Self::Lan if !matches!(sender, Self::Loopback | Self::Lan) => Err(RelayAddrError::Lan),
            _ => Ok(()),
        }
    }

    /// Checks every IP address in `enr`, a record relayed by a node of scope `sender`, with
    /// [`Self::check_relayed_addr`]. A record with no IP address fails, as it has none to dial.
    pub fn check_relayed_enr(sender: Self, enr: &Enr) -> Result<(), RelayAddrError> {
        let ips = [enr.ip4().map(IpAddr::V4), enr.ip6().map(IpAddr::V6)];
        if ips.iter().all(Option::is_none) {
            return Err(RelayAddrError::NoAddress);
        }
        ips.into_iter().flatten().try_for_each(|ip| Self::check_relayed_addr(sender, ip))
    }

    /// Checks whether a node of scope `own` can reach `addr`, an address it was given without
    /// knowing who it came from, such as a bootnode's enode URL. It is [`Self::check_relayed_addr`]
    /// with `own` in place of the sender, as [`Self::check_reachable_enr`] is.
    pub fn check_reachable_addr(own: Self, addr: IpAddr) -> Result<(), RelayAddrError> {
        Self::check_relayed_addr(own, addr)
    }

    /// Checks whether a node of scope `own` can reach every IP address in `enr`, a record another
    /// node relayed, without knowing which node that was.
    ///
    /// It is [`Self::check_relayed_enr`] with `own` in place of the sender: what matters for a
    /// record we would dial is that none of its addresses is more local than our own.
    pub fn check_reachable_enr(own: Self, enr: &Enr) -> Result<(), RelayAddrError> {
        Self::check_relayed_enr(own, enr)
    }

    /// Returns discv5's table filter for a node of scope `own`. It keeps the records
    /// [`Self::check_reachable_enr`] rejects out of the table, so that they are not relayed to
    /// others, and out of lookups, so that discovery doesn't contact them.
    ///
    /// discv5 takes a plain `fn`, which can't hold the scope, so there is one for each scope. An
    /// unspecified or special-purpose own scope gets the public one.
    pub fn table_filter(own: Self) -> fn(&Enr) -> bool {
        match own {
            Self::Loopback => |enr| Self::check_reachable_enr(Self::Loopback, enr).is_ok(),
            Self::Lan => |enr| Self::check_reachable_enr(Self::Lan, enr).is_ok(),
            Self::Unspecified | Self::Special | Self::Public => {
                |enr| Self::check_reachable_enr(Self::Public, enr).is_ok()
            }
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
    fn test_check_relayed_addr(
        #[case] sender: IpAddr,
        #[case] addr: IpAddr,
        #[case] expected: Result<(), RelayAddrError>,
    ) {
        assert_eq!(AddrScope::check_relayed_addr(AddrScope::of(sender), addr), expected);
    }

    /// Our own scope is that of the IP we advertise. Without one, it is the most public among our
    /// bootnodes', ignoring special-purpose ones, and public if there are none: one local bootnode
    /// among public ones must not make the node accept local records from public peers. A
    /// special-purpose IP is public too. Our scope is never unspecified or special-purpose, which
    /// say nothing about what we can reach, but the most conservative scope. See
    /// [`AddrScope::own`].
    #[rstest]
    #[case::public("1.2.3.4", &[], AddrScope::Public)]
    #[case::lan("10.0.0.1", &[], AddrScope::Lan)]
    #[case::advertised_wins_over_bootnodes("1.2.3.4", &["127.0.0.1"], AddrScope::Public)]
    #[case::special("192.0.2.1", &[], AddrScope::Public)]
    #[case::special_ignores_bootnodes("192.0.2.1", &["127.0.0.1"], AddrScope::Public)]
    #[case::unspecified_no_bootnodes("0.0.0.0", &[], AddrScope::Public)]
    #[case::unspecified_public_bootnodes("0.0.0.0", &["1.2.3.4", "5.6.7.8"], AddrScope::Public)]
    #[case::unspecified_public_and_lan_bootnodes(
        "0.0.0.0",
        &["1.2.3.4", "10.0.0.1"],
        AddrScope::Public
    )]
    #[case::unspecified_public_and_loopback_bootnodes(
        "0.0.0.0",
        &["127.0.0.1", "1.2.3.4"],
        AddrScope::Public
    )]
    #[case::unspecified_lan_and_loopback_bootnodes(
        "0.0.0.0",
        &["10.0.0.1", "127.0.0.1"],
        AddrScope::Lan
    )]
    #[case::unspecified_loopback_bootnodes(
        "0.0.0.0",
        &["127.0.0.1", "127.0.0.2"],
        AddrScope::Loopback
    )]
    #[case::unspecified_special_bootnode("0.0.0.0", &["192.0.2.1"], AddrScope::Public)]
    #[case::unspecified_special_and_loopback_bootnodes(
        "0.0.0.0",
        &["192.0.2.1", "127.0.0.1"],
        AddrScope::Loopback
    )]
    #[case::unspecified_v6("::", &["::1"], AddrScope::Loopback)]
    fn test_own_scope(
        #[case] ip: IpAddr,
        #[case] bootnode_ips: &[&str],
        #[case] expected: AddrScope,
    ) {
        let bootnode_ips = bootnode_ips.iter().map(|ip| ip.parse().expect("a valid IP"));
        // TODO(#23766): the mixed cases fail, as the most local bootnode decides.
        assert_eq!(AddrScope::own(ip, bootnode_ips), expected);
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

    /// Our own scope reaches an address as a sender of that scope may relay it.
    #[rstest]
    #[case::loopback_while_public("127.0.0.1", AddrScope::Public, Err(RelayAddrError::Loopback))]
    #[case::lan_while_public("10.0.0.1", AddrScope::Public, Err(RelayAddrError::Lan))]
    #[case::public_while_public("23.55.1.242", AddrScope::Public, Ok(()))]
    #[case::loopback_while_loopback("127.0.0.1", AddrScope::Loopback, Ok(()))]
    fn test_check_reachable_addr(
        #[case] addr: IpAddr,
        #[case] own: AddrScope,
        #[case] expected: Result<(), RelayAddrError>,
    ) {
        assert_eq!(AddrScope::check_reachable_addr(own, addr), expected);
    }

    /// Every address in a record must pass, and a record without one fails.
    #[rstest]
    #[case::ip4_only(Some("23.55.1.242"), None, Ok(()))]
    #[case::ip6_only(None, Some("2a00:1450:4001::1"), Ok(()))]
    #[case::both(Some("23.55.1.242"), Some("2a00:1450:4001::1"), Ok(()))]
    #[case::bad_ip4(Some("192.168.0.1"), Some("2a00:1450:4001::1"), Err(RelayAddrError::Lan))]
    #[case::bad_ip6(Some("23.55.1.242"), Some("fe80::1"), Err(RelayAddrError::Lan))]
    #[case::none(None, None, Err(RelayAddrError::NoAddress))]
    fn test_check_relayed_enr(
        #[case] ip4: Option<&str>,
        #[case] ip6: Option<&str>,
        #[case] expected: Result<(), RelayAddrError>,
    ) {
        let enr = enr(ip4, ip6);
        assert_eq!(AddrScope::check_relayed_enr(AddrScope::Public, &enr), expected);
        assert_eq!(AddrScope::check_reachable_enr(AddrScope::Public, &enr), expected);
    }

    /// Each own scope gets the table filter that keeps out the records it can't reach. An
    /// unspecified or special-purpose own scope gets the public one.
    #[rstest]
    #[case::loopback(AddrScope::Loopback, [true, true, true])]
    #[case::lan(AddrScope::Lan, [false, true, true])]
    #[case::public(AddrScope::Public, [false, false, true])]
    #[case::unspecified(AddrScope::Unspecified, [false, false, true])]
    #[case::special(AddrScope::Special, [false, false, true])]
    fn test_table_filter(#[case] own: AddrScope, #[case] expected: [bool; 3]) {
        let filter = AddrScope::table_filter(own);
        let records = ["127.0.0.1", "10.0.0.1", "23.55.1.242"].map(|ip| enr(Some(ip), None));
        assert_eq!(records.each_ref().map(filter), expected);
        assert!(!filter(&enr(Some("192.0.2.1"), None)), "a special-purpose record never passes");
        assert!(!filter(&enr(None, None)), "a record without an address never passes");
    }
}
