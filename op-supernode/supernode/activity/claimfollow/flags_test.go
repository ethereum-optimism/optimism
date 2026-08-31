package claimfollow

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
)

const (
	testRegistryHex    = "0x4200000000000000000000000000000000000777"
	testGenesisHashHex = "0x00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
)

// run drives the REAL supernode CLI: the flag group's registration through
// flags.RegisterActivityFlags, its names, and the read that turns them into a config.
func run(t *testing.T, args ...string) CLIConfig {
	t.Helper()
	var got CLIConfig
	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.FullDynamicFlags([]uint64{424243}))
	app.Action = func(ctx *cli.Context) error {
		got = ReadCLIConfig(ctx)
		return nil
	}
	require.NoError(t, app.Run(append([]string{"op-supernode"}, args...)))
	return got
}

// THE DORMANCY GATE. A stock supernode invocation leaves the whole group inert: nothing is enabled,
// nothing validates as configured, and Resolve refuses. Nothing else in the supernode reads any of
// this, so an unset group cannot change a single byte of behaviour.
func TestDormantWhenUnset(t *testing.T) {
	cfg := run(t, "--chains=424243", "--l1=http://l1:8545")
	require.False(t, cfg.Enabled)
	require.Zero(t, cfg.ChainID)
	require.Empty(t, cfg.ClaimRegistry)
	require.Empty(t, cfg.GenesisHash)
	require.Zero(t, cfg.ScanStartBlock)
	// The route flag has a default, which is what makes the enabled path need no extra argument.
	// It is inert all the same: nothing consults it while Enabled is false.
	require.Equal(t, DefaultRoute, cfg.Route)

	// A disabled group always passes Check, whatever is in it: refusing to start over a stale value
	// in an unused group would be a worse failure than ignoring it.
	require.NoError(t, cfg.Check())
	require.NoError(t, CLIConfig{ClaimRegistry: "nonsense", Route: "a/b"}.Check())

	_, _, _, err := cfg.Resolve()
	require.Error(t, err, "a dormant group resolves to nothing")
}

// A nil CLI context — an embedder that never built one — reads as disabled rather than panicking.
func TestNilContextIsDisabled(t *testing.T) {
	cfg := ReadCLIConfig(nil)
	require.False(t, cfg.Enabled)
	require.NoError(t, cfg.Check())
}

func TestEnabledGroupParsesAndResolves(t *testing.T) {
	cfg := run(t,
		"--chains=424243", "--l1=http://l1:8545",
		"--private-interop.enabled",
		"--private-interop.chain-id=424243",
		"--private-interop.claim-registry="+testRegistryHex,
		"--private-interop.genesis-hash="+testGenesisHashHex,
		"--private-interop.claim-scan-start-block=100",
	)
	require.True(t, cfg.Enabled)
	require.Equal(t, uint64(424243), cfg.ChainID)
	require.Equal(t, testRegistryHex, cfg.ClaimRegistry)
	require.Equal(t, testGenesisHashHex, cfg.GenesisHash)
	require.Equal(t, uint64(100), cfg.ScanStartBlock)
	require.Equal(t, DefaultRoute, cfg.Route)
	require.NoError(t, cfg.Check())

	chainID, modCfg, route, err := cfg.Resolve()
	require.NoError(t, err)
	require.Equal(t, uint64(424243), chainID.ToBig().Uint64())
	require.Equal(t, common.HexToAddress(testRegistryHex), modCfg.Registry)
	require.Equal(t, common.HexToHash(testGenesisHashHex), modCfg.GenesisHash)
	require.Equal(t, uint64(100), modCfg.StartBlock)
	require.Equal(t, "/claimed", route, "the route is one path segment under the chain's own")
}

func TestRouteIsOverridable(t *testing.T) {
	cfg := run(t,
		"--chains=424243", "--l1=http://l1:8545",
		"--private-interop.enabled",
		"--private-interop.chain-id=424243",
		"--private-interop.claim-registry="+testRegistryHex,
		"--private-interop.genesis-hash="+testGenesisHashHex,
		"--private-interop.route=private",
	)
	_, _, route, err := cfg.Resolve()
	require.NoError(t, err)
	require.Equal(t, "/private", route)
}

// Enabled, the group is ALL-OR-NOTHING: there is no half-configured module, because a module that
// scanned the wrong address would serve confident refs about nothing.
func TestCheckRefusesAHalfConfiguredGroup(t *testing.T) {
	base := CLIConfig{Enabled: true, ChainID: 424243, ClaimRegistry: testRegistryHex,
		GenesisHash: testGenesisHashHex, Route: DefaultRoute}
	require.NoError(t, base.Check())

	for _, tc := range []struct {
		name   string
		mutate func(c *CLIConfig)
		want   string
	}{
		{"no chain id", func(c *CLIConfig) { c.ChainID = 0 }, "chain-id"},
		{"no registry", func(c *CLIConfig) { c.ClaimRegistry = "" }, "claim-registry"},
		{"registry is not an address", func(c *CLIConfig) { c.ClaimRegistry = "0xnope" }, "hex address"},
		{"registry is the zero address", func(c *CLIConfig) {
			c.ClaimRegistry = "0x0000000000000000000000000000000000000000"
		}, "zero address"},
		// The genesis hash is REQUIRED, and it is required because the alternative is a bootstrap
		// deadlock: a module that serves nothing leaves the op-node it feeds with a zero current_l1,
		// and the operator's batcher will not load a block without one -- so the claim that would end
		// the not-yet state never gets built. A startup failure is the cheap version of that.
		{"no genesis hash", func(c *CLIConfig) { c.GenesisHash = "" }, "genesis-hash is required"},
		{"genesis hash is not a hash", func(c *CLIConfig) { c.GenesisHash = "0xnope" }, "32-byte hex hash"},
		// common.HexToHash left-pads anything short and truncates anything long, so a typo would
		// silently become a DIFFERENT valid-looking hash and the module would vouch for a block that
		// does not exist. The length is checked rather than assumed.
		{"genesis hash is too short", func(c *CLIConfig) { c.GenesisHash = "0xdeadbeef" }, "32-byte hex hash"},
		{"genesis hash is too long", func(c *CLIConfig) { c.GenesisHash = testGenesisHashHex + "00" }, "32-byte hex hash"},
		{"genesis hash is zero", func(c *CLIConfig) {
			c.GenesisHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
		}, "zero hash"},
		{"empty route", func(c *CLIConfig) { c.Route = "" }, "must not be empty"},
		{"route with a slash", func(c *CLIConfig) { c.Route = "chain/claimed" }, "single path segment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.mutate(&c)
			err := c.Check()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// The group reaches the supernode's flag set through RegisterActivityFlags, which is the idiom an
// activity uses to own its own flags without the config package importing it.
func TestFlagsAreRegisteredWithTheSupernode(t *testing.T) {
	all := flags.FullDynamicFlags([]uint64{424243})
	names := map[string]bool{}
	for _, f := range all {
		names[f.Names()[0]] = true
	}
	for _, f := range Flags {
		require.True(t, names[f.Names()[0]], "flag %s is not registered with op-supernode", f.Names()[0])
	}
}
