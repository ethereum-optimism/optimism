package driver

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestInteropMessageQueue(t *testing.T) {
	log := testlog.Logger(t, log.LvlDebug)
	localChainId := big.NewInt(990)

	// ignore all rpcs
	rpc := &testutils.MockRPCClient{}
	rpc.On("CallContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	remoteRpcs := map[uint64]client.RPC{991: rpc, 992: rpc}
	q, err := NewInteropMessageQueue(log, localChainId, remoteRpcs)
	require.NoError(t, err)

	// Two Batches Ready in 991. T=1, T= 2
	q.msgFetchers[991].buffer <- derive.InteropMessages{}
	q.msgFetchers[991].buffer <- derive.InteropMessages{}

	// One Batch Ready for 992. T = 0
	q.msgFetchers[992].buffer <- derive.InteropMessages{}

	// Queue will fill up with one item per chain
	wait.For(context.Background(), 100*time.Millisecond, func() (bool, error) { return len(q.newMsgs) == 2, nil })
	msgs := q.NewMessages()
	require.Len(t, msgs, 2)

	// Second batch in 991 flushed to the queue
	wait.For(context.Background(), 100*time.Millisecond, func() (bool, error) { return len(q.newMsgs) == 1, nil })
	msgs = q.NewMessages()
	require.Len(t, msgs, 1)
}
