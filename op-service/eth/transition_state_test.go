package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestTransitionStateCodec(t *testing.T) {
	t.Run("TransitionState", func(t *testing.T) {
		superRoot := &SuperV1{
			Timestamp: 9842494,
			Chains: []ChainIDAndOutput{
				{ChainID: ChainIDFromUInt64(34), Output: Bytes32{0x01}},
				{ChainID: ChainIDFromUInt64(35), Output: Bytes32{0x02}},
			},
		}
		state := &TransitionState{
			SuperRoot: superRoot.Marshal(),
			PendingProgress: []OptimisticBlock{
				{BlockHash: common.Hash{0x05}, OutputRoot: Bytes32{0x03}},
				{BlockHash: common.Hash{0x06}, OutputRoot: Bytes32{0x04}},
			},
			Step: 2,
		}
		data := state.Marshal()
		actual, err := UnmarshalTransitionState(data)
		require.NoError(t, err)
		require.Equal(t, state, actual)
	})

	t.Run("SuperRoot", func(t *testing.T) {
		superRoot := &SuperV1{
			Timestamp: 9842494,
			Chains: []ChainIDAndOutput{
				{ChainID: ChainIDFromUInt64(34), Output: Bytes32{0x01}},
				{ChainID: ChainIDFromUInt64(35), Output: Bytes32{0x02}},
			},
		}
		expected := &TransitionState{
			SuperRoot: superRoot.Marshal(),
		}
		data := superRoot.Marshal()
		actual, err := UnmarshalTransitionState(data)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}
