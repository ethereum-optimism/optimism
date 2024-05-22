package manet

import (
	"net"
	"strings"

	ma "github.com/multiformats/go-multiaddr"
)

// Private4 and Private6 are well-known private networks
var Private4, Private6 []*net.IPNet
var privateCIDR4 = []string{
	// localhost
	"127.0.0.0/8",
	// private networks
	"10.0.0.0/8",
	"100.64.0.0/10",
	"172.16.0.0/12",
	"192.168.0.0/16",
	// link local
	"169.254.0.0/16",
}
var privateCIDR6 = []string{
	// localhost
	"::1/128",
	// ULA reserved
	"fc00::/7",
	// link local
	"fe80::/10",
}

// Unroutable4 and Unroutable6 are well known unroutable address ranges
var Unroutable4, Unroutable6 []*net.IPNet
var unroutableCIDR4 = []string{
	"0.0.0.0/8",
	"192.0.0.0/26",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"255.255.255.255/32",
}
var unroutableCIDR6 = []string{
	"ff00::/8",      // multicast
	"2001:db8::/32", // documentation
}

var globalUnicast []*net.IPNet
var globalUnicastCIDR6 = []string{
	"2000::/3",
}

var nat64CIDRs = []string{
	"64:ff9b:1::/48", // RFC 8215
	"64:ff9b::/96",   // RFC 6052
}

var nat64 []*net.IPNet

// unResolvableDomains do not resolve to an IP address.
// Ref: https://en.wikipedia.org/wiki/Special-use_domain_name#Reserved_domain_names
var unResolvableDomains = []string{
	// Reverse DNS Lookup
	".in-addr.arpa",
	".ip6.arpa",

	// RFC 6761: Users MAY assume that queries for "invalid" names will always return NXDOMAIN
	// responses
	".invalid",
}

// privateUseDomains are reserved for private use and have no central authority for consistent
// address resolution
// Ref: https://en.wikipedia.org/wiki/Special-use_domain_name#Reserved_domain_names
var privateUseDomains = []string{
	// RFC 8375: Reserved for home networks
	".home.arpa",

	// MDNS
	".local",

	// RFC 6761: No central authority for .test names
	".test",
}

// RFC 6761: Users may assume that IPv4 and IPv6 address queries for localhost names will
// always resolve to the respective IP loopback address
const localHostDomain = ".localhost"

func init() {
	Private4 = parseCIDR(privateCIDR4)
	Private6 = parseCIDR(privateCIDR6)
	Unroutable4 = parseCIDR(unroutableCIDR4)
	Unroutable6 = parseCIDR(unroutableCIDR6)
	globalUnicast = parseCIDR(globalUnicastCIDR6)
	nat64 = parseCIDR(nat64CIDRs)
}

func parseCIDR(cidrs []string) []*net.IPNet {
	ipnets := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		ipnets[i] = ipnet
	}
	return ipnets
}

// IsPublicAddr returns true if the IP part of the multiaddr is a publicly routable address
// or if it's a dns address without a special use domain e.g. .local.
func IsPublicAddr(a ma.Multiaddr) bool {
	isPublic := false
	ma.ForEach(a, func(c ma.Component) bool {
		switch c.Protocol().Code {
		case ma.P_IP6ZONE:
			return true
		case ma.P_IP4:
			ip := net.IP(c.RawValue())
			isPublic = !inAddrRange(ip, Private4) && !inAddrRange(ip, Unroutable4)
		case ma.P_IP6:
			ip := net.IP(c.RawValue())
			// IP6 documentation prefix(part of Unroutable6) is a subset of the ip6
			// global unicast allocation so we ensure that it's not a documentation
			// prefix by diffing with Unroutable6
			isPublicUnicastAddr := inAddrRange(ip, globalUnicast) && !inAddrRange(ip, Unroutable6)
			if isPublicUnicastAddr {
				isPublic = true
				return false
			}
			// The WellKnown NAT64 prefix(RFC 6052) can only reference a public IPv4
			// address.
			// The Local use NAT64 prefix(RFC 8215) can reference private IPv4
			// addresses. But since the translation from Local use NAT64 prefix to IPv4
			// address is left to the user we have no way of knowing which IPv4 address
			// is referenced. We count these as Public addresses because a false
			// negative for this method here is generally worse than a false positive.
			isPublic = inAddrRange(ip, nat64)
			return false
		case ma.P_DNS, ma.P_DNS4, ma.P_DNS6, ma.P_DNSADDR:
			dnsAddr := c.Value()
			isPublic = true
			if isSubdomain(dnsAddr, localHostDomain) {
				isPublic = false
				return false
			}
			for _, ud := range unResolvableDomains {
				if isSubdomain(dnsAddr, ud) {
					isPublic = false
					return false
				}
			}
			for _, pd := range privateUseDomains {
				if isSubdomain(dnsAddr, pd) {
					isPublic = false
					break
				}
			}
		}
		return false
	})
	return isPublic
}

// isSubdomain checks if child is sub domain of parent. It also returns true if child and parent are
// the same domain.
// Parent must have a "." prefix.
func isSubdomain(child, parent string) bool {
	return strings.HasSuffix(child, parent) || child == parent[1:]
}

// IsPrivateAddr returns true if the IP part of the mutiaddr is in a private network
func IsPrivateAddr(a ma.Multiaddr) bool {
	isPrivate := false
	ma.ForEach(a, func(c ma.Component) bool {
		switch c.Protocol().Code {
		case ma.P_IP6ZONE:
			return true
		case ma.P_IP4:
			isPrivate = inAddrRange(net.IP(c.RawValue()), Private4)
		case ma.P_IP6:
			isPrivate = inAddrRange(net.IP(c.RawValue()), Private6)
		case ma.P_DNS, ma.P_DNS4, ma.P_DNS6, ma.P_DNSADDR:
			dnsAddr := c.Value()
			if isSubdomain(dnsAddr, localHostDomain) {
				isPrivate = true
			}
			// We don't check for privateUseDomains because private use domains can
			// resolve to public IP addresses
		}
		return false
	})
	return isPrivate
}

func inAddrRange(ip net.IP, ipnets []*net.IPNet) bool {
	for _, ipnet := range ipnets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}
