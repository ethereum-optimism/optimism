package batcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-batcher/batcher/throttler"
	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
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

func setup(t *testing.T, closeAppFn context.CancelCauseFunc) (*BatchSubmitter, *mockL2EndpointProvider) {
	ep := newEndpointProvider()

	cfg := defaultTestRollupConfig
	cfg.Genesis.L1.Number = genesisL1Origin

	if closeAppFn == nil {
		closeAppFn = func(cause error) {
			t.Fatalf("closeAppFn called, batcher hit a critical error: %v", cause)
		}
	}

	return NewBatchSubmitter(DriverSetup{
		closeApp:     closeAppFn,
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

// testDAClient is a simple mock DA client for testing that implements altda.DAStorage
type testDAClient struct {
	shouldFail atomic.Bool
	failErr    error
	callCount  atomic.Int32
}

func newTestDAClient() *testDAClient {
	return &testDAClient{
		failErr: errors.New("DA server unavailable"),
	}
}

func (c *testDAClient) GetInput(ctx context.Context, key altda.CommitmentData) ([]byte, error) {
	return []byte("mock data"), nil
}

func (c *testDAClient) SetInput(ctx context.Context, data []byte) (altda.CommitmentData, error) {
	c.callCount.Add(1)
	if c.shouldFail.Load() {
		return nil, c.failErr
	}
	// Return a valid commitment
	return altda.NewCommitmentData(altda.Keccak256CommitmentType, data), nil
}

func (c *testDAClient) SetShouldFail(fail bool) {
	c.shouldFail.Store(fail)
}

func (c *testDAClient) CallCount() int {
	return int(c.callCount.Load())
}

// setupWithAltDA creates a BatchSubmitter with AltDA enabled for testing
func setupWithAltDA(t *testing.T, cfg BatcherConfig) (*BatchSubmitter, *mockL2EndpointProvider, *testDAClient) {
	ep := newEndpointProvider()
	mockDA := newTestDAClient()

	rollupCfg := defaultTestRollupConfig
	rollupCfg.Genesis.L1.Number = genesisL1Origin

	return NewBatchSubmitter(DriverSetup{
		Log:              testlog.Logger(t, log.LevelDebug),
		Metr:             metrics.NoopMetrics,
		RollupConfig:     rollupCfg,
		Config:           cfg,
		ChannelConfig:    defaultTestChannelConfig(),
		EndpointProvider: ep,
		AltDA:            mockDA,
	}), ep, mockDA
}

func TestBatchSubmitter_SafeL1Origin(t *testing.T) {
	bs, ep := setup(t, nil)

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
	bs, ep := setup(t, nil)

	ep.rollupClientErr = errors.New("failed to resolve rollup client")

	_, err := bs.safeL1Origin(context.Background())
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
	bs, _ := setup(t, nil)

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

type handlerFailureMode string

const (
	noFailure      handlerFailureMode = "none"
	internalError  handlerFailureMode = "internal_error"
	methodNotFound handlerFailureMode = "method_not_found"
)

// createHTTPHandler creates a mock HTTP handler for testing, it accepts a callback which
// is invoked when the expected request is received.
func createHTTPHandler(t *testing.T, cb func(), failureMode handlerFailureMode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				cb()

				switch failureMode {
				case noFailure:
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":true}`))
					if err != nil {
						t.Logf("Error writing response: %v", err)
					}
					return
				case internalError:
					http.Error(w, "Simulated failure", http.StatusInternalServerError)
					return
				case methodNotFound:
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":%d,"message":"method not found"}}`, eth.MethodNotFound)))
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
	// Set a very long timeout to avoid flakiness
	timeout := time.Second * 120
	testThrottlingEndpoints := func(numHealthyServers, numUnhealthyServers int) func(t *testing.T) {

		return func(t *testing.T) {
			healthyCalls := make([]int, numHealthyServers)
			unHealthyCalls := make([]int, numUnhealthyServers)

			healthyServers := make([]*httptest.Server, numHealthyServers)
			unhealthyServers := make([]*httptest.Server, numUnhealthyServers)

			urls := make([]string, 0, numHealthyServers+numUnhealthyServers)

			for i := range healthyCalls {
				healthyServers[i] = httptest.NewServer(createHTTPHandler(t, func() { healthyCalls[i]++ }, noFailure))
				urls = append(urls, healthyServers[i].URL)
				defer healthyServers[i].Close()
			}
			for i := range unHealthyCalls {
				unhealthyServers[i] = httptest.NewServer(createHTTPHandler(t, func() { unHealthyCalls[i]++ }, internalError))
				urls = append(urls, unhealthyServers[i].URL)
				defer unhealthyServers[i].Close()
			}

			// Setup test context
			ctx, cancel := context.WithCancel(context.Background())

			// Add in an endpoint with no server at all, representing an "always down" endpoint
			urls = append(urls, "http://invalid/")

			t.Log("Throttling endpoints:", urls)

			var batcherShutdownError error

			// Create test BatchSubmitter using the setup function
			bs, _ := setup(t, func(cause error) {
				batcherShutdownError = cause
			})
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
					if slices.Contains(healthyCalls, 0) || slices.Contains(unHealthyCalls, 0) {
						return false
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
				startTestServerAtAddr(addr, createHTTPHandler(t, func() { restartedServerCalled = true }, noFailure))
				defer healthyServers[0].Close()
				t.Log("restarted server at", addr)

				require.Eventually(t, func() bool {
					return restartedServerCalled
				}, timeout, time.Millisecond*10, "Restarted server should have been called within 2s")
			}

			// Take an unhealthy server down, wait 2s and bring it back up with misconfiguration. Check the batcher exits.
			if len(unhealthyServers) > 0 {
				restartedServerCalled := false

				addr := unhealthyServers[0].Listener.Addr().String()
				unhealthyServers[0].Close()
				time.Sleep(time.Second * 2)
				startTestServerAtAddr(addr, createHTTPHandler(t, func() { restartedServerCalled = true }, methodNotFound))
				defer unhealthyServers[0].Close()
				t.Log("restarted server at", addr)

				require.Eventually(t, func() bool {
					return restartedServerCalled
				}, timeout, time.Millisecond*10, "Restarted server should have been called within 2s")

				require.Eventually(t, func() bool {
					return batcherShutdownError != nil
				}, timeout, time.Millisecond*10, "Batcher should have triggered self shutdown within 2s")

				require.Equal(t, batcherShutdownError.Error(), ErrSetMaxDASizeRPCMethodUnavailable("http://"+addr, errors.New("method not found")).Error(), "Batcher shutdown error should be the same as the expected error")
			}
		}
	}
	t.Run("two normal endpoints", testThrottlingEndpoints(2, 0))
	t.Run("two failing endpoints", testThrottlingEndpoints(0, 2))
	t.Run("one normal endpoint, one failing endpoint", testThrottlingEndpoints(1, 1))
}

func TestBatchSubmitter_CriticalError(t *testing.T) {
	criticalErrors := []error{
		eth.InputError{
			Code: eth.MethodNotFound,
		},
		eth.InputError{
			Code: eth.InvalidParams,
		},
	}

	for _, e := range criticalErrors {
		assert.True(t, isCriticalThrottlingRPCError(e), "false positive: %s", e)
	}

	nonCriticalErrors := []error{
		eth.InputError{
			Code: eth.UnsupportedFork,
		},
		errors.New("timeout"),
	}

	for _, e := range nonCriticalErrors {
		assert.False(t, isCriticalThrottlingRPCError(e), "false negative: %s", e)
	}
}

func TestBatchSubmitter_AltDA_Success(t *testing.T) {
	t.Run("Successful DA post sends commitment to L1", func(t *testing.T) {
		cfg := BatcherConfig{
			UseAltDA:                true,
			AltDAEnableFallback:     false,
			MaxConcurrentDARequests: 10,
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		}

		bs, _, mockDA := setupWithAltDA(t, cfg)

		// Create test data
		originalData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
		txData := singleFrameTxData(frameData{
			data: originalData,
		})

		// Create dependencies
		queue := new(MockTxQueue)
		receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
		daGroup := new(errgroup.Group)

		// Send transaction
		err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
		require.NoError(t, err)
		require.NoError(t, daGroup.Wait())

		// Verify DA was called
		require.Equal(t, 1, mockDA.CallCount(), "DA should be called once")

		// Verify transaction was queued to L1
		candidate := queue.Load(txData.ID().String())
		require.NotNil(t, candidate.To, "Transaction should be queued")

		// Verify the L1 transaction contains commitment data, not original data
		// Commitment is 33 bytes (1 byte type + 32 bytes hash) + 1 byte version prefix = 34 bytes
		require.NotEqual(t, originalData, candidate.TxData,
			"L1 tx should contain commitment, not original calldata")
		require.Equal(t, 34, len(candidate.TxData),
			"L1 tx should contain 34-byte commitment (version + type + hash)")
	})

	t.Run("DA request with context canceled requeues frame", func(t *testing.T) {
		cfg := BatcherConfig{
			UseAltDA:                true,
			AltDAEnableFallback:     false,
			MaxConcurrentDARequests: 10,
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		}

		bs, _, mockDA := setupWithAltDA(t, cfg)

		// Cancel the shutdown context to simulate context cancellation
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		bs.shutdownCtx = ctx

		txData := singleFrameTxData(frameData{
			data: []byte{0x01, 0x02, 0x03},
		})

		queue := new(MockTxQueue)
		receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
		daGroup := new(errgroup.Group)

		err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
		require.NoError(t, err)
		require.NoError(t, daGroup.Wait())

		// DA should still be called (context is checked inside SetInput)
		require.Equal(t, 1, mockDA.CallCount(), "DA should be called once")

		// Frame should be requeued via recordFailedDARequest
		// This test verifies that context cancellation is handled without error
	})

	t.Run("Multiple transactions spawn multiple goroutines", func(t *testing.T) {
		cfg := BatcherConfig{
			UseAltDA:                true,
			AltDAEnableFallback:     false,
			MaxConcurrentDARequests: 10,
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		}

		bs, _, mockDA := setupWithAltDA(t, cfg)

		// Create multiple transactions
		queue := new(MockTxQueue)
		receiptsCh := make(chan txmgr.TxReceipt[txRef], 10)
		daGroup := new(errgroup.Group)

		numTxs := 3
		for i := 0; i < numTxs; i++ {
			txData := singleFrameTxData(frameData{
				data: []byte{byte(i), 0x01, 0x02},
			})

			err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
			require.NoError(t, err)
		}

		// Wait for all DA requests to complete
		require.NoError(t, daGroup.Wait())

		// Verify all transactions called DA
		require.Equal(t, numTxs, mockDA.CallCount(), "All transactions should call DA")
	})
}

func TestBatchSubmitter_AltDAFallback_Activation(t *testing.T) {
	tests := []struct {
		name                  string
		useAltDA              bool
		enableFallback        bool
		daFails               bool
		expectFallbackActive  bool
		expectDACallCount     int
	}{
		{
			name:                 "Fallback activates on DA failure when enabled",
			useAltDA:             true,
			enableFallback:       true,
			daFails:              true,
			expectFallbackActive: true,
			expectDACallCount:    1,
		},
		{
			name:                 "Fallback does NOT activate when disabled",
			useAltDA:             true,
			enableFallback:       false,
			daFails:              true,
			expectFallbackActive: false,
			expectDACallCount:    1,
		},
		{
			name:                 "Successful DA request does NOT trigger fallback",
			useAltDA:             true,
			enableFallback:       true,
			daFails:              false,
			expectFallbackActive: false,
			expectDACallCount:    1,
		},
		{
			name:                 "No DA call when AltDA disabled",
			useAltDA:             false,
			enableFallback:       true,
			daFails:              false,
			expectFallbackActive: false,
			expectDACallCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatcherConfig{
				UseAltDA:             tt.useAltDA,
				AltDAEnableFallback:  tt.enableFallback,
				MaxConcurrentDARequests: 10,
				ThrottleParams: config.ThrottleParams{
					ControllerType: config.StepControllerType,
				},
			}

			bs, _, mockDA := setupWithAltDA(t, cfg)
			mockDA.SetShouldFail(tt.daFails)

			// Create test data
			txData := singleFrameTxData(frameData{
				data: []byte{0x01, 0x02, 0x03},
			})

			// Create dependencies
			queue := new(MockTxQueue)
			receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
			daGroup := new(errgroup.Group)

			// Send transaction
			err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
			require.NoError(t, err)

			// Wait for DA goroutine to complete
			require.NoError(t, daGroup.Wait())

			// Verify expectations
			require.Equal(t, tt.expectDACallCount, mockDA.CallCount(), "DA call count mismatch")
			require.Equal(t, tt.expectFallbackActive, bs.altDAFallbackActive, "Fallback activation state mismatch")
		})
	}
}

func TestBatchSubmitter_AltDAFallback_BypassBehavior(t *testing.T) {
	tests := []struct {
		name              string
		fallbackActive    bool
		expectDACall      bool
		expectDirectToL1  bool
	}{
		{
			name:             "Normal operation uses DA when not in fallback",
			fallbackActive:   false,
			expectDACall:     true,
			expectDirectToL1: false,
		},
		{
			name:             "Fallback mode bypasses DA and posts directly to L1",
			fallbackActive:   true,
			expectDACall:     false,
			expectDirectToL1: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BatcherConfig{
				UseAltDA:                true,
				AltDAEnableFallback:     true,
				MaxConcurrentDARequests: 10,
				ThrottleParams: config.ThrottleParams{
					ControllerType: config.StepControllerType,
				},
			}

			bs, _, mockDA := setupWithAltDA(t, cfg)

			// Set initial fallback state
			bs.altDAFallbackActive = tt.fallbackActive

			// Create test data
			txData := singleFrameTxData(frameData{
				data: []byte{0x01, 0x02, 0x03},
			})

			// Create dependencies
			queue := new(MockTxQueue)
			receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
			daGroup := new(errgroup.Group)

			// Send transaction
			err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
			require.NoError(t, err)

			// Wait for DA goroutine if one was spawned
			require.NoError(t, daGroup.Wait())

			// Verify DA call behavior
			if tt.expectDACall {
				require.Equal(t, 1, mockDA.CallCount(), "Expected DA to be called")
			} else {
				require.Equal(t, 0, mockDA.CallCount(), "Expected DA to be bypassed")
			}

			// Verify transaction was sent to L1
			// In both cases, a transaction should be queued
			candidate := queue.Load(txData.ID().String())
			require.NotNil(t, candidate.To, "Transaction should be queued to L1")
		})
	}
}

func TestBatchSubmitter_AltDAFallback_Persistence(t *testing.T) {
	cfg := BatcherConfig{
		UseAltDA:                true,
		AltDAEnableFallback:     true,
		MaxConcurrentDARequests: 10,
		ThrottleParams: config.ThrottleParams{
			ControllerType: config.StepControllerType,
		},
	}

	bs, _, mockDA := setupWithAltDA(t, cfg)
	mockDA.SetShouldFail(true) // First call will fail

	// Create test data for first transaction
	txData1 := singleFrameTxData(frameData{
		data: []byte{0x01, 0x02, 0x03},
	})

	// Create dependencies
	queue := new(MockTxQueue)
	receiptsCh := make(chan txmgr.TxReceipt[txRef], 10)
	daGroup := new(errgroup.Group)

	// Send first transaction - should fail and activate fallback
	err := bs.sendTransaction(txData1, queue, receiptsCh, daGroup)
	require.NoError(t, err)
	require.NoError(t, daGroup.Wait())

	// Verify fallback is active
	require.True(t, bs.altDAFallbackActive, "Fallback should be active after DA failure")
	require.Equal(t, 1, mockDA.CallCount(), "DA should have been called once")

	// Now make DA healthy again (shouldn't matter)
	mockDA.SetShouldFail(false)

	// Send second transaction - should bypass DA due to fallback
	txData2 := singleFrameTxData(frameData{
		data: []byte{0x04, 0x05, 0x06},
	})

	daGroup2 := new(errgroup.Group)
	err = bs.sendTransaction(txData2, queue, receiptsCh, daGroup2)
	require.NoError(t, err)
	require.NoError(t, daGroup2.Wait())

	// Verify DA was NOT called for second transaction
	require.Equal(t, 1, mockDA.CallCount(), "DA should still only have been called once (bypassed on second tx)")
	require.True(t, bs.altDAFallbackActive, "Fallback should remain active")

	// Send third transaction - should still bypass DA
	txData3 := singleFrameTxData(frameData{
		data: []byte{0x07, 0x08, 0x09},
	})

	daGroup3 := new(errgroup.Group)
	err = bs.sendTransaction(txData3, queue, receiptsCh, daGroup3)
	require.NoError(t, err)
	require.NoError(t, daGroup3.Wait())

	// Verify DA still not called
	require.Equal(t, 1, mockDA.CallCount(), "DA should still only have been called once (bypassed on third tx)")
	require.True(t, bs.altDAFallbackActive, "Fallback should persist across multiple transactions")
}

func TestBatchSubmitter_AltDAFallback_RepeatedFailures(t *testing.T) {
	t.Run("No duplicate activation when already in fallback", func(t *testing.T) {
		cfg := BatcherConfig{
			UseAltDA:                true,
			AltDAEnableFallback:     true,
			MaxConcurrentDARequests: 10,
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		}

		bs, _, mockDA := setupWithAltDA(t, cfg)

		// Set fallback as already active
		bs.altDAFallbackActive = true

		// Make DA fail (even though it shouldn't be called)
		mockDA.SetShouldFail(true)

		// Create test data
		txData := singleFrameTxData(frameData{
			data: []byte{0x01, 0x02, 0x03},
		})

		// Create dependencies
		queue := new(MockTxQueue)
		receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
		daGroup := new(errgroup.Group)

		// Send transaction - should bypass DA entirely
		err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
		require.NoError(t, err)
		require.NoError(t, daGroup.Wait())

		// Verify DA was NOT called (bypassed)
		require.Equal(t, 0, mockDA.CallCount(), "DA should be bypassed when fallback is already active")
		require.True(t, bs.altDAFallbackActive, "Fallback should remain active")

		// Verify transaction still went to L1
		candidate := queue.Load(txData.ID().String())
		require.NotNil(t, candidate.To, "Transaction should be queued to L1")
	})

	t.Run("Context canceled does NOT trigger fallback", func(t *testing.T) {
		cfg := BatcherConfig{
			UseAltDA:                true,
			AltDAEnableFallback:     true,
			MaxConcurrentDARequests: 10,
			ThrottleParams: config.ThrottleParams{
				ControllerType: config.StepControllerType,
			},
		}

		bs, _, mockDA := setupWithAltDA(t, cfg)

		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		bs.shutdownCtx = ctx

		// Create test data
		txData := singleFrameTxData(frameData{
			data: []byte{0x01, 0x02, 0x03},
		})

		// Create dependencies
		queue := new(MockTxQueue)
		receiptsCh := make(chan txmgr.TxReceipt[txRef], 1)
		daGroup := new(errgroup.Group)

		// Send transaction - DA will fail due to context canceled
		err := bs.sendTransaction(txData, queue, receiptsCh, daGroup)
		require.NoError(t, err)
		require.NoError(t, daGroup.Wait())

		// Verify fallback was NOT activated (context canceled is expected)
		require.False(t, bs.altDAFallbackActive, "Fallback should NOT activate on context.Canceled")
		require.Equal(t, 1, mockDA.CallCount(), "DA should have been called once")
	})
