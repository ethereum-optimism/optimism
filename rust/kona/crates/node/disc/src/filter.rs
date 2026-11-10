//! Which addresses discovery uses in other nodes' records.

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr};

use discv5::Enr;

/// A [`discv5::Config::table_filter`] that accepts a record only if it has an IP address, and all
/// its IP addresses are public: none is unspecified, loopback, private, link-local, shared
/// (carrier-grade NAT, RFC 6598), benchmarking, reserved, multicast, broadcast or documentation.
///
/// discv5 applies it to the records in other nodes' answers to our lookups, and to those added with
/// `add_enr`, such as bootnodes. It keeps a record it rejects out of the table, so that it isn't
/// relayed to others, and out of lookups, so that discovery doesn't contact it. Leave it out on a
/// devnet, where nodes use loopback or LAN addresses.
pub fn public_only(enr: &Enr) -> bool {
    let ips = [enr.ip4().map(IpAddr::V4), enr.ip6().map(IpAddr::V6)];
    ips.iter().any(Option::is_some) && ips.into_iter().flatten().all(is_public)
}

/// Returns whether `ip` can name a host on the public internet, see [`public_only`]. An
/// IPv4-mapped IPv6 address is judged by the IPv4 address in it.
const fn is_public(ip: IpAddr) -> bool {
    match ip.to_canonical() {
        IpAddr::V4(ip) => is_public_v4(ip),
        IpAddr::V6(ip) => is_public_v6(ip),
    }
}

const fn is_public_v4(ip: Ipv4Addr) -> bool {
    // `Ipv4Addr::is_shared`, `is_benchmarking` and `is_reserved` aren't stable yet.
    let [first, second, ..] = ip.octets();
    // 100.64.0.0/10.
    let is_shared = first == 100 && second & 0b1100_0000 == 0b0100_0000;
    // 198.18.0.0/15, which labs and proxy tools use internally.
    let is_benchmarking = first == 198 && second & 0b1111_1110 == 18;
    // 240.0.0.0/4, which some clouds use for internal ranges, such as GKE's Class E pod ranges.
    let is_reserved = first & 0b1111_0000 == 0b1111_0000;
    !(ip.is_unspecified() ||
        ip.is_loopback() ||
        ip.is_private() ||
        ip.is_link_local() ||
        is_shared ||
        is_benchmarking ||
        is_reserved ||
        ip.is_multicast() ||
        ip.is_broadcast() ||
        ip.is_documentation())
}

const fn is_public_v6(ip: Ipv6Addr) -> bool {
    // `Ipv6Addr::is_documentation` isn't stable yet: `2001:db8::/32` and `3fff::/20`.
    let [first, second, ..] = ip.segments();
    let is_documentation =
        (first == 0x2001 && second == 0x0db8) || (first == 0x3fff && second < 0x1000);
    !(ip.is_unspecified() ||
        ip.is_loopback() ||
        ip.is_unique_local() ||
        ip.is_unicast_link_local() ||
        ip.is_multicast() ||
        is_documentation)
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case::public_v4("1.1.1.1", true)]
    #[case::past_private_172("172.32.0.1", true)]
    #[case::past_shared("100.128.0.0", true)]
    #[case::public_v6("2a00:1450:4001::1", true)]
    #[case::mapped_public("::ffff:88.99.100.2", true)]
    #[case::unspecified_v4("0.0.0.0", false)]
    #[case::loopback_v4("127.0.2.19", false)]
    #[case::private_10("10.0.1.1", false)]
    #[case::private_172("172.31.252.251", false)]
    #[case::private_192("192.168.1.4", false)]
    #[case::link_local_v4("169.254.1.1", false)]
    #[case::shared_first("100.64.0.0", false)]
    #[case::shared_last("100.127.255.255", false)]
    #[case::multicast_v4("224.0.0.22", false)]
    #[case::broadcast("255.255.255.255", false)]
    #[case::documentation_v4("192.0.2.1", false)]
    #[case::unspecified_v6("::", false)]
    #[case::loopback_v6("::1", false)]
    #[case::unique_local_v6("fc00::4", false)]
    #[case::link_local_v6("fe80::f4a1:8eff:fec5:9d9d", false)]
    #[case::multicast_v6("ff05::1:3", false)]
    #[case::documentation_v6("2001:db8:85a3:8d3:1319:8a2e:370:7348", false)]
    #[case::documentation_v6_new("3fff:fff::1", false)]
    #[case::past_documentation_v6_new("3fff:1000::1", true)]
    #[case::mapped_loopback("::ffff:127.0.0.1", false)]
    #[case::mapped_private("::ffff:10.10.0.2", false)]
    // Reserved, which some clouds use for internal ranges, such as GKE's Class E pod ranges.
    #[case::reserved_first("240.0.0.0", false)]
    #[case::reserved("250.1.2.3", false)]
    #[case::reserved_last("255.255.255.254", false)]
    #[case::mapped_reserved("::ffff:240.0.0.1", false)]
    // Benchmarking, which labs and proxy tools use internally.
    #[case::benchmarking_first("198.18.0.0", false)]
    #[case::benchmarking_last("198.19.255.255", false)]
    #[case::before_benchmarking("198.17.255.255", true)]
    #[case::past_benchmarking("198.20.0.0", true)]
    fn test_is_public(#[case] ip: IpAddr, #[case] expected: bool) {
        assert_eq!(is_public(ip), expected);
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

    /// Every address in a record must be public, and a record without one fails.
    #[rstest]
    #[case::ip4_only(Some("23.55.1.242"), None, true)]
    #[case::ip6_only(None, Some("2a00:1450:4001::1"), true)]
    #[case::both(Some("23.55.1.242"), Some("2a00:1450:4001::1"), true)]
    #[case::private_ip4(Some("192.168.0.1"), Some("2a00:1450:4001::1"), false)]
    #[case::private_ip6(Some("23.55.1.242"), Some("fe80::1"), false)]
    #[case::none(None, None, false)]
    fn test_public_only(
        #[case] ip4: Option<&str>,
        #[case] ip6: Option<&str>,
        #[case] expected: bool,
    ) {
        assert_eq!(public_only(&enr(ip4, ip6)), expected);
    }
}
