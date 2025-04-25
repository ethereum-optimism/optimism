package batcher

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
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
		Log:              testlog.Logger(t, log.LevelDebug),
		Metr:             metrics.NoopMetrics,
		RollupConfig:     cfg,
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

func TestBatchSubmitter_ThrottlingEndpoints(t *testing.T) {
	// Track request counts for verification
	var server1Calls, server2Calls int64

	// Create mock HTTP servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify this is a JSON-RPC call to miner_setMaxDASize with expected params
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					// Successfully handled the expected RPC call
					server1Calls++
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
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Same handler as server1
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					server2Calls++
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
	}))
	defer server2.Close()

	// Setup test context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create test BatchSubmitter using the setup function
	bs, _ := setup(t)
	bs.shutdownCtx = ctx
	bs.Config = BatcherConfig{
		NetworkTimeout:      time.Second,
		ThrottleThreshold:   10000,
		ThrottleTxSize:      5000,
		ThrottleBlockSize:   20000,
		ThrottlingEndpoints: []string{server1.URL, server2.URL},
	}

	// Test the throttling loop
	pendingBytesUpdated := make(chan int64, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	// Start throttling loop in a goroutine
	go bs.throttlingLoop(&wg, pendingBytesUpdated)

	// Send test data to trigger throttling
	pendingBytesUpdated <- 20000 // Over threshold, should trigger throttling

	// Allow time for processing
	time.Sleep(time.Millisecond * 200)

	// Check that both endpoints were called
	require.Greater(t, server1Calls, int64(0), "Server 1 should have been called")
	require.Greater(t, server2Calls, int64(0), "Server 2 should have been called")

	// Clean up previous test
	close(pendingBytesUpdated)
	cancel()
	wg.Wait()

	// Test fallback to L2 client when no throttling endpoints provided
	var defaultCalls int64

	// Create a mock server for the default endpoint
	defaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					defaultCalls++
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
	}))
	defer defaultServer.Close()

	// Create new context for the second test
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	// Setup for default endpoint test
	bs2, ep2 := setup(t)
	bs2.shutdownCtx = ctx2
	bs2.Config = BatcherConfig{
		NetworkTimeout:      time.Second,
		ThrottleThreshold:   10000,
		ThrottleTxSize:      5000,
		ThrottleBlockSize:   20000,
		ThrottlingEndpoints: []string{defaultServer.URL}, // this is populated during init if no throttling endpoints are provided
	}

	// Create RPC client for our test server
	rpcClient, err := rpc.Dial(defaultServer.URL)
	require.NoError(t, err)

	// Setup the mock L2 client to return our rpc client
	mockL2Client := new(testutils.MockL2Client)
	mockL2Client.On("Client").Return(rpcClient)

	// Configure endpoint provider to return our mock client
	ep2.ethClient = mockL2Client

	pendingBytesUpdated2 := make(chan int64, 1)
	var wg2 sync.WaitGroup
	wg2.Add(1)

	// Start throttling loop with default endpoint
	go bs2.throttlingLoop(&wg2, pendingBytesUpdated2)

	// Send test data
	pendingBytesUpdated2 <- 20000 // Over threshold

	// Allow time for processing
	time.Sleep(time.Millisecond * 200)

	// Check that default endpoint was called
	require.Greater(t, defaultCalls, int64(0), "Default endpoint should have been called")

	// Clean up
	close(pendingBytesUpdated2)
	cancel2()
	wg2.Wait()

	// Test independent endpoint behavior with partial endpoint failure
	var (
		successCalls int64
		failureCalls int64
	)

	// Server that always fails
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					failureCalls++
					http.Error(w, "Simulated failure", http.StatusInternalServerError)
					return
				}
			}
		}
		http.Error(w, "Unexpected request", http.StatusBadRequest)
	}))
	defer failingServer.Close()

	// Server that always succeeds
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var req struct {
				JSONRPC string        `json:"jsonrpc"`
				Method  string        `json:"method"`
				Params  []interface{} `json:"params"`
				ID      interface{}   `json:"id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				if req.Method == "miner_setMaxDASize" && len(req.Params) == 2 {
					successCalls++
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
	}))
	defer successServer.Close()

	// Create new context for the third test
	ctx3, cancel3 := context.WithCancel(context.Background())
	defer cancel3()

	// Setup for partial failure test
	bs3, _ := setup(t)
	bs3.shutdownCtx = ctx3
	bs3.Config = BatcherConfig{
		NetworkTimeout:      time.Second,
		ThrottleThreshold:   10000,
		ThrottleTxSize:      5000,
		ThrottleBlockSize:   20000,
		ThrottlingEndpoints: []string{failingServer.URL, successServer.URL},
	}

	pendingBytesUpdated3 := make(chan int64, 1)
	var wg3 sync.WaitGroup
	wg3.Add(1)

	// Start throttling loop with partial failure
	go bs3.throttlingLoop(&wg3, pendingBytesUpdated3)

	// Send test data
	pendingBytesUpdated3 <- 20000 // Over threshold

	// Allow time for processing
	time.Sleep(time.Millisecond * 200)

	// With independent endpoint throttling:
	// 1. The failing endpoint should have been called at least once
	require.Greater(t, failureCalls, int64(0), "Failing endpoint should have been called")

	// 2. The success endpoint should also have been called and should succeed independently
	require.Greater(t, successCalls, int64(0), "Success endpoint should have been called")

	// Clean up
	close(pendingBytesUpdated3)
	cancel3()
	wg3.Wait()

	// Test direct singleEndpointThrottler function
	t.Run("TestSingleEndpointThrottler", func(t *testing.T) {
		// Reset counters
		successCalls = 0
		failureCalls = 0

		// Create a new BatchSubmitter with specific throttling config and a valid shutdown context
		testBS, _ := setup(t)
		testCtx, testCancel := context.WithCancel(context.Background())
		defer testCancel()

		testBS.shutdownCtx = testCtx
		testBS.Config = BatcherConfig{
			NetworkTimeout:    time.Second,
			ThrottleThreshold: 10000,
			ThrottleTxSize:    5000,
			ThrottleBlockSize: 20000,
		}

		// Create channels for endpoint updates
		successUpdates := make(chan int64, 1)
		failureUpdates := make(chan int64, 1)

		var wg sync.WaitGroup

		// Start the throttlers
		wg.Add(2)
		go testBS.singleEndpointThrottler(&wg, successUpdates, successServer.URL)
		go testBS.singleEndpointThrottler(&wg, failureUpdates, failingServer.URL)

		// Send updates to trigger throttling
		successUpdates <- 20000 // Over threshold
		failureUpdates <- 20000 // Over threshold

		// Allow time for processing
		time.Sleep(time.Millisecond * 300)

		// Verify the endpoints were called
		require.Greater(t, successCalls, int64(0), "Success endpoint should have been called at least once")
		require.Greater(t, failureCalls, int64(0), "Failing endpoint should have been called at least once")

		// Clean up
		close(successUpdates)
		close(failureUpdates)
		wg.Wait()
	})
}
