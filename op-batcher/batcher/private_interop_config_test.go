package batcher

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/params"
)

const (
	piTestKey   = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	piTestHashA = "0x1111111111111111111111111111111111111111111111111111111111111111"
	piTestHashB = "0x2222222222222222222222222222222222222222222222222222222222222222"
)

func validPrivateInteropCLIConfig() PrivateInteropCLIConfig {
	return PrivateInteropCLIConfig{
		PrivateChainGenesisPath:   "/etc/private-chain-genesis.json",
		PublicProjectionRPC:       "http://public-projection:8545",
		PublicProjectionRollupRPC: "http://public-projection:9545",
		MaxBlocksPerRange:         300,
		MaxRangeBytes:             512 * 1024,
		RollupConfigHash:          piTestHashA,
		DepSetHash:                piTestHashB,
		GasLimitExport:            500_000,
		GasLimitImport:            500_000,
		GasLimitEvent:             500_000,
		GasLimitClaim:             500_000,
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
		{"no public projection rollup rpc", func(c *PrivateInteropCLIConfig) { c.PublicProjectionRollupRPC = "" }, "public-projection-rollup-rpc"},
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
		"--private-interop.public-projection-rollup-rpc=http://public-projection-node:9545",
		"--private-interop.max-blocks-per-range=300",
		"--private-interop.rollup-config-hash=" + piTestHashA,
		"--private-interop.dep-set-hash=" + piTestHashB,
		"--private-key=" + piTestKey,
	}))
	require.NotNil(t, got)
	pi := got.PrivateInterop
	require.Equal(t, "/etc/private-chain-genesis.json", pi.PrivateChainGenesisPath)
	require.Equal(t, "http://public-projection-el:8545", pi.PublicProjectionRPC)
	require.Equal(t, "http://public-projection-node:9545", pi.PublicProjectionRollupRPC)
	require.Equal(t, uint64(300), pi.MaxBlocksPerRange)
	require.NoError(t, pi.Check())
	// The sound-profile flags and their defaults.
	require.Equal(t, "network", pi.SP1Prover)
	require.Equal(t, 2*time.Minute, pi.ProofTimeout)
	require.Equal(t, 100*time.Millisecond, pi.ProofTimeoutPerBlock)
	require.Empty(t, pi.PrivateRollupConfigPath)

	var sound *CLIConfig
	app3 := cli.NewApp()
	app3.Flags = flags.Flags
	app3.Action = func(ctx *cli.Context) error {
		sound = NewConfig(ctx)
		return nil
	}
	require.NoError(t, app3.Run([]string{"op-batcher",
		"--l1-eth-rpc=http://l1:8545", "--l2-eth-rpc=http://l2:8545", "--rollup-rpc=http://node:9545",
		"--private-interop.genesis=/etc/private-chain-genesis.json",
		"--private-interop.public-projection-rpc=http://public-projection-el:8545",
		"--private-interop.public-projection-rollup-rpc=http://public-projection-node:9545",
		"--private-interop.private-rollup-config=/etc/private-rollup.json",
		"--private-interop.l1-chain-config=/etc/l1-chain-config.json",
		"--private-interop.sp1-prover=native-mock",
		"--private-interop.proof-timeout=20m",
		"--private-interop.proof-timeout-per-block=250ms",
		"--private-interop.proof-command=/usr/bin/kona-sp1-private-projection-executor",
		"--private-key=" + piTestKey,
	}))
	spi := sound.PrivateInterop
	require.NoError(t, spi.Check())
	require.Equal(t, "/etc/private-rollup.json", spi.PrivateRollupConfigPath)
	require.Equal(t, "/etc/l1-chain-config.json", spi.L1ChainConfigPath)
	require.Equal(t, "native-mock", spi.SP1Prover)
	require.Equal(t, 20*time.Minute, spi.ProofTimeout)
	require.Equal(t, 250*time.Millisecond, spi.ProofTimeoutPerBlock)

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

func TestPrivateInteropSoundProfileFlagsCheck(t *testing.T) {
	c := validPrivateInteropCLIConfig()
	c.SP1Prover = "gpu"
	require.ErrorContains(t, c.Check(), "sp1-prover")
	c = validPrivateInteropCLIConfig()
	c.PrivateRollupConfigPath = "/etc/private-rollup.json"
	require.ErrorContains(t, c.Check(), "given together")
	c = validPrivateInteropCLIConfig()
	c.ProofTimeout = -time.Second
	require.ErrorContains(t, c.Check(), "negative")
	// Programmatic configs (the devstack) may leave the defaults unset.
	c = validPrivateInteropCLIConfig()
	s, err := c.Resolve()
	require.NoError(t, err)
	require.Equal(t, ProverNetwork, s.SP1Prover)
	require.Equal(t, flags.DefaultPrivateInteropProofTimeout, s.ProofTimeout)
	require.Equal(t, flags.DefaultPrivateInteropProofTimeoutPerBlock, s.ProofTimeoutPerBlock)
}

// piProfileFixture is a deployed sp1-private-projection-v1 profile and the matching local view.
type piProfileFixture struct {
	in privateInteropProfileInputs
}

func newPIProfileFixture(t *testing.T) *piProfileFixture {
	t.Helper()
	private := piRollupCfg()
	private.L1ChainID = params.MergedTestChainConfig.ChainID
	privateJSON, err := json.Marshal(private)
	require.NoError(t, err)
	l1JSON, err := json.Marshal(params.MergedTestChainConfig)
	require.NoError(t, err)
	depSet := []eth.ChainID{eth.ChainIDFromUInt64(901), eth.ChainIDFromUInt64(902)}
	local := piRollupCfg()
	local.Genesis.L2.Hash = common.Hash{0x9e}
	local.PrivateProjection = &projection.Config{Verifier: projection.InsecureStub, GenesisOutputRoot: common.Hash{0x60}}
	deployed := *local
	deployed.PrivateProjection = &projection.Config{
		Verifier:          projection.SP1PrivateProjectionV1,
		GenesisOutputRoot: common.Hash{0x60},
		ProgramVKey:       common.Hash{31: 1},
		PrivateConfigHash: projection.PrivateConfigHash(privateJSON, l1JSON),
		DependencySetHash: projection.DependencySetHash(depSet),
		MockProofs:        true,
	}
	settings := &PrivateInteropSettings{ProofCommand: "/usr/bin/producer", SP1Prover: ProverNativeMock}
	return &piProfileFixture{in: privateInteropProfileInputs{
		Settings: settings, PrivateRollup: private, Local: local, Deployed: &deployed,
		DependencySet: depSet, PrivateRollupJSON: privateJSON, L1ChainConfigJSON: l1JSON,
	}}
}

// TestResolvePrivateInteropProfile: the batcher takes private_projection from the deployed config,
// derives both claim commitments from it, and refuses to start on any mismatch.
func TestResolvePrivateInteropProfile(t *testing.T) {
	f := newPIProfileFixture(t)
	got, err := resolvePrivateInteropProfile(f.in)
	require.NoError(t, err)
	pp := f.in.Deployed.PrivateProjection
	require.Equal(t, pp, got.Rollup.PrivateProjection)
	require.Equal(t, projection.ConfigHash(pp, projection.Context{
		ChainID: got.Rollup.L2ChainID, GenesisTime: got.Rollup.Genesis.L2Time, BlockTime: got.Rollup.BlockTime,
		GenesisHash: got.Rollup.Genesis.L2.Hash,
	}), got.RollupConfigHash)
	require.Equal(t, pp.DependencySetHash, got.DepSetHash)
	require.Equal(t, ProverNativeMock, got.Prover)
	require.Equal(t, f.in.PrivateRollupJSON, got.PrivateConfigJSON, "the pinned bytes, never a re-marshal")
	require.Equal(t, f.in.L1ChainConfigJSON, got.L1ConfigJSON)

	// A matching pin of either commitment is accepted.
	f = newPIProfileFixture(t)
	f.in.Settings.RollupConfigHash, f.in.Settings.DepSetHash = got.RollupConfigHash, got.DepSetHash
	_, err = resolvePrivateInteropProfile(f.in)
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		mutate func(f *piProfileFixture)
		err    string
	}{
		{"mismatched private config file", func(f *piProfileFixture) {
			f.in.L1ChainConfigJSON = append(append([]byte{}, f.in.L1ChainConfigJSON...), ' ')
		}, "private_config_hash"},
		{"mismatched rollup-config-hash flag", func(f *piProfileFixture) {
			f.in.Settings.RollupConfigHash = common.Hash{0x1b}
		}, "rollup-config-hash"},
		{"mismatched dep-set-hash flag", func(f *piProfileFixture) {
			f.in.Settings.DepSetHash = common.Hash{0x1c}
		}, "dep-set-hash"},
		{"mock prover without mock_proofs", func(f *piProfileFixture) {
			f.in.Deployed.PrivateProjection.MockProofs = false
			f.in.Settings.SP1Prover = ProverMock
		}, "needs mock_proofs"},
		{"native-mock prover without mock_proofs", func(f *piProfileFixture) {
			f.in.Deployed.PrivateProjection.MockProofs = false
		}, "needs mock_proofs"},
		{"deployed config differs", func(f *piProfileFixture) { f.in.Deployed.BlockTime = 3 }, "differs from the one projected"},
		{"deployed genesis output differs", func(f *piProfileFixture) {
			f.in.Deployed.PrivateProjection.GenesisOutputRoot = common.Hash{0x61}
		}, "genesis_output_root"},
		{"dependency set differs", func(f *piProfileFixture) {
			f.in.DependencySet = f.in.DependencySet[:1]
		}, "dependency set hashes"},
		{"sp1 without pinned files", func(f *piProfileFixture) {
			f.in.PrivateRollupJSON, f.in.L1ChainConfigJSON = nil, nil
		}, "requires --private-interop.private-rollup-config"},
		{"pinned file for another chain", func(f *piProfileFixture) {
			other := piRollupCfg()
			other.L2ChainID = big902()
			raw, err := json.Marshal(other)
			require.NoError(t, err)
			f.in.PrivateRollupJSON = raw
		}, "not this batcher's private chain"},
		{"no proof command", func(f *piProfileFixture) { f.in.Settings.ProofCommand = "" }, "requires --private-interop.proof-command"},
		{"no deployed profile", func(f *piProfileFixture) { f.in.Deployed.PrivateProjection = nil }, "no private_projection"},
		{"deployed profile fails its gate", func(f *piProfileFixture) {
			f.in.Local.L2ChainID, f.in.Deployed.L2ChainID = big10(), big10()
		}, "deployed private_projection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newPIProfileFixture(t)
			tc.mutate(f)
			_, err := resolvePrivateInteropProfile(f.in)
			require.ErrorContains(t, err, tc.err)
		})
	}

	// execution-mock-v1: the producer runs natively, and the pinned files are optional.
	f = newPIProfileFixture(t)
	f.in.Deployed.PrivateProjection = &projection.Config{
		Verifier: projection.ExecutionMock, GenesisOutputRoot: common.Hash{0x60},
		DependencySetHash: projection.DependencySetHash(f.in.DependencySet),
	}
	f.in.PrivateRollupJSON, f.in.L1ChainConfigJSON = nil, nil
	got, err = resolvePrivateInteropProfile(f.in)
	require.NoError(t, err)
	require.Equal(t, ProverNative, got.Prover)
	var private rollup.Config
	require.NoError(t, json.Unmarshal(got.PrivateConfigJSON, &private))
	require.Equal(t, f.in.PrivateRollup.L2ChainID, private.L2ChainID)

	// insecure-stub-v1 takes no producer.
	f = newPIProfileFixture(t)
	f.in.Deployed.PrivateProjection = &projection.Config{
		Verifier: projection.InsecureStub, GenesisOutputRoot: common.Hash{0x60},
		DependencySetHash: projection.DependencySetHash(f.in.DependencySet),
	}
	_, err = resolvePrivateInteropProfile(f.in)
	require.ErrorContains(t, err, "takes no --private-interop.proof-command")
	f.in.Settings.ProofCommand = ""
	got, err = resolvePrivateInteropProfile(f.in)
	require.NoError(t, err)
	require.Empty(t, got.Prover)
}

func big902() *big.Int { return big.NewInt(902) }
func big10() *big.Int  { return big.NewInt(10) }
