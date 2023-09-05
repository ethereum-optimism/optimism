package genesis

import (
	"testing"
	"time"

	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/node/nodefakes"
	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/common/hexutility"
	"github.com/ledgerwatch/erigon/cmd/rpcdaemon/commands"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/stretchr/testify/require"
)

func TestDBConnect(t *testing.T) {
	l2PrivateClient := &nodefakes.FakeRPC{}
	l2PublicClient := &nodefakes.FakeRPC{}

	timestamp := time.Now().Unix()
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
	block := node.Block{Number: 0, GasLimit: 1000000, Transactions: []*common.Hash{&txHash}, Time: (hexutil.Uint64)(timestamp)}

	c := &ConnectEngine{
		l2PrivateClient:   l2PrivateClient,
		l2PublicClient:    l2PublicClient,
		startingTimestamp: int(timestamp),
		rpcTimeout:        time.Second,
	}

	l2PublicClient.GetLatestBlockReturns(nil, ErrGetLatestBlock)
	err := c.Start()
	require.ErrorIs(t, err, ErrGetLatestBlock)

	l2PublicClient.GetLatestBlockReturns(&node.Block{Number: 0}, nil)
	err = c.Start()
	require.ErrorContains(t, err, "difficulty is not 2")

	l2PublicClient.GetLatestBlockReturns(&node.Block{Number: 0, Difficulty: (hexutil.Big)(*common.Big2)}, nil)

	l2PrivateClient.ForkchoiceUpdateV1Returns(nil, ErrForkchoice)
	err = c.Start()
	require.ErrorIs(t, err, ErrForkchoice)

	l2PrivateClient.ForkchoiceUpdateV1Returns(&node.ForkchoiceUpdatedResult{
		PayloadStatus: node.PayloadStatusV1{
			Status: "VALID",
		},
		PayloadID: &node.PayloadID{1},
	}, nil)
	l2PrivateClient.GetPayloadV1Returns(nil, ErrNewPayload)
	err = c.Start()
	require.ErrorIs(t, err, ErrNewPayload)

	executionPayloadTx := make([]hexutility.Bytes, 0)
	l2PrivateClient.GetPayloadV1Returns(&commands.ExecutionPayload{
		BlockHash:    block.Hash,
		Transactions: executionPayloadTx,
	}, nil)
	l2PrivateClient.NewPayloadV1Returns(nil, ErrNewPayload)
	err = c.Start()
	require.ErrorContains(t, err, "timestamp is not expected")
	l2PrivateClient.GetPayloadV1Returns(&commands.ExecutionPayload{
		Timestamp:    (hexutil.Uint64)(timestamp),
		BlockHash:    block.Hash,
		Transactions: executionPayloadTx,
	}, nil)
	err = c.Start()
	require.ErrorIs(t, err, ErrNewPayload)

	l2PrivateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "INVALID",
		LatestValidHash: &block.Hash,
	}, nil)
	err = c.Start()
	require.ErrorContains(t, err, "payload is invalid")

	l2PrivateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "VALID",
		LatestValidHash: &common.Hash{1},
	}, nil)
	err = c.Start()
	require.ErrorContains(t, err, "latest valid hash is not correct")

	l2PrivateClient.NewPayloadV1Returns(&node.PayloadStatusV1{
		Status:          "VALID",
		LatestValidHash: &block.Hash,
	}, nil)
	err = c.Start()
	require.NoError(t, err)
}
