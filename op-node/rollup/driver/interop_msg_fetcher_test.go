package driver

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func rpcOnLogs(rpc *testutils.MockRPCClient, logs []types.Log) *mock.Call {
	return rpc.On("CallContext", mock.Anything, mock.Anything, "eth_getLogs", mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			*args.Get(1).(*[]types.Log) = logs
		})
}

func encodeMessages(blockNumber uint64, messages []bindings.CrossL2OutboxMessagePassed) []types.Log {
	outboxAbi, _ := bindings.CrossL2OutboxMetaData.GetAbi()
	msgPassedArgs := outboxAbi.Events["MessagePassed"].Inputs.NonIndexed()

	logs := make([]types.Log, len(messages))
	for i, msg := range messages {
		logData, _ := msgPassedArgs.Pack(msg.TargetChain, msg.Value, msg.GasLimit, msg.Data, msg.MessageRoot)
		logs[i] = types.Log{
			Topics: []common.Hash{
				outboxAbi.Events["MessagePassed"].ID,
				common.BigToHash(msg.Nonce),
				common.BytesToHash(msg.From[:]),
				common.BytesToHash(msg.To[:]),
			},
			Data:        logData,
			BlockNumber: blockNumber,
		}
	}
	return logs
}

func TestInteropMessageFetcherFlushesMessages(t *testing.T) {
	log := testlog.Logger(t, log.LvlDebug)
	localChain, remoteChain := common.BigToHash(big.NewInt(990)), common.BigToHash(big.NewInt(991))

	// ignore the poller
	rpc := &testutils.MockRPCClient{}
	rpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.Anything).Return(nil)

	msgsSink := make(chan derive.InteropMessages)
	f := NewInteropMessageFetcher(log, localChain, remoteChain, rpc, msgsSink)

	// single message available in block 5
	rpcOnLogs(rpc, encodeMessages(5, []bindings.CrossL2OutboxMessagePassed{
		{Nonce: big.NewInt(1), TargetChain: localChain, Value: big.NewInt(0), GasLimit: big.NewInt(0)},
	}))

	// finalization update
	f.finalized <- eth.L1BlockRef{Number: 10}

	// message gets dumped into the sink
	m := <-msgsSink
	require.Equal(t, 1, len(m.Messages))
	require.Equal(t, uint64(0), m.SourceInfo.FromBlockNumber.Uint64())
	require.Equal(t, uint64(10), m.SourceInfo.ToBlockNumber.Uint64())
}

func TestInteropMessageFetcherMessageBuffer(t *testing.T) {
	log := testlog.Logger(t, log.LvlDebug)
	localChain, remoteChain := common.BigToHash(big.NewInt(990)), common.BigToHash(big.NewInt(991))

	// ignore the poller
	rpc := &testutils.MockRPCClient{}
	rpc.On("CallContext", mock.Anything, mock.Anything, "eth_getBlockByNumber", mock.Anything).Return(nil)

	msgsSink := make(chan derive.InteropMessages)
	f := NewInteropMessageFetcher(log, localChain, remoteChain, rpc, msgsSink)

	// single message in block 5
	rpcOnLogs(rpc, encodeMessages(5, []bindings.CrossL2OutboxMessagePassed{
		{Nonce: big.NewInt(1), TargetChain: localChain, Value: big.NewInt(0), GasLimit: big.NewInt(0)},
	})).Times(1)

	// finalization update causing a read (immediately available in the sink)
	f.finalized <- eth.L1BlockRef{Number: 10}

	// another message in block 15
	rpcOnLogs(rpc, encodeMessages(15, []bindings.CrossL2OutboxMessagePassed{
		{Nonce: big.NewInt(2), TargetChain: localChain, Value: big.NewInt(0), GasLimit: big.NewInt(0)},
	}))

	// finalization update causing a read (placed in the buffer)
	f.finalized <- eth.L1BlockRef{Number: 20}
	wait.For(context.Background(), 100*time.Millisecond, func() (bool, error) { return len(f.buffer) == 1, nil })

	// both messages can be read through the sink
	firstMsg := <-msgsSink
	require.Equal(t, 1, len(firstMsg.Messages))
	require.Equal(t, uint64(0), firstMsg.SourceInfo.FromBlockNumber.Uint64())
	require.Equal(t, uint64(10), firstMsg.SourceInfo.ToBlockNumber.Uint64())
	secondMsg := <-msgsSink
	require.Equal(t, 1, len(secondMsg.Messages))
	require.Equal(t, uint64(11), secondMsg.SourceInfo.FromBlockNumber.Uint64())
	require.Equal(t, uint64(20), secondMsg.SourceInfo.ToBlockNumber.Uint64())
}
