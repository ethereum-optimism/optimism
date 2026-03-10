package derive

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestAttributesQueue checks that it properly uses the PreparePayloadAttributes function
// (which is well tested) and that it properly sets NoTxPool and adds in the candidate
// transactions.
func TestAttributesQueue(t *testing.T) {
	// test config, only init the necessary fields
	cfg := &rollup.Config{
		BlockTime:              2,
		L1ChainID:              big.NewInt(101),
		L2ChainID:              big.NewInt(102),
		DepositContractAddress: common.Address{0xbb},
		L1SystemConfigAddress:  common.Address{0xcc},
	}
	rng := rand.New(rand.NewSource(1234))
	l1Info := testutils.RandomBlockInfo(rng)

	l1Fetcher := &testutils.MockL1Source{}
	defer l1Fetcher.AssertExpectations(t)

	l1Fetcher.ExpectInfoByHash(l1Info.InfoHash, l1Info, nil)

	safeHead := testutils.RandomL2BlockRef(rng)
	safeHead.L1Origin = l1Info.ID()
	safeHead.Time = l1Info.InfoTime

	batch := SingularBatch{
		ParentHash:   safeHead.Hash,
		EpochNum:     rollup.Epoch(l1Info.InfoNum),
		EpochHash:    l1Info.InfoHash,
		Timestamp:    safeHead.Time + cfg.BlockTime,
		Transactions: []eth.Data{eth.Data("foobar"), eth.Data("example")},
	}

	parentL1Cfg := eth.SystemConfig{
		BatcherAddr: common.Address{42},
		Overhead:    [32]byte{},
		Scalar:      [32]byte{},
		GasLimit:    1234,
	}
	expectedL1Cfg := eth.SystemConfig{
		BatcherAddr: common.Address{42},
		Overhead:    [32]byte{},
		Scalar:      [32]byte{},
		GasLimit:    1234,
	}

	l2Fetcher := &testutils.MockL2Client{}
	l2Fetcher.ExpectSystemConfigByL2Hash(safeHead.Hash, parentL1Cfg, nil)

	rollupCfg := rollup.Config{}
	l1InfoTx, err := L1InfoDepositBytes(&rollupCfg, params.MergedTestChainConfig, expectedL1Cfg, safeHead.SequenceNumber+1, l1Info, 0)
	require.NoError(t, err)
	attrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(safeHead.Time + cfg.BlockTime),
		PrevRandao:            eth.Bytes32(l1Info.InfoMixDigest),
		SuggestedFeeRecipient: predeploys.SequencerFeeVaultAddr,
		Transactions:          []eth.Data{l1InfoTx, eth.Data("foobar"), eth.Data("example")},
		NoTxPool:              true,
		GasLimit:              (*eth.Uint64Quantity)(&expectedL1Cfg.GasLimit),
	}
	attrBuilder := NewFetchingAttributesBuilder(cfg, params.MergedTestChainConfig, nil, l1Fetcher, l2Fetcher)

	aq := NewAttributesQueue(testlog.Logger(t, log.LevelError), cfg, attrBuilder, nil)

	actual, err := aq.createNextAttributes(context.Background(), &batch, safeHead)

	require.NoError(t, err)
	require.Equal(t, attrs, *actual)
}

// mockSingularBatchProvider is a minimal mock for testing ApplyDepositsOnly.
type mockSingularBatchProvider struct {
	flushed bool
}

func (m *mockSingularBatchProvider) FlushChannel()          { m.flushed = true }
func (m *mockSingularBatchProvider) Origin() eth.L1BlockRef { return eth.L1BlockRef{} }
func (m *mockSingularBatchProvider) NextBatch(context.Context, eth.L2BlockRef) (*SingularBatch, bool, error) {
	return nil, false, nil
}
func (m *mockSingularBatchProvider) Reset(context.Context, eth.L1BlockRef, eth.SystemConfig) error {
	return nil
}

func TestApplyDepositsOnly_WithNilLastAttribs(t *testing.T) {
	// Simulate the race condition: lastAttribs is nil (cleared by a pipeline reset)
	// but the caller provides attributes directly via the event.
	mock := &mockSingularBatchProvider{}
	aq := NewAttributesQueue(testlog.Logger(t, log.LevelError), &rollup.Config{}, nil, mock)

	// Verify lastAttribs is nil (simulating post-reset state)
	require.Nil(t, aq.lastAttribs)

	// The original DepositsOnlyAttributes would fail here
	_, err := aq.DepositsOnlyAttributes(eth.BlockID{}, eth.L1BlockRef{})
	require.Error(t, err, "no attributes generated yet")

	// But ApplyDepositsOnly works with provided attributes
	depositTx := eth.Data{0x7e, 0x01} // deposit tx type
	userTx := eth.Data{0x02, 0x01}    // non-deposit tx
	attribs := &AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:    123,
			Transactions: []eth.Data{depositTx, userTx},
		},
		Parent:      eth.L2BlockRef{Number: 1},
		DerivedFrom: eth.L1BlockRef{Number: 10},
	}

	result := aq.ApplyDepositsOnly(attribs)

	require.True(t, mock.flushed, "FlushChannel should have been called")
	require.NotNil(t, result)
	require.True(t, result.Attributes.IsDepositsOnly())
	require.Len(t, result.Attributes.Transactions, 1, "should only have the deposit tx")
	require.Equal(t, depositTx, result.Attributes.Transactions[0])
	// lastAttribs should be updated
	require.Equal(t, result, aq.lastAttribs)
}

func TestApplyDepositsOnly_PreservesParentAndDerivedFrom(t *testing.T) {
	mock := &mockSingularBatchProvider{}
	aq := NewAttributesQueue(testlog.Logger(t, log.LevelError), &rollup.Config{}, nil, mock)

	parent := eth.L2BlockRef{Number: 42, Hash: common.Hash{0xaa}}
	derivedFrom := eth.L1BlockRef{Number: 100, Hash: common.Hash{0xbb}}
	attribs := &AttributesWithParent{
		Attributes: &eth.PayloadAttributes{
			Timestamp:    456,
			Transactions: []eth.Data{{0x7e, 0x01}},
		},
		Parent:      parent,
		DerivedFrom: derivedFrom,
	}

	result := aq.ApplyDepositsOnly(attribs)

	require.Equal(t, parent, result.Parent)
	require.Equal(t, derivedFrom, result.DerivedFrom)
}
