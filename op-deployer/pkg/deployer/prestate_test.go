package deployer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestPrestateRoleSourceResolution(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedDefault := testPrestate("11")
	fallbackDefault := testPrestate("22")
	valueA := testPrestate("a3")
	valueB := testPrestate("44")

	tests := []struct {
		name    string
		command string
		global  string
		chain   string
		want    string
		wantErr bool
	}{
		{name: "command only", command: valueA, want: valueA},
		{name: "global override only", global: valueA, want: valueA},
		{name: "chain override only", chain: valueA, want: valueA},
		{name: "chain override shadows global", global: valueA, chain: valueB, want: valueB},
		{name: "command and global agree", command: valueA, global: strings.ToUpper(valueA[2:4]) + valueA[4:], want: valueA},
		{name: "command and global conflict", command: valueA, global: valueB, wantErr: true},
		{name: "command and chain agree", command: valueA, chain: valueA, want: valueA},
		{name: "command and chain conflict", command: valueA, chain: valueB, wantErr: true},
	}

	roles := []struct {
		name       string
		errName    string
		intentKey  string
		configure  func(*PrestateConfig, string)
		otherKey   string
		otherValue string
		check      func(*state.ChainState) common.Hash
	}{
		{
			name:      "selected",
			errName:   "selected",
			intentKey: faultGameAbsolutePrestateOverride,
			configure: func(cfg *PrestateConfig, value string) { cfg.Prestate = value },
			otherKey:  cannonFallbackPrestateOverride, otherValue: fallbackDefault,
			check: func(chain *state.ChainState) common.Hash { return chain.Prestate },
		},
		{
			name:      "fallback",
			errName:   "Cannon fallback",
			intentKey: cannonFallbackPrestateOverride,
			configure: func(cfg *PrestateConfig, value string) { cfg.CannonFallbackPrestate = value },
			otherKey:  faultGameAbsolutePrestateOverride, otherValue: selectedDefault,
			check: func(chain *state.ChainState) common.Hash { return chain.CannonFallbackPrestate },
		},
	}

	for _, role := range roles {
		for _, test := range tests {
			t.Run(role.name+"/"+test.name, func(t *testing.T) {
				global := map[string]any{role.otherKey: role.otherValue}
				if test.global != "" {
					global[role.intentKey] = with0xPrefix(test.global)
				}
				overrides := map[string]any{"respectedGameType": embedded.GameTypeCannonKona}
				if test.chain != "" {
					overrides[role.intentKey] = test.chain
				}
				workdir := writePrestateWorkdir(t, global, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
				cfg := newTestPrestateConfig(t, workdir)
				if test.command != "" {
					role.configure(&cfg, test.command)
				}

				err := Prestate(context.Background(), cfg)
				if test.wantErr {
					require.ErrorContains(t, err, "conflicting "+role.errName)
					require.ErrorContains(t, err, chainID.Hex())
					return
				}
				require.NoError(t, err)
				got := readPrestateChain(t, workdir, chainID)
				require.Equal(t, common.HexToHash(test.want), role.check(got))
			})
		}
	}
}

func TestPrestateStrictValidationAcrossRolesAndSources(t *testing.T) {
	chainID := common.HexToHash("0x01")
	validSelected := testPrestate("11")
	validFallback := testPrestate("22")
	invalidValues := []struct {
		name    string
		value   any
		wantErr string
	}{
		{name: "short", value: "0xabc123", wantErr: "exactly 64 hex characters"},
		{name: "overlong", value: validSelected + "11", wantErr: "exactly 64 hex characters"},
		{name: "malformed", value: "0x" + strings.Repeat("11", 31) + "zz", wantErr: "valid hex"},
		{name: "zero", value: "0x" + strings.Repeat("00", 32), wantErr: "must not be zero"},
		{name: "missing prefix", value: strings.Repeat("11", 32), wantErr: "must start with 0x"},
		{name: "explicit empty", value: "", wantErr: "must start with 0x"},
		{name: "whitespace", value: " " + validSelected, wantErr: "must start with 0x"},
		{name: "non-string", value: int64(7), wantErr: "must be a string"},
	}

	roles := []struct {
		name      string
		key       string
		setConfig func(*PrestateConfig, any)
	}{
		{
			name: "selected", key: faultGameAbsolutePrestateOverride,
			setConfig: func(cfg *PrestateConfig, value any) {
				if raw, ok := value.(string); ok {
					cfg.Prestate = raw
				}
				cfg.PrestateSet = true
			},
		},
		{
			name: "fallback", key: cannonFallbackPrestateOverride,
			setConfig: func(cfg *PrestateConfig, value any) {
				if raw, ok := value.(string); ok {
					cfg.CannonFallbackPrestate = raw
				}
				cfg.CannonFallbackPrestateSet = true
			},
		},
	}

	for _, role := range roles {
		for _, source := range []string{"command", "chain override", "global override"} {
			for _, invalid := range invalidValues {
				if source == "command" && invalid.name == "non-string" {
					continue
				}
				t.Run(role.name+"/"+source+"/"+invalid.name, func(t *testing.T) {
					global := map[string]any{
						faultGameAbsolutePrestateOverride: validSelected,
						cannonFallbackPrestateOverride:    validFallback,
					}
					overrides := map[string]any{"respectedGameType": embedded.GameTypeCannonKona}
					cfgValue := false
					switch source {
					case "command":
						delete(global, role.key)
						cfgValue = true
					case "chain override":
						overrides[role.key] = invalid.value
					case "global override":
						global[role.key] = invalid.value
					}
					workdir := writePrestateWorkdir(t, global, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
					cfg := newTestPrestateConfig(t, workdir)
					if cfgValue {
						role.setConfig(&cfg, invalid.value)
					}
					err := Prestate(context.Background(), cfg)
					require.Error(t, err)
					require.ErrorContains(t, err, invalid.wantErr)
				})
			}
		}
	}
}

func TestPrestateGameTypeRequirements(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	selected := testPrestate("11")
	fallback := testPrestate("22")
	staleSelected := common.HexToHash(testPrestate("aa"))
	staleFallback := common.HexToHash(testPrestate("bb"))

	tests := []struct {
		name         string
		global       map[string]any
		chains       []prestateTestChain
		configure    func(*PrestateConfig)
		want         map[common.Hash][2]common.Hash
		wantErrParts []string
	}{
		{
			name:   "permissioned only needs no PCD source and clears stale values",
			chains: []prestateTestChain{{id: chainA, prepared: true, initialSelected: staleSelected, initialFallback: staleFallback}},
			want:   map[common.Hash][2]common.Hash{chainA: {}},
		},
		{
			name:   "legacy permissioned absolute prestate is not copied",
			chains: []prestateTestChain{{id: chainA, prepared: true, overrides: map[string]any{faultGameAbsolutePrestateOverride: selected}}},
			want:   map[common.Hash][2]common.Hash{chainA: {}},
		},
		{
			name:      "CANNON_KONA commits distinct pair",
			chains:    []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			configure: commandPair(selected, fallback),
			want:      map[common.Hash][2]common.Hash{chainA: {common.HexToHash(selected), common.HexToHash(fallback)}},
		},
		{
			name:         "CANNON_KONA rejects missing selected",
			global:       map[string]any{cannonFallbackPrestateOverride: fallback},
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			wantErrParts: []string{"CANNON_KONA", "requires selected prestate"},
		},
		{
			name:         "CANNON_KONA rejects missing fallback",
			global:       map[string]any{faultGameAbsolutePrestateOverride: selected},
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			wantErrParts: []string{"CANNON_KONA", "requires Cannon fallback prestate"},
		},
		{
			name:         "CANNON_KONA rejects equal pair",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			configure:    commandPair(selected, selected),
			wantErrParts: []string{"different selected and Cannon fallback prestates", selected},
		},
		{
			name:      "SUPER_CANNON_KONA commits selected and clears stale fallback",
			chains:    []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialFallback: staleFallback}},
			configure: func(cfg *PrestateConfig) { cfg.Prestate = selected },
			want:      map[common.Hash][2]common.Hash{chainA: {common.HexToHash(selected), common.Hash{}}},
		},
		{
			name:         "SUPER_CANNON_KONA rejects unconsumed fallback flag",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)}},
			configure:    commandPair(selected, fallback),
			wantErrParts: []string{"--" + CannonFallbackPrestateFlagName, "no undeployed chain resolves to CANNON_KONA"},
		},
		{
			name:         "permissioned only rejects unconsumed selected flag",
			chains:       []prestateTestChain{{id: chainA, prepared: true}},
			configure:    func(cfg *PrestateConfig) { cfg.Prestate = selected },
			wantErrParts: []string{"--" + PrestateFlagName, "no undeployed chain resolves", "respectedGameType"},
		},
		{
			name:         "permissioned only rejects unconsumed fallback flag",
			chains:       []prestateTestChain{{id: chainA, prepared: true}},
			configure:    func(cfg *PrestateConfig) { cfg.CannonFallbackPrestate = fallback },
			wantErrParts: []string{"--" + CannonFallbackPrestateFlagName, "no undeployed chain resolves", "respectedGameType"},
		},
		{
			name:         "misspelled respectedGameType override rejects supplied pair",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: map[string]any{"respectedGamType": embedded.GameTypeCannonKona}}},
			configure:    commandPair(selected, fallback),
			wantErrParts: []string{"--" + PrestateFlagName, "no undeployed chain resolves", "respectedGameType"},
		},
		{
			name:         "unsupported initial type fails",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannon)}},
			wantErrParts: []string{"unsupported initial dispute game type", "0"},
		},
		{
			name: "permissioned plus CANNON_KONA succeeds",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, initialSelected: staleSelected, initialFallback: staleFallback},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: commandPair(selected, fallback),
			want: map[common.Hash][2]common.Hash{
				chainA: {},
				chainB: {common.HexToHash(selected), common.HexToHash(fallback)},
			},
		},
		{
			name: "CANNON_KONA plus SUPER_CANNON_KONA fails",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
			},
			configure:    commandPair(selected, fallback),
			wantErrParts: []string{"cannot mix CANNON_KONA and SUPER_CANNON_KONA"},
		},
		{
			name: "multi-chain SUPER_CANNON_KONA fails",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
			},
			configure:    func(cfg *PrestateConfig) { cfg.Prestate = selected },
			wantErrParts: []string{"SUPER_CANNON_KONA", "multi-chain"},
		},
		{
			name: "deployed CANNON_KONA preserves prestates and ignores conflicting sources",
			chains: []prestateTestChain{
				{
					id:              chainA,
					prepared:        true,
					deployed:        true,
					overrides:       map[string]any{"respectedGameType": embedded.GameTypeCannonKona, faultGameAbsolutePrestateOverride: testPrestate("33"), cannonFallbackPrestateOverride: testPrestate("44")},
					initialSelected: staleSelected,
					initialFallback: staleFallback,
				},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: commandPair(selected, fallback),
			want: map[common.Hash][2]common.Hash{
				chainA: {staleSelected, staleFallback},
				chainB: {common.HexToHash(selected), common.HexToHash(fallback)},
			},
		},
		{
			name: "deployed CANNON_KONA needs no sources alongside active permissioned",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: staleSelected, initialFallback: staleFallback},
				{id: chainB, prepared: true, initialSelected: staleSelected, initialFallback: staleFallback},
			},
			want: map[common.Hash][2]common.Hash{
				chainA: {staleSelected, staleFallback},
				chainB: {},
			},
		},
		{
			name:         "selected flag with only deployed consumer is rejected",
			chains:       []prestateTestChain{{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: staleSelected, initialFallback: staleFallback}},
			configure:    func(cfg *PrestateConfig) { cfg.Prestate = selected },
			wantErrParts: []string{"--" + PrestateFlagName, "no undeployed chain resolves"},
		},
		{
			name:         "fallback flag with only deployed consumer is rejected",
			chains:       []prestateTestChain{{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: staleSelected, initialFallback: staleFallback}},
			configure:    func(cfg *PrestateConfig) { cfg.CannonFallbackPrestate = fallback },
			wantErrParts: []string{"--" + CannonFallbackPrestateFlagName, "no undeployed chain resolves"},
		},
		{
			name: "deployed unsupported game type is ignored",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannon), initialSelected: staleSelected, initialFallback: staleFallback},
				{id: chainB, prepared: true, initialSelected: staleSelected, initialFallback: staleFallback},
			},
			want: map[common.Hash][2]common.Hash{
				chainA: {staleSelected, staleFallback},
				chainB: {},
			},
		},
		{
			name: "deployed CANNON_KONA still conflicts with active SUPER_CANNON_KONA",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: staleSelected, initialFallback: staleFallback},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
			},
			configure:    func(cfg *PrestateConfig) { cfg.Prestate = selected },
			wantErrParts: []string{"cannot mix CANNON_KONA and SUPER_CANNON_KONA"},
		},
		{
			name: "deployed SUPER_CANNON_KONA is preserved alongside active permissioned",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: staleSelected, initialFallback: staleFallback},
				{id: chainB, prepared: true, initialSelected: staleSelected, initialFallback: staleFallback},
			},
			want: map[common.Hash][2]common.Hash{
				chainA: {staleSelected, staleFallback},
				chainB: {},
			},
		},
		{
			name:         "deployed malformed game type still fails resolution",
			chains:       []prestateTestChain{{id: chainA, prepared: true, deployed: true, overrides: map[string]any{"respectedGameType": "invalid"}, initialSelected: staleSelected, initialFallback: staleFallback}},
			wantErrParts: []string{"failed to resolve initial dispute game type", chainA.Hex()},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workdir := writePrestateWorkdir(t, test.global, test.chains, true, 1)
			cfg := newTestPrestateConfig(t, workdir)
			if test.configure != nil {
				test.configure(&cfg)
			}
			err := Prestate(context.Background(), cfg)
			if len(test.wantErrParts) > 0 {
				require.Error(t, err)
				for _, part := range test.wantErrParts {
					require.ErrorContains(t, err, part)
				}
				return
			}
			require.NoError(t, err)
			for chainID, want := range test.want {
				got := readPrestateChain(t, workdir, chainID)
				require.Equal(t, want[0], got.Prestate)
				require.Equal(t, want[1], got.CannonFallbackPrestate)
			}
		})
	}
}

func TestPrestateCommandValuesFanOutOnlyToApplicableChains(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	chainC := common.HexToHash("0x03")
	selected := testPrestate("11")
	fallback := testPrestate("22")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{
		{id: chainA, prepared: true},
		{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
		{id: chainC, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
	}, true, 1)
	cfg := newTestPrestateConfig(t, workdir)
	commandPair(selected, fallback)(&cfg)

	require.NoError(t, Prestate(context.Background(), cfg))
	require.Equal(t, common.Hash{}, readPrestateChain(t, workdir, chainA).Prestate)
	for _, chainID := range []common.Hash{chainB, chainC} {
		chain := readPrestateChain(t, workdir, chainID)
		require.Equal(t, common.HexToHash(selected), chain.Prestate)
		require.Equal(t, common.HexToHash(fallback), chain.CannonFallbackPrestate)
	}
}

func TestPrestatePreconditionsAndAtomicity(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	selected := testPrestate("11")
	fallback := testPrestate("22")
	oldSelected := common.HexToHash(testPrestate("aa"))
	oldFallback := common.HexToHash(testPrestate("bb"))

	tests := []struct {
		name      string
		chains    []prestateTestChain
		prepared  bool
		version   int
		global    map[string]any
		configure func(*PrestateConfig)
		wantErr   string
	}{
		{
			name: "unprepared state names prepare", prepared: false, version: 1,
			chains: []prestateTestChain{{id: chainA, prepared: true}}, wantErr: "op-deployer prepare",
		},
		{
			name: "unsupported state version", prepared: true, version: 2,
			chains: []prestateTestChain{{id: chainA, prepared: true}}, wantErr: "unsupported state version: 2",
		},
		{
			name: "unknown later chain creates no entry", prepared: true, version: 1,
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: oldSelected, initialFallback: oldFallback},
				{id: chainB, prepared: false, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: commandPair(selected, fallback), wantErr: "run op-deployer prepare",
		},
		{
			name: "conflict after earlier chain resolves", prepared: true, version: 1,
			global: map[string]any{faultGameAbsolutePrestateOverride: selected, cannonFallbackPrestateOverride: fallback},
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: oldSelected, initialFallback: oldFallback},
				{id: chainB, prepared: true, overrides: map[string]any{"respectedGameType": embedded.GameTypeCannonKona, cannonFallbackPrestateOverride: testPrestate("44")}},
			},
			configure: commandPair(selected, fallback), wantErr: "conflicting Cannon fallback prestate sources",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workdir := writePrestateWorkdir(t, test.global, test.chains, test.prepared, test.version)
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)
			cfg := newTestPrestateConfig(t, workdir)
			if test.configure != nil {
				test.configure(&cfg)
			}
			err = Prestate(context.Background(), cfg)
			require.ErrorContains(t, err, test.wantErr)
			after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})
	}
}

func TestPrestatePersistenceReplacementAndIdempotency(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedA := testPrestate("11")
	fallbackA := testPrestate("22")
	selectedB := testPrestate("33")
	fallbackB := testPrestate("44")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}}, true, 1)

	cfg := newTestPrestateConfig(t, workdir)
	commandPair(selectedA, fallbackA)(&cfg)
	require.NoError(t, Prestate(context.Background(), cfg))
	first, err := os.ReadFile(filepath.Join(workdir, "state.json"))
	require.NoError(t, err)
	require.NoError(t, Prestate(context.Background(), cfg))
	second, err := os.ReadFile(filepath.Join(workdir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, first, second)

	commandPair(selectedB, fallbackB)(&cfg)
	require.NoError(t, Prestate(context.Background(), cfg))
	chain := readPrestateChain(t, workdir, chainID)
	require.Equal(t, common.HexToHash(selectedB), chain.Prestate)
	require.Equal(t, common.HexToHash(fallbackB), chain.CannonFallbackPrestate)
}

func TestNewPrestateConfigFlagAndEnvironmentPrecedence(t *testing.T) {
	selectedEnv := testPrestate("11")
	selectedCLI := testPrestate("22")
	fallbackEnv := testPrestate("33")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", selectedEnv)
	t.Setenv("DEPLOYER_CANNON_FALLBACK_PRESTATE", fallbackEnv)

	cfg := parsePrestateCLIConfig(t, []string{"--" + PrestateFlagName, selectedCLI})
	require.Equal(t, selectedCLI, cfg.Prestate)
	require.True(t, cfg.PrestateSet)
	require.Equal(t, fallbackEnv, cfg.CannonFallbackPrestate)
	require.True(t, cfg.CannonFallbackPrestateSet)
}

func TestPrestateEnvironmentOnly(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selected := testPrestate("11")
	fallback := testPrestate("22")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", selected)
	t.Setenv("DEPLOYER_CANNON_FALLBACK_PRESTATE", fallback)
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}}, true, 1)
	cfg := parsePrestateCLIConfig(t, []string{"--" + WorkdirFlagName, workdir})

	require.NoError(t, Prestate(context.Background(), cfg))
	chain := readPrestateChain(t, workdir, chainID)
	require.Equal(t, common.HexToHash(selected), chain.Prestate)
	require.Equal(t, common.HexToHash(fallback), chain.CannonFallbackPrestate)
}

func TestNewPrestateConfigPreservesExplicitEmptyEnvironment(t *testing.T) {
	chainID := common.HexToHash("0x01")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", testPrestate("11"))
	t.Setenv("DEPLOYER_CANNON_FALLBACK_PRESTATE", "")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}}, true, 1)
	cfg := parsePrestateCLIConfig(t, []string{"--" + WorkdirFlagName, workdir})
	require.Empty(t, cfg.CannonFallbackPrestate)
	require.True(t, cfg.CannonFallbackPrestateSet)
	require.ErrorContains(t, Prestate(context.Background(), cfg), "invalid --"+CannonFallbackPrestateFlagName)
}

func TestPrestateCLISurface(t *testing.T) {
	require.Equal(t, "cannon-fallback-prestate", CannonFallbackPrestateFlagName)
	require.Equal(t, PrefixEnvVar("DISPUTE_ABSOLUTE_PRESTATE"), PrestateFlag.EnvVars)
	require.Equal(t, PrefixEnvVar("CANNON_FALLBACK_PRESTATE"), CannonFallbackPrestateFlag.EnvVars)
	require.Contains(t, PrestateFlag.Usage, "Selected")
	require.Contains(t, PrestateFlag.Usage, "differing intent value")
	require.Contains(t, CannonFallbackPrestateFlag.Usage, "PERMISSIONED_CANNON")
	require.Contains(t, CannonFallbackPrestateFlag.Usage, "only with CANNON_KONA")
	require.Contains(t, CannonFallbackPrestateFlag.Usage, "differing intent value")

	registered := make(map[string]bool)
	for _, cliFlag := range PrestateFlags {
		for _, name := range cliFlag.Names() {
			registered[name] = true
		}
	}
	require.True(t, registered[PrestateFlagName])
	require.True(t, registered[CannonFallbackPrestateFlagName])
}

func FuzzParsePrestateRoundTrip(f *testing.F) {
	f.Add(append(make([]byte, 31), byte(1)))
	f.Fuzz(func(t *testing.T, value []byte) {
		hash := common.BytesToHash(value)
		if hash == (common.Hash{}) {
			t.Skip()
		}
		parsed, err := parsePrestate(hash.Hex())
		require.NoError(t, err)
		require.Equal(t, hash, parsed)
	})
}

type prestateTestChain struct {
	id              common.Hash
	prepared        bool
	deployed        bool
	overrides       map[string]any
	initialSelected common.Hash
	initialFallback common.Hash
}

func writePrestateWorkdir(t *testing.T, global map[string]any, chains []prestateTestChain, prepared bool, version int) string {
	t.Helper()
	ids := make([]common.Hash, 0, len(chains))
	for _, chain := range chains {
		ids = append(ids, chain.id)
	}
	intent, err := state.NewIntentCustom(1, ids)
	require.NoError(t, err)
	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	intent.SuperchainRoles = &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: addr,
		SuperchainGuardian:        addr,
		Challenger:                addr,
	}
	intent.GlobalDeployOverrides = cloneOverrides(global)
	for i, chain := range chains {
		intent.Chains[i].BaseFeeVaultRecipient = addr
		intent.Chains[i].L1FeeVaultRecipient = addr
		intent.Chains[i].SequencerFeeVaultRecipient = addr
		intent.Chains[i].OperatorFeeVaultRecipient = addr
		intent.Chains[i].Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
		intent.Chains[i].Eip1559Denominator = standard.Eip1559Denominator
		intent.Chains[i].Eip1559Elasticity = standard.Eip1559Elasticity
		intent.Chains[i].Roles = state.ChainRoles{
			L1ProxyAdminOwner: addr,
			L2ProxyAdminOwner: addr,
			SystemConfigOwner: addr,
			UnsafeBlockSigner: addr,
			Batcher:           addr,
			Proposer:          addr,
			Challenger:        addr,
		}
		intent.Chains[i].DeployOverrides = cloneOverrides(chain.overrides)
	}

	st := &state.State{Version: version, Prepared: prepared}
	for _, chain := range chains {
		if chain.prepared {
			st.Chains = append(st.Chains, &state.ChainState{
				ID:                     chain.id,
				Deployed:               &chain.deployed,
				Prestate:               chain.initialSelected,
				CannonFallbackPrestate: chain.initialFallback,
			})
		}
	}

	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))
	return workdir
}

func newTestPrestateConfig(t *testing.T, workdir string) PrestateConfig {
	t.Helper()
	return PrestateConfig{Workdir: workdir, Logger: testlog.Logger(t, slog.LevelInfo)}
}

func parsePrestateCLIConfig(t *testing.T, args []string) PrestateConfig {
	t.Helper()
	app := cli.NewApp()
	var cfg PrestateConfig
	app.Commands = []*cli.Command{{
		Name:  "prestate",
		Flags: clonePrestateTestFlags(),
		Action: func(ctx *cli.Context) error {
			cfg = newPrestateConfig(ctx, log.NewLogger(log.DiscardHandler()))
			return nil
		},
	}}
	require.NoError(t, app.Run(append([]string{"test", "prestate"}, args...)))
	return cfg
}

func clonePrestateTestFlags() []cli.Flag {
	cloned := make([]cli.Flag, 0, len(PrestateFlags))
	for _, cliFlag := range PrestateFlags {
		stringFlag, ok := cliFlag.(*cli.StringFlag)
		if !ok {
			panic("prestate test flag is not a string flag")
		}
		copy := *stringFlag
		cloned = append(cloned, &copy)
	}
	return cloned
}

func readPrestateChain(t *testing.T, workdir string, chainID common.Hash) *state.ChainState {
	t.Helper()
	st, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	chain, err := st.Chain(chainID)
	require.NoError(t, err)
	return chain
}

func commandPair(selected, fallback string) func(*PrestateConfig) {
	return func(cfg *PrestateConfig) {
		cfg.Prestate = selected
		cfg.CannonFallbackPrestate = fallback
	}
}

func gameOverride(gameType embedded.GameType) map[string]any {
	return map[string]any{"respectedGameType": gameType}
}

func cloneOverrides(overrides map[string]any) map[string]any {
	if overrides == nil {
		return nil
	}
	cloned := make(map[string]any, len(overrides))
	for key, value := range overrides {
		cloned[key] = value
	}
	return cloned
}

func with0xPrefix(value string) string {
	if strings.HasPrefix(value, "0x") {
		return value
	}
	return "0x" + value
}

func testPrestate(pair string) string {
	return "0x" + strings.Repeat(pair, 32)
}
