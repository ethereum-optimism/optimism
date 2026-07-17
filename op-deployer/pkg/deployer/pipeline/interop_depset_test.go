package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestBuildInteropDepSetIncludesEveryIntentChain(t *testing.T) {
	chainIDs := []common.Hash{
		common.BigToHash(common.Big1),
		common.BigToHash(common.Big2),
		common.BigToHash(common.Big3),
	}

	depSet, err := BuildInteropDepSet([]*state.ChainIntent{
		{ID: chainIDs[0]},
		{ID: chainIDs[1]},
		{ID: chainIDs[2]},
	})
	require.NoError(t, err)

	require.Len(t, depSet.Chains(), len(chainIDs))
	for _, chainID := range chainIDs {
		require.True(t, depSet.HasChain(eth.ChainIDFromBytes32(chainID)))
	}
}

func TestBuildInteropDepSetPropagatesConstructorErrors(t *testing.T) {
	expectedErr := errors.New("constructor failed")

	depSet, err := buildInteropDepSet(
		[]*state.ChainIntent{{ID: common.BigToHash(common.Big1)}},
		func(map[eth.ChainID]*depset.StaticConfigDependency) (*depset.StaticConfigDependencySet, error) {
			return nil, expectedErr
		},
	)

	require.Nil(t, depSet)
	require.ErrorIs(t, err, expectedErr)
}

func TestValidateInteropDepSetMatchesIntent(t *testing.T) {
	chainA := common.HexToHash("0x01")
	chainB := common.HexToHash("0x02")
	chainC := common.HexToHash("0x03")
	chainD := common.HexToHash("0x04")

	tests := []struct {
		name        string
		intentIDs   []common.Hash
		preparedIDs []common.Hash
		missing     bool
		wantErr     string
	}{
		{
			name:        "equal sets in different orders",
			intentIDs:   []common.Hash{chainC, chainA, chainB},
			preparedIDs: []common.Hash{chainB, chainC, chainA},
		},
		{
			name:      "missing dependency set",
			intentIDs: []common.Hash{chainA},
			missing:   true,
			wantErr:   "prepared interop dependency set is missing; rerun op-deployer prepare",
		},
		{
			name:        "duplicate intent ID",
			intentIDs:   []common.Hash{chainA, chainB, chainA},
			preparedIDs: []common.Hash{chainA, chainB},
			wantErr:     "intent contains duplicate chain IDs [" + chainA.Hex() + "]; rerun op-deployer prepare",
		},
		{
			name:        "added chain",
			intentIDs:   []common.Hash{chainA, chainB},
			preparedIDs: []common.Hash{chainA},
			wantErr:     "intent chain set does not match prepared chain set: added chain IDs [" + chainB.Hex() + "]; removed chain IDs []; rerun op-deployer prepare",
		},
		{
			name:        "removed chain",
			intentIDs:   []common.Hash{chainA},
			preparedIDs: []common.Hash{chainA, chainB},
			wantErr:     "intent chain set does not match prepared chain set: added chain IDs []; removed chain IDs [" + chainB.Hex() + "]; rerun op-deployer prepare",
		},
		{
			name:        "added and removed chains are sorted",
			intentIDs:   []common.Hash{chainD, chainB},
			preparedIDs: []common.Hash{chainC, chainA},
			wantErr: "intent chain set does not match prepared chain set: added chain IDs [" +
				chainB.Hex() + ", " + chainD.Hex() + "]; removed chain IDs [" +
				chainA.Hex() + ", " + chainC.Hex() + "]; rerun op-deployer prepare",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intentChains := make([]*state.ChainIntent, 0, len(test.intentIDs))
			for _, id := range test.intentIDs {
				intentChains = append(intentChains, &state.ChainIntent{ID: id})
			}

			var prepared *depset.StaticConfigDependencySet
			if !test.missing {
				var err error
				prepared, err = BuildInteropDepSet(chainIntents(test.preparedIDs))
				require.NoError(t, err)
			}

			err := ValidateInteropDepSetMatchesIntent(intentChains, prepared)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}

func chainIntents(ids []common.Hash) []*state.ChainIntent {
	chains := make([]*state.ChainIntent, 0, len(ids))
	for _, id := range ids {
		chains = append(chains, &state.ChainIntent{ID: id})
	}
	return chains
}

func TestGenerateInteropDepsetPersistsState(t *testing.T) {
	chainID := common.BigToHash(common.Big1)
	intent := &state.Intent{
		Chains: []*state.ChainIntent{{ID: chainID}},
	}
	st := &state.State{}

	var persistedState *state.State
	pEnv := &Env{
		Logger: testlog.Logger(t, slog.LevelInfo),
		StateWriter: stateWriterFunc(func(st *state.State) error {
			persistedState = st
			return nil
		}),
	}

	require.NoError(t, GenerateInteropDepset(context.Background(), pEnv, intent, st))
	require.NotNil(t, st.InteropDepSet)
	require.True(t, st.InteropDepSet.HasChain(eth.ChainIDFromBytes32(chainID)))
	require.Same(t, st, persistedState)
}
