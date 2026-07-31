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
	if !vcli.IsChainSet(opnodeflags.P2PPrivPathName) {
		vcli.WithStringOverride(opnodeflags.P2PPrivPathName, filepath.Join(p2pDir, "opnode_p2p_priv.txt"))
	}
	vcli.WithStringOverride(opnodeflags.PeerstorePathName, filepath.Join(p2pDir, "peerstore_db"))
	vcli.WithStringOverride(opnodeflags.DiscoveryPathName, filepath.Join(p2pDir, "discovery_db"))
	// Default listen ports to 0 (dynamic) to prevent collisions when the user
	// has not pinned a per-chain port. A non-zero vn.all.<flag> would be reused
	// by every virtual node, so require per-chain listen ports instead.
	if err := withNamespacedP2PListenPort(vcli, opnodeflags.ListenTCPPortName); err != nil {
		return err
	}
	if err := withNamespacedP2PListenPort(vcli, opnodeflags.ListenUDPPortName); err != nil {
		return err
	}
	return nil
}

func withNamespacedP2PListenPort(vcli *flags.VirtualCLI, name string) error {
	if !vcli.IsSet(name) {
		vcli.WithUintOverride(name, 0)
		return nil
	}
	if vcli.IsGlobalSet(name) && vcli.GlobalUint(name) != 0 {
		return fmt.Errorf("%s%s cannot be non-zero for virtual-node P2P listen ports; use %s<chainID>.%s for fixed per-chain ports", flags.VNFlagGlobalPrefix, name, flags.VNFlagNamePrefix, name)
	}
	return nil
}
