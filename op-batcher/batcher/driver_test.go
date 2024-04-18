package batcher

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr/mocks"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockL1Client struct {
	mock.Mock
}

func (m *MockL1Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	args := m.Called(ctx, account, blockNumber)
	return args.Get(0).(uint64), args.Error(1)
}

func (m *MockL1Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	if header, ok := args.Get(0).(*types.Header); ok {
		return header, args.Error(1)
	}
	return nil, args.Error(1)
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

const genesisL1Origin = uint64(123)

func setup(t *testing.T) (*BatchSubmitter, *mockL2EndpointProvider) {
	ep := newEndpointProvider()

	cfg := defaultTestRollupConfig
	cfg.Genesis.L1.Number = genesisL1Origin

	return NewBatchSubmitter(DriverSetup{
		Log:              testlog.Logger(t, log.LevelDebug),
		Metr:             metrics.NoopMetrics,
		RollupConfig:     &cfg,
		EndpointProvider: ep,
		Txmgr:            new(mocks.TxManager),
		L1Client:         new(MockL1Client),
	}), ep
}

func TestBatchSubmitter_SafeL1Origin(t *testing.T) {
	bs, ep := setup(t)

	tests := []struct {
		name                   string
		currentSafeOrigin      uint64
		failsToFetchSyncStatus bool
		expectResult           uint64
		expectErr              bool
	}{
		{
			name:              "ExistingSafeL1Origin",
			currentSafeOrigin: 999,
			expectResult:      999,
		},
		{
			name:              "NoExistingSafeL1OriginUsesGenesis",
			currentSafeOrigin: 0,
			expectResult:      genesisL1Origin,
		},
		{
			name:                   "ErrorFetchingSyncStatus",
			failsToFetchSyncStatus: true,
			expectErr:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.failsToFetchSyncStatus {
				ep.rollupClient.ExpectSyncStatus(&eth.SyncStatus{}, errors.New("failed to fetch sync status"))

			} else {
				ep.rollupClient.ExpectSyncStatus(&eth.SyncStatus{
					SafeL2: eth.L2BlockRef{
						L1Origin: eth.BlockID{
							Number: tt.currentSafeOrigin,
						},
					},
				}, nil)
			}

			id, err := bs.safeL1Origin(context.Background())

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectResult, id.Number)
			}
		})
	}

}

func TestBatchSubmitter_SafeL1Origin_FailsToResolveRollupClient(t *testing.T) {
	bs, ep := setup(t)

	ep.rollupClientErr = errors.New("failed to resolve rollup client")

	_, err := bs.safeL1Origin(context.Background())
	require.Error(t, err)
}

func TestBatchSubmitter_CheckRecentTxsOnStart(t *testing.T) {
	bs, ep := setup(t)
	txMgr := bs.Txmgr.(*mocks.TxManager)
	l1Client := bs.L1Client.(*MockL1Client)

	tests := []struct {
		name               string
		currentBlock       uint64
		blockConfirms      uint64
		previousNonceBlock uint64
		expectWaitSync     bool
	}{
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          5   5   5   6   6   6
			// call NonceAt   x   -   x   x   x   x
			name:               "NonceChanged_3Blocks",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 497,
			expectWaitSync:     true,
		},
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          5   5   5   5   5   6
			// call NonceAt   x   -   -   -   x   x
			name:               "NonceChanged_1BlockAgo",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 499,
			expectWaitSync:     true,
		},
		{
			// Blocks       495 496 497 498 499 500
			// Nonce          6   6   6   6   6   6
			// call NonceAt   x   -   -   -   -   x
			name:               "NonceUnchanged",
			currentBlock:       500,
			blockConfirms:      5,
			previousNonceBlock: 400,
			expectWaitSync:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l1Client.ExpectedCalls = nil
			l1Client.Calls = nil

			currentNonce := uint64(6)
			previousNonce := uint64(5)

			txMgr.On("BlockNumber", bs.shutdownCtx).Return(tt.currentBlock, nil)
			txMgr.On("From").Return(common.Address{})

			// Setup mock calls for NonceAt, depending on how many times its expected to be called
			if tt.previousNonceBlock < tt.currentBlock-tt.blockConfirms {
				l1Client.On("NonceAt", bs.shutdownCtx, common.Address{}, big.NewInt(int64(tt.currentBlock))).Return(currentNonce, nil)
				l1Client.On("NonceAt", bs.shutdownCtx, common.Address{}, big.NewInt(int64(tt.currentBlock-tt.blockConfirms))).Return(currentNonce, nil)
			} else {
				for block := tt.currentBlock; block >= (tt.currentBlock - tt.blockConfirms); block-- {
					blockBig := big.NewInt(int64(block))
					if block > (tt.currentBlock-tt.blockConfirms) && block < tt.previousNonceBlock {
						t.Log("skipped block: ", block)
						continue
					} else if block <= tt.previousNonceBlock {
						t.Log("previousNonce set at block: ", block)
						l1Client.On("NonceAt", bs.shutdownCtx, common.Address{}, blockBig).Return(previousNonce, nil)
					} else {
						t.Log("currentNonce set at block: ", block)
						l1Client.On("NonceAt", bs.shutdownCtx, common.Address{}, blockBig).Return(currentNonce, nil)
					}
				}
			}

			ep.rollupClient.ExpectRollupConfig(&rollup.Config{VerifierConfDepth: tt.blockConfirms}, nil)
			if tt.expectWaitSync {
				ep.rollupClient.ExpectSyncStatus(&eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: tt.currentBlock + tt.blockConfirms}}, nil)
			}

			bs.checkRecentTxsOnStart()

			txMgr.AssertExpectations(t)
			l1Client.AssertExpectations(t)
			ep.rollupClient.AssertExpectations(t)
		})
	}
}
