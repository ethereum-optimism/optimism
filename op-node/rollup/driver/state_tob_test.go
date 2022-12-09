// On develop
package driver

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
)

type TestDummyOutputImpl struct {
	willError bool

	cfg *rollup.Config

	l1Origin eth.L1BlockRef
	l2Head   eth.L2BlockRef
}

func (d *TestDummyOutputImpl) PlanNextSequencerAction(sequenceErr error) (delay time.Duration, seal bool, onto eth.BlockID) {
	return 0, d.l1Origin != (eth.L1BlockRef{}), d.l2Head.ParentID()
}

func (d *TestDummyOutputImpl) StartBuildingBlock(ctx context.Context, l1Origin eth.L1BlockRef) error {
	d.l1Origin = l1Origin
	return nil
}

func (d *TestDummyOutputImpl) CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error) {
	// If we're meant to error, return one
	if d.willError {
		return nil, errors.New("the TestDummyOutputImpl.createNewBlock operation failed")
	}
	info := &testutils.MockBlockInfo{
		InfoHash:        d.l1Origin.Hash,
		InfoParentHash:  d.l1Origin.ParentHash,
		InfoCoinbase:    common.Address{},
		InfoRoot:        common.Hash{},
		InfoNum:         d.l1Origin.Number,
		InfoTime:        d.l1Origin.Time,
		InfoMixDigest:   [32]byte{},
		InfoBaseFee:     big.NewInt(123),
		InfoReceiptRoot: common.Hash{},
	}
	infoTx, err := derive.L1InfoDepositBytes(d.l2Head.SequenceNumber, info, eth.SystemConfig{})
	if err != nil {
		panic(err)
	}
	payload := eth.ExecutionPayload{
		ParentHash:    d.l2Head.Hash,
		FeeRecipient:  common.Address{},
		StateRoot:     eth.Bytes32{},
		ReceiptsRoot:  eth.Bytes32{},
		LogsBloom:     eth.Bytes256{},
		PrevRandao:    eth.Bytes32{},
		BlockNumber:   eth.Uint64Quantity(d.l2Head.Number + 1),
		GasLimit:      0,
		GasUsed:       0,
		Timestamp:     eth.Uint64Quantity(d.l2Head.Time + d.cfg.BlockTime),
		ExtraData:     nil,
		BaseFeePerGas: eth.Uint256Quantity{},
		BlockHash:     common.Hash{123},
		Transactions:  []eth.Data{infoTx},
	}
	return &payload, nil
}

var _ SequencerIface = (*TestDummyOutputImpl)(nil)

type TestDummyDerivationPipeline struct {
	DerivationPipeline
	l2Head      eth.L2BlockRef
	l2SafeHead  eth.L2BlockRef
	l2Finalized eth.L2BlockRef
}

func (d TestDummyDerivationPipeline) Reset()                                         {}
func (d TestDummyDerivationPipeline) Step(ctx context.Context) error                 { return nil }
func (d TestDummyDerivationPipeline) SetUnsafeHead(head eth.L2BlockRef)              {}
func (d TestDummyDerivationPipeline) AddUnsafePayload(payload *eth.ExecutionPayload) {}
func (d TestDummyDerivationPipeline) Finalized() eth.L2BlockRef                      { return d.l2Head }
func (d TestDummyDerivationPipeline) SafeL2Head() eth.L2BlockRef                     { return d.l2SafeHead }
func (d TestDummyDerivationPipeline) UnsafeL2Head() eth.L2BlockRef                   { return d.l2Finalized }

type TestDummyL1OriginSelector struct {
	retval eth.L1BlockRef
}

func (l TestDummyL1OriginSelector) FindL1Origin(ctx context.Context, l1Head eth.L1BlockRef, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	return l.retval, nil
}

// TestRejectCreateBlockBadTimestamp tests that a block creation with invalid timestamps will be caught.
// This does not test:
// - The findL1Origin call (it is hardcoded to be the head)
// - The outputInterface used to create a new block from a given payload.
// - The DerivationPipeline setting unsafe head (a mock provider is used to pretend to set it)
// - Metrics (only mocked enough to let the method proceed)
// - Publishing (network is set to nil so publishing won't occur)
func TestRejectCreateBlockBadTimestamp(t *testing.T) {
	// Create our random provider
	rng := rand.New(rand.NewSource(rand.Int63()))

	// Create our context for methods to execute under
	ctx := context.Background()

	// Create our fake L1/L2 heads and link them accordingly
	l1HeadRef := testutils.RandomBlockRef(rng)
	l2HeadRef := testutils.RandomL2BlockRef(rng)
	l2l1OriginBlock := l1HeadRef
	l2HeadRef.L1Origin = l2l1OriginBlock.ID()

	// Create a rollup config
	cfg := rollup.Config{
		BlockTime: uint64(60),
		Genesis: rollup.Genesis{
			L1:     l1HeadRef.ID(),
			L2:     l2HeadRef.ID(),
			L2Time: 0x7000, // dummy value
		},
	}

	// Patch our timestamp so we fail
	l2HeadRef.Time = l2l1OriginBlock.Time - (cfg.BlockTime * 2)

	// Create our outputter
	outputProvider := &TestDummyOutputImpl{cfg: &cfg, l2Head: l2HeadRef, willError: false}

	// Create our state
	s := Driver{
		l1State: &L1State{
			l1Head:  l1HeadRef,
			log:     log.New(),
			metrics: metrics.NoopMetrics,
		},
		log:              log.New(),
		l1OriginSelector: TestDummyL1OriginSelector{retval: l1HeadRef},
		config:           &cfg,
		sequencer:        outputProvider,
		derivation:       TestDummyDerivationPipeline{},
		metrics:          metrics.NoopMetrics,
	}

	// Create a new block
	// - L2Head's L1Origin, its timestamp should be greater than L1 genesis.
	// - L2Head timestamp + BlockTime should be greater than or equal to the L1 Time.
	err := s.startNewL2Block(ctx)
	if err == nil {
		err = s.completeNewBlock(ctx)
	}

	// Verify the L1Origin's block number is greater than L1 genesis in our config.
	if l2l1OriginBlock.Number < s.config.Genesis.L1.Number {
		require.NoError(t, err, "L1Origin block number should be greater than the L1 genesis block number")
	}

	// Verify the new L2 block to create will have a time stamp equal or newer than our L1 origin block we derive from.
	if l2HeadRef.Time+cfg.BlockTime < l2l1OriginBlock.Time {
		// If not, we expect a specific error.
		// TODO: This isn't the cleanest, we should construct + compare the whole error message.
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "cannot build L2 block on top")
		require.Contains(t, err.Error(), "for time")
		require.Contains(t, err.Error(), "before L1 origin")
		return
	}

	// If we expected the outputter to error, capture that here
	if outputProvider.willError {
		require.NotNil(t, err, "outputInterface failed to createNewBlock, so createNewL2Block should also have failed")
		return
	}

	// Otherwise we should have no error.
	require.NoError(t, err, "error raised in TestRejectCreateBlockBadTimestamp")
}

// FuzzRejectCreateBlockBadTimestamp is a property test derived from the TestRejectCreateBlockBadTimestamp unit test.
// It fuzzes timestamps and block times to find a configuration to violate error checking.
func FuzzRejectCreateBlockBadTimestamp(f *testing.F) {
	f.Fuzz(func(t *testing.T, randSeed int64, l2Time uint64, blockTime uint64, forceOutputFail bool, currentL2HeadTime uint64) {
		// Create our random provider
		rng := rand.New(rand.NewSource(randSeed))

		// Create our context for methods to execute under
		ctx := context.Background()

		// Create our fake L1/L2 heads and link them accordingly
		l1HeadRef := testutils.RandomBlockRef(rng)
		l2HeadRef := testutils.RandomL2BlockRef(rng)
		l2l1OriginBlock := l1HeadRef
		l2HeadRef.L1Origin = l2l1OriginBlock.ID()

		// TODO: Cap our block time so it doesn't overflow
		if blockTime > 0x100000 {
			blockTime = 0x100000
		}

		// Create a rollup config
		cfg := rollup.Config{
			BlockTime: blockTime,
			Genesis: rollup.Genesis{
				L1:     l1HeadRef.ID(),
				L2:     l2HeadRef.ID(),
				L2Time: l2Time, // dummy value
			},
		}

		// Patch our timestamp so we fail
		l2HeadRef.Time = currentL2HeadTime

		// Create our outputter
		outputProvider := &TestDummyOutputImpl{cfg: &cfg, l2Head: l2HeadRef, willError: forceOutputFail}

		// Create our state
		s := Driver{
			l1State: &L1State{
				l1Head:  l1HeadRef,
				log:     log.New(),
				metrics: metrics.NoopMetrics,
			},
			log:              log.New(),
			l1OriginSelector: TestDummyL1OriginSelector{retval: l1HeadRef},
			config:           &cfg,
			sequencer:        outputProvider,
			derivation:       TestDummyDerivationPipeline{},
			metrics:          metrics.NoopMetrics,
		}

		// Create a new block
		// - L2Head's L1Origin, its timestamp should be greater than L1 genesis.
		// - L2Head timestamp + BlockTime should be greater than or equal to the L1 Time.
		err := s.startNewL2Block(ctx)
		if err == nil {
			err = s.completeNewBlock(ctx)
		}

		// Verify the L1Origin's timestamp is greater than L1 genesis in our config.
		if l2l1OriginBlock.Number < s.config.Genesis.L1.Number {
			require.NoError(t, err)
			return
		}

		// Verify the new L2 block to create will have a time stamp equal or newer than our L1 origin block we derive from.
		if l2HeadRef.Time+cfg.BlockTime < l2l1OriginBlock.Time {
			// If not, we expect a specific error.
			// TODO: This isn't the cleanest, we should construct + compare the whole error message.
			require.NotNil(t, err)
			require.Contains(t, err.Error(), "cannot build L2 block on top")
			require.Contains(t, err.Error(), "for time")
			require.Contains(t, err.Error(), "before L1 origin")
			return
		}

		// Otherwise we should have no error.
		require.Nil(t, err)

		// If we expected the outputter to error, capture that here
		if outputProvider.willError {
			require.NotNil(t, err, "outputInterface failed to createNewBlock, so createNewL2Block should also have failed")
			return
		}

		// Otherwise we should have no error.
		require.NoError(t, err, "L1Origin block number should be greater than the L1 genesis block number")
	})
}
