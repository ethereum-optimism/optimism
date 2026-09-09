package atomic

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestCallTracerLogOrdering(t *testing.T) {
	log := func(value byte, position uint) callLog {
		return callLog{Data: []byte{value}, Position: hexutil.Uint(position)}
	}
	frame := callFrame{
		Logs: []callLog{log(1, 0), log(4, 1), log(5, 2)},
		Calls: []callFrame{
			{Logs: []callLog{log(2, 0), log(3, 1)}, Calls: []callFrame{{Error: "execution reverted", Logs: []callLog{log(99, 0)}}}},
			{Error: "execution reverted", Calls: []callFrame{{Logs: []callLog{log(99, 0)}}}},
		},
	}
	execution, err := executionFromFrame(frame)
	require.NoError(t, err)
	var actual []byte
	for _, log := range execution.Logs {
		actual = append(actual, log.Data...)
	}
	require.Equal(t, []byte{1, 2, 3, 4, 5}, actual)
	frame.Error = "execution reverted"
	execution, err = executionFromFrame(frame)
	require.NoError(t, err)
	require.True(t, execution.Reverted)
	require.Empty(t, execution.Logs)
}
