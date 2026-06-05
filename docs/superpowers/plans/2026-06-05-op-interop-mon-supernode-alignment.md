# op-interop-mon Supernode/Filter Alignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `op-interop-mon` validate executing messages with the current interop validity model (checksum binding + expiry), add read-only cross-checks against op-interop-filter and op-supernode, and run it against interop-reorg-0 to gather data.

**Architecture:** The monitor stays an independent watchdog: finders read each chain's L2 receipts and emit `Job`s, updaters validate each EM against the *initiating* chain's receipts, and a metric collector emits Prometheus gauges. Phase 1 upgrades validation + adds distinct statuses sourced from a depset file. Phases 2-3 add optional, read-only observers (filter, supernode) that never gate core behavior. Phase 4 deploys and gathers data.

**Tech Stack:** Go, urfave/cli v2, prometheus client, `op-core/interop/messages`, `op-core/interop/depset`, `op-service/eth`, testify.

**Out of scope (do not touch):** `monitor/supervisor_client.go`, `--supervisor-endpoints`, `--trigger-failsafe`, `MetricCollector.TriggerFailsafe`, `MetricCollector.CheckFailsafeStatus`. The failsafe lives in op-interop-filter now; leave the monitor's failsafe-trigger path exactly as-is.

**Reference spec:** `docs/superpowers/specs/2026-06-05-op-interop-mon-supernode-alignment-design.md`

---

## File Structure

**Phase 1 (validity correctness):**
- Modify `op-interop-mon/monitor/job.go` — add `executingTimestamp`, new statuses.
- Modify `op-interop-mon/monitor/finder.go` — set `executingTimestamp` in `processBlock`.
- Modify `op-interop-mon/monitor/updater.go` — new validation logic, expiry window.
- Modify `op-interop-mon/monitor/metric_collector.go` — emit new statuses + reorg metric.
- Modify `op-interop-mon/metrics/metrics.go`, `op-interop-mon/metrics/noop.go` — new metrics.
- Modify `op-interop-mon/flags/flags.go`, `op-interop-mon/monitor/config.go`, `op-interop-mon/monitor/service.go` — `--dependency-set` flag, load depset, pass expiry window.
- Modify `op-interop-mon/README.md`.

**Phase 2 (filter observer):**
- Create `op-interop-mon/monitor/filter_client.go` + `filter_client_test.go`.
- Create `op-interop-mon/monitor/filter_observer.go` + `filter_observer_test.go`.
- Modify flags/config/service/metrics for `--interop-filter-endpoint`, `--interop-filter-min-safety`.

**Phase 3 (supernode observer):**
- Create `op-interop-mon/monitor/supernode_client.go` + `supernode_client_test.go`.
- Create `op-interop-mon/monitor/supernode_observer.go` + `supernode_observer_test.go`.
- Modify flags/config/service/metrics for `--supernode-endpoints`.

**Phase 4 (deploy & gather):** ops only, no source changes beyond a depset JSON artifact.

---

## PHASE 1 — Validity correctness

### Task 1: Add `executingTimestamp` to Job and set it in the finder

**Files:**
- Modify: `op-interop-mon/monitor/job.go` (struct `Job`)
- Modify: `op-interop-mon/monitor/finder.go` (`processBlock`)
- Test: `op-interop-mon/monitor/finder_test.go`

- [ ] **Step 1: Write the failing test**

Add to `op-interop-mon/monitor/finder_test.go`:

```go
func TestProcessBlockSetsExecutingTimestamp(t *testing.T) {
	logger := log.New()
	var got *Job
	finder := NewFinder(
		eth.ChainIDFromUInt64(1),
		&mockClient{},
		// toJobs: emit a single empty job so we can inspect timestamp wiring
		func(receipts []*types.Receipt, executingChain eth.ChainID) []*Job {
			return []*Job{{initiating: &messages.Identifier{ChainID: executingChain}}}
		},
		func(j *Job) { got = j },
		func(chainID eth.ChainID, block eth.BlockInfo) {},
		10,
		logger,
	)

	blockInfo := eth.HeaderBlockInfo(&types.Header{Number: big.NewInt(5), Time: 1234})
	require.NoError(t, finder.processBlock(blockInfo, types.Receipts{}))
	require.NotNil(t, got)
	require.Equal(t, uint64(1234), got.executingTimestamp)
}
```

Ensure imports in `finder_test.go` include `"math/big"`, `messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"`, and `"github.com/stretchr/testify/require"` (add any missing).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd op-interop-mon && go test ./monitor/ -run TestProcessBlockSetsExecutingTimestamp -v`
Expected: FAIL — `got.executingTimestamp` undefined (compile error: `Job` has no field `executingTimestamp`).

- [ ] **Step 3: Add the field to Job**

In `op-interop-mon/monitor/job.go`, add to the `Job` struct (next to the other `executing*` fields):

```go
	executingAddress   common.Address
	executingChain     eth.ChainID
	executingBlock     eth.BlockID
	executingLogIndex  uint
	executingPayload   common.Hash
	executingTimestamp uint64
```

- [ ] **Step 4: Set the field in the finder**

In `op-interop-mon/monitor/finder.go` `processBlock`, set the timestamp in the per-job loop:

```go
	jobs := t.toJobs([]*types.Receipt(receipts), t.chainID)
	firstSeen := time.Now()
	for _, job := range jobs {
		job.firstSeen = firstSeen
		job.executingTimestamp = blockInfo.Time()
		job.UpdateStatus(jobStatusUnknown)
		t.newCallback(job)
	}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd op-interop-mon && go test ./monitor/ -run TestProcessBlockSetsExecutingTimestamp -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add op-interop-mon/monitor/job.go op-interop-mon/monitor/finder.go op-interop-mon/monitor/finder_test.go
git commit -m "feat(op-interop-mon): capture executing block timestamp on jobs"
```

---

### Task 2: Add `expired` and `timestamp_mismatch` job statuses

**Files:**
- Modify: `op-interop-mon/monitor/job.go` (`jobStatus` enum, `isTerminal`, `String`)
- Test: `op-interop-mon/monitor/job_test.go`

- [ ] **Step 1: Write the failing test**

Add to `op-interop-mon/monitor/job_test.go`:

```go
func TestJobStatusStringsAndTerminal(t *testing.T) {
	require.Equal(t, "expired", jobStatusExpired.String())
	require.Equal(t, "timestamp_mismatch", jobStatusTimestampMismatch.String())
	require.True(t, jobStatusExpired.isTerminal())
	require.True(t, jobStatusTimestampMismatch.isTerminal())
	require.False(t, jobStatusUnknown.isTerminal())
}
```

Add `"github.com/stretchr/testify/require"` to imports if missing.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd op-interop-mon && go test ./monitor/ -run TestJobStatusStringsAndTerminal -v`
Expected: FAIL — `jobStatusExpired` / `jobStatusTimestampMismatch` undefined.

- [ ] **Step 3: Add the statuses**

In `op-interop-mon/monitor/job.go`, extend the enum:

```go
const (
	jobStatusUnknown jobStatus = iota
	jobStatusValid
	jobStatusInvalid
	jobStatusExpired
	jobStatusTimestampMismatch
)
```

Update `isTerminal`:

```go
func (j jobStatus) isTerminal() bool {
	switch j {
	case jobStatusValid, jobStatusInvalid, jobStatusExpired, jobStatusTimestampMismatch:
		return true
	default:
		return false
	}
}
```

Update `String`:

```go
func (s jobStatus) String() string {
	switch s {
	case jobStatusUnknown:
		return "unknown"
	case jobStatusValid:
		return "valid"
	case jobStatusInvalid:
		return "invalid"
	case jobStatusExpired:
		return "expired"
	case jobStatusTimestampMismatch:
		return "timestamp_mismatch"
	default:
		return fmt.Sprintf("unknown status: %d", s)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd op-interop-mon && go test ./monitor/ -run TestJobStatusStringsAndTerminal -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add op-interop-mon/monitor/job.go op-interop-mon/monitor/job_test.go
git commit -m "feat(op-interop-mon): add expired and timestamp_mismatch job statuses"
```

---

### Task 3: Thread the message expiry window into the updater

**Files:**
- Modify: `op-interop-mon/monitor/updater.go` (`RPCUpdater`, `NewUpdater`)
- Modify: `op-interop-mon/monitor/service.go` (`initUpdaters` call site — temporary, finalized in Task 6)
- Test: `op-interop-mon/monitor/updater_test.go` (`setupTestUpdater`)

Rationale: the updater only needs the expiry window (a `uint64`), not the whole `DependencySet`. Passing the scalar keeps the updater decoupled and trivially testable. The depset is loaded once in `service.go` (Task 6) and the window extracted there.

- [ ] **Step 1: Update the test helper to pass an expiry window**

In `op-interop-mon/monitor/updater_test.go`, change `setupTestUpdater`:

```go
func setupTestUpdater(t *testing.T) (*RPCUpdater, *mockClient) {
	logger := log.New()
	client := &mockClient{}
	expiry := locks.RWMapFromMap(map[eth.ChainID]eth.NumberAndHash{})
	updater := NewUpdater(eth.ChainIDFromUInt64(1), client, expiry, 604800, logger)
	return updater, client
}
```

- [ ] **Step 2: Run to verify it fails (signature mismatch)**

Run: `cd op-interop-mon && go test ./monitor/ -run TestUpdater -v`
Expected: FAIL — too many arguments to `NewUpdater`.

- [ ] **Step 3: Add the field and parameter**

In `op-interop-mon/monitor/updater.go`, add `messageExpiryWindow uint64` to `RPCUpdater` and `NewUpdater`:

```go
type RPCUpdater struct {
	client  UpdaterClient
	chainID eth.ChainID

	expireTime time.Duration

	// messageExpiryWindow is the interop message expiry window in seconds,
	// sourced from the dependency set. A message is expired if the executing
	// block timestamp exceeds the initiating message timestamp by more than this.
	messageExpiryWindow uint64

	inbox  chan *Job
	closed chan struct{}

	jobs      sync.Map
	finalized *locks.RWMap[eth.ChainID, eth.NumberAndHash]

	log log.Logger
}

func NewUpdater(
	chainID eth.ChainID,
	client UpdaterClient,
	finalized *locks.RWMap[eth.ChainID, eth.NumberAndHash],
	messageExpiryWindow uint64,
	log log.Logger) *RPCUpdater {
	return &RPCUpdater{
		chainID:             chainID,
		client:              client,
		log:                 log.New("component", "rpc_updater", "chain_id", chainID),
		inbox:               make(chan *Job, inboxDepth),
		closed:              make(chan struct{}),
		expireTime:          2 * time.Minute,
		finalized:           finalized,
		messageExpiryWindow: messageExpiryWindow,
	}
}
```

In `op-interop-mon/monitor/service.go` `initUpdaters`, update the call temporarily (finalized in Task 6) to compile:

```go
func (ms *InteropMonitorService) initUpdaters(clients map[eth.ChainID]*sources.EthClient) error {
	for chainID, ethClient := range clients {
		updater := NewUpdater(chainID, ethClient, ms.finalized, ms.messageExpiryWindow, ms.Log)
		ms.updaters[chainID] = updater
	}
	return nil
}
```

Add a field `messageExpiryWindow uint64` to `InteropMonitorService` and, for now, default it in `initFromClients` before `initUpdaters`:

```go
	if ms.messageExpiryWindow == 0 {
		ms.messageExpiryWindow = 604800
	}
```

(Task 6 sets it from the depset.)

- [ ] **Step 4: Run to verify build + existing tests pass**

Run: `cd op-interop-mon && go test ./monitor/ -run TestUpdater -v`
Expected: PASS (existing updater tests still pass with a zero timestamp/window).

- [ ] **Step 5: Commit**

```bash
git add op-interop-mon/monitor/updater.go op-interop-mon/monitor/service.go op-interop-mon/monitor/updater_test.go
git commit -m "refactor(op-interop-mon): thread message expiry window into updater"
```

---

### Task 4: New validation logic in `UpdateJobStatus`

**Files:**
- Modify: `op-interop-mon/monitor/updater.go` (`UpdateJobStatus`)
- Test: `op-interop-mon/monitor/updater_test.go`

Validity check order (equivalent to the full `MessageChecksum` binding, since block number, log index, and chainID are correct by construction — receipts are fetched by `job.initiating.BlockNumber`, the log is found at `job.initiating.LogIndex`, and the updater is keyed by `job.initiating.ChainID`):

1. fetch error -> `unknown`
2. log not found at index -> `invalid`
3. `log.Address != initiating.Origin` -> `invalid`
4. `block.Time() != initiating.Timestamp` -> `timestamp_mismatch`
5. `keccak(LogToMessagePayload(log)) != executingPayload` -> `invalid`
6. `executingTimestamp < initiating.Timestamp` -> `invalid`
7. `executingTimestamp > initiating.Timestamp + messageExpiryWindow` -> `expired`
8. otherwise -> `valid`

- [ ] **Step 1: Write the failing tests**

Add to `op-interop-mon/monitor/updater_test.go` a new table-driven test. Note the valid log's payload hash uses an empty-topics log (`Data` only), matching the existing helper.

```go
func TestUpdaterValidityInvariants(t *testing.T) {
	validLog := &ethtypes.Log{Index: 0, Address: common.HexToAddress("0xabc"), Data: []byte{0x01, 0x02, 0x03}}
	validHash := crypto.Keccak256Hash(messages.LogToMessagePayload(validLog))

	tests := []struct {
		name           string
		origin         common.Address
		initTimestamp  uint64
		execTimestamp  uint64
		blockTime      uint64
		expiryWindow   uint64
		payload        common.Hash
		expectedStatus []jobStatus
	}{
		{
			name:           "valid within expiry window",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusValid},
		},
		{
			name:           "origin mismatch is invalid",
			origin:         common.HexToAddress("0xdead"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
		{
			name:           "block timestamp mismatch",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1100,
			blockTime:      999,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusTimestampMismatch},
		},
		{
			name:           "expired beyond window",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  1000 + 604800 + 1,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusExpired},
		},
		{
			name:           "executing before initiating is invalid",
			origin:         common.HexToAddress("0xabc"),
			initTimestamp:  1000,
			execTimestamp:  999,
			blockTime:      1000,
			expiryWindow:   604800,
			payload:        validHash,
			expectedStatus: []jobStatus{jobStatusInvalid},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.New()
			client := &mockClient{}
			expiry := locks.RWMapFromMap(map[eth.ChainID]eth.NumberAndHash{})
			updater := NewUpdater(eth.ChainIDFromUInt64(1), client, expiry, tt.expiryWindow, logger)

			client.fetchReceiptsByNumber = func(ctx context.Context, number uint64) (eth.BlockInfo, ethtypes.Receipts, error) {
				blk := eth.HeaderBlockInfo(&ethtypes.Header{Number: big.NewInt(100), Time: tt.blockTime})
				return blk, ethtypes.Receipts{{Logs: []*ethtypes.Log{validLog}}}, nil
			}

			job := &Job{
				initiating: &messages.Identifier{
					ChainID:     eth.ChainIDFromUInt64(1),
					BlockNumber: 100,
					LogIndex:    0,
					Origin:      tt.origin,
					Timestamp:   tt.initTimestamp,
				},
				executingBlock:     eth.BlockID{Number: 200},
				executingChain:     eth.ChainIDFromUInt64(2),
				executingPayload:   tt.payload,
				executingTimestamp: tt.execTimestamp,
			}

			updater.UpdateJobStatus(job)
			require.Equal(t, tt.expectedStatus, job.status)
		})
	}
}
```

Add `"math/big"` to the `updater_test.go` imports.

- [ ] **Step 2: Run to verify it fails**

Run: `cd op-interop-mon && go test ./monitor/ -run TestUpdaterValidityInvariants -v`
Expected: FAIL — origin/timestamp/expiry cases produce `invalid`/`valid` instead of the new statuses (old logic only checks payload hash).

- [ ] **Step 3: Implement the new validation**

Replace `UpdateJobStatus` in `op-interop-mon/monitor/updater.go`:

```go
func (t *RPCUpdater) UpdateJobStatus(job *Job) {
	blockInfo, receipts, err := t.client.FetchReceiptsByNumber(context.Background(), job.initiating.BlockNumber)
	if err != nil {
		t.log.Error("error getting block receipts", "error", err)
		job.UpdateStatus(jobStatusUnknown)
		return
	}

	job.AddInitiatingHash(blockInfo.Hash())

	log, err := t.findLogEvent(receipts, job)
	if err == ErrLogNotFound {
		t.log.Warn("initiating log not found", "job", job.String())
		job.UpdateStatus(jobStatusInvalid)
		return
	} else if err != nil {
		t.log.Error("error finding log event", "error", err)
		job.UpdateStatus(jobStatusUnknown)
		return
	}

	// Origin must match the initiating message's declared origin address.
	if log.Address != job.initiating.Origin {
		t.log.Warn("initiating log origin mismatch", "expected", job.initiating.Origin, "got", log.Address)
		job.UpdateStatus(jobStatusInvalid)
		return
	}

	// The initiating block timestamp must match the timestamp bound into the message identifier.
	if blockInfo.Time() != job.initiating.Timestamp {
		t.log.Warn("initiating timestamp mismatch", "expected", job.initiating.Timestamp, "got", blockInfo.Time())
		job.UpdateStatus(jobStatusTimestampMismatch)
		return
	}

	// Payload hash must match (binds the message contents).
	actualHash := crypto.Keccak256Hash(messages.LogToMessagePayload(log))
	if actualHash != job.executingPayload {
		t.log.Warn("log hash mismatch", "expected", job.executingPayload, "got", actualHash)
		job.UpdateStatus(jobStatusInvalid)
		return
	}

	// Expiry invariants: an executing message cannot precede its initiating
	// message, and must be executed within the message expiry window.
	if job.executingTimestamp < job.initiating.Timestamp {
		t.log.Warn("executing message precedes initiating message",
			"executing_ts", job.executingTimestamp, "initiating_ts", job.initiating.Timestamp)
		job.UpdateStatus(jobStatusInvalid)
		return
	}
	if job.executingTimestamp > job.initiating.Timestamp+t.messageExpiryWindow {
		t.log.Warn("executing message is expired",
			"executing_ts", job.executingTimestamp, "initiating_ts", job.initiating.Timestamp, "window", t.messageExpiryWindow)
		job.UpdateStatus(jobStatusExpired)
		return
	}

	job.UpdateStatus(jobStatusValid)
}
```

- [ ] **Step 4: Run to verify the new + existing updater tests pass**

Run: `cd op-interop-mon && go test ./monitor/ -run TestUpdater -v`
Expected: PASS (both `TestUpdaterValidityInvariants` and the existing `TestUpdaterJobStatusUpdate`; the existing cases use zero origin/timestamp/window and still resolve as before).

- [ ] **Step 5: Commit**

```bash
git add op-interop-mon/monitor/updater.go op-interop-mon/monitor/updater_test.go
git commit -m "feat(op-interop-mon): validate EM origin, timestamp binding, and expiry"
```

---

### Task 5: Emit new statuses and an initiating-reorg metric

**Files:**
- Modify: `op-interop-mon/metrics/metrics.go`, `op-interop-mon/metrics/noop.go`
- Modify: `op-interop-mon/monitor/metric_collector.go`
- Test: `op-interop-mon/monitor/metric_collector_test.go`

- [ ] **Step 1: Add the metric to the interface, impl, and noop**

In `op-interop-mon/metrics/metrics.go`, add to the `Metricer` interface:

```go
	RecordInitiatingReorg(executingChainID string, initiatingChainID string)
```

Add a field to `Metrics`:

```go
	initiatingReorgs prometheus.CounterVec
```

Construct it in `NewMetrics` (after `initiatingBlockRange`):

```go
		initiatingReorgs: *factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "initiating_reorgs_total",
			Help:      "Count of jobs whose initiating block was observed at more than one hash (initiating-chain reorg)",
		}, []string{
			"executing_chain_id",
			"initiating_chain_id",
		}),
```

Add the method:

```go
// RecordInitiatingReorg increments when a job's initiating block is seen at multiple hashes.
func (m *Metrics) RecordInitiatingReorg(executingChainID string, initiatingChainID string) {
	m.initiatingReorgs.WithLabelValues(executingChainID, initiatingChainID).Inc()
}
```

In `op-interop-mon/metrics/noop.go`, add:

```go
func (*noopMetrics) RecordInitiatingReorg(executingChainID string, initiatingChainID string) {}
```

- [ ] **Step 2: Add the new statuses to the collector's status init and wire the reorg metric**

In `op-interop-mon/monitor/metric_collector.go` `CollectMetrics`, extend the status-initialization list:

```go
				for _, status := range []string{
					jobStatusValid.String(),
					jobStatusInvalid.String(),
					jobStatusExpired.String(),
					jobStatusTimestampMismatch.String(),
					jobStatusUnknown.String(),
				} {
					messageStatus[exeChain][initChain][status] = 0
				}
```

In the per-job loop, where multiple initiating hashes are detected, add the metric call (keep the existing `log.Warn`):

```go
		if len(initiatingHashes) > 1 {
			m.log.Warn("Initiating BlockNumber found multiple Blocks (reorg of initiating block)",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block_height", job.executingBlock.Number,
				"initiating_block_height", job.initiating.BlockNumber,
				"executing_block_hash", job.executingBlock.Hash,
				"initiating_hashes", initiatingHashes,
			)
			m.m.RecordInitiatingReorg(job.executingChain.String(), job.initiating.ChainID.String())
		}
```

Note: do **not** modify the `shouldFailsafe` logic — it stays driven by `jobStatusInvalid` and the valid<->invalid transition only.

- [ ] **Step 3: Add the method to the test mock**

In `op-interop-mon/monitor/metric_collector_test.go`, add to `mockMetrics`:

```go
	recordInitiatingReorgFn func(executingChainID string, initiatingChainID string)
	actualInitiatingReorgs  []expectedTerminalCall
```

And a method (place near the other mock methods):

```go
func (m *mockMetrics) RecordInitiatingReorg(executingChainID string, initiatingChainID string) {
	if m.recordInitiatingReorgFn != nil {
		m.recordInitiatingReorgFn(executingChainID, initiatingChainID)
	}
	m.actualInitiatingReorgs = append(m.actualInitiatingReorgs, expectedTerminalCall{
		executingChainID:  executingChainID,
		initiatingChainID: initiatingChainID,
	})
}
```

- [ ] **Step 4: Write a test that an expired job is counted and a reorg is recorded**

Add to `op-interop-mon/monitor/metric_collector_test.go`:

```go
func TestCollectMetricsExpiredAndReorg(t *testing.T) {
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 100},
		executingChain: eth.ChainIDFromUInt64(2),
		executingBlock: eth.BlockID{Number: 200},
	}
	job.SetDidMetrics()
	job.UpdateStatus(jobStatusExpired)
	job.AddInitiatingHash(common.HexToHash("0x1"))
	job.AddInitiatingHash(common.HexToHash("0x2"))

	updater := &mockUpdater{collectForMetricsFn: func(m map[JobID]*Job) map[JobID]*Job {
		m[job.ID()] = job
		return m
	}}
	mm := &mockMetrics{}
	mc := NewMetricCollector(log.New(), mm, map[eth.ChainID]Updater{
		eth.ChainIDFromUInt64(1): updater,
		eth.ChainIDFromUInt64(2): &mockUpdater{},
	}, nil, false)

	mc.CollectMetrics()

	var expiredCount float64
	for _, c := range mm.actualMessageStatusCalls {
		if c.status == "expired" && c.executingChainID == eth.ChainIDFromUInt64(2).String() {
			expiredCount = c.count
		}
	}
	require.Equal(t, float64(1), expiredCount)
	require.Len(t, mm.actualInitiatingReorgs, 1)
}
```

Add `"github.com/ethereum/go-ethereum/common"` to imports if missing.

- [ ] **Step 5: Run tests**

Run: `cd op-interop-mon && go test ./monitor/ -run 'TestCollectMetrics' -v && go test ./metrics/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add op-interop-mon/metrics/metrics.go op-interop-mon/metrics/noop.go op-interop-mon/monitor/metric_collector.go op-interop-mon/monitor/metric_collector_test.go
git commit -m "feat(op-interop-mon): emit expired/timestamp_mismatch statuses and reorg metric"
```

---

### Task 6: Add `--dependency-set` flag, load depset, source chains + expiry window

**Files:**
- Modify: `op-interop-mon/flags/flags.go`
- Modify: `op-interop-mon/monitor/config.go`
- Modify: `op-interop-mon/monitor/service.go`
- Test: `op-interop-mon/monitor/config_test.go` (new)

- [ ] **Step 1: Add the flag**

In `op-interop-mon/flags/flags.go`, add to the `var (...)` block:

```go
	DependencySetFlag = &cli.StringFlag{
		Name:      "dependency-set",
		Usage:     "Path to the interop dependency-set JSON file (sources the chain set and message expiry window)",
		EnvVars:   prefixEnvVars("DEPENDENCY_SET"),
		Required:  true,
		TakesFile: true,
	}
```

Add `DependencySetFlag` to `requiredFlags`:

```go
var requiredFlags = []cli.Flag{
	L2RpcsFlag,
	DependencySetFlag,
}
```

- [ ] **Step 2: Add to config**

In `op-interop-mon/monitor/config.go`, add `DependencySetPath string` to `CLIConfig`, read it in `NewConfig`:

```go
		DependencySetPath: ctx.String(flags.DependencySetFlag.Name),
```

and validate in `Check`:

```go
	if c.DependencySetPath == "" {
		return errors.New("dependency-set is required")
	}
```

- [ ] **Step 3: Load depset in service init and source the expiry window + reconcile chains**

In `op-interop-mon/monitor/service.go`, add the import:

```go
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
```

In `initFromCLIConfig`, after `initClients`, load the depset and set the window:

```go
	loader := &depset.JSONDependencySetLoader{Path: cfg.DependencySetPath}
	depSet, err := loader.LoadDependencySet()
	if err != nil {
		return fmt.Errorf("failed to load dependency set: %w", err)
	}
	ms.messageExpiryWindow = depSet.MessageExpiryWindow()

	// Reconcile configured RPCs against the dependency set.
	for chainID := range clients {
		if !depSet.HasChain(chainID) {
			log.Warn("configured L2 RPC chain is not in the dependency set", "chain_id", chainID)
		}
	}
	for _, chainID := range depSet.Chains() {
		if _, ok := clients[chainID]; !ok {
			return fmt.Errorf("dependency set chain %s has no configured L2 RPC; cannot validate its initiating messages", chainID)
		}
	}
```

Remove the temporary `if ms.messageExpiryWindow == 0 { ms.messageExpiryWindow = 604800 }` block added in Task 3 from `initFromClients` (the window now always comes from the depset for the CLI path). Keep a guard so `InteropMonitorServiceFromClients` (test path) still works:

```go
	if ms.messageExpiryWindow == 0 {
		ms.messageExpiryWindow = depset.MessageExpiryTimeSecondsInterop
	}
```

(Place this guard in `initFromClients` so the pre-built-clients constructor defaults sanely. The CLI path sets it before calling `initFromClients`.)

- [ ] **Step 4: Write a config test**

Create `op-interop-mon/monitor/config_test.go`:

```go
package monitor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigCheckRequiresDependencySet(t *testing.T) {
	c := &CLIConfig{L2Rpcs: []string{"http://localhost:8545"}}
	require.ErrorContains(t, c.Check(), "dependency-set is required")
}
```

- [ ] **Step 5: Run + build the binary**

Run: `cd op-interop-mon && go test ./monitor/ -run TestConfigCheck -v && go build ./...`
Expected: PASS + clean build.

- [ ] **Step 6: Commit**

```bash
git add op-interop-mon/flags/flags.go op-interop-mon/monitor/config.go op-interop-mon/monitor/service.go op-interop-mon/monitor/config_test.go
git commit -m "feat(op-interop-mon): source chains and expiry window from --dependency-set"
```

---

### Task 7: Update the README and run the full package test + lint

**Files:**
- Modify: `op-interop-mon/README.md`

- [ ] **Step 1: Update the README**

In `op-interop-mon/README.md`, update the Purpose/Architecture text to reflect that the monitor validates executing messages against the current interop validity model (checksum binding + expiry), that it sources chains/expiry from a `--dependency-set` file, and that op-supervisor has been replaced by op-supernode (CL) + op-interop-filter (EL, failsafe). Add the new metrics (`expired`, `timestamp_mismatch`, `initiating_reorgs_total`) to the MetricCollector section. Do not document the failsafe-trigger path as changed — it is unchanged.

- [ ] **Step 2: Run the full package tests + lint**

Run: `cd op-interop-mon && go test ./... && cd .. && golangci-lint run ./op-interop-mon/... 2>/dev/null || true`
(The repo standard is to lint before committing Go changes — see user memory.)
Expected: tests PASS.

- [ ] **Step 3: Commit**

```bash
git add op-interop-mon/README.md
git commit -m "docs(op-interop-mon): describe checksum/expiry validation and depset sourcing"
```

---

## PHASE 2 — interop-filter observer (optional, read-only)

### Task 8: FilterClient (read-only RPC wrapper)

**Files:**
- Create: `op-interop-mon/monitor/filter_client.go`
- Test: `op-interop-mon/monitor/filter_client_test.go`

The interop-filter exposes (confirmed in `op-interop-filter/filter/frontend.go`): public `interop_checkAccessList(inboxEntries []common.Hash, minSafety safety.Level, executingDescriptor messages.ExecutingDescriptor) error` and public read-only `admin_getFailsafeEnabled() (bool, error)`.

- [ ] **Step 1: Write the failing test**

Create `op-interop-mon/monitor/filter_client_test.go`. The stub must satisfy the `client.RPC` interface from `op-service/client`; confirm its method set (`grep -n "type RPC interface" -A8 op-service/client/*.go`) and adjust the stub accordingly during Step 2.

```go
package monitor

import (
	"context"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// stubRPC implements the client.RPC surface used by FilterClient/SupernodeClient.
type stubRPC struct {
	calls  []string
	result interface{}
	err    error
}

func (s *stubRPC) CallContext(ctx context.Context, out interface{}, method string, args ...interface{}) error {
	s.calls = append(s.calls, method)
	if s.err != nil {
		return s.err
	}
	if b, ok := out.(*bool); ok {
		if v, ok := s.result.(bool); ok {
			*b = v
		}
	}
	return nil
}
func (s *stubRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error { return nil }
func (s *stubRPC) Subscribe(ctx context.Context, namespace string, ch interface{}, args ...interface{}) (ethereum.Subscription, error) {
	return nil, nil
}
func (s *stubRPC) Close() {}

func TestFilterClientGetFailsafeEnabled(t *testing.T) {
	rpc := &stubRPC{result: true}
	fc := &FilterClient{client: rpc, minSafety: safety.CrossUnsafe}
	enabled, err := fc.GetFailsafeEnabled(context.Background())
	require.NoError(t, err)
	require.True(t, enabled)
	require.Contains(t, rpc.calls, "admin_getFailsafeEnabled")
}

func TestFilterClientCheckMessage(t *testing.T) {
	rpc := &stubRPC{}
	fc := &FilterClient{client: rpc, minSafety: safety.CrossUnsafe}
	msg := messages.Message{
		Identifier: messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10, LogIndex: 0, Timestamp: 1000, Origin: common.HexToAddress("0xabc")},
	}
	err := fc.CheckMessage(context.Background(), msg, eth.ChainIDFromUInt64(2), 1100)
	require.NoError(t, err)
	require.Contains(t, rpc.calls, "interop_checkAccessList")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd op-interop-mon && go test ./monitor/ -run TestFilterClient -v`
Expected: FAIL — `FilterClient` undefined (and/or stub interface mismatch to fix).

- [ ] **Step 3: Implement FilterClient**

Create `op-interop-mon/monitor/filter_client.go`:

```go
package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// FilterChecker is the read-only interop-filter surface the observer needs.
type FilterChecker interface {
	// CheckMessage replays a single executing message as an access list to the filter.
	CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error
	GetFailsafeEnabled(ctx context.Context) (bool, error)
	Close()
}

// FilterClient calls the op-interop-filter public RPC (read-only).
type FilterClient struct {
	client    client.RPC
	minSafety safety.Level
	log       log.Logger
}

var _ FilterChecker = (*FilterClient)(nil)

func NewFilterClient(endpoint string, minSafety safety.Level, log log.Logger) (*FilterClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("interop-filter endpoint not configured")
	}
	c, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create interop-filter client: %w", err)
	}
	return &FilterClient{client: c, minSafety: minSafety, log: log}, nil
}

// CheckMessage builds the access-list for one executing message and calls interop_checkAccessList.
// A nil error means the filter considers the message valid at minSafety; a non-nil error is the filter's rejection.
func (fc *FilterClient) CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error {
	access := msg.Access()
	entries := messages.EncodeAccessList([]messages.Access{access})
	execDesc := messages.ExecutingDescriptor{ChainID: executingChain, Timestamp: executingTimestamp}
	return fc.client.CallContext(ctx, nil, "interop_checkAccessList", entries, fc.minSafety, execDesc)
}

func (fc *FilterClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := fc.client.CallContext(ctx, &enabled, "admin_getFailsafeEnabled")
	return enabled, err
}

func (fc *FilterClient) Close() { fc.client.Close() }
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd op-interop-mon && go test ./monitor/ -run TestFilterClient -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add op-interop-mon/monitor/filter_client.go op-interop-mon/monitor/filter_client_test.go
git commit -m "feat(op-interop-mon): add read-only interop-filter client"
```

---

### Task 9: FilterObserver + flags + metrics

**Files:**
- Create: `op-interop-mon/monitor/filter_observer.go` + `filter_observer_test.go`
- Modify: `op-interop-mon/metrics/metrics.go`, `op-interop-mon/metrics/noop.go`, `metric_collector_test.go` (mock)
- Modify: `op-interop-mon/flags/flags.go`, `op-interop-mon/monitor/config.go`, `op-interop-mon/monitor/service.go`, `op-interop-mon/monitor/metric_collector.go`

- [ ] **Step 1: Add metrics (interface, impl, noop, mock)**

In `op-interop-mon/metrics/metrics.go` `Metricer`:

```go
	RecordFilterDivergence(executingChainID string, initiatingChainID string, monitorStatus string, filterStatus string)
	RecordFilterFailsafe(enabled bool)
```

Add fields `filterDivergence prometheus.CounterVec` (labels `executing_chain_id, initiating_chain_id, monitor_status, filter_status`, name `filter_divergence_total`) and `filterFailsafe prometheus.Gauge` (name `interop_filter_failsafe`), construct them in `NewMetrics`, and add methods:

```go
func (m *Metrics) RecordFilterDivergence(executingChainID, initiatingChainID, monitorStatus, filterStatus string) {
	m.filterDivergence.WithLabelValues(executingChainID, initiatingChainID, monitorStatus, filterStatus).Inc()
}
func (m *Metrics) RecordFilterFailsafe(enabled bool) {
	if enabled {
		m.filterFailsafe.Set(1)
	} else {
		m.filterFailsafe.Set(0)
	}
}
```

Add noop implementations. In `metric_collector_test.go`'s `mockMetrics`, add fields `actualFilterDivergences []expectedMessageStatusCall` and `lastFilterFailsafe bool` plus the two methods that populate them (mirror the Task 5 mock pattern).

- [ ] **Step 2: Write the observer test**

Create `op-interop-mon/monitor/filter_observer_test.go`:

```go
package monitor

import (
	"context"
	"errors"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockFilterChecker struct {
	checkErr error
	failsafe bool
}

func (m *mockFilterChecker) CheckMessage(ctx context.Context, msg messages.Message, ec eth.ChainID, ts uint64) error {
	return m.checkErr
}
func (m *mockFilterChecker) GetFailsafeEnabled(ctx context.Context) (bool, error) { return m.failsafe, nil }
func (m *mockFilterChecker) Close()                                               {}

func TestFilterObserverDivergence(t *testing.T) {
	// monitor says valid, filter says invalid -> divergence recorded
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: errors.New("filter rejects")}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})

	require.Len(t, mm.actualFilterDivergences, 1)
}

func TestFilterObserverFailsafeGauge(t *testing.T) {
	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{failsafe: true}, mm, log.New())
	obs.PollFailsafe(context.Background())
	require.True(t, mm.lastFilterFailsafe)
}
```

- [ ] **Step 3: Implement FilterObserver**

Create `op-interop-mon/monitor/filter_observer.go`:

```go
package monitor

import (
	"context"
	"time"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum/go-ethereum/log"
)

// FilterObserver cross-checks the monitor's verdict against the interop-filter.
type FilterObserver struct {
	filter  FilterChecker
	m       metrics.Metricer
	log     log.Logger
	timeout time.Duration
}

func NewFilterObserver(filter FilterChecker, m metrics.Metricer, log log.Logger) *FilterObserver {
	return &FilterObserver{filter: filter, m: m, log: log, timeout: 2 * time.Second}
}

// Observe replays each terminal job's executing message to the filter and records divergences.
func (o *FilterObserver) Observe(ctx context.Context, jobs map[JobID]*Job) {
	for _, job := range jobs {
		status := job.LatestStatus()
		if !status.isTerminal() {
			continue
		}
		msg := messages.Message{Identifier: *job.initiating, PayloadHash: job.executingPayload}
		cctx, cancel := context.WithTimeout(ctx, o.timeout)
		err := o.filter.CheckMessage(cctx, msg, job.executingChain, job.executingTimestamp)
		cancel()
		monitorValid := status == jobStatusValid
		filterValid := err == nil
		if monitorValid != filterValid {
			filterStatus := "valid"
			if !filterValid {
				filterStatus = "invalid"
			}
			o.log.Warn("monitor/filter verdict divergence",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"monitor_status", status.String(),
				"filter_status", filterStatus,
				"filter_err", err,
			)
			o.m.RecordFilterDivergence(job.executingChain.String(), job.initiating.ChainID.String(), status.String(), filterStatus)
		}
	}
}

// PollFailsafe reads the filter's failsafe state and records it as a gauge.
func (o *FilterObserver) PollFailsafe(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	enabled, err := o.filter.GetFailsafeEnabled(cctx)
	if err != nil {
		o.log.Error("failed to read interop-filter failsafe state", "error", err)
		return
	}
	o.m.RecordFilterFailsafe(enabled)
}
```

- [ ] **Step 4: Wire flags + service + collector**

Add optional flags `--interop-filter-endpoint` (StringFlag) and `--interop-filter-min-safety` (StringFlag, default `"cross-unsafe"`) to `flags.go`; read into `CLIConfig` (`InteropFilterEndpoint string`, `InteropFilterMinSafety string`). In `service.go`, if `InteropFilterEndpoint != ""`, build a `FilterClient` (parse min-safety via `safety.Level(cfg.InteropFilterMinSafety)`; reject with an error if `!level.Validate()`) and pass a `*FilterObserver` to `NewMetricCollector`. Add an optional `filterObserver *FilterObserver` field to `MetricCollector` (extend the constructor signature); at the end of `CollectMetrics`, if non-nil, call `m.filterObserver.Observe(context.Background(), jobMap)` and `m.filterObserver.PollFailsafe(context.Background())`.

- [ ] **Step 5: Run tests + build**

Run: `cd op-interop-mon && go test ./... && go build ./...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add op-interop-mon/
git commit -m "feat(op-interop-mon): add optional interop-filter observer (divergence + failsafe gauge)"
```

---

## PHASE 3 — supernode observer (optional, read-only)

### Task 10: SupernodeClient

**Files:**
- Create: `op-interop-mon/monitor/supernode_client.go`
- Test: `op-interop-mon/monitor/supernode_client_test.go`

> **Verify-at-impl:** Confirm the exact supernode RPC method names + response shapes against `op-supernode` before finalizing (`grep -rn "RPCNamespace\|func.*SyncStatus\|Heartbeat" op-supernode/supernode/activity/`). Expected: `supernode_syncStatus` returning per-chain safe/finalized heads, and `heartbeat_check`. Adjust the `SupernodeSyncStatus` struct to match the real response.

- [ ] **Step 1: Write the failing test** — `SupernodeClient` with the `stubRPC` (reuse the stub from Task 8; it lives in `filter_client_test.go` in the same package) verifying `SyncStatus` calls `supernode_syncStatus` and `Heartbeat` calls `heartbeat_check`:

```go
func TestSupernodeClientCalls(t *testing.T) {
	rpc := &stubRPC{}
	sc := &SupernodeClient{client: rpc}
	_, _ = sc.SyncStatus(context.Background())
	_ = sc.Heartbeat(context.Background())
	require.Contains(t, rpc.calls, "supernode_syncStatus")
	require.Contains(t, rpc.calls, "heartbeat_check")
}
```

- [ ] **Step 2: Run to verify it fails.** Run: `cd op-interop-mon && go test ./monitor/ -run TestSupernodeClient -v` — FAIL (undefined).

- [ ] **Step 3: Implement** `op-interop-mon/monitor/supernode_client.go`:

```go
package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

// SupernodeSyncStatus is the subset of supernode_syncStatus the monitor consumes.
// VERIFY shape against op-supernode at implementation time.
type SupernodeSyncStatus struct {
	Chains map[eth.ChainID]struct {
		CrossSafe eth.BlockID `json:"crossSafe"`
		Finalized eth.BlockID `json:"finalized"`
	} `json:"chains"`
}

type SupernodeObserverClient interface {
	SyncStatus(ctx context.Context) (*SupernodeSyncStatus, error)
	Heartbeat(ctx context.Context) error
	Close()
}

type SupernodeClient struct {
	client client.RPC
	log    log.Logger
}

var _ SupernodeObserverClient = (*SupernodeClient)(nil)

func NewSupernodeClient(endpoint string, log log.Logger) (*SupernodeClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("supernode endpoint not configured")
	}
	c, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create supernode client: %w", err)
	}
	return &SupernodeClient{client: c, log: log}, nil
}

func (sc *SupernodeClient) SyncStatus(ctx context.Context) (*SupernodeSyncStatus, error) {
	var out SupernodeSyncStatus
	err := sc.client.CallContext(ctx, &out, "supernode_syncStatus")
	return &out, err
}

func (sc *SupernodeClient) Heartbeat(ctx context.Context) error {
	return sc.client.CallContext(ctx, nil, "heartbeat_check")
}

func (sc *SupernodeClient) Close() { sc.client.Close() }
```

- [ ] **Step 4: Run to verify it passes.** Run: `cd op-interop-mon && go test ./monitor/ -run TestSupernodeClient -v` — PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(op-interop-mon): add read-only supernode client"`.

---

### Task 11: SupernodeObserver + flags + metrics

**Files:**
- Create: `op-interop-mon/monitor/supernode_observer.go` + `supernode_observer_test.go`
- Modify: `op-interop-mon/metrics/metrics.go`, `noop.go`, `metric_collector_test.go` (mock)
- Modify: `op-interop-mon/flags/flags.go`, `config.go`, `service.go`, `metric_collector.go`

- [ ] **Step 1: Add metrics** — to the `Metricer` interface, impl, noop, and `mockMetrics`:
  - `RecordSupernodeUp(endpoint string, up bool)` → gauge `supernode_up{endpoint}`
  - `RecordSupernodeSafeHead(chainID string, level string, blockNumber uint64)` → gauge `supernode_safe_head{chain_id,level}`
  - `RecordCrossSafetyViolation(executingChainID string, initiatingChainID string, level string)` → counter `cross_safety_violations_total{executing_chain_id,initiating_chain_id,level}`
  Add a `mockMetrics` field `actualCrossSafetyViolations []expectedTerminalCall` (and an `up`/`safeHead` recorder as needed) populated by the methods.

- [ ] **Step 2: Write the observer test** — `op-interop-mon/monitor/supernode_observer_test.go` with a mock `SupernodeObserverClient`:

```go
package monitor

import (
	"context"
	"errors"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockSupernodeClient struct {
	status *SupernodeSyncStatus
	hbErr  error
}

func (m *mockSupernodeClient) SyncStatus(ctx context.Context) (*SupernodeSyncStatus, error) {
	return m.status, nil
}
func (m *mockSupernodeClient) Heartbeat(ctx context.Context) error { return m.hbErr }
func (m *mockSupernodeClient) Close()                              {}

func TestSupernodeObserverCrossSafetyViolation(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := &SupernodeSyncStatus{Chains: map[eth.ChainID]struct {
		CrossSafe eth.BlockID `json:"crossSafe"`
		Finalized eth.BlockID `json:"finalized"`
	}{
		execChain: {CrossSafe: eth.BlockID{Number: 250}},
	}}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200}, // <= cross-safe head 250 => violation
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Len(t, mm.actualCrossSafetyViolations, 1)
}

func TestSupernodeObserverHeartbeatDown(t *testing.T) {
	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{hbErr: errors.New("down")}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{})
	require.False(t, mm.lastSupernodeUp)
}
```

(Add `lastSupernodeUp bool` to `mockMetrics`, set in `RecordSupernodeUp`.)

- [ ] **Step 3: Implement SupernodeObserver** `op-interop-mon/monitor/supernode_observer.go`:

```go
package monitor

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-interop-mon/metrics"
	"github.com/ethereum/go-ethereum/log"
)

type SupernodeObserver struct {
	endpoint string
	client   SupernodeObserverClient
	m        metrics.Metricer
	log      log.Logger
	timeout  time.Duration
}

func NewSupernodeObserver(endpoint string, c SupernodeObserverClient, m metrics.Metricer, log log.Logger) *SupernodeObserver {
	return &SupernodeObserver{endpoint: endpoint, client: c, m: m, log: log, timeout: 2 * time.Second}
}

func (o *SupernodeObserver) Observe(ctx context.Context, jobs map[JobID]*Job) {
	cctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	if err := o.client.Heartbeat(cctx); err != nil {
		o.log.Error("supernode heartbeat failed", "endpoint", o.endpoint, "error", err)
		o.m.RecordSupernodeUp(o.endpoint, false)
		return
	}
	o.m.RecordSupernodeUp(o.endpoint, true)

	status, err := o.client.SyncStatus(cctx)
	if err != nil {
		o.log.Error("supernode syncStatus failed", "endpoint", o.endpoint, "error", err)
		return
	}
	for chainID, s := range status.Chains {
		o.m.RecordSupernodeSafeHead(chainID.String(), "cross_safe", s.CrossSafe.Number)
		o.m.RecordSupernodeSafeHead(chainID.String(), "finalized", s.Finalized.Number)
	}

	// Highest-signal check: a bad EM that the supernode already promoted to cross-safe.
	for _, job := range jobs {
		st := job.LatestStatus()
		if st == jobStatusValid || st == jobStatusUnknown {
			continue
		}
		s, ok := status.Chains[job.executingChain]
		if !ok {
			continue
		}
		if job.executingBlock.Number <= s.CrossSafe.Number {
			o.log.Error("bad executing message at/below supernode cross-safe head",
				"executing_chain_id", job.executingChain,
				"initiating_chain_id", job.initiating.ChainID,
				"executing_block", job.executingBlock.Number,
				"cross_safe_head", s.CrossSafe.Number,
				"status", st.String(),
			)
			o.m.RecordCrossSafetyViolation(job.executingChain.String(), job.initiating.ChainID.String(), "cross_safe")
		}
	}
}
```

- [ ] **Step 4: Wire flags + service + collector** — add optional `--supernode-endpoints` (StringSliceFlag) → `CLIConfig.SupernodeEndpoints []string`. In `service.go`, build one `SupernodeObserver` per endpoint (via `NewSupernodeClient`) and pass the slice to `NewMetricCollector` (extend signature with `supernodeObservers []*SupernodeObserver`). In `CollectMetrics`, after the filter observer, iterate and call each `Observe(context.Background(), jobMap)`.

- [ ] **Step 5: Run tests + build.** Run: `cd op-interop-mon && go test ./... && go build ./...` — PASS.

- [ ] **Step 6: Commit** — `git commit -m "feat(op-interop-mon): add optional supernode observer with cross-safety violation metric"`.

---

## PHASE 4 — Deploy against interop-reorg-0 & gather data

> Ops task. No TDD. Run from the repo root unless noted.

- [ ] **Step 1: Build**

```bash
just op-interop-mon                       # local binary smoke
docker buildx bake op-interop-mon         # image
```

- [ ] **Step 2: Write the reorg-0 depset file**

Create `/tmp/interop-reorg-0-depset.json`:

```json
{"dependencies":{"420120132":{},"420120133":{}}}
```

If the devnet overrides the expiry window, add `"overrideMessageExpiryWindow": <seconds>`. Confirm from the devnet config:
`grep -rn "MessageExpiry\|expiry\|dependency" /Users/jacobelias/workspace/ethereum-optimism/devnets-private/dev/interop-reorg-0/` — default `604800` if absent.

- [ ] **Step 3: Port-forward the endpoints** (cluster `oplabs-dev-ent-networks-us-ue1-0`; see neti-ops / k8s-ops skills and user memory for the pattern). Confirm exact service names with `kubectl get svc -A | grep interop-reorg-0`.

```bash
kubectl port-forward -n an-interop-reorg-0-0-proxyd-public svc/an-interop-reorg-0-0-proxyd-public 8645:8545 &
kubectl port-forward -n an-interop-reorg-0-1-proxyd-public svc/an-interop-reorg-0-1-proxyd-public 8745:8545 &
# interop-filter (front of proxyd-public) and supernode (proxyd-cl): identify svc + port from `kubectl get svc`
```

- [ ] **Step 4: Run the monitor locally against the devnet**

```bash
./bin/op-interop-mon \
  --dependency-set /tmp/interop-reorg-0-depset.json \
  --l2-rpcs http://localhost:8645 --l2-rpcs http://localhost:8745 \
  --interop-filter-endpoint http://localhost:<filter-port> \
  --supernode-endpoints http://localhost:<supernode-cl-port> \
  --metrics.enabled --metrics.addr 127.0.0.1 --metrics.port 7300 \
  --log.level info
```

(Failsafe flags intentionally omitted: `--supervisor-endpoints` unset; `--trigger-failsafe` irrelevant.)

- [ ] **Step 5: Gather data**

```bash
curl -s localhost:7300/metrics | grep -E 'op_interop_mon_(message_status|terminal_status_changes|initiating_reorgs_total|interop_filter_failsafe|filter_divergence_total|cross_safety_violations_total|supernode_)'
```

Let it run across a reorg cycle. Capture: valid/invalid/expired/timestamp_mismatch counts per chain pair, filter divergences, filter failsafe engagements, initiating reorgs, cross-safety violations, and supernode head progression. Summarize findings back to the user.

---

## Self-Review

**Spec coverage:**
- Validity (checksum binding + expiry) → Tasks 1, 4. ✓
- Distinct statuses → Task 2 (defined), Tasks 4-5 (used/emitted). ✓
- Depset file (`--dependency-set`) → Task 6. ✓
- Filter observer (divergence + failsafe gauge) → Tasks 8-9. ✓
- Supernode observer (liveness/heads + cross-safety violation) → Tasks 10-11. ✓
- Reorg metric → Task 5. ✓
- Deploy & gather → Phase 4. ✓
- Failsafe untouched → header + Task 5 Step 2 note. ✓

**Type consistency:** `messageExpiryWindow uint64` (Tasks 3,4); `FilterChecker.CheckMessage(ctx, messages.Message, eth.ChainID, uint64)` (Tasks 8,9); `SupernodeObserverClient.SyncStatus/Heartbeat` + `SupernodeSyncStatus` (Tasks 10,11); metric method names consistent across metrics.go/noop.go/mockMetrics; `NewMetricCollector` signature is extended in Tasks 9 and 11 (filter observer, then supernode observers) — when executing, apply both extensions together if done out of order. ✓

**Known verification points (not placeholders):** the `client.RPC` interface method set for the test stub (Task 8) and the supernode RPC response shape (Task 10) are explicitly flagged to confirm against the live types at implementation time.
