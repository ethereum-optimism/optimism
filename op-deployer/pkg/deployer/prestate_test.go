package deployer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
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
		{name: "global and chain agree", global: valueA, chain: valueA, want: valueA},
		{name: "global and chain conflict", global: valueA, chain: valueB, wantErr: true},
		{name: "command and global agree", command: valueA, global: strings.ToUpper(valueA[2:4]) + valueA[4:], want: valueA},
		{name: "command and global conflict", command: valueA, global: valueB, wantErr: true},
		{name: "command and chain agree", command: valueA, chain: valueA, want: valueA},
		{name: "command and chain conflict", command: valueA, chain: valueB, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			global := make(map[string]any)
			if test.global != "" {
				global[state.FaultGameAbsolutePrestateOverrideKey] = with0xPrefix(test.global)
			}
			overrides := gameOverride(embedded.GameTypeCannonKona)
			if test.chain != "" {
				overrides[state.FaultGameAbsolutePrestateOverrideKey] = test.chain
			}
			workdir := writePrestateWorkdir(t, global, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
			cfg := newTestPrestateConfig(t, workdir)
			cfg.Prestate = test.command

			err := Prestate(context.Background(), cfg)
			if test.wantErr {
				require.ErrorContains(t, err, "conflicting selected prestate")
				require.ErrorContains(t, err, chainID.Hex())
				return
			}
			require.NoError(t, err)
			require.Equal(t, common.HexToHash(test.want), readPrestateChain(t, workdir, chainID).Prestate)
		})
	}
}

func TestPrestateStrictValidationAcrossSources(t *testing.T) {
	chainID := common.HexToHash("0x01")
	valid := testPrestate("11")
	invalidValues := []struct {
		name             string
		value            any
		wantErr          string
		canonicalWantErr string
	}{
		{name: "short", value: "0xabc123", wantErr: "exactly 64 hex characters", canonicalWantErr: "hex string has length 6, want 64"},
		{name: "overlong", value: valid + "11", wantErr: "exactly 64 hex characters", canonicalWantErr: "hex string has length 66, want 64"},
		{name: "malformed", value: "0x" + strings.Repeat("11", 31) + "zz", wantErr: "valid hex", canonicalWantErr: "invalid hex string"},
		{name: "zero", value: "0x" + strings.Repeat("00", 32), wantErr: "must not be zero"},
		{name: "missing prefix", value: strings.Repeat("11", 32), wantErr: "must start with 0x", canonicalWantErr: "without 0x prefix"},
		{name: "explicit empty", value: "", wantErr: "must start with 0x", canonicalWantErr: "hex string has length 0, want 64"},
		{name: "whitespace", value: " " + valid, wantErr: "must start with 0x", canonicalWantErr: "without 0x prefix"},
		{name: "non-string", value: int64(7), wantErr: "must be a string", canonicalWantErr: "cannot unmarshal non-string"},
	}

	for _, source := range []string{"command", "chain override", "global override"} {
		for _, invalid := range invalidValues {
			if source == "command" && invalid.name == "non-string" {
				continue
			}
			t.Run(source+"/"+invalid.name, func(t *testing.T) {
				global := map[string]any{state.FaultGameAbsolutePrestateOverrideKey: valid}
				overrides := gameOverride(embedded.GameTypeCannonKona)
				cfgValue := false
				switch source {
				case "command":
					delete(global, state.FaultGameAbsolutePrestateOverrideKey)
					cfgValue = true
				case "chain override":
					overrides[state.FaultGameAbsolutePrestateOverrideKey] = invalid.value
				case "global override":
					global[state.FaultGameAbsolutePrestateOverrideKey] = invalid.value
				}
				workdir := writePrestateWorkdir(t, global, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
				cfg := newTestPrestateConfig(t, workdir)
				if cfgValue {
					if raw, ok := invalid.value.(string); ok {
						cfg.Prestate = raw
					}
					cfg.PrestateSet = true
				}
				before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
				require.NoError(t, err)

				err = Prestate(context.Background(), cfg)
				require.Error(t, err)
				if source != "command" && invalid.canonicalWantErr != "" {
					require.ErrorContains(t, err, "failed to resolve initial dispute game type")
					require.ErrorContains(t, err, chainID.Hex())
					require.ErrorContains(t, err, invalid.canonicalWantErr)
				} else {
					require.ErrorContains(t, err, invalid.wantErr)
				}
				after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
				require.NoError(t, readErr)
				require.Equal(t, before, after)
			})
		}
	}

	hiddenValues := []struct {
		name             string
		value            string
		wantErr          string
		canonicalWantErr string
	}{
		{name: "malformed", value: "0x" + strings.Repeat("11", 31) + "zz", wantErr: "valid hex", canonicalWantErr: "invalid hex string"},
		{name: "zero", value: "0x" + strings.Repeat("00", 32), wantErr: "must not be zero"},
	}
	for _, invalid := range hiddenValues {
		t.Run("invalid global hidden by chain/"+invalid.name, func(t *testing.T) {
			global := map[string]any{state.FaultGameAbsolutePrestateOverrideKey: invalid.value}
			overrides := map[string]any{
				"respectedGameType":                        embedded.GameTypeCannonKona,
				state.FaultGameAbsolutePrestateOverrideKey: valid,
			}
			workdir := writePrestateWorkdir(t, global, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)

			err = Prestate(context.Background(), newTestPrestateConfig(t, workdir))
			require.Error(t, err)
			require.ErrorContains(t, err, invalid.wantErr)
			after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})

		t.Run("invalid chain alongside command/"+invalid.name, func(t *testing.T) {
			overrides := map[string]any{
				"respectedGameType":                        embedded.GameTypeCannonKona,
				state.FaultGameAbsolutePrestateOverrideKey: invalid.value,
			}
			workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{id: chainID, prepared: true, overrides: overrides}}, true, 1)
			cfg := newTestPrestateConfig(t, workdir)
			cfg.Prestate = valid
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)

			err = Prestate(context.Background(), cfg)
			require.Error(t, err)
			if invalid.canonicalWantErr != "" {
				require.ErrorContains(t, err, invalid.canonicalWantErr)
			} else {
				require.ErrorContains(t, err, invalid.wantErr)
			}
			after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})
	}
}

func TestPrestateRejectsPermissionedPlaceholderAcrossSources(t *testing.T) {
	chainID := common.HexToHash("0x01")
	placeholder := opcm.PermissionedCannonFallbackPrestatePlaceholder.Hex()
	valid := testPrestate("11")

	tests := []struct {
		name        string
		gameType    embedded.GameType
		source      string
		environment bool
		global      map[string]any
	}{
		{
			name:        "CANNON_KONA command/environment",
			gameType:    embedded.GameTypeCannonKona,
			source:      "command/environment",
			environment: true,
		},
		{
			name:     "CANNON_KONA global override",
			gameType: embedded.GameTypeCannonKona,
			source:   "global override",
			global:   map[string]any{state.FaultGameAbsolutePrestateOverrideKey: placeholder},
		},
		{
			name:     "CANNON_KONA chain override",
			gameType: embedded.GameTypeCannonKona,
			source:   "chain override",
		},
		{
			name:     "SUPER_CANNON_KONA command/environment",
			gameType: embedded.GameTypeSuperCannonKona,
			source:   "command/environment",
		},
		{
			name:     "SUPER_CANNON_KONA global override",
			gameType: embedded.GameTypeSuperCannonKona,
			source:   "global override",
			global:   map[string]any{state.FaultGameAbsolutePrestateOverrideKey: placeholder},
		},
		{
			name:     "SUPER_CANNON_KONA chain override",
			gameType: embedded.GameTypeSuperCannonKona,
			source:   "chain override",
		},
		{
			name:     "reserved lower-precedence command alongside valid global override",
			gameType: embedded.GameTypeCannonKona,
			source:   "command/environment",
			global:   map[string]any{state.FaultGameAbsolutePrestateOverrideKey: valid},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			overrides := gameOverride(test.gameType)
			if test.source == "chain override" {
				overrides[state.FaultGameAbsolutePrestateOverrideKey] = placeholder
			}
			workdir := writePrestateWorkdir(t, test.global, []prestateTestChain{{
				id: chainID, prepared: true, overrides: overrides,
			}}, true, 1)
			cfg := newTestPrestateConfig(t, workdir)
			if test.source == "command/environment" {
				if test.environment {
					t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", placeholder)
					cfg = parsePrestateCLIConfig(t, []string{"--" + WorkdirFlagName, workdir})
				} else {
					cfg.Prestate = placeholder
				}
			}
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)

			err = Prestate(context.Background(), cfg)
			require.Error(t, err)
			for _, part := range []string{
				test.source,
				chainID.Hex(),
				gameTypeName(test.gameType),
				placeholder,
				"reserved for the CANNON_KONA permissioned fallback",
			} {
				require.ErrorContains(t, err, part)
			}
			if test.name == "reserved lower-precedence command alongside valid global override" {
				require.NotContains(t, err.Error(), "conflicting selected prestate sources")
			}

			after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})
	}
}

func TestPrestatePermissionedPlaceholderScope(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	placeholder := opcm.PermissionedCannonFallbackPrestatePlaceholder
	valid := testPrestate("11")
	stale := common.HexToHash(testPrestate("aa"))

	t.Run("permissioned-only placeholder command remains unused", func(t *testing.T) {
		workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
			id: chainA, prepared: true, initialSelected: stale,
		}}, true, 1)
		cfg := newTestPrestateConfig(t, workdir)
		cfg.Prestate = placeholder.Hex()
		before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
		require.NoError(t, err)

		err = Prestate(context.Background(), cfg)
		require.ErrorContains(t, err, "--"+PrestateFlagName)
		require.ErrorContains(t, err, "no undeployed chain resolves")
		after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
		require.NoError(t, readErr)
		require.Equal(t, before, after)
	})

	t.Run("deployed permissionless placeholder override is ignored", func(t *testing.T) {
		workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
			id:              chainA,
			prepared:        true,
			deployed:        true,
			overrides:       map[string]any{"respectedGameType": embedded.GameTypeCannonKona, state.FaultGameAbsolutePrestateOverrideKey: placeholder.Hex()},
			initialSelected: stale,
		}}, true, 1)

		require.NoError(t, Prestate(context.Background(), newTestPrestateConfig(t, workdir)))
		require.Equal(t, stale, readPrestateChain(t, workdir, chainA).Prestate)
	})

	t.Run("deployed placeholder does not poison undeployed valid prestate", func(t *testing.T) {
		workdir := writePrestateWorkdir(t, nil, []prestateTestChain{
			{
				id:              chainA,
				prepared:        true,
				deployed:        true,
				overrides:       map[string]any{"respectedGameType": embedded.GameTypeCannonKona, state.FaultGameAbsolutePrestateOverrideKey: placeholder.Hex()},
				initialSelected: placeholder,
			},
			{
				id:        chainB,
				prepared:  true,
				overrides: gameOverride(embedded.GameTypeCannonKona),
			},
		}, true, 1)
		cfg := newTestPrestateConfig(t, workdir)
		cfg.Prestate = valid

		require.NoError(t, Prestate(context.Background(), cfg))
		require.Equal(t, placeholder, readPrestateChain(t, workdir, chainA).Prestate)
		require.Equal(t, common.HexToHash(valid), readPrestateChain(t, workdir, chainB).Prestate)
	})
}

func TestPrestateRejectsObsoleteFallbackOverrides(t *testing.T) {
	chainID := common.HexToHash("0x01")
	value := testPrestate("11")
	tests := []struct {
		name     string
		global   map[string]any
		override map[string]any
		want     []string
	}{
		{
			name:   "global",
			global: map[string]any{cannonFallbackPrestateOverride: value},
			want:   []string{"global override", cannonFallbackPrestateOverride, "obsolete", "remove it"},
		},
		{
			name:     "chain",
			override: map[string]any{cannonFallbackPrestateOverride: value},
			want:     []string{"chain override", cannonFallbackPrestateOverride, chainID.Hex(), "obsolete", "remove it"},
		},
		{
			name:     "deployed chain",
			override: map[string]any{cannonFallbackPrestateOverride: value},
			want:     []string{"chain override", cannonFallbackPrestateOverride, chainID.Hex(), "obsolete", "remove it"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployed := test.name == "deployed chain"
			workdir := writePrestateWorkdir(t, test.global, []prestateTestChain{{
				id: chainID, prepared: true, deployed: deployed, overrides: test.override,
			}}, true, 1)
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)

			err = Prestate(context.Background(), newTestPrestateConfig(t, workdir))
			require.Error(t, err)
			for _, part := range test.want {
				require.ErrorContains(t, err, part)
			}
			after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})
	}
}

func TestPrestateGameTypeRequirements(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	selected := testPrestate("11")
	stale := common.HexToHash(testPrestate("aa"))

	tests := []struct {
		name         string
		global       map[string]any
		chains       []prestateTestChain
		configure    func(*PrestateConfig)
		want         map[common.Hash]common.Hash
		wantErrParts []string
	}{
		{
			name:   "permissioned only needs no PCD source and clears stale value",
			chains: []prestateTestChain{{id: chainA, prepared: true, initialSelected: stale}},
			want:   map[common.Hash]common.Hash{chainA: {}},
		},
		{
			name:   "legacy permissioned absolute prestate is not copied",
			chains: []prestateTestChain{{id: chainA, prepared: true, overrides: map[string]any{state.FaultGameAbsolutePrestateOverrideKey: selected}}},
			want:   map[common.Hash]common.Hash{chainA: {}},
		},
		{
			name:      "CANNON_KONA commits selected",
			chains:    []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			configure: setCommandPrestate(selected),
			want:      map[common.Hash]common.Hash{chainA: common.HexToHash(selected)},
		},
		{
			name:         "CANNON_KONA rejects missing selected",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)}},
			wantErrParts: []string{"CANNON_KONA", "requires selected prestate"},
		},
		{
			name:      "SUPER_CANNON_KONA commits selected",
			chains:    []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale}},
			configure: setCommandPrestate(selected),
			want:      map[common.Hash]common.Hash{chainA: common.HexToHash(selected)},
		},
		{
			name:         "SUPER_CANNON_KONA rejects missing selected",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)}},
			wantErrParts: []string{"SUPER_CANNON_KONA", "requires selected prestate"},
		},
		{
			name:         "permissioned only rejects unconsumed selected flag",
			chains:       []prestateTestChain{{id: chainA, prepared: true}},
			configure:    setCommandPrestate(selected),
			wantErrParts: []string{"--" + PrestateFlagName, "no undeployed chain resolves", "respectedGameType"},
		},
		{
			name:         "misspelled respectedGameType rejects supplied prestate",
			chains:       []prestateTestChain{{id: chainA, prepared: true, overrides: map[string]any{"respectedGamType": embedded.GameTypeCannonKona}}},
			configure:    setCommandPrestate(selected),
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
				{id: chainA, prepared: true, initialSelected: stale},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: {},
				chainB: common.HexToHash(selected),
			},
		},
		{
			name: "CANNON_KONA plus SUPER_CANNON_KONA fails",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
			},
			configure:    setCommandPrestate(selected),
			wantErrParts: []string{"cannot mix CANNON_KONA and SUPER_CANNON_KONA"},
		},
		{
			name: "multiple SUPER_CANNON_KONA chains commit selected",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: common.HexToHash(selected),
				chainB: common.HexToHash(selected),
			},
		},
		{
			name: "permissioned plus SUPER_CANNON_KONA succeeds",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, initialSelected: stale},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: {},
				chainB: common.HexToHash(selected),
			},
		},
		{
			name: "deployed CANNON_KONA preserves prestate and ignores conflicting source",
			chains: []prestateTestChain{
				{
					id: chainA, prepared: true, deployed: true,
					overrides:       map[string]any{"respectedGameType": embedded.GameTypeCannonKona, state.FaultGameAbsolutePrestateOverrideKey: testPrestate("33")},
					initialSelected: stale,
				},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: common.HexToHash(selected),
			},
		},
		{
			name: "deployed CANNON_KONA needs no source alongside active permissioned",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: stale},
				{id: chainB, prepared: true, initialSelected: stale},
			},
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: {},
			},
		},
		{
			name:         "selected flag with only deployed consumer is rejected",
			chains:       []prestateTestChain{{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale}},
			configure:    setCommandPrestate(selected),
			wantErrParts: []string{"--" + PrestateFlagName, "no undeployed chain resolves"},
		},
		{
			name: "deployed unsupported game type is ignored",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannon), initialSelected: stale},
				{id: chainB, prepared: true, initialSelected: stale},
			},
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: {},
			},
		},
		{
			name: "deployed CANNON_KONA migration preserves historical prestate alongside active SUPER_CANNON_KONA",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: stale},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona)},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: common.HexToHash(selected),
			},
		},
		{
			name: "deployed SUPER_CANNON_KONA is preserved alongside active permissioned",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
				{id: chainB, prepared: true, initialSelected: stale},
			},
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: {},
			},
		},
		{
			name: "deployed SUPER_CANNON_KONA is preserved alongside active SUPER_CANNON_KONA",
			chains: []prestateTestChain{
				{id: chainA, prepared: true, deployed: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
				{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeSuperCannonKona), initialSelected: stale},
			},
			configure: setCommandPrestate(selected),
			want: map[common.Hash]common.Hash{
				chainA: stale,
				chainB: common.HexToHash(selected),
			},
		},
		{
			name:   "deployed malformed game type is ignored and preserved",
			chains: []prestateTestChain{{id: chainA, prepared: true, deployed: true, overrides: map[string]any{"respectedGameType": "invalid"}, initialSelected: stale}},
			want:   map[common.Hash]common.Hash{chainA: stale},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workdir := writePrestateWorkdir(t, test.global, test.chains, true, 1)
			before, err := os.ReadFile(filepath.Join(workdir, "state.json"))
			require.NoError(t, err)
			cfg := newTestPrestateConfig(t, workdir)
			if test.configure != nil {
				test.configure(&cfg)
			}

			err = Prestate(context.Background(), cfg)
			if len(test.wantErrParts) > 0 {
				require.Error(t, err)
				for _, part := range test.wantErrParts {
					require.ErrorContains(t, err, part)
				}
				after, readErr := os.ReadFile(filepath.Join(workdir, "state.json"))
				require.NoError(t, readErr)
				require.Equal(t, before, after)
				return
			}
			require.NoError(t, err)
			for chainID, want := range test.want {
				require.Equal(t, want, readPrestateChain(t, workdir, chainID).Prestate)
			}
		})
	}
}

func TestPrestateCommandValueFansOutOnlyToApplicableChains(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	chainC := common.HexToHash("0x03")
	selected := testPrestate("11")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{
		{id: chainA, prepared: true},
		{id: chainB, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
		{id: chainC, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona)},
	}, true, 1)
	cfg := newTestPrestateConfig(t, workdir)
	cfg.Prestate = selected

	require.NoError(t, Prestate(context.Background(), cfg))
	require.Zero(t, readPrestateChain(t, workdir, chainA).Prestate)
	for _, chainID := range []common.Hash{chainB, chainC} {
		require.Equal(t, common.HexToHash(selected), readPrestateChain(t, workdir, chainID).Prestate)
	}
}

func TestPrestatePreconditionsAndAtomicity(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	selected := testPrestate("11")
	oldSelected := common.HexToHash(testPrestate("aa"))

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
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: oldSelected},
				{id: chainB, prepared: false, overrides: gameOverride(embedded.GameTypeCannonKona)},
			},
			configure: setCommandPrestate(selected), wantErr: "run op-deployer prepare",
		},
		{
			name: "conflict after earlier chain resolves", prepared: true, version: 1,
			global: map[string]any{state.FaultGameAbsolutePrestateOverrideKey: selected},
			chains: []prestateTestChain{
				{id: chainA, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona), initialSelected: oldSelected},
				{id: chainB, prepared: true, overrides: map[string]any{"respectedGameType": embedded.GameTypeCannonKona, state.FaultGameAbsolutePrestateOverrideKey: testPrestate("44")}},
			},
			configure: setCommandPrestate(selected), wantErr: "conflicting selected prestate sources",
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

func TestPrestateRejectsIntentChainChangesAfterPrepare(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")

	tests := []struct {
		name     string
		chains   []prestateTestChain
		mutate   func(*state.Intent, *state.State)
		wantErrs []string
	}{
		{
			name:   "missing prepared dependency set",
			chains: []prestateTestChain{{id: chainA, prepared: true}},
			mutate: func(_ *state.Intent, st *state.State) {
				st.InteropDepSet = nil
			},
			wantErrs: []string{"prepared interop dependency set is missing", "rerun op-deployer prepare"},
		},
		{
			name:   "missing prepared game type",
			chains: []prestateTestChain{{id: chainA, prepared: true}},
			mutate: func(_ *state.Intent, st *state.State) {
				chain, err := st.Chain(chainA)
				require.NoError(t, err)
				chain.InitialGameType = nil
			},
			wantErrs: []string{chainA.Hex(), "no initial game type recorded by prepare", "rerun op-deployer prepare"},
		},
		{
			name:   "game type changed",
			chains: []prestateTestChain{{id: chainA, prepared: true}},
			mutate: func(intent *state.Intent, _ *state.State) {
				intent.Chains[0].DeployOverrides = gameOverride(embedded.GameTypeCannonKona)
			},
			wantErrs: []string{
				chainA.Hex(),
				"prepared SUPER_PERMISSIONED (5)",
				"intent CANNON_KONA (8)",
				"rerun op-deployer prepare",
			},
		},
		{
			name:   "added chain",
			chains: []prestateTestChain{{id: chainA, prepared: true}},
			mutate: func(intent *state.Intent, _ *state.State) {
				intent.Chains = append(intent.Chains, &state.ChainIntent{
					ID:              chainB,
					DeployOverrides: map[string]any{"respectedGameType": "malformed"},
				})
			},
			wantErrs: []string{"added chain IDs", chainB.Hex(), "rerun op-deployer prepare"},
		},
		{
			name: "removed chain",
			chains: []prestateTestChain{
				{id: chainA, prepared: true},
				{id: chainB, prepared: true},
			},
			mutate: func(intent *state.Intent, _ *state.State) {
				intent.Chains = intent.Chains[:1]
			},
			wantErrs: []string{"removed chain IDs", chainB.Hex(), "rerun op-deployer prepare"},
		},
		{
			name:   "duplicate chain",
			chains: []prestateTestChain{{id: chainA, prepared: true}},
			mutate: func(intent *state.Intent, _ *state.State) {
				intent.Chains = append(intent.Chains, intent.Chains[0])
			},
			wantErrs: []string{"duplicate chain IDs", chainA.Hex(), "rerun op-deployer prepare"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workdir := writePrestateWorkdir(t, nil, test.chains, true, 1)
			intent, err := pipeline.ReadIntent(workdir)
			require.NoError(t, err)
			st, err := pipeline.ReadState(workdir)
			require.NoError(t, err)
			test.mutate(intent, st)
			require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
			require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))

			statePath := filepath.Join(workdir, "state.json")
			before, err := os.ReadFile(statePath)
			require.NoError(t, err)

			err = Prestate(context.Background(), newTestPrestateConfig(t, workdir))
			require.Error(t, err)
			for _, part := range test.wantErrs {
				require.ErrorContains(t, err, part)
			}

			after, readErr := os.ReadFile(statePath)
			require.NoError(t, readErr)
			require.Equal(t, before, after)
		})
	}
}

func TestPrestateRejectsUnsupportedPreparedGameTypeWithoutWritingState(t *testing.T) {
	chainID := common.HexToHash("0x01")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
		id:        chainID,
		prepared:  true,
		overrides: gameOverride(embedded.GameTypeZKDisputeGame),
	}}, true, 1)
	statePath := filepath.Join(workdir, "state.json")
	before, err := os.ReadFile(statePath)
	require.NoError(t, err)

	err = Prestate(context.Background(), newTestPrestateConfig(t, workdir))
	require.ErrorContains(t, err, chainID.Hex())
	require.ErrorContains(t, err, "unsupported initial dispute game type 10")

	after, readErr := os.ReadFile(statePath)
	require.NoError(t, readErr)
	require.Equal(t, before, after)
}

func TestPrestateAllowsAbsolutePrestateChangeAfterPrepare(t *testing.T) {
	chainID := common.HexToHash("0x01")
	initial := testPrestate("11")
	updated := testPrestate("22")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
		id:       chainID,
		prepared: true,
		overrides: map[string]any{
			"respectedGameType":                        embedded.GameTypeCannonKona,
			state.FaultGameAbsolutePrestateOverrideKey: initial,
		},
	}}, true, 1)

	intent, err := pipeline.ReadIntent(workdir)
	require.NoError(t, err)
	intent.Chains[0].DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = updated
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))

	require.NoError(t, Prestate(context.Background(), newTestPrestateConfig(t, workdir)))
	require.Equal(t, common.HexToHash(updated), readPrestateChain(t, workdir, chainID).Prestate)
}

func TestPrestatePersistenceReplacementAndIdempotency(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedA := testPrestate("11")
	selectedB := testPrestate("33")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
		id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona),
	}}, true, 1)

	cfg := newTestPrestateConfig(t, workdir)
	cfg.Prestate = selectedA
	require.NoError(t, Prestate(context.Background(), cfg))
	first, err := os.ReadFile(filepath.Join(workdir, "state.json"))
	require.NoError(t, err)
	require.NoError(t, Prestate(context.Background(), cfg))
	second, err := os.ReadFile(filepath.Join(workdir, "state.json"))
	require.NoError(t, err)
	require.Equal(t, first, second)

	cfg.Prestate = selectedB
	require.NoError(t, Prestate(context.Background(), cfg))
	require.Equal(t, common.HexToHash(selectedB), readPrestateChain(t, workdir, chainID).Prestate)
}

func TestNewPrestateConfigFlagAndEnvironmentPrecedence(t *testing.T) {
	selectedEnv := testPrestate("11")
	selectedCLI := testPrestate("22")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", selectedEnv)

	cfg := parsePrestateCLIConfig(t, []string{"--" + PrestateFlagName, selectedCLI})
	require.Equal(t, selectedCLI, cfg.Prestate)
	require.True(t, cfg.PrestateSet)
}

func TestPrestateEnvironmentOnly(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selected := testPrestate("11")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", selected)
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
		id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona),
	}}, true, 1)
	cfg := parsePrestateCLIConfig(t, []string{"--" + WorkdirFlagName, workdir})

	require.NoError(t, Prestate(context.Background(), cfg))
	require.Equal(t, common.HexToHash(selected), readPrestateChain(t, workdir, chainID).Prestate)
}

func TestNewPrestateConfigPreservesExplicitEmptyEnvironment(t *testing.T) {
	chainID := common.HexToHash("0x01")
	t.Setenv("DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE", "")
	workdir := writePrestateWorkdir(t, nil, []prestateTestChain{{
		id: chainID, prepared: true, overrides: gameOverride(embedded.GameTypeCannonKona),
	}}, true, 1)
	cfg := parsePrestateCLIConfig(t, []string{"--" + WorkdirFlagName, workdir})
	require.Empty(t, cfg.Prestate)
	require.True(t, cfg.PrestateSet)
	require.ErrorContains(t, Prestate(context.Background(), cfg), "invalid --"+PrestateFlagName)
}

func TestPrestateCLISurface(t *testing.T) {
	require.Equal(t, PrefixEnvVar("DISPUTE_ABSOLUTE_PRESTATE"), PrestateFlag.EnvVars)
	require.Contains(t, PrestateFlag.Usage, "Selected")
	require.Contains(t, PrestateFlag.Usage, "differing intent value")

	registered := make(map[string]bool)
	for _, cliFlag := range PrestateFlags {
		for _, name := range cliFlag.Names() {
			registered[name] = true
		}
	}
	require.True(t, registered[PrestateFlagName])
	require.NotContains(t, registered, "cannon-fallback-prestate")

	app := cli.NewApp()
	app.Commands = []*cli.Command{{
		Name:  "prestate",
		Flags: clonePrestateTestFlags(),
	}}
	err := app.Run([]string{"test", "prestate", "--cannon-fallback-prestate", testPrestate("11")})
	require.ErrorContains(t, err, "flag provided but not defined")
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

	interopDepSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	require.NoError(t, err)
	st := &state.State{Version: version, Prepared: prepared, InteropDepSet: interopDepSet}
	for i, chain := range chains {
		if chain.prepared {
			var initialGameType *uint32
			if !chain.deployed {
				gameType := fixtureInitialGameType(t, &intent, intent.Chains[i])
				initialGameType = &gameType
			}
			st.Chains = append(st.Chains, &state.ChainState{
				ID:              chain.id,
				Deployed:        &chain.deployed,
				Prestate:        chain.initialSelected,
				InitialGameType: initialGameType,
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

func setCommandPrestate(selected string) func(*PrestateConfig) {
	return func(cfg *PrestateConfig) {
		cfg.Prestate = selected
	}
}

func gameOverride(gameType embedded.GameType) map[string]any {
	return map[string]any{"respectedGameType": gameType}
}

func fixtureInitialGameType(t *testing.T, intent *state.Intent, chain *state.ChainIntent) uint32 {
	t.Helper()

	global := make(map[string]any)
	if gameType, ok := intent.GlobalDeployOverrides["respectedGameType"]; ok {
		global["respectedGameType"] = gameType
	}
	chainOverrides := make(map[string]any)
	if gameType, ok := chain.DeployOverrides["respectedGameType"]; ok {
		chainOverrides["respectedGameType"] = gameType
	}
	proofParams, err := pipeline.ResolveChainProofParams(
		&state.Intent{GlobalDeployOverrides: global},
		&state.ChainIntent{DeployOverrides: chainOverrides},
	)
	require.NoError(t, err)
	return proofParams.DisputeGameType
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
