package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Test helper types
type expectedExecutingCall struct {
	chainID   string
	blockNum  uint64
	blockHash string
	status    string
	value     float64
}

type expectedInitiatingCall struct {
	chainID  string
	blockNum uint64
	status   string
	value    float64
}

type expectedTerminalCall struct {
	executingChainID  string
	initiatingChainID string
	value             float64
}

// mockUpdater implements the Updater interface with configurable function implementations
type mockUpdater struct {
	collectForMetricsFn func(map[JobID]*Job) map[JobID]*Job
	enqueueFn           func(*Job)
}

func (m *mockUpdater) CollectForMetrics(jobMap map[JobID]*Job) map[JobID]*Job {
	if m.collectForMetricsFn != nil {
		return m.collectForMetricsFn(jobMap)
	}
	return jobMap
}

func (m *mockUpdater) Enqueue(job *Job) {
	if m.enqueueFn != nil {
		m.enqueueFn(job)
	}
}

func (m *mockUpdater) Start(ctx context.Context) error {
	return nil
}

func (m *mockUpdater) Stop() error {
	return nil
}

// mockMetrics implements the metrics.Metricer interface with configurable function implementations
// by default, it records the calls to the metrics functions
type mockMetrics struct {
	recordInfoFn                   func(version string)
	recordUpFn                     func()
	recordInitiatingMessageStatsFn func(chainID string, blockHeight uint64, status string, value float64)
	recordExecutingMessageStatsFn  func(chainID string, blockHeight uint64, blockHash string, status string, value float64)
	recordTerminalStatusChangeFn   func(executingChainID string, initiatingChainID string, value float64)

	// Recording slices for test verification
	actualExecutingCalls  []expectedExecutingCall
	actualInitiatingCalls []expectedInitiatingCall
	actualTerminalCalls   []expectedTerminalCall
}

func (m *mockMetrics) RecordInfo(version string) {
	if m.recordInfoFn != nil {
		m.recordInfoFn(version)
	}
}

func (m *mockMetrics) RecordUp() {
	if m.recordUpFn != nil {
		m.recordUpFn()
	}
}

func (m *mockMetrics) RecordInitiatingMessageStats(chainID string, blockHeight uint64, status string, value float64) {
	if m.recordInitiatingMessageStatsFn != nil {
		m.recordInitiatingMessageStatsFn(chainID, blockHeight, status, value)
	} else {
		m.actualInitiatingCalls = append(m.actualInitiatingCalls, expectedInitiatingCall{
			chainID:  chainID,
			blockNum: blockHeight,
			status:   status,
			value:    value,
		})
	}
}

func (m *mockMetrics) RecordExecutingMessageStats(chainID string, blockHeight uint64, blockHash string, status string, value float64) {
	if m.recordExecutingMessageStatsFn != nil {
		m.recordExecutingMessageStatsFn(chainID, blockHeight, blockHash, status, value)
	} else {
		m.actualExecutingCalls = append(m.actualExecutingCalls, expectedExecutingCall{
			chainID:   chainID,
			blockNum:  blockHeight,
			blockHash: blockHash,
			status:    status,
			value:     value,
		})
	}
}

func (m *mockMetrics) RecordTerminalStatusChange(executingChainID string, initiatingChainID string, value float64) {
	if m.recordTerminalStatusChangeFn != nil {
		m.recordTerminalStatusChangeFn(executingChainID, initiatingChainID, value)
	} else {
		m.actualTerminalCalls = append(m.actualTerminalCalls, expectedTerminalCall{
			executingChainID:  executingChainID,
			initiatingChainID: initiatingChainID,
			value:             value,
		})
	}
}

func jobForTest(
	executingChainID uint64,
	executingBlockNum uint64,
	executingBlockHash string,
	initiatingChainID uint64,
	initiatingBlockNum uint64,
	status ...jobStatus,
) *Job {
	return &Job{
		id:             JobID(uuid.New().String()),
		executingChain: eth.ChainIDFromUInt64(executingChainID),
		executingBlock: eth.BlockID{Number: executingBlockNum, Hash: common.HexToHash(executingBlockHash)},
		initiating:     &supervisortypes.Identifier{ChainID: eth.ChainIDFromUInt64(initiatingChainID), BlockNumber: initiatingBlockNum},
		status:         status,
	}
}

// TestNewMetricCollector tests the creation of a new MetricCollector
func TestNewMetricCollector(t *testing.T) {
	// Setup test dependencies
	logger := log.New()
	mockMetrics := &mockMetrics{}
	updaters := map[eth.ChainID]Updater{
		eth.ChainIDFromUInt64(1): &mockUpdater{},
		eth.ChainIDFromUInt64(2): &mockUpdater{},
	}

	// Create new MetricCollector
	collector := NewMetricCollector(logger, mockMetrics, updaters)

	// Verify the collector was created correctly
	require.NotNil(t, collector)
	require.Equal(t, logger, collector.log)
	require.Equal(t, mockMetrics, collector.m)
	require.Equal(t, updaters, collector.updaters)
	require.NotNil(t, collector.closed)
	require.False(t, collector.Stopped(), "New collector should not be stopped")
}

// TestMetricCollectorStartStop tests the Start and Stop functionality
func TestMetricCollectorStartStop(t *testing.T) {
	// Setup test dependencies
	logger := log.New()
	mockMetrics := &mockMetrics{}
	updaters := map[eth.ChainID]Updater{
		eth.ChainIDFromUInt64(1): &mockUpdater{},
	}

	// Create new MetricCollector
	collector := NewMetricCollector(logger, mockMetrics, updaters)

	// Start the collector
	err := collector.Start()
	require.NoError(t, err, "Start should not return an error")
	require.False(t, collector.Stopped(), "Collector should not be stopped after Start()")

	// Wait a short time to ensure the goroutine is running
	time.Sleep(100 * time.Millisecond)

	// Stop the collector
	err = collector.Stop()
	require.NoError(t, err, "Stop should not return an error")
	require.True(t, collector.Stopped(), "Collector should be stopped after Stop()")
}

// TestCollectMetrics tests the metric collection functionality
func TestCollectMetrics(t *testing.T) {
	type testCase struct {
		name string
		// Input jobs from each updater
		updater1Jobs map[JobID]*Job
		updater2Jobs map[JobID]*Job
		updater3Jobs map[JobID]*Job
		// Expected metric calls
		expectedExecutingCalls  []expectedExecutingCall
		expectedInitiatingCalls []expectedInitiatingCall
		expectedTerminalCalls   []expectedTerminalCall
	}

	tests := []testCase{
		{
			name:                    "empty job maps",
			updater1Jobs:            map[JobID]*Job{},
			updater2Jobs:            map[JobID]*Job{},
			updater3Jobs:            map[JobID]*Job{},
			expectedExecutingCalls:  []expectedExecutingCall{},
			expectedInitiatingCalls: []expectedInitiatingCall{},
			expectedTerminalCalls:   []expectedTerminalCall{},
		},
		{
			name: "single job with future status",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusFuture),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedExecutingCalls: []expectedExecutingCall{
				{
					chainID:   "1",
					blockNum:  100,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000123",
					status:    "future",
					value:     1,
				},
			},
			expectedInitiatingCalls: []expectedInitiatingCall{
				{
					chainID:  "2",
					blockNum: 200,
					status:   "future",
					value:    1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
		},
		{
			name: "job with terminal status change",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid, jobStatusInvalid),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedExecutingCalls: []expectedExecutingCall{
				{
					chainID:   "1",
					blockNum:  100,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000123",
					status:    "invalid",
					value:     1,
				},
			},
			expectedInitiatingCalls: []expectedInitiatingCall{
				{
					chainID:  "2",
					blockNum: 200,
					status:   "invalid",
					value:    1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					value:             1,
				},
			},
		},
		{
			name: "multiple jobs with same status",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusFuture),
				"job2": jobForTest(1, 101, "0x456", 2, 201, jobStatusFuture),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedExecutingCalls: []expectedExecutingCall{
				{
					chainID:   "1",
					blockNum:  100,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000123",
					status:    "future",
					value:     1,
				},
				{
					chainID:   "1",
					blockNum:  101,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000456",
					status:    "future",
					value:     1,
				},
			},
			expectedInitiatingCalls: []expectedInitiatingCall{
				{
					chainID:  "2",
					blockNum: 200,
					status:   "future",
					value:    1,
				},
				{
					chainID:  "2",
					blockNum: 201,
					status:   "future",
					value:    1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
		},
		{
			name: "jobs across different chains",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusFuture),
			},
			updater2Jobs: map[JobID]*Job{
				"job2": jobForTest(2, 300, "0x456", 3, 400, jobStatusValid),
			},
			updater3Jobs: map[JobID]*Job{
				"job3": jobForTest(3, 500, "0x789", 1, 600, jobStatusInvalid),
			},
			expectedExecutingCalls: []expectedExecutingCall{
				{
					chainID:   "1",
					blockNum:  100,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000123",
					status:    "future",
					value:     1,
				},
				{
					chainID:   "2",
					blockNum:  300,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000456",
					status:    "valid",
					value:     1,
				},
				{
					chainID:   "3",
					blockNum:  500,
					blockHash: "0x0000000000000000000000000000000000000000000000000000000000000789",
					status:    "invalid",
					value:     1,
				},
			},
			expectedInitiatingCalls: []expectedInitiatingCall{
				{
					chainID:  "2",
					blockNum: 200,
					status:   "future",
					value:    1,
				},
				{
					chainID:  "3",
					blockNum: 400,
					status:   "valid",
					value:    1,
				},
				{
					chainID:  "1",
					blockNum: 600,
					status:   "invalid",
					value:    1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test dependencies
			logger := log.New()
			mockMetrics := &mockMetrics{}

			// Create mock updaters with predefined responses
			updater1 := &mockUpdater{
				collectForMetricsFn: func(jobs map[JobID]*Job) map[JobID]*Job {
					for _, job := range tt.updater1Jobs {
						jobs[job.ID()] = job
					}
					return jobs
				},
			}
			updater2 := &mockUpdater{
				collectForMetricsFn: func(jobs map[JobID]*Job) map[JobID]*Job {
					for _, job := range tt.updater2Jobs {
						jobs[job.ID()] = job
					}
					return jobs
				},
			}
			updater3 := &mockUpdater{
				collectForMetricsFn: func(jobs map[JobID]*Job) map[JobID]*Job {
					for _, job := range tt.updater3Jobs {
						jobs[job.ID()] = job
					}
					return jobs
				},
			}

			// Create collector with mock updaters
			collector := NewMetricCollector(logger, mockMetrics, map[eth.ChainID]Updater{
				eth.ChainIDFromUInt64(1): updater1,
				eth.ChainIDFromUInt64(2): updater2,
				eth.ChainIDFromUInt64(3): updater3,
			})

			// Run metric collection
			collector.CollectMetrics()

			// Verify metric calls
			require.ElementsMatch(t, tt.expectedExecutingCalls, mockMetrics.actualExecutingCalls, "executing message stats calls should match")
			require.ElementsMatch(t, tt.expectedInitiatingCalls, mockMetrics.actualInitiatingCalls, "initiating message stats calls should match")
			require.ElementsMatch(t, tt.expectedTerminalCalls, mockMetrics.actualTerminalCalls, "terminal status change calls should match")
		})
	}
}
