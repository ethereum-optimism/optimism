package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

const (
	piTestKey   = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	piTestHashA = "0x1111111111111111111111111111111111111111111111111111111111111111"
	piTestHashB = "0x2222222222222222222222222222222222222222222222222222222222222222"
)

func validPrivateInteropCLIConfig() PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		PrivateChainGenesisPath: "/etc/private-chain-genesis.json",
		PublicProjectionRPC:     "http://public-projection:8545",
		MaxBlocksPerRange:       300,
		MaxRangeBytes:           512 * 1024,
		RollupConfigHash:        piTestHashA,
		DepSetHash:              piTestHashB,
		GasLimitExport:          500_000,
		GasLimitImport:          500_000,
		GasLimitEvent:           500_000,
		GasLimitClaim:           500_000,
	}
}

// TestPrivateInteropConfigCheck is the validation table. Every row is a way the operator can
// misconfigure the group, and the point of each is that it fails at STARTUP rather than becoming a
// zero value inside bytes that go on L1.
func TestPrivateInteropConfigCheck(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(c *PrivateInteropCLIConfig)
		err    string
	}{
		{"valid", func(*PrivateInteropCLIConfig) {}, ""},
		{"no private genesis", func(c *PrivateInteropCLIConfig) { c.PrivateChainGenesisPath = "" }, "private-interop.genesis"},
		{"no public projection rpc", func(c *PrivateInteropCLIConfig) { c.PublicProjectionRPC = "" }, "public-projection-rpc"},
		{"zero cadence", func(c *PrivateInteropCLIConfig) { c.MaxBlocksPerRange = 0 }, "max-blocks-per-range"},
		{"zero range bytes", func(c *PrivateInteropCLIConfig) { c.MaxRangeBytes = 0 }, "max-range-bytes"},

		{
			"zero rollup config hash",
			func(c *PrivateInteropCLIConfig) {
				c.RollupConfigHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
			},
			"rollup-config-hash is the zero hash",
		},
		{"short dep set hash", func(c *PrivateInteropCLIConfig) { c.DepSetHash = "0x1234" }, "dep-set-hash is not a 32-byte hash"},
		{"claim gas below intrinsic", func(c *PrivateInteropCLIConfig) { c.GasLimitClaim = 20_999 }, "gas-limit-claim"},
		{"export gas below intrinsic", func(c *PrivateInteropCLIConfig) { c.GasLimitExport = 0 }, "gas-limit-export"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validPrivateInteropCLIConfig()
			tc.mutate(&cfg)
			err := cfg.Check()
			if tc.err == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.err)
		})
	}
}

// TestPrivateInteropConfigResolve pins the typed form: what Check accepted is what the seam is
// built from, with no silent reinterpretation.
func TestPrivateInteropConfigResolve(t *testing.T) {
	cfg := validPrivateInteropCLIConfig()
	s, err := cfg.Resolve()
	require.NoError(t, err)

	require.Equal(t, predeploys.ClaimRegistryAddr, s.ClaimRegistry)
	require.Equal(t, predeploys.EventReplayerAddr, s.EventReplayer)
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, s.ReplayMessenger)
	require.Equal(t, common.HexToHash(piTestHashA), s.RollupConfigHash)
	require.Equal(t, common.HexToHash(piTestHashB), s.DepSetHash)
	require.Equal(t, uint64(500_000), s.Gas.GasLimitClaim)
	require.Equal(t, uint64(512*1024), s.MaxRangeBytes)
	// A resolve of an unchecked configuration is refused rather than silently zero-valued.
	bad := validPrivateInteropCLIConfig()
	bad.RollupConfigHash = "0x1234"
	_, err = bad.Resolve()
	require.Error(t, err)

	// Unset commitments resolve to zero, which the service replaces with derived values.
	derived := validPrivateInteropCLIConfig()
	derived.RollupConfigHash = ""
	derived.DepSetHash = ""
	require.NoError(t, derived.Check())
	ds, err := derived.Resolve()
	require.NoError(t, err)
	require.Equal(t, common.Hash{}, ds.RollupConfigHash)
	require.Equal(t, common.Hash{}, ds.DepSetHash)
}

// TestPrivateInteropFlagsParse drives the real CLI: the flag names, the group's registration in
// op-batcher's flag set, and NewConfig's read of it.
func TestPrivateInteropFlagsParse(t *testing.T) {
	var got *CLIConfig
	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Action = func(ctx *cli.Context) error {
		got = NewConfig(ctx)
		return nil
	}
	require.NoError(t, app.Run([]string{"op-batcher",
		"--l1-eth-rpc=http://l1:8545",
		"--l2-eth-rpc=http://private-el:8545",
		"--rollup-rpc=http://private-node:9545",
		"--private-interop.genesis=/etc/private-chain-genesis.json",
		"--private-interop.public-projection-rpc=http://public-projection-el:8545",
		"--private-interop.max-blocks-per-range=300",
		"--private-interop.rollup-config-hash=" + piTestHashA,
		"--private-interop.dep-set-hash=" + piTestHashB,
		"--private-key=" + piTestKey,
	}))
	require.NotNil(t, got)
	pi := got.PrivateInterop
	require.Equal(t, "/etc/private-chain-genesis.json", pi.PrivateChainGenesisPath)
	require.Equal(t, "http://public-projection-el:8545", pi.PublicProjectionRPC)
	require.Equal(t, uint64(300), pi.MaxBlocksPerRange)
	require.NoError(t, pi.Check())

	// And a stock batcher run leaves the whole group inert.
	var stock *CLIConfig
	app2 := cli.NewApp()
	app2.Flags = flags.Flags
	app2.Action = func(ctx *cli.Context) error {
		stock = NewConfig(ctx)
		return nil
	}
	require.NoError(t, app2.Run([]string{"op-batcher",
		"--l1-eth-rpc=http://l1:8545", "--l2-eth-rpc=http://l2:8545", "--rollup-rpc=http://node:9545"}))
	require.Empty(t, stock.PrivateInterop.PrivateChainGenesisPath)
}
