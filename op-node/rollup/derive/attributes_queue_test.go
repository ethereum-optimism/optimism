package derive

import (
	"context"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type MockAttributesQueueOutput struct {
	MockOriginStage
}

func (m *MockAttributesQueueOutput) AddSafeAttributes(attributes *eth.PayloadAttributes) {
	m.Mock.MethodCalled("AddSafeAttributes", attributes)
}

func (m *MockAttributesQueueOutput) ExpectAddSafeAttributes(attributes *eth.PayloadAttributes) {
	m.Mock.On("AddSafeAttributes", attributes).Once().Return()
}

func (m *MockAttributesQueueOutput) SafeL2Head() eth.L2BlockRef {
	return m.Mock.MethodCalled("SafeL2Head").Get(0).(eth.L2BlockRef)
}

func (m *MockAttributesQueueOutput) ExpectSafeL2Head(head eth.L2BlockRef) {
	m.Mock.On("SafeL2Head").Once().Return(head)
}

var _ AttributesQueueOutput = (*MockAttributesQueueOutput)(nil)

func TestAttributesQueue_Step(t *testing.T) {
	// test config, only init the necessary fields
	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		FeeRecipientAddress:    common.Address{0xaa},
		DepositContractAddress: common.Address{0xbb},
	}
	rng := rand.New(rand.NewSource(1234))
	l1Info := testutils.RandomL1Info(rng)

	l1Fetcher := &testutils.MockL1Source{}
	defer l1Fetcher.AssertExpectations(t)

	l1Fetcher.ExpectInfoByHash(l1Info.InfoHash, l1Info, nil)

	out := &MockAttributesQueueOutput{}
	out.progress = Progress{
		Origin: l1Info.BlockRef(),
		Closed: false,
	}
	defer out.AssertExpectations(t)

	safeHead := testutils.RandomL2BlockRef(rng)
	safeHead.L1Origin = l1Info.ID()

	out.ExpectSafeL2Head(safeHead)

	batch := &BatchData{BatchV1{
		EpochNum:     rollup.Epoch(l1Info.InfoNum),
		EpochHash:    l1Info.InfoHash,
		Timestamp:    12345,
		Transactions: []eth.Data{eth.Data("foobar"), eth.Data("example")},
	}}

	l1InfoTx, err := L1InfoDepositBytes(safeHead.SequenceNumber+1, l1Info)
	require.NoError(t, err)
	attrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(safeHead.Time + cfg.BlockTime),
		PrevRandao:            eth.Bytes32(l1Info.InfoMixDigest),
		SuggestedFeeRecipient: cfg.FeeRecipientAddress,
		Transactions:          []eth.Data{l1InfoTx, eth.Data("foobar"), eth.Data("example")},
		NoTxPool:              true,
	}
	out.ExpectAddSafeAttributes(&attrs)

	aq := NewAttributesQueue(testlog.Logger(t, log.LvlError), cfg, l1Fetcher, out)
	require.NoError(t, RepeatResetStep(t, aq.ResetStep, l1Fetcher, 1))

	aq.AddBatch(batch)

	require.NoError(t, aq.Step(context.Background(), out.progress), "adding batch to next stage, no EOF yet")
	require.Equal(t, io.EOF, aq.Step(context.Background(), out.progress), "done with batches")
}
