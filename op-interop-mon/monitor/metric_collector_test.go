package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// Test helper types
type expectedMessageStatusCall struct {
	executingChainID  string
	initiatingChainID string
	status            string
	count             float64
}

type expectedTerminalCall struct {
	executingChainID  string
	initiatingChainID string
	count             float64
}

type expectedBlockRangeCall struct {
	chainID string
	min     uint64
	max     uint64
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
	recordInfoFn                 func(version string)
	recordUpFn                   func()
	recordMessageStatusFn        func(executingChainID string, initiatingChainID string, status string, count float64)
	recordTerminalStatusChangeFn func(executingChainID string, initiatingChainID string, count float64)
	recordExecutingBlockRangeFn  func(chainID string, min uint64, max uint64)
	recordInitiatingBlockRangeFn func(chainID string, min uint64, max uint64)

	// Recording slices for test verification
	actualMessageStatusCalls   []expectedMessageStatusCall
	actualTerminalCalls        []expectedTerminalCall
	actualExecutingRangeCalls  []expectedBlockRangeCall
	actualInitiatingRangeCalls []expectedBlockRangeCall
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

func (m *mockMetrics) RecordMessageStatus(executingChainID string, initiatingChainID string, status string, count float64) {
	if m.recordMessageStatusFn != nil {
		m.recordMessageStatusFn(executingChainID, initiatingChainID, status, count)
	} else {
		m.actualMessageStatusCalls = append(m.actualMessageStatusCalls, expectedMessageStatusCall{
			executingChainID:  executingChainID,
			initiatingChainID: initiatingChainID,
			status:            status,
			count:             count,
		})
	}
}

func (m *mockMetrics) RecordTerminalStatusChange(executingChainID string, initiatingChainID string, count float64) {
	if m.recordTerminalStatusChangeFn != nil {
		m.recordTerminalStatusChangeFn(executingChainID, initiatingChainID, count)
	} else {
		m.actualTerminalCalls = append(m.actualTerminalCalls, expectedTerminalCall{
			executingChainID:  executingChainID,
			initiatingChainID: initiatingChainID,
			count:             count,
		})
	}
}

func (m *mockMetrics) RecordExecutingBlockRange(chainID string, min uint64, max uint64) {
	if m.recordExecutingBlockRangeFn != nil {
		m.recordExecutingBlockRangeFn(chainID, min, max)
	} else {
		m.actualExecutingRangeCalls = append(m.actualExecutingRangeCalls, expectedBlockRangeCall{
			chainID: chainID,
			min:     min,
			max:     max,
		})
	}
}

func (m *mockMetrics) RecordInitiatingBlockRange(chainID string, min uint64, max uint64) {
	if m.recordInitiatingBlockRangeFn != nil {
		m.recordInitiatingBlockRangeFn(chainID, min, max)
	} else {
		m.actualInitiatingRangeCalls = append(m.actualInitiatingRangeCalls, expectedBlockRangeCall{
			chainID: chainID,
			min:     min,
			max:     max,
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
		expectedMessageStatusCalls   []expectedMessageStatusCall
		expectedTerminalCalls        []expectedTerminalCall
		expectedExecutingRangeCalls  []expectedBlockRangeCall
		expectedInitiatingRangeCalls []expectedBlockRangeCall
	}

	tests := []testCase{
		{
			name:                         "empty job maps",
			updater1Jobs:                 map[JobID]*Job{},
			updater2Jobs:                 map[JobID]*Job{},
			updater3Jobs:                 map[JobID]*Job{},
			expectedMessageStatusCalls:   []expectedMessageStatusCall{},
			expectedTerminalCalls:        []expectedTerminalCall{},
			expectedExecutingRangeCalls:  []expectedBlockRangeCall{},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{},
		},
		{
			name: "single job with future status",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusFuture),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					status:            "future",
					count:             1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     100,
					max:     100,
				},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "2",
					min:     200,
					max:     200,
				},
			},
		},
		{
			name: "job with terminal status change",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid, jobStatusInvalid),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					status:            "invalid",
					count:             1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					count:             1,
				},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     100,
					max:     100,
				},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "2",
					min:     200,
					max:     200,
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
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					status:            "future",
					count:             2,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     100,
					max:     101,
				},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "2",
					min:     200,
					max:     201,
				},
			},
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
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					status:            "future",
					count:             1,
				},
				{
					executingChainID:  "2",
					initiatingChainID: "3",
					status:            "valid",
					count:             1,
				},
				{
					executingChainID:  "3",
					initiatingChainID: "1",
					status:            "invalid",
					count:             1,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     100,
					max:     100,
				},
				{
					chainID: "2",
					min:     300,
					max:     300,
				},
				{
					chainID: "3",
					min:     500,
					max:     500,
				},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     600,
					max:     600,
				},
				{
					chainID: "2",
					min:     200,
					max:     200,
				},
				{
					chainID: "3",
					min:     400,
					max:     400,
				},
			},
		},
		{
			name: "complex block ranges",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusFuture),
				"job2": jobForTest(1, 50, "0x456", 2, 250, jobStatusFuture),
				"job3": jobForTest(1, 150, "0x789", 2, 150, jobStatusFuture),
			},
			updater2Jobs: map[JobID]*Job{
				"job4": jobForTest(2, 300, "0xabc", 1, 400, jobStatusValid),
				"job5": jobForTest(2, 250, "0xdef", 1, 450, jobStatusValid),
				"job6": jobForTest(2, 350, "0xghi", 1, 350, jobStatusValid),
			},
			updater3Jobs: map[JobID]*Job{
				"job7": jobForTest(3, 500, "0xjkl", 3, 600, jobStatusInvalid),
				"job8": jobForTest(3, 450, "0xmno", 3, 650, jobStatusInvalid),
				"job9": jobForTest(3, 550, "0xpqr", 3, 550, jobStatusInvalid),
			},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{
					executingChainID:  "1",
					initiatingChainID: "2",
					status:            "future",
					count:             3,
				},
				{
					executingChainID:  "2",
					initiatingChainID: "1",
					status:            "valid",
					count:             3,
				},
				{
					executingChainID:  "3",
					initiatingChainID: "3",
					status:            "invalid",
					count:             3,
				},
			},
			expectedTerminalCalls: []expectedTerminalCall{},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     50,
					max:     150,
				},
				{
					chainID: "2",
					min:     250,
					max:     350,
				},
				{
					chainID: "3",
					min:     450,
					max:     550,
				},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{
					chainID: "1",
					min:     350,
					max:     450,
				},
				{
					chainID: "2",
					min:     150,
					max:     250,
				},
				{
					chainID: "3",
					min:     550,
					max:     650,
				},
			},
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
			require.ElementsMatch(t, tt.expectedMessageStatusCalls, mockMetrics.actualMessageStatusCalls, "message status calls should match")
			require.ElementsMatch(t, tt.expectedTerminalCalls, mockMetrics.actualTerminalCalls, "terminal status change calls should match")
			require.ElementsMatch(t, tt.expectedExecutingRangeCalls, mockMetrics.actualExecutingRangeCalls, "executing block range calls should match")
			require.ElementsMatch(t, tt.expectedInitiatingRangeCalls, mockMetrics.actualInitiatingRangeCalls, "initiating block range calls should match")
		})
	}
}
