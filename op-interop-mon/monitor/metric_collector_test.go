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

// mockFailsafeClient implements the FailsafeClient interface for testing
type mockFailsafeClient struct {
	setFailsafeEnabledCalled bool
	setFailsafeEnabledValue  bool
}

func (m *mockFailsafeClient) SetFailsafeEnabled(ctx context.Context, enabled bool) error {
	m.setFailsafeEnabledCalled = true
	m.setFailsafeEnabledValue = enabled
	return nil
}

func (m *mockFailsafeClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	return m.setFailsafeEnabledValue, nil
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
	mockFailsafeClients := []FailsafeClient{}

	// Create new MetricCollector
	collector := NewMetricCollector(logger, mockMetrics, updaters, mockFailsafeClients, true)

	// Verify the collector was created correctly
	require.NotNil(t, collector)
	require.Equal(t, logger, collector.log)
	require.Equal(t, mockMetrics, collector.m)
	require.Equal(t, updaters, collector.updaters)
	require.Equal(t, mockFailsafeClients, collector.failsafeClients)
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
	mockFailsafeClients := []FailsafeClient{&mockFailsafeClient{}}

	// Create new MetricCollector
	collector := NewMetricCollector(logger, mockMetrics, updaters, mockFailsafeClients, true)

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

// TestFailsafeTriggering tests that the failsafe API is called when invalid messages are detected
func TestFailsafeTriggering(t *testing.T) {
	type testCase struct {
		name                    string
		job                     *Job
		expectFailsafeCalled    bool
		expectFailsafeEnabled   bool
		expectFailsafeClientNil bool
	}

	tests := []testCase{
		{
			name:                  "invalid message triggers failsafe",
			job:                   jobForTest(1, 100, "0x123", 2, 200, jobStatusInvalid),
			expectFailsafeCalled:  true,
			expectFailsafeEnabled: true,
		},
		{
			name:                  "terminal state change triggers failsafe",
			job:                   jobForTest(1, 100, "0x123", 2, 200, jobStatusValid, jobStatusInvalid),
			expectFailsafeCalled:  true,
			expectFailsafeEnabled: true,
		},
		{
			name:                  "valid message does not trigger failsafe",
			job:                   jobForTest(1, 100, "0x123", 2, 200, jobStatusValid),
			expectFailsafeCalled:  false,
			expectFailsafeEnabled: false,
		},
		{
			name:                    "nil failsafe client means no failsafe",
			job:                     jobForTest(1, 100, "0x123", 2, 200, jobStatusInvalid),
			expectFailsafeCalled:    false,
			expectFailsafeEnabled:   false,
			expectFailsafeClientNil: true,
		},
		{
			name:                  "triggerFailsafe false prevents API call",
			job:                   jobForTest(1, 100, "0x123", 2, 200, jobStatusInvalid),
			expectFailsafeCalled:  false,
			expectFailsafeEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test dependencies
			logger := log.New()
			mockMetrics := &mockMetrics{}

			// Create mock updater that returns the test job
			updater := &mockUpdater{
				collectForMetricsFn: func(jobs map[JobID]*Job) map[JobID]*Job {
					jobs[tt.job.ID()] = tt.job
					return jobs
				},
			}

			updaters := map[eth.ChainID]Updater{
				eth.ChainIDFromUInt64(1): updater,
				eth.ChainIDFromUInt64(2): &mockUpdater{},
			}

			// Create collector with or without failsafe client
			var collector *MetricCollector
			if tt.expectFailsafeClientNil {
				collector = NewMetricCollector(logger, mockMetrics, updaters, nil, true)
			} else {
				mockFailsafeClients := []FailsafeClient{&mockFailsafeClient{}}
				// Use triggerFailsafe based on whether we expect the API to be called
				triggerFailsafe := tt.expectFailsafeCalled
				collector = NewMetricCollector(logger, mockMetrics, updaters, mockFailsafeClients, triggerFailsafe)

				// Run metric collection
				collector.CollectMetrics()

				// Verify failsafe behavior
				mockFailsafeClient := mockFailsafeClients[0].(*mockFailsafeClient)
				require.Equal(t, tt.expectFailsafeCalled, mockFailsafeClient.setFailsafeEnabledCalled, "Failsafe API call should match expectation")
				if tt.expectFailsafeCalled {
					require.Equal(t, tt.expectFailsafeEnabled, mockFailsafeClient.setFailsafeEnabledValue, "Failsafe enabled value should match expectation")
				}
			}

			// For nil client test, just verify no panic
			if tt.expectFailsafeClientNil {
				collector.CollectMetrics()
				// Test passes if no panic occurs
			}
		})
	}
}

// TestCollectMetrics tests the metric collection functionality
func TestCollectMetrics(t *testing.T) {
	type testCase struct {
		name string
		// Input jobs from each updater
		updater1Jobs map[JobID]*Job
		updater2Jobs map[JobID]*Job
		updater3Jobs map[JobID]*Job
		// Expected metric calls (only non-zero expectations)
		expectedMessageStatusCalls   []expectedMessageStatusCall
		expectedTerminalCalls        []expectedTerminalCall
		expectedExecutingRangeCalls  []expectedBlockRangeCall
		expectedInitiatingRangeCalls []expectedBlockRangeCall
	}

	tests := []testCase{
		{
			name:         "empty job maps",
			updater1Jobs: map[JobID]*Job{},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			// All expectations are default (zero)
		},
		{
			name: "single job with valid status",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{"1", "2", "valid", 1},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{"1", 100, 100},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{"2", 200, 200},
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
				{"1", "2", "invalid", 1},
			},
			expectedTerminalCalls: []expectedTerminalCall{
				{"1", "2", 1},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{"1", 100, 100},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{"2", 200, 200},
			},
		},
		{
			name: "multiple jobs with same status",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid),
				"job2": jobForTest(1, 101, "0x456", 2, 201, jobStatusValid),
			},
			updater2Jobs: map[JobID]*Job{},
			updater3Jobs: map[JobID]*Job{},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{"1", "2", "valid", 2},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{"1", 100, 101},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{"2", 200, 201},
			},
		},
		{
			name: "jobs across different chains",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid),
			},
			updater2Jobs: map[JobID]*Job{
				"job2": jobForTest(2, 300, "0x456", 3, 400, jobStatusValid),
			},
			updater3Jobs: map[JobID]*Job{
				"job3": jobForTest(3, 500, "0x789", 1, 600, jobStatusInvalid),
			},
			expectedMessageStatusCalls: []expectedMessageStatusCall{
				{"1", "2", "valid", 1},
				{"2", "3", "valid", 1},
				{"3", "1", "invalid", 1},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{"1", 100, 100},
				{"2", 300, 300},
				{"3", 500, 500},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{"1", 600, 600},
				{"2", 200, 200},
				{"3", 400, 400},
			},
		},
		{
			name: "complex block ranges",
			updater1Jobs: map[JobID]*Job{
				"job1": jobForTest(1, 100, "0x123", 2, 200, jobStatusValid),
				"job2": jobForTest(1, 50, "0x456", 2, 250, jobStatusValid),
				"job3": jobForTest(1, 150, "0x789", 2, 150, jobStatusValid),
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
				{"1", "2", "valid", 3},
				{"2", "1", "valid", 3},
				{"3", "3", "invalid", 3},
			},
			expectedExecutingRangeCalls: []expectedBlockRangeCall{
				{"1", 50, 150},
				{"2", 250, 350},
				{"3", 450, 550},
			},
			expectedInitiatingRangeCalls: []expectedBlockRangeCall{
				{"1", 350, 450},
				{"2", 150, 250},
				{"3", 550, 650},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test dependencies
			logger := log.New()
			mockMetrics := &mockMetrics{}
			mockFailsafeClients := []FailsafeClient{&mockFailsafeClient{}}

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
			}, mockFailsafeClients, true)

			// Run metric collection
			collector.CollectMetrics()

			// Generate expected calls. By default, all different combinations of executing and initiating chains and statuses are expected,
			// but will have a zero value if not specified in the test case. Specific expectations are overloaded over the defaults.

			// Default Message Status Calls with specific expectations merged in
			var expectedMessageStatusCalls []expectedMessageStatusCall
			for _, executing := range []string{"1", "2", "3"} {
				for _, initiating := range []string{"1", "2", "3"} {
					for _, status := range []string{"valid", "invalid", "unknown"} {
						call := expectedMessageStatusCall{executing, initiating, status, 0}
						for _, specific := range tt.expectedMessageStatusCalls {
							if specific.executingChainID == executing &&
								specific.initiatingChainID == initiating &&
								specific.status == status {
								call = specific
								break
							}
						}
						expectedMessageStatusCalls = append(expectedMessageStatusCalls, call)
					}
				}
			}

			// Default Terminal Calls with specific expectations merged in
			var expectedTerminalCalls []expectedTerminalCall
			for _, executing := range []string{"1", "2", "3"} {
				for _, initiating := range []string{"1", "2", "3"} {
					call := expectedTerminalCall{executing, initiating, 0}
					for _, specific := range tt.expectedTerminalCalls {
						if specific.executingChainID == executing &&
							specific.initiatingChainID == initiating {
							call = specific
							break
						}
					}
					expectedTerminalCalls = append(expectedTerminalCalls, call)
				}
			}

			// Default Executing Range Calls with specific expectations merged in
			var expectedExecutingRangeCalls []expectedBlockRangeCall
			for _, chainID := range []string{"1", "2", "3"} {
				call := expectedBlockRangeCall{chainID, 0, 0}
				for _, specific := range tt.expectedExecutingRangeCalls {
					if specific.chainID == chainID {
						call = specific
						break
					}
				}
				expectedExecutingRangeCalls = append(expectedExecutingRangeCalls, call)
			}

			// Default Initiating Range Calls with specific expectations merged in
			var expectedInitiatingRangeCalls []expectedBlockRangeCall
			for _, chainID := range []string{"1", "2", "3"} {
				call := expectedBlockRangeCall{chainID, 0, 0}
				for _, specific := range tt.expectedInitiatingRangeCalls {
					if specific.chainID == chainID {
						call = specific
						break
					}
				}
				expectedInitiatingRangeCalls = append(expectedInitiatingRangeCalls, call)
			}

			// Verify metric calls
			require.ElementsMatch(t, expectedMessageStatusCalls, mockMetrics.actualMessageStatusCalls, "message status calls should match")
			require.ElementsMatch(t, expectedTerminalCalls, mockMetrics.actualTerminalCalls, "terminal status change calls should match")
			require.ElementsMatch(t, expectedExecutingRangeCalls, mockMetrics.actualExecutingRangeCalls, "executing block range calls should match")
			require.ElementsMatch(t, expectedInitiatingRangeCalls, mockMetrics.actualInitiatingRangeCalls, "initiating block range calls should match")
		})
	}
}
