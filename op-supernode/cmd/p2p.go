package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

func withNoP2P(vcli *flags.VirtualCLI) error {
	vcli.WithBoolOverride(opnodeflags.DisableP2PName, true)
	vcli.WithStringOverride(opnodeflags.P2PPrivPathName, "")
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, "")
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, "")
	vcli.WithUintOverride(opnodeflags.ListenTCPPortName, 0)
	vcli.WithUintOverride(opnodeflags.ListenUDPPortName, 0)
	return nil
}

func withNamespacedP2P(vcli *flags.VirtualCLI, datadir string, namespace string) error {
	p2pDir := filepath.Join(datadir, namespace, "p2p")
	if err := os.MkdirAll(p2pDir, 0o700); err != nil {
		return fmt.Errorf("failed creating p2p dir for chain %s: %w", namespace, err)
	}
	if !vcli.IsChainSet(opnodeflags.P2PPrivPathName) {
		vcli.WithStringOverride(opnodeflags.P2PPrivPathName, filepath.Join(p2pDir, "opnode_p2p_priv.txt"))
	}
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, filepath.Join(p2pDir, "peerstore_db"))
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, filepath.Join(p2pDir, "discovery_db"))
	// A listen port left at 0 (dynamic) is resolved to a free port here, once,
	// so that every virtual node instance of this chain binds the same port for
	// the life of the process. The chain container replaces the virtual node on
	// every rewind; with port 0 each replacement would listen on a new random
	// port, and peers connected to the old one would lose it. A non-zero
	// vn.all.<flag> would be reused by every virtual node, so require per-chain
	// listen ports instead.
	if err := withNamespacedP2PListenPort(vcli, opnodeflags.ListenTCPPortName); err != nil {
		return err
	}
	if err := withNamespacedP2PListenPort(vcli, opnodeflags.ListenUDPPortName); err != nil {
		return err
	}
	return nil
}

func withNamespacedP2PListenPort(vcli *flags.VirtualCLI, name string) error {
	if vcli.IsGlobalSet(name) && vcli.GlobalUint(name) != 0 {
		return fmt.Errorf("%s%s cannot be non-zero for virtual-node P2P listen ports; use %s<chainID>.%s for fixed per-chain ports", flags.VNFlagGlobalPrefix, name, flags.VNFlagNamePrefix, name)
	}
	if vcli.IsSet(name) && vcli.Uint(name) != 0 {
		return nil
	}
	network := "tcp"
	if name == opnodeflags.ListenUDPPortName {
		network = "udp"
	}
	port, err := pickFreePort(network)
	if err != nil {
		return fmt.Errorf("pick a %s port for %s: %w", network, name, err)
	}
	vcli.WithUintOverride(name, port)
	return nil
}

var (
	pickedPortsMu sync.Mutex
	pickedPorts   = map[string]bool{}
)

// pickFreePort asks the OS for a free port and returns one that no earlier
// call in this process has returned, so virtual nodes never share a port.
func pickFreePort(network string) (uint, error) {
	pickedPortsMu.Lock()
	defer pickedPortsMu.Unlock()
	for range 16 {
		var port int
		switch network {
		case "udp":
			c, err := net.ListenPacket("udp", ":0")
			if err != nil {
				return 0, err
			}
			port = c.LocalAddr().(*net.UDPAddr).Port
			_ = c.Close()
		default:
			l, err := net.Listen("tcp", ":0")
			if err != nil {
				return 0, err
			}
			port = l.Addr().(*net.TCPAddr).Port
			_ = l.Close()
		}
		key := fmt.Sprintf("%s/%d", network, port)
		if !pickedPorts[key] {
			pickedPorts[key] = true
			return uint(port), nil
		}
	}
	return 0, fmt.Errorf("no unused %s port after 16 tries", network)
}
