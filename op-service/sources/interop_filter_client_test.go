package sources

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestInteropFilterClient_CheckAccessList(t *testing.T) {
	ctx := context.Background()
	rpcClient := new(mockRPC)
	defer rpcClient.AssertExpectations(t)
	client := NewInteropFilterClient(rpcClient)

	inboxEntries := []common.Hash{common.HexToHash("0x01")}
	minSafety := safety.CrossUnsafe
	execDescriptor := messages.ExecutingDescriptor{
		ChainID:   eth.ChainIDFromUInt64(900),
		Timestamp: 123,
	}

	rpcClient.On(
		"CallContext",
		ctx,
		nil,
		"interop_checkAccessList",
		[]any{inboxEntries, minSafety, execDescriptor},
	).Return([]error{nil})

	require.NoError(t, client.CheckAccessList(ctx, inboxEntries, minSafety, execDescriptor))
}

func TestInteropFilterClient_GetBlockHashByNumber(t *testing.T) {
	ctx := context.Background()
	rpcClient := new(mockRPC)
	defer rpcClient.AssertExpectations(t)
	client := NewInteropFilterClient(rpcClient)

	chainID := eth.ChainIDFromUInt64(900)
	blockNum := rpc.BlockNumber(123)
	expected := common.HexToHash("0x1234")

	rpcClient.On(
		"CallContext",
		ctx,
		new(common.Hash),
		"interop_getBlockHashByNumber",
		[]any{chainID, blockNum},
	).Run(func(args mock.Arguments) {
		*args[1].(*common.Hash) = expected
	}).Return([]error{nil})

	actual, err := client.GetBlockHashByNumber(ctx, chainID, blockNum)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestInteropFilterClient_GetBlockHashByNumber_NotFound(t *testing.T) {
	ctx := context.Background()
	rpcClient := new(mockRPC)
	defer rpcClient.AssertExpectations(t)
	client := NewInteropFilterClient(rpcClient)

	chainID := eth.ChainIDFromUInt64(900)
	blockNum := rpc.BlockNumber(123)

	rpcClient.On(
		"CallContext",
		ctx,
		new(common.Hash),
		"interop_getBlockHashByNumber",
		[]any{chainID, blockNum},
	).Return([]error{errors.New("block 123 for chain 900: not found")})

	_, err := client.GetBlockHashByNumber(ctx, chainID, blockNum)
	require.ErrorIs(t, err, ethereum.NotFound)
}
