package manet

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
)

// ResolveUnspecifiedAddress expands an unspecified ip addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces. If ifaceAddr is nil, we request interface addresses
// from the network stack. (this is so you can provide a cached value if resolving many addrs)
func ResolveUnspecifiedAddress(resolve ma.Multiaddr, ifaceAddrs []ma.Multiaddr) ([]ma.Multiaddr, error) {
	// split address into its components
	first, rest := ma.SplitFirst(resolve)

	// if first component (ip) is not unspecified, use it as is.
	if !IsIPUnspecified(first) {
		return []ma.Multiaddr{resolve}, nil
	}

	resolveProto := resolve.Protocols()[0].Code
	out := make([]ma.Multiaddr, 0, len(ifaceAddrs))
	for _, ia := range ifaceAddrs {
		iafirst, _ := ma.SplitFirst(ia)
		// must match the first protocol to be resolve.
		if iafirst.Protocol().Code != resolveProto {
			continue
		}

		joined := ia
		if rest != nil {
			joined = ma.Join(ia, rest)
		}
		out = append(out, joined)
	}
	if len(out) < 1 {
		return nil, fmt.Errorf("failed to resolve: %s", resolve)
	}
	return out, nil
}

// ResolveUnspecifiedAddresses expands unspecified ip addresses (/ip4/0.0.0.0, /ip6/::) to
// use the known local interfaces.
func ResolveUnspecifiedAddresses(unspecAddrs, ifaceAddrs []ma.Multiaddr) ([]ma.Multiaddr, error) {
	// todo optimize: only fetch these if we have a "any" addr.
	if len(ifaceAddrs) < 1 {
		var err error
		ifaceAddrs, err = interfaceAddresses()
		if err != nil {
			return nil, err
		}
	}

	var outputAddrs []ma.Multiaddr
	for _, a := range unspecAddrs {
		// unspecified?
		resolved, err := ResolveUnspecifiedAddress(a, ifaceAddrs)
		if err != nil {
			continue // optimistic. if we can't resolve anything, we'll know at the bottom.
		}
		outputAddrs = append(outputAddrs, resolved...)
	}

	if len(outputAddrs) < 1 {
		return nil, fmt.Errorf("failed to specify addrs: %s", unspecAddrs)
	}
	return outputAddrs, nil
}

// interfaceAddresses returns a list of addresses associated with local machine
// Note: we do not return link local addresses. IP loopback is ok, because we
// may be connecting to other nodes in the same machine.
func interfaceAddresses() ([]ma.Multiaddr, error) {
	maddrs, err := InterfaceMultiaddrs()
	if err != nil {
		return nil, err
	}

	var out []ma.Multiaddr
	for _, a := range maddrs {
		if IsIP6LinkLocal(a) {
			continue
		}
		out = append(out, a)
	}
	return out, nil
}
