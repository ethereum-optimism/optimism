package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

const (
	piTestKey      = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	piTestRegistry = "0x00000000000000000000000000000000000e9e9e"
	piTestReplayer = "0x00000000000000000000000000000000000e0e0e"
	piTestHashA    = "0x1111111111111111111111111111111111111111111111111111111111111111"
	piTestHashB    = "0x2222222222222222222222222222222222222222222222222222222222222222"
)

func validPrivateInteropCLIConfig() PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		Enabled:                   true,
		RenderingRollupConfigPath: "/etc/rendering-rollup.json",
		RenderingRPC:              "http://rendering:8545",
		MaxBlocksPerRange:         300,
		ClaimRegistry:             piTestRegistry,
		EventReplayer:             piTestReplayer,
		ReplayMessenger:           predeploys.L2toL2CrossDomainMessenger,
		RollupConfigHash:          piTestHashA,
		DepSetHash:                piTestHashB,
		OperatorKey:               piTestKey,
		GasLimitExport:            500_000,
		GasLimitImport:            500_000,
		GasLimitEvent:             500_000,
		GasLimitClaim:             500_000,
		GasFeeCap:                 1_000_000,
		GasTipCap:                 0,
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
		{
			"disabled ignores everything else",
			func(c *PrivateInteropCLIConfig) { *c = PrivateInteropCLIConfig{} },
			"",
		},
		{
			"disabled with garbage is still inert",
			func(c *PrivateInteropCLIConfig) { c.Enabled = false; c.ClaimRegistry = "not-an-address" },
			"",
		},
		{"no rendering rollup config", func(c *PrivateInteropCLIConfig) { c.RenderingRollupConfigPath = "" }, "rendering-rollup-config"},
		{"no rendering rpc", func(c *PrivateInteropCLIConfig) { c.RenderingRPC = "" }, "rendering-rpc"},
		{"zero cadence", func(c *PrivateInteropCLIConfig) { c.MaxBlocksPerRange = 0 }, "max-blocks-per-range"},
		{"no claim registry", func(c *PrivateInteropCLIConfig) { c.ClaimRegistry = "" }, "claim-registry is required"},
		{
			"zero claim registry",
			func(c *PrivateInteropCLIConfig) { c.ClaimRegistry = "0x0000000000000000000000000000000000000000" },
			"claim-registry is the zero address",
		},
		{"malformed claim registry", func(c *PrivateInteropCLIConfig) { c.ClaimRegistry = "0xdeadbeef" }, "is not an address"},
		{
			"zero event replayer",
			func(c *PrivateInteropCLIConfig) { c.EventReplayer = "0x0000000000000000000000000000000000000000" },
			"event-replayer is the zero address",
		},
		{
			"zero replay messenger",
			func(c *PrivateInteropCLIConfig) { c.ReplayMessenger = "0x0000000000000000000000000000000000000000" },
			"replay-messenger is the zero address",
		},
		{"malformed extra emitter", func(c *PrivateInteropCLIConfig) { c.ExtraEmitters = []string{"nope"} }, "extra-emitters is not an address"},
		{
			"good extra emitter",
			func(c *PrivateInteropCLIConfig) { c.ExtraEmitters = []string{piTestReplayer} },
			"",
		},
		{"no rollup config hash", func(c *PrivateInteropCLIConfig) { c.RollupConfigHash = "" }, "rollup-config-hash is required"},
		{
			"zero rollup config hash",
			func(c *PrivateInteropCLIConfig) {
				c.RollupConfigHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
			},
			"rollup-config-hash is the zero hash",
		},
		{"short dep set hash", func(c *PrivateInteropCLIConfig) { c.DepSetHash = "0x1234" }, "dep-set-hash is not a 32-byte hash"},
		{"no operator key", func(c *PrivateInteropCLIConfig) { c.OperatorKey = "" }, "operator-key is required"},
		{"bad operator key", func(c *PrivateInteropCLIConfig) { c.OperatorKey = "zz" }, "not a valid private key"},
		{"claim gas below intrinsic", func(c *PrivateInteropCLIConfig) { c.GasLimitClaim = 20_999 }, "gas-limit-claim"},
		{"export gas below intrinsic", func(c *PrivateInteropCLIConfig) { c.GasLimitExport = 0 }, "gas-limit-export"},
		{"zero fee cap", func(c *PrivateInteropCLIConfig) { c.GasFeeCap = 0 }, "gas-fee-cap must be greater than zero"},
		{"tip above cap", func(c *PrivateInteropCLIConfig) { c.GasTipCap = c.GasFeeCap + 1 }, "exceeds"},
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
	cfg.ExtraEmitters = []string{piTestReplayer}
	s, err := cfg.Resolve()
	require.NoError(t, err)

	require.Equal(t, common.HexToAddress(piTestRegistry), s.ClaimRegistry)
	require.Equal(t, common.HexToAddress(piTestReplayer), s.EventReplayer)
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, s.ReplayMessenger)
	require.Equal(t, common.HexToHash(piTestHashA), s.RollupConfigHash)
	require.Equal(t, common.HexToHash(piTestHashB), s.DepSetHash)
	require.NotNil(t, s.OperatorKey)
	require.Equal(t, uint64(500_000), s.Gas.GasLimitClaim)
	require.Equal(t, uint64(1_000_000), s.Gas.GasFeeCap.Uint64())
	require.Equal(t, uint64(0), s.Gas.GasTipCap.Uint64())
	// The extra emitter is in the set, and a random address is not.
	require.True(t, s.Emitters.Renders(&types.Log{Address: common.HexToAddress(piTestReplayer)}))
	require.False(t, s.Emitters.Renders(&types.Log{Address: common.HexToAddress("0x00000000000000000000000000000000000abcde")}))

	// A resolve of an unchecked configuration is refused rather than silently zero-valued.
	bad := validPrivateInteropCLIConfig()
	bad.ClaimRegistry = ""
	_, err = bad.Resolve()
	require.Error(t, err)
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
		"--private-interop.enabled",
		"--private-interop.rendering-rollup-config=/etc/rendering-rollup.json",
		"--private-interop.rendering-rpc=http://rendering-el:8545",
		"--private-interop.max-blocks-per-range=300",
		"--private-interop.claim-registry=" + piTestRegistry,
		"--private-interop.event-replayer=" + piTestReplayer,
		"--private-interop.extra-emitters=" + piTestReplayer,
		"--private-interop.rollup-config-hash=" + piTestHashA,
		"--private-interop.dep-set-hash=" + piTestHashB,
		"--private-interop.operator-key=" + piTestKey,
	}))
	require.NotNil(t, got)
	pi := got.PrivateInterop
	require.True(t, pi.Enabled)
	require.Equal(t, "http://rendering-el:8545", pi.RenderingRPC)
	require.Equal(t, uint64(300), pi.MaxBlocksPerRange)
	require.Equal(t, predeploys.L2toL2CrossDomainMessenger, pi.ReplayMessenger)
	require.Equal(t, []string{piTestReplayer}, pi.ExtraEmitters)
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
	require.False(t, stock.PrivateInterop.Enabled)
	require.NoError(t, stock.PrivateInterop.Check())
}
