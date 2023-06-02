package genesis

import (
	"context"
	"testing"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestRegenerateBlock(t *testing.T) {
	publicClient := &node.MockRPC{}
	legacyClient := &node.MockRPC{}
	privateClient := &node.MockRPC{}

	to := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	// R := uint256.NewInt(1)
	transaction := &types.LegacyTx{
		GasPrice: uint256.NewInt(0),
		CommonTx: types.CommonTx{
			Gas:   50000,
			To:    &to,
			Value: uint256.NewInt(1),
			R:     *uint256.NewInt(1),
			S:     *uint256.NewInt(1),
			V:     *uint256.NewInt(1),
		},
	}
	txHash := transaction.Hash()
	block := node.Block{Number: 0, GasLimit: 1000000, Transactions: []*common.Hash{&txHash}}

	b := &BuilderEngine{
		ctx:                 context.Background(),
		stop:                make(chan struct{}),
		l2PrivateClient:     privateClient,
		l2PublicClient:      publicClient,
		l2LegacyClient:      legacyClient,
		rpcTimeout:          0,
		pollingInterval:     0,
		hardforkBlockNumber: 1,
	}

	publicClient.SetResponse("GetLatestBlock", "")
	err := b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrGetLatestBlock)

	publicClient.SetResponse("GetLatestBlock", &node.Block{Number: 0})
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrGetBlockByNum)

	legacyClient.SetResponse("GetBlockByNumber", &block)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrGetTxByHash)

	fakeTransaction := &types.LegacyTx{
		GasPrice: uint256.NewInt(100),
		CommonTx: types.CommonTx{
			Gas: 50000000,
			To:  &to,
		},
	}
	legacyClient.SetResponse("GetTransactionByHash", fakeTransaction)
	err = b.RegenerateBlock()
	require.Error(t, err)

	legacyClient.SetResponse("GetTransactionByHash", transaction)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrForkchoice)

	privateClient.SetResponse("ForkchoiceUpdateV1", &node.ForkchoiceUpdatedResult{
		PayloadStatus: node.PayloadStatusV1{
			Status: "VALID",
		},
		PayloadID: &node.PayloadID{1},
	})
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrGetPayload)

	marshalTransactions, err := types.MarshalTransactionsBinary(types.Transactions{transaction})
	require.NoError(t, err)
	executionPayloadTx := make([]hexutility.Bytes, 1)
	executionPayloadTx[0] = hexutility.Bytes(marshalTransactions[0])
	privateClient.SetResponse("GetPayloadV1", &commands.ExecutionPayload{
		BlockHash:    block.Hash,
		Transactions: executionPayloadTx,
	})
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, node.ErrNewPayload)

	privateClient.SetResponse("NewPayloadV1", &node.PayloadStatusV1{
		Status:          "INVALID",
		LatestValidHash: &block.Hash,
	})
	err = b.RegenerateBlock()
	require.Error(t, err)

	privateClient.SetResponse("NewPayloadV1", &node.PayloadStatusV1{
		Status:          "INVALID",
		LatestValidHash: &common.Hash{1},
	})
	err = b.RegenerateBlock()
	require.Error(t, err)

	privateClient.SetResponse("NewPayloadV1", &node.PayloadStatusV1{
		Status:          "VALID",
		LatestValidHash: &block.Hash,
	})
	err = b.RegenerateBlock()
	require.NoError(t, err)
}
