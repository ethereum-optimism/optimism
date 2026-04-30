package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const testChainID uint64 = 11155420

// portFlags returns the supernode-cloned variants of the listen TCP/UDP flags
// so tests can drive them through cli context parsing the same way the real
// supernode does at runtime.
func portFlags() []cli.Flag {
	return []cli.Flag{
		&cli.UintFlag{Name: flags.VNFlagGlobalPrefix + opnodeflags.ListenTCPPortName},
		&cli.UintFlag{Name: flags.VNFlagGlobalPrefix + opnodeflags.ListenUDPPortName},
		&cli.UintFlag{Name: fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenTCPPortName)},
		&cli.UintFlag{Name: fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenUDPPortName)},
	}
}

func newCtx(t *testing.T, flagSet []cli.Flag, args []string) *cli.Context {
	t.Helper()
	app := &cli.App{Flags: flagSet}
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range flagSet {
		require.NoError(t, f.Apply(set))
	}
	require.NoError(t, set.Parse(args))
	return cli.NewContext(app, set, nil)
}

func newVCLI(t *testing.T, args []string) *flags.VirtualCLI {
	t.Helper()
	return flags.NewVirtualCLI(newCtx(t, portFlags(), args), testChainID)
}

func TestWithNamespacedP2P_DefaultsListenPortsToZero(t *testing.T) {
	vcli := newVCLI(t, nil)
	require.NoError(t, withNamespacedP2P(vcli, t.TempDir(), fmt.Sprintf("%d", testChainID)))

	require.Equal(t, uint(0), vcli.Uint(opnodeflags.ListenTCPPortName))
	require.Equal(t, uint(0), vcli.Uint(opnodeflags.ListenUDPPortName))
}

func TestWithNamespacedP2P_HonoursGlobalListenPorts(t *testing.T) {
	// A global vn.all.* port collides at bind time when more than one chain is
	// running, but the supernode honours it as configured -- test for
	// completeness so a single-chain deployment can pin a fixed port.
	vcli := newVCLI(t, []string{
		"--" + flags.VNFlagGlobalPrefix + opnodeflags.ListenTCPPortName + "=9222",
		"--" + flags.VNFlagGlobalPrefix + opnodeflags.ListenUDPPortName + "=9223",
	})
	require.NoError(t, withNamespacedP2P(vcli, t.TempDir(), fmt.Sprintf("%d", testChainID)))

	require.Equal(t, uint(9222), vcli.Uint(opnodeflags.ListenTCPPortName))
	require.Equal(t, uint(9223), vcli.Uint(opnodeflags.ListenUDPPortName))
}

func TestWithNamespacedP2P_HonoursPerChainListenPorts(t *testing.T) {
	tcpName := fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenTCPPortName)
	udpName := fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenUDPPortName)
	vcli := newVCLI(t, []string{
		"--" + tcpName + "=9000",
		"--" + udpName + "=9001",
	})
	require.NoError(t, withNamespacedP2P(vcli, t.TempDir(), fmt.Sprintf("%d", testChainID)))

	require.Equal(t, uint(9000), vcli.Uint(opnodeflags.ListenTCPPortName))
	require.Equal(t, uint(9001), vcli.Uint(opnodeflags.ListenUDPPortName))
}

func TestWithNamespacedP2P_PerChainOverridesGlobal(t *testing.T) {
	tcpName := fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenTCPPortName)
	vcli := newVCLI(t, []string{
		"--" + flags.VNFlagGlobalPrefix + opnodeflags.ListenTCPPortName + "=9222",
		"--" + tcpName + "=9000",
	})
	require.NoError(t, withNamespacedP2P(vcli, t.TempDir(), fmt.Sprintf("%d", testChainID)))

	require.Equal(t, uint(9000), vcli.Uint(opnodeflags.ListenTCPPortName))
	// UDP fell through to the default-zero override.
	require.Equal(t, uint(0), vcli.Uint(opnodeflags.ListenUDPPortName))
}

func TestWithNamespacedP2P_CreatesP2PDir(t *testing.T) {
	dataDir := t.TempDir()
	ns := fmt.Sprintf("%d", testChainID)
	require.NoError(t, withNamespacedP2P(newVCLI(t, nil), dataDir, ns))

	expected := filepath.Join(dataDir, ns, "p2p")
	info, err := os.Stat(expected)
	require.NoError(t, err)
	require.True(t, info.IsDir(), "expected p2p dir at %s", expected)
}

func TestWithNoP2P_AlwaysForcesListenPortsToZero(t *testing.T) {
	// Even when the user pinned ports, withNoP2P should zero them out because
	// P2P is being disabled outright.
	tcpName := fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenTCPPortName)
	udpName := fmt.Sprintf("%s%d.%s", flags.VNFlagNamePrefix, testChainID, opnodeflags.ListenUDPPortName)
	vcli := newVCLI(t, []string{
		"--" + tcpName + "=9000",
		"--" + udpName + "=9001",
	})
	require.NoError(t, withNoP2P(vcli))

	require.Equal(t, uint(0), vcli.Uint(opnodeflags.ListenTCPPortName))
	require.Equal(t, uint(0), vcli.Uint(opnodeflags.ListenUDPPortName))
}
