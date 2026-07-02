package deployer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPrestateResolutionAndCommitment(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	chainC := common.HexToHash("0x03")
	prestateA := testPrestate("11")
	prestateB := testPrestate("22")

	tests := []struct {
		name          string
		flag          string
		globalSet     bool
		globalValue   string
		chains        []prestateTestChain
		want          map[common.Hash]common.Hash
		wantZero      []common.Hash
		wantErrParts  []string
		wantNoWriteTo map[common.Hash]common.Hash
	}{
		{
			name: "flag only",
			flag: prestateA,
			chains: []prestateTestChain{{
				id:       chainA,
				prepared: true,
			}},
			want: map[common.Hash]common.Hash{chainA: common.HexToHash(prestateA)},
		},
		{
			name: "chain override only",
			chains: []prestateTestChain{{
				id:          chainA,
				prepared:    true,
				overrideSet: true,
				override:    prestateA,
			}},
			want: map[common.Hash]common.Hash{chainA: common.HexToHash(prestateA)},
		},
		{
			name:        "global override only",
			globalSet:   true,
			globalValue: prestateA,
			chains: []prestateTestChain{{
				id:       chainA,
				prepared: true,
			}},
			want: map[common.Hash]common.Hash{chainA: common.HexToHash(prestateA)},
		},
		{
			name:        "chain override beats global",
			globalSet:   true,
			globalValue: prestateA,
			chains: []prestateTestChain{{
				id:          chainA,
				prepared:    true,
				overrideSet: true,
				override:    prestateB,
			}},
			want: map[common.Hash]common.Hash{chainA: common.HexToHash(prestateB)},
		},
		{
			name: "flag and override agree",
			flag: prestateA,
			chains: []prestateTestChain{{
				id:          chainA,
				prepared:    true,
				overrideSet: true,
				override:    prestateA,
			}},
			want: map[common.Hash]common.Hash{chainA: common.HexToHash(prestateA)},
		},
		{
			name: "flag and override disagree",
			flag: prestateA,
			chains: []prestateTestChain{{
				id:          chainA,
				prepared:    true,
				overrideSet: true,
				override:    prestateB,
			}},
			wantErrParts: []string{
				"conflicting prestate sources",
				chainA.Hex(),
				prestateA,
				prestateB,
			},
			wantNoWriteTo: map[common.Hash]common.Hash{chainA: common.Hash{}},
		},
		{
			name: "no source anywhere",
			chains: []prestateTestChain{{
				id:       chainA,
				prepared: true,
			}},
			wantErrParts: []string{
				"no prestates committed",
			},
			wantNoWriteTo: map[common.Hash]common.Hash{chainA: common.Hash{}},
		},
		{
			name: "unprepared chain",
			flag: prestateA,
			chains: []prestateTestChain{{
				id: chainA,
			}},
			wantErrParts: []string{
				chainA.Hex(),
				"op-deployer prepare",
			},
		},
		{
			name: "mixed no-source chain skips while another commits",
			chains: []prestateTestChain{
				{
					id:              chainA,
					prepared:        true,
					deployOverrides: map[string]any{"respectedGameType": standard.DisputeGameType},
				},
				{
					id:          chainB,
					prepared:    true,
					overrideSet: true,
					override:    prestateA,
				},
			},
			want:     map[common.Hash]common.Hash{chainB: common.HexToHash(prestateA)},
			wantZero: []common.Hash{chainA},
		},
		{
			name: "multi-chain flag fan-out",
			flag: prestateA,
			chains: []prestateTestChain{
				{id: chainA, prepared: true},
				{id: chainB, prepared: true},
				{id: chainC, prepared: true},
			},
			want: map[common.Hash]common.Hash{
				chainA: common.HexToHash(prestateA),
				chainB: common.HexToHash(prestateA),
				chainC: common.HexToHash(prestateA),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			workdir := writePrestateWorkdir(t, test.globalSet, test.globalValue, test.chains)

			err := Prestate(context.Background(), PrestateConfig{
				Workdir:  workdir,
				Logger:   testlog.Logger(t, slog.LevelInfo),
				Prestate: test.flag,
			})

			if len(test.wantErrParts) > 0 {
				require.Error(t, err)
				for _, part := range test.wantErrParts {
					require.Contains(t, err.Error(), part)
				}
				if len(test.wantNoWriteTo) > 0 {
					gotState, readErr := pipeline.ReadState(workdir)
					require.NoError(t, readErr)
					for chainID, want := range test.wantNoWriteTo {
						requireChainPrestate(t, gotState, chainID, want)
					}
				}
				return
			}

			require.NoError(t, err)
			gotState, err := pipeline.ReadState(workdir)
			require.NoError(t, err)
			for chainID, want := range test.want {
				requireChainPrestate(t, gotState, chainID, want)
			}
			for _, chainID := range test.wantZero {
				requireChainPrestate(t, gotState, chainID, common.Hash{})
			}
		})
	}
}

func TestPrestateStrictValidationAcrossSources(t *testing.T) {
	chainID := common.HexToHash("0x01")
	validPrestate := testPrestate("11")
	invalidValues := []struct {
		name    string
		value   string
		wantErr string
	}{
		{
			name:    "short",
			value:   "0xabc123",
			wantErr: "exactly 64 hex characters",
		},
		{
			name:    "overlong",
			value:   validPrestate + "11",
			wantErr: "exactly 64 hex characters",
		},
		{
			name:    "malformed",
			value:   "0x" + strings.Repeat("11", 31) + "zz",
			wantErr: "valid hex",
		},
		{
			name:    "zero",
			value:   "0x" + strings.Repeat("00", 32),
			wantErr: "must not be zero",
		},
		{
			name:    "missing prefix",
			value:   strings.Repeat("11", 32),
			wantErr: "must start with 0x",
		},
	}

	sources := []string{"flag", "chain-override", "global-override"}
	for _, source := range sources {
		source := source
		for _, invalid := range invalidValues {
			invalid := invalid
			t.Run(source+"/"+invalid.name, func(t *testing.T) {
				chain := prestateTestChain{id: chainID, prepared: true}
				flag := ""
				globalSet := false
				globalValue := ""

				switch source {
				case "flag":
					flag = invalid.value
				case "chain-override":
					chain.overrideSet = true
					chain.override = invalid.value
				case "global-override":
					globalSet = true
					globalValue = invalid.value
				}

				workdir := writePrestateWorkdir(t, globalSet, globalValue, []prestateTestChain{chain})
				err := Prestate(context.Background(), PrestateConfig{
					Workdir:  workdir,
					Logger:   testlog.Logger(t, slog.LevelInfo),
					Prestate: flag,
				})

				require.Error(t, err)
				require.Contains(t, err.Error(), invalid.wantErr)
				if source == "flag" {
					require.Contains(t, err.Error(), PrestateFlagName)
				} else {
					require.Contains(t, err.Error(), source)
					require.Contains(t, err.Error(), chainID.Hex())
				}
			})
		}
	}
}

func TestPrestatePersistsStateRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	prestate := testPrestate("33")
	workdir := writePrestateWorkdir(t, false, "", []prestateTestChain{{
		id:       chainID,
		prepared: true,
	}})

	require.NoError(t, Prestate(context.Background(), PrestateConfig{
		Workdir:  workdir,
		Logger:   testlog.Logger(t, slog.LevelInfo),
		Prestate: prestate,
	}))

	rawState, err := os.ReadFile(filepath.Join(workdir, "state.json"))
	require.NoError(t, err)
	require.Contains(t, string(rawState), `"prestate": "`+common.HexToHash(prestate).Hex()+`"`)

	gotState, err := pipeline.ReadState(workdir)
	require.NoError(t, err)
	requireChainPrestate(t, gotState, chainID, common.HexToHash(prestate))
}

func TestPrestateFlagEnvVar(t *testing.T) {
	require.Equal(t, PrefixEnvVar("DISPUTE_ABSOLUTE_PRESTATE"), PrestateFlag.EnvVars)
}

type prestateTestChain struct {
	id              common.Hash
	prepared        bool
	overrideSet     bool
	override        string
	deployOverrides map[string]any
}

func writePrestateWorkdir(t *testing.T, globalSet bool, globalValue string, chains []prestateTestChain) string {
	t.Helper()

	intent := &state.Intent{}
	if globalSet {
		intent.GlobalDeployOverrides = map[string]any{
			faultGameAbsolutePrestateOverride: globalValue,
		}
	}

	st := &state.State{}
	for _, chain := range chains {
		deployOverrides := chain.deployOverrides
		if chain.overrideSet {
			if deployOverrides == nil {
				deployOverrides = make(map[string]any)
			}
			deployOverrides[faultGameAbsolutePrestateOverride] = chain.override
		}

		intent.Chains = append(intent.Chains, &state.ChainIntent{
			ID:              chain.id,
			DeployOverrides: deployOverrides,
		})
		if chain.prepared {
			st.Chains = append(st.Chains, &state.ChainState{ID: chain.id})
		}
	}

	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, st.WriteToFile(filepath.Join(workdir, "state.json")))
	return workdir
}

func requireChainPrestate(t *testing.T, st *state.State, chainID common.Hash, want common.Hash) {
	t.Helper()

	chain, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, want, chain.Prestate)
}

func testPrestate(pair string) string {
	return "0x" + strings.Repeat(pair, 32)
}
