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
