package engine_controller

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

type mockL2 struct {
	lastNum uint64
	ref     eth.L2BlockRef
}

func (m *mockL2) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockL2) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	m.lastNum = num
	return m.ref, nil
}

func TestEngineController_TargetBlockNumber(t *testing.T) {
	t.Parallel()
	rcfg := &rollup.Config{Genesis: rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: 1_000}, BlockTime: 2, L2ChainID: big.NewInt(420)}
	m := &mockL2{ref: eth.L2BlockRef{Number: 0, Time: 0}}
	ec := &simpleEngineController{l2: m, rollup: rcfg}

	// ts = genesis + 2*3 => block #3
	numRef, err := ec.BlockAtTimestamp(context.Background(), 1_000+2*3)
	require.NoError(t, err)
	require.Equal(t, uint64(3), m.lastNum)
	require.Equal(t, m.ref, numRef)
}

func TestEngineController_SentinelErrors(t *testing.T) {
	t.Parallel()
	ec := &simpleEngineController{l2: nil, rollup: nil}
	_, err := ec.BlockAtTimestamp(context.Background(), 0)
	require.ErrorIs(t, err, ErrNoEngineClient)

	ec = &simpleEngineController{l2: &mockL2{}, rollup: nil}
	_, err = ec.BlockAtTimestamp(context.Background(), 0)
	require.ErrorIs(t, err, ErrNoRollupConfig)
}
