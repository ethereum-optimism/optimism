package main

import (
	"fmt"
	"os"
	"path/filepath"

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
	vcli.WithStringOverride(opnodeflags.P2PPrivPathName, filepath.Join(p2pDir, "opnode_p2p_priv.txt"))
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, filepath.Join(p2pDir, "peerstore_db"))
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, filepath.Join(p2pDir, "discovery_db"))
	// Default listen ports to 0 (dynamic) to prevent collisions when the user
	// has not pinned a port. Honour vn.all.<flag> and vn.<id>.<flag> when set.
	if !vcli.IsSet(opnodeflags.ListenTCPPortName) {
		vcli.WithUintOverride(opnodeflags.ListenTCPPortName, 0)
	}
	if !vcli.IsSet(opnodeflags.ListenUDPPortName) {
		vcli.WithUintOverride(opnodeflags.ListenUDPPortName, 0)
	}
	return nil
}
