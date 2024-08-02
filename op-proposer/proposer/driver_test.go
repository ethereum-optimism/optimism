package proposer

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/bindings"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
)

type MockL2OOContract struct {
	mock.Mock
}

func (m *MockL2OOContract) Version(opts *bind.CallOpts) (string, error) {
	args := m.Called(opts)
	return args.String(0), args.Error(1)
}

func (m *MockL2OOContract) NextBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	args := m.Called(opts)
	return args.Get(0).(*big.Int), args.Error(1)
}

type mockL2EndpointProvider struct {
	ethClient       *testutils.MockL2Client
	ethClientErr    error
	rollupClient    *testutils.MockRollupClient
	rollupClientErr error
}

func newEndpointProvider() *mockL2EndpointProvider {
	return &mockL2EndpointProvider{
		ethClient:    new(testutils.MockL2Client),
		rollupClient: new(testutils.MockRollupClient),
	}
}

func (p *mockL2EndpointProvider) EthClient(context.Context) (dial.EthClientInterface, error) {
	return p.ethClient, p.ethClientErr
}

func (p *mockL2EndpointProvider) RollupClient(context.Context) (dial.RollupClientInterface, error) {
	return p.rollupClient, p.rollupClientErr
}

func (p *mockL2EndpointProvider) Close() {}

func setup(t *testing.T) (*L2OutputSubmitter, *mockL2EndpointProvider, *MockL2OOContract) {
	ep := newEndpointProvider()

	l2OutputOracleAddr := common.HexToAddress("0x3F8A862E63E759a77DA22d384027D21BF096bA9E")

	proposerConfig := ProposerConfig{
		PollInterval:        20 * time.Millisecond,
		ProposalInterval:    20 * time.Millisecond,
		OutputRetryInterval: 1 * time.Millisecond,
		L2OutputOracleAddr:  &l2OutputOracleAddr,
	}

	txmgr := mocks.TxManager{}
	txmgr.On("From").Return(common.Address{})
	txmgr.On("BlockNumber", mock.Anything).Return(uint64(100), nil)
	txmgr.On("Send", mock.Anything, mock.Anything).Return(&types.Receipt{Status: uint64(1), TxHash: common.Hash{}}, nil)

	setup := DriverSetup{
		Log:            testlog.Logger(t, log.LevelDebug),
		Metr:           metrics.NoopMetrics,
		Cfg:            proposerConfig,
		Txmgr:          &txmgr,
		RollupProvider: ep,
	}

	parsed, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	l2ooContract := MockL2OOContract{}
	l2OutputSubmitter := L2OutputSubmitter{
		DriverSetup:  setup,
		done:         make(chan struct{}),
		l2ooContract: &l2ooContract,
		l2ooABI:      parsed,
		ctx:          ctx,
		cancel:       cancel,
	}

	return &l2OutputSubmitter, ep, &l2ooContract
}

func TestL2OutputSubmitter_OutputRetry(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "loopL2OO"},
		{name: "loopDGF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps, ep, l2ooContract := setup(t)

			ep.rollupClient.On("SyncStatus").Return(&eth.SyncStatus{FinalizedL2: eth.L2BlockRef{Number: 42}}, nil).Once()
			ep.rollupClient.ExpectOutputAtBlock(42, nil, fmt.Errorf("failed to fetch output"))
			ep.rollupClient.ExpectOutputAtBlock(
				42,
				&eth.OutputResponse{
					Version:  supportedL2OutputVersion,
					BlockRef: eth.L2BlockRef{Number: 42},
					Status: &eth.SyncStatus{
						CurrentL1:   eth.L1BlockRef{Hash: common.Hash{}},
						FinalizedL2: eth.L2BlockRef{Number: 42},
					}},
				nil,
			)

			if tt.name == "loopDGF" {
				go ps.loopDGF(ps.ctx)
			} else {
				l2ooContract.On("NextBlockNumber", mock.AnythingOfType("*bind.CallOpts")).Return(big.NewInt(42), nil).Once()
				l2ooContract.On("NextBlockNumber", mock.AnythingOfType("*bind.CallOpts")).Return(big.NewInt(42), nil).Once()
				ep.rollupClient.On("SyncStatus").Return(&eth.SyncStatus{FinalizedL2: eth.L2BlockRef{Number: 42}}, nil).Once()
				go ps.loopL2OO(ps.ctx)
			}

			time.Sleep(25 * time.Millisecond)
			close(ps.done)

			ep.rollupClient.AssertExpectations(t)
			l2ooContract.AssertExpectations(t)
		})
	}
}
