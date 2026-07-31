package proposer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer/source"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	txmgrmocks "github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type StubDGFContract struct {
	hasProposedCount int
	proposedRecently bool
	cutoff           time.Time
}

func (m *StubDGFContract) HasProposedSince(_ context.Context, _ common.Address, cutoff time.Time, _ uint32) (bool, time.Time, common.Hash, error) {
	m.hasProposedCount++
	m.cutoff = cutoff
	return m.proposedRecently, time.Unix(1000, 0), common.Hash{0xdd}, nil
}

func (m *StubDGFContract) ProposalTx(_ context.Context, _ uint32, _ common.Hash, _ []byte) (txmgr.TxCandidate, error) {
	return txmgr.TxCandidate{}, nil
}

func (m *StubDGFContract) Version(_ context.Context) (string, error) {
	panic("not implemented")
}

type mockRollupEndpointProvider struct {
	rollupClient    *testutils.MockRollupClient
	rollupClientErr error
}

func newEndpointProvider() *mockRollupEndpointProvider {
	return &mockRollupEndpointProvider{
		rollupClient: new(testutils.MockRollupClient),
	}
}

func (p *mockRollupEndpointProvider) RollupClient(context.Context) (dial.RollupClientInterface, error) {
	return p.rollupClient, p.rollupClientErr
}

func (p *mockRollupEndpointProvider) Close() {}

func setup(t *testing.T) (*L2OutputSubmitter, *mockRollupEndpointProvider, *StubDGFContract, *txmgrmocks.TxManager, *testlog.CapturingHandler) {
	ep := newEndpointProvider()

	proposerConfig := ProposerConfig{
		PollInterval:     time.Microsecond,
		ProposalInterval: time.Microsecond,
	}

	txmgr := txmgrmocks.NewTxManager(t)

	lgr, logs := testlog.CaptureLogger(t, log.LevelDebug)
	setup := DriverSetup{
		Log:            lgr,
		Metr:           metrics.NoopMetrics,
		Cfg:            proposerConfig,
		Txmgr:          txmgr,
		ProposalSource: source.NewRollupProposalSource(ep),
	}

	ctx, cancel := context.WithCancel(context.Background())

	l2OutputSubmitter := L2OutputSubmitter{
		DriverSetup: setup,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}
	mockDGFContract := new(StubDGFContract)
	l2OutputSubmitter.dgfContract = mockDGFContract

	txmgr.On("Send", mock.Anything, mock.Anything).
		Return(&types.Receipt{Status: uint64(1), TxHash: common.Hash{}}, nil).
		Once().
		Run(func(_ mock.Arguments) {
			// let loops return after first Send call
			t.Log("Closing proposer.")
			close(l2OutputSubmitter.done)
		})

	return &l2OutputSubmitter, ep, mockDGFContract, txmgr, logs
}

func TestL2OutputSubmitter_FetchDGFOutputCutoffUsesClock(t *testing.T) {
	const proposalInterval = 15 * time.Minute
	fetchCutoff := func(t *testing.T, proposerClock clock.Clock) time.Time {
		dgfContract := &StubDGFContract{proposedRecently: true}
		txmgr := txmgrmocks.NewTxManager(t)
		txmgr.On("From").Return(common.Address{0xab}).Once()
		submitter := &L2OutputSubmitter{
			DriverSetup: DriverSetup{
				Log:   testlog.Logger(t, log.LevelDebug),
				Cfg:   ProposerConfig{ProposalInterval: proposalInterval},
				Clock: proposerClock,
				Txmgr: txmgr,
			},
			dgfContract: dgfContract,
		}

		_, shouldPropose, err := submitter.FetchDGFOutput(t.Context())
		require.NoError(t, err)
		require.False(t, shouldPropose)
		return dgfContract.cutoff
	}

	t.Run("InjectedClock", func(t *testing.T) {
		now := time.Unix(1_234_567, 0)
		cutoff := fetchCutoff(t, clock.NewDeterministicClock(now))
		require.Equal(t, now.Add(-proposalInterval), cutoff)
	})

	t.Run("NilClockUsesSystemTime", func(t *testing.T) {
		before := time.Now().Add(-proposalInterval)
		cutoff := fetchCutoff(t, nil)
		after := time.Now().Add(-proposalInterval)
		require.False(t, cutoff.Before(before), "cutoff must not predate the call")
		require.False(t, cutoff.After(after), "cutoff must not postdate the call")
	})
}

func TestL2OutputSubmitter_OutputRetry(t *testing.T) {
	proposerAddr := common.Address{0xab}
	const numFails = 3

	ps, ep, dgfContract, txmgr, logs := setup(t)

	ep.rollupClient.On("SyncStatus").Return(&eth.SyncStatus{FinalizedL2: eth.L2BlockRef{Number: 42}}, nil).Times(numFails + 1)
	ep.rollupClient.ExpectOutputAtBlock(42, nil, fmt.Errorf("TEST: failed to fetch output")).Times(numFails)
	ep.rollupClient.ExpectOutputAtBlock(
		42,
		&eth.OutputResponse{
			Version:  eth.OutputVersionV0,
			BlockRef: eth.L2BlockRef{Number: 42},
			Status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Hash: common.Hash{}},
				FinalizedL2: eth.L2BlockRef{Number: 42},
			},
		},
		nil,
	)

	txmgr.On("From").Return(proposerAddr).Times(numFails + 1)

	ps.wg.Add(1)
	ps.loop()

	ep.rollupClient.AssertExpectations(t)

	require.Equal(t, numFails+1, dgfContract.hasProposedCount)

	require.Len(t, logs.FindLogs(testlog.NewMessageContainsFilter("Error getting proposal")), numFails)
	require.NotNil(t, logs.FindLog(testlog.NewMessageFilter("Proposer tx successfully published")))
	require.NotNil(t, logs.FindLog(testlog.NewMessageFilter("loop returning")))
}
