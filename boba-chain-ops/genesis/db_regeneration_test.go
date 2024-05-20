package genesis

import (
	"context"
	"errors"
	"testing"

	"github.com/bobanetwork/boba/boba-chain-ops/node"
	"github.com/bobanetwork/boba/boba-chain-ops/node/nodefakes"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/turbo/engineapi/engine_types"
	"github.com/stretchr/testify/require"
)

var (
	ErrGetLatestBlock = errors.New("GetLatestBlock mock rpc error")
	ErrGetBlockByNum  = errors.New("GetBlockByNumber mock rpc error")
	ErrGetTxByHash    = errors.New("GetTransactionByHash mock rpc error")
	ErrForkchoice     = errors.New("ForkchoiceUpdateV1 mock rpc error")
	ErrGetPayload     = errors.New("GetPayloadV1 mock rpc error")
	ErrNewPayload     = errors.New("NewPayloadV1 mock rpc error")
)

func TestRegenerateBlock(t *testing.T) {
	publicClient := &nodefakes.FakeRPC{}
	legacyClient := &nodefakes.FakeRPC{}
	privateClient := &nodefakes.FakeRPC{}

	to := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
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

	publicClient.GetLatestBlockReturns(nil, ErrGetLatestBlock)
	err := b.RegenerateBlock()
	require.ErrorIs(t, err, ErrGetLatestBlock)

	publicClient.GetLatestBlockReturns(&node.Block{Number: 0}, nil)
	legacyClient.GetBlockByNumberReturns(nil, ErrGetBlockByNum)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, ErrGetBlockByNum)

	legacyClient.GetBlockByNumberReturns(&block, nil)
	legacyClient.GetTransactionByHashReturns(nil, ErrGetTxByHash)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, ErrGetTxByHash)

	fakeTransaction := &types.LegacyTx{
		GasPrice: uint256.NewInt(100),
		CommonTx: types.CommonTx{
			Gas: 50000000,
			To:  &to,
		},
	}
	legacyClient.GetTransactionByHashReturns(fakeTransaction, nil)
	err = b.RegenerateBlock()
	require.ErrorContains(t, err, "not match")

	legacyClient.GetTransactionByHashReturns(transaction, nil)
	privateClient.ForkchoiceUpdateV1Returns(nil, ErrForkchoice)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, ErrForkchoice)

	privateClient.ForkchoiceUpdateV1Returns(&node.ForkchoiceUpdatedResult{
		PayloadStatus: node.PayloadStatusV1{
			Status: "VALID",
		},
		PayloadID: &node.PayloadID{1},
	}, nil)
	privateClient.GetPayloadV1Returns(nil, ErrNewPayload)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, ErrNewPayload)

	marshalTransactions, err := types.MarshalTransactionsBinary(types.Transactions{transaction})
	require.NoError(t, err)
	executionPayloadTx := make([]hexutility.Bytes, 1)
	executionPayloadTx[0] = hexutility.Bytes(marshalTransactions[0])
	privateClient.GetPayloadV1Returns(&engine_types.ExecutionPayload{
		BlockHash:    block.Hash,
		Transactions: executionPayloadTx,
	}, nil)
	privateClient.NewPayloadV1Returns(nil, ErrNewPayload)
	err = b.RegenerateBlock()
	require.ErrorIs(t, err, ErrNewPayload)

	privateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "INVALID",
		LatestValidHash: &block.Hash,
	}, nil)
	err = b.RegenerateBlock()
	require.ErrorContains(t, err, "payload is invalid")

	privateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "VALID",
		LatestValidHash: &common.Hash{1},
	}, nil)
	err = b.RegenerateBlock()
	require.ErrorContains(t, err, "latest valid hash is not correct")

	privateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "VALID",
		LatestValidHash: &block.Hash,
	}, nil)
	err = b.RegenerateBlock()
	require.NoError(t, err)
}
