package batcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher/throttler"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

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
		Log:          testlog.Logger(t, log.LevelDebug),
		Metr:         metrics.NoopMetrics,
		RollupConfig: cfg,
		Config: BatcherConfig{
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		},
		ChannelConfig:    defaultTestChannelConfig(),
		EndpointProvider: ep,
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
					LocalSafeL2: eth.L2BlockRef{
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
	fmt.Println(err)
	require.Error(t, err)
}

type MockTxQueue struct {
	m sync.Map
}

func (q *MockTxQueue) Send(ref txRef, candidate txmgr.TxCandidate, receiptCh chan txmgr.TxReceipt[txRef]) {
	q.m.Store(ref.id.String(), candidate)
}

func (q *MockTxQueue) Load(id string) txmgr.TxCandidate {
	c, _ := q.m.Load(id)
	return c.(txmgr.TxCandidate)
}

func TestBatchSubmitter_sendTx_FloorDataGas(t *testing.T) {
	bs, _ := setup(t)

	q := new(MockTxQueue)

	txData := txData{
		frames: []frameData{
			{
				data: []byte{0x01, 0x02, 0x03}, // 3 nonzero bytes = 12 tokens https://github.com/ethereum/EIPs/blob/master/EIPS/eip-7623.md
			},
		},
	}
	candidate := txmgr.TxCandidate{
		To:     &bs.RollupConfig.BatchInboxAddress,
		TxData: txData.CallData(),
	}

	bs.sendTx(txData,
		false,
		&candidate,
		q,
		make(chan txmgr.TxReceipt[txRef]))

	candidateOut := q.Load(txData.ID().String())

	expectedFloorDataGas := uint64(21_000 + 12*10)
	require.GreaterOrEqual(t, candidateOut.GasLimit, expectedFloorDataGas)
}

// ======= ALTDA TESTS =======

// fakeL1Client is just a dummy struct. All fault injection is done via the fakeTxMgr (which doesn't interact with this fakeL1Client).
type fakeL1Client struct {
}

func (f *fakeL1Client) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number == nil {
		number = big.NewInt(0)
	}
	return &types.Header{
		Number:     number,
		ParentHash: common.Hash{},
		Time:       0,
	}, nil
}
func (f *fakeL1Client) NonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return 0, nil
}

func altDASetup(t *testing.T, log log.Logger) (*BatchSubmitter, *mockL2EndpointProvider, *altda.MockDAClient, *testutils.FakeTxMgr) {
	ep := newEndpointProvider()

	rollupCfg := &rollup.Config{
		Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L1: eth.BlockID{Number: genesisL1Origin}},
		L2ChainID: big.NewInt(1234),
	}
	batcherCfg := BatcherConfig{
		PollInterval: 10 * time.Millisecond,
		UseAltDA:     true,
	}

	fakeTxMgr := testutils.NewFakeTxMgr(log.With("subsystem", "fake-txmgr"), common.Address{0})
	l1Client := &fakeL1Client{}

	channelCfg := ChannelConfig{
		// SeqWindowSize:      15,
		// SubSafetyMargin:    4,
		ChannelTimeout:  10,
		MaxFrameSize:    300,
		TargetNumFrames: 1,
		BatchType:       derive.SingularBatchType,
		CompressorConfig: compressor.Config{
			Kind: compressor.NoneKind,
		},
		DaType: DaTypeAltDA,
	}
	mockAltDAClient := altda.NewCountingGenericCommitmentMockDAClient(log.With("subsystem", "da-client"))
	return NewBatchSubmitter(DriverSetup{
		Log:              log,
		Metr:             metrics.NoopMetrics,
		RollupConfig:     rollupCfg,
		ChannelConfig:    channelCfg,
		Config:           batcherCfg,
		EndpointProvider: ep,
		Txmgr:            fakeTxMgr,
		L1Client:         l1Client,
		AltDA:            mockAltDAClient,
	}), ep, mockAltDAClient, fakeTxMgr
}

func fakeSyncStatus(unsafeL2BlockNum uint64, L1BlockRef eth.L1BlockRef) *eth.SyncStatus {
	return &eth.SyncStatus{
		UnsafeL2: eth.L2BlockRef{
			Number: unsafeL2BlockNum,
			Hash:   common.MaxHash,
			L1Origin: eth.BlockID{
				Number: 0,
			},
		},
		LocalSafeL2: eth.L2BlockRef{
			Number: 0,
			Hash:   common.MaxHash,
			L1Origin: eth.BlockID{
				Number: 0,
			},
		},
		HeadL1: L1BlockRef,
	}
}

// There are 4 failure cases (unhappy paths) that the op-batcher has to deal with.
// They are outlined in https://github.com/ethereum-optimism/optimism/tree/develop/op-batcher#happy-path
// This test suite covers these 4 cases in the context of AltDA.
func TestBatchSubmitter_AltDA_FailureCase1_L2Reorg(t *testing.T) {
	t.Parallel()
	log := testlog.Logger(t, log.LevelDebug)
	bs, ep, mockAltDAClient, fakeTxMgr := altDASetup(t, log)

	L1Block0 := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil, types.DefaultBlockConfig)
	L1Block0Ref := eth.L1BlockRef{
		Hash:   L1Block0.Hash(),
		Number: L1Block0.NumberU64(),
	}
	// We return incremental syncStatuses to force the op-batcher to entirely process each L2 block one by one.
	// To test multi channel behavior, we could return a sync status that is multiple blocks ahead of the current L2 block.
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(1, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(2, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(3, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(1, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(2, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Return(fakeSyncStatus(3, L1Block0Ref), nil)

	L2Block0 := newMiniL2BlockWithNumberParent(1, big.NewInt(0), common.HexToHash("0x0"))
	L2Block1 := newMiniL2BlockWithNumberParent(1, big.NewInt(1), L2Block0.Hash())
	L2Block2 := newMiniL2BlockWithNumberParent(1, big.NewInt(2), L2Block1.Hash())
	L2Block2Prime := newMiniL2BlockWithNumberParentAndL1Information(1, big.NewInt(2), L2Block1.Hash(), 101, 0)
	L2Block3Prime := newMiniL2BlockWithNumberParent(1, big.NewInt(3), L2Block2Prime.Hash())

	// L2block0 is the genesis block which is considered safe, so never loaded into the state.
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(1)).Twice().Return(L2Block1, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(2)).Once().Return(L2Block2, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(2)).Once().Return(L2Block2Prime, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(3)).Twice().Return(L2Block3Prime, nil)

	err := bs.StartBatchSubmitting()
	require.NoError(t, err)
	time.Sleep(1 * time.Second) // 1 second is enough to process all blocks at 10ms poll interval
	err = bs.StopBatchSubmitting(context.Background())
	require.NoError(t, err)

	// After the reorg, block 1 needs to be reprocessed, hence why we see 5 store calls: 1, 2, 1, 2', 3'
	require.Equal(t, 5, mockAltDAClient.StoreCount)
	require.Equal(t, uint64(5), fakeTxMgr.Nonce)

}

func TestBatchSubmitter_AltDA_FailureCase2_FailedL1Tx(t *testing.T) {
	t.Parallel()
	log := testlog.Logger(t, log.LevelDebug)
	bs, ep, mockAltDAClient, fakeTxMgr := altDASetup(t, log)

	L1Block0 := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil, types.DefaultBlockConfig)
	L1Block0Ref := eth.L1BlockRef{
		Hash:   L1Block0.Hash(),
		Number: L1Block0.NumberU64(),
	}
	// We return incremental syncStatuses to force the op-batcher to entirely process each L2 block one by one.
	// To test multi channel behavior, we could return a sync status that is multiple blocks ahead of the current L2 block.
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(1, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(2, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Times(10).Return(fakeSyncStatus(3, L1Block0Ref), nil)
	ep.rollupClient.Mock.On("SyncStatus").Return(fakeSyncStatus(4, L1Block0Ref), nil)

	L2Block0 := newMiniL2BlockWithNumberParent(1, big.NewInt(0), common.HexToHash("0x0"))
	L2Block1 := newMiniL2BlockWithNumberParent(1, big.NewInt(1), L2Block0.Hash())
	L2Block2 := newMiniL2BlockWithNumberParent(1, big.NewInt(2), L2Block1.Hash())
	L2Block3 := newMiniL2BlockWithNumberParent(1, big.NewInt(3), L2Block2.Hash())
	L2Block4 := newMiniL2BlockWithNumberParent(1, big.NewInt(4), L2Block3.Hash())

	// L2block0 is the genesis block which is considered safe, so never loaded into the state.
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(1)).Once().Return(L2Block1, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(2)).Once().Return(L2Block2, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(3)).Once().Return(L2Block3, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(4)).Once().Return(L2Block4, nil)

	fakeTxMgr.ErrorEveryNthSend(2)
	err := bs.StartBatchSubmitting()
	require.NoError(t, err)
	time.Sleep(1 * time.Second) // 1 second is enough to process all blocks at 10ms poll interval
	err = bs.StopBatchSubmitting(context.Background())
	require.NoError(t, err)

	require.Equal(t, 4, mockAltDAClient.StoreCount)
	// TODO: we should prob also check that the commitments are in order?
	require.Equal(t, uint64(4), fakeTxMgr.Nonce)
}

func TestBatchSubmitter_AltDA_FailureCase3_ChannelTimeout(t *testing.T) {
	// This function is not implemented because the batcher channel logic makes it very difficult to inject faults.
	// A version of this test was implemented here: https://github.com/Layr-Labs/optimism/blob/4b79c981a13bf096ae2984634d976956fbbfddff/op-batcher/batcher/driver_test.go#L300
	// However we opted to not merge it into the main branch because it has an external dependency on the https://github.com/pingcap/failpoint package,
	// and requires a lot of custom test setup and failpoint code injection into the batcher's codebase.
	// See https://github.com/ethereum-optimism/optimism/commit/4b79c981a13bf096ae2984634d976956fbbfddff for the full implementation.
}

func TestBatchSubmitter_AltDA_FailureCase4_FailedBlobSubmission(t *testing.T) {
	t.Parallel()
	log := testlog.Logger(t, log.LevelDebug)
	bs, ep, mockAltDAClient, fakeTxMgr := altDASetup(t, log)

	L1Block0 := types.NewBlock(&types.Header{
		Number: big.NewInt(0),
	}, nil, nil, nil, types.DefaultBlockConfig)
	L1Block0Ref := eth.L1BlockRef{
		Hash:   L1Block0.Hash(),
		Number: L1Block0.NumberU64(),
	}
	ep.rollupClient.Mock.On("SyncStatus").Return(fakeSyncStatus(4, L1Block0Ref), nil)

	L2Block0 := newMiniL2BlockWithNumberParent(1, big.NewInt(0), common.HexToHash("0x0"))
	L2Block1 := newMiniL2BlockWithNumberParent(1, big.NewInt(1), L2Block0.Hash())
	L2Block2 := newMiniL2BlockWithNumberParent(1, big.NewInt(2), L2Block1.Hash())
	L2Block3 := newMiniL2BlockWithNumberParent(1, big.NewInt(3), L2Block2.Hash())
	L2Block4 := newMiniL2BlockWithNumberParent(1, big.NewInt(4), L2Block3.Hash())

	// L2block0 is the genesis block which is considered safe, so never loaded into the state.
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(1)).Once().Return(L2Block1, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(2)).Once().Return(L2Block2, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(3)).Once().Return(L2Block3, nil)
	ep.ethClient.Mock.On("BlockByNumber", big.NewInt(4)).Once().Return(L2Block4, nil)

	mockAltDAClient.DropEveryNthPut(2)

	err := bs.StartBatchSubmitting()
	require.NoError(t, err)
	time.Sleep(1 * time.Second) // 1 second is enough to process all blocks at 10ms poll interval
	err = bs.StopBatchSubmitting(context.Background())
	require.NoError(t, err)

	require.Equal(t, 4, mockAltDAClient.StoreCount)
	require.Equal(t, uint64(4), fakeTxMgr.Nonce)
}

// createHTTPHandler creates a mock HTTP handler for testing, it accepts a callback which
// is invoked when the expected request is received.
func createHTTPHandler(t *testing.T, cb func(), alwaysFails bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {

				if alwaysFails {
					http.Error(w, "Simulated failure", http.StatusInternalServerError)
					cb()
					return
				}
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					cb()
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":true}`))
					if err != nil {
						t.Logf("Error writing response: %v", err)
					}
					return
				}
			}
		}
		http.Error(w, "Unexpected request", http.StatusBadRequest)
	}
}

func TestBatchSubmitter_ThrottlingEndpoints(t *testing.T) {

	testThrottlingEndpoints := func(numHealthyServers, numUnhealthyServers int) func(t *testing.T) {

		return func(t *testing.T) {
			healthyCalls := make([]int, numHealthyServers)
			unHealthyCalls := make([]int, numUnhealthyServers)

			healthyServers := make([]*httptest.Server, numHealthyServers)
			unhealthyServers := make([]*httptest.Server, numUnhealthyServers)

			urls := make([]string, 0, numHealthyServers+numUnhealthyServers)

			for i := range healthyCalls {
				healthyServers[i] = httptest.NewServer(createHTTPHandler(t, func() { healthyCalls[i]++ }, false))
				urls = append(urls, healthyServers[i].URL)
				defer healthyServers[i].Close()
			}
			for i := range unHealthyCalls {
				unhealthyServers[i] = httptest.NewServer(createHTTPHandler(t, func() { unHealthyCalls[i]++ }, true))
				urls = append(urls, unhealthyServers[i].URL)
				defer unhealthyServers[i].Close()
			}

			// Setup test context
			ctx, cancel := context.WithCancel(context.Background())

			// Add in an endpoint with no server at all, representing an "always down" endpoint
			urls = append(urls, "http://invalid/")

			t.Log("Throttling endpoints:", urls)

			// Create test BatchSubmitter using the setup function
			bs, _ := setup(t)
			bs.shutdownCtx = ctx
			bs.Config.NetworkTimeout = time.Second
			bs.Config.ThrottleParams.Endpoints = urls
			bs.throttleController = throttler.NewThrottleController(
				throttler.NewStepStrategy(10000),
				throttler.ThrottleConfig{
					TxSizeLowerLimit:    5000,
					TxSizeUpperLimit:    10000,
					BlockSizeLowerLimit: 20000,
					BlockSizeUpperLimit: 30000,
				})

			// Test the throttling loop
			pendingBytesUpdated := make(chan int64, 1)
			wg1 := sync.WaitGroup{}
			wg1.Add(1)

			// Start throttling loop in a goroutine
			go bs.throttlingLoop(&wg1, pendingBytesUpdated)

			// Simulate block loading by sending periodically on pendingBytesUpdated
			wg2 := sync.WaitGroup{}
			blockLoadingCtx, cancelBlockLoading := context.WithCancel(context.Background())
			go func() {
				defer wg2.Done()
				// Simulate block loading
				for range time.NewTicker(100 * time.Millisecond).C {
					select {
					case <-blockLoadingCtx.Done():
						return
					default:
						// Simulate block loading
						pendingBytesUpdated <- 20000 // the value doesn't actually matter for this test
					}
				}

			}()
			wg2.Add(1)

			t.Cleanup(func() {
				cancelBlockLoading()
				wg2.Wait()
				close(pendingBytesUpdated)
				wg1.Wait()
				cancel()
			})

			require.Eventually(t,
				func() bool {
					// Check that all endpoints were called
					for i := range healthyCalls {
						if healthyCalls[i] == 0 {
							return false
						}
					}
					for i := range unHealthyCalls {
						if unHealthyCalls[i] == 0 {
							return false
						}
					}
					return true
				}, time.Second*10, time.Millisecond*10, "All endpoints should have been called within 10s")

			startTestServerAtAddr := func(addr string, handler http.HandlerFunc) *httptest.Server {
				ln, err := net.Listen("tcp", addr)
				require.NoError(t, err, "Failed to create new listener for test server")

				s := &httptest.Server{
					Listener: ln,
					Config:   &http.Server{Handler: handler},
				}
				s.Start()
				return s
			}

			// Take one of the healthy servers down, wait 2s and restart. Check it is called again.
			if len(healthyServers) > 0 {
				restartedServerCalled := false

				addr := healthyServers[0].Listener.Addr().String()
				healthyServers[0].Close()
				time.Sleep(time.Second * 2)
				startTestServerAtAddr(addr, createHTTPHandler(t, func() { restartedServerCalled = true }, false))
				defer healthyServers[0].Close()
				t.Log("restarted server at", addr)

				require.Eventually(t, func() bool {
					return restartedServerCalled
				}, time.Second*2, time.Millisecond*10, "Restarted server should have been called within 2s")
			}

		}
	}
	t.Run("two normal endpoints", testThrottlingEndpoints(2, 0))
	t.Run("two failing endpoints", testThrottlingEndpoints(0, 2))
	t.Run("one normal endpoint, one failing endpoint", testThrottlingEndpoints(1, 1))
}
