# op-supernode Fuzzing Targets

## Properties Reference

### Interop Message Verification (verifyInteropMessages)
| ID | Property | Tested By |
|----|----------|-----------|
| P1 | Valid cross-chain messages never produce `InvalidHeads` | `FuzzVerifyInteropMessagesValid` |
| P2 | Every invalidation type is correctly detected (unknown chain, timestamp violation, expired, conflict, hash mismatch) | `FuzzVerifyInteropMessagesFails` |
| P3 | `Result.IsValid()` ↔ `len(InvalidHeads) == 0` | `FuzzVerifyInteropMessagesValid` |
| P4 | `execMsg.Timestamp + ExpiryTime` overflow doesn't cause false positive/negative; exact boundary values handled correctly | `FuzzVerifyExpiryBoundary`, `FuzzVerifyExpiryOverflow` |
| P5 | First block (`ErrSkipped` path) correctly handles hash match vs mismatch against `FirstSealedBlock` | `FuzzVerifyFirstBlockSkipped` |
| P6 | Block with multiple invalid executing messages is still marked invalid (first-invalid short-circuits) | `FuzzVerifyMultipleInvalidMessages` |
| P7 | Chains not present in `logsDBs` are silently excluded from `Result.L2Heads` | `FuzzVerifyMissingChains` |

### LogsDB Timestamp & Block Processing
| ID | Property | Tested By |
|----|----------|-----------|
| P9 | Gap violations are always detected (`gap > blockTime` triggers error) | `FuzzVerifyCanAddTimestamp` |
| P11 | First block with empty parent hash is accepted exactly once (virtual parent seal) | `FuzzProcessBlockLogs` |
| P12 | `AddLog` called once per log, `SealBlock` called expected number of times, log indices sequential | `FuzzProcessBlockLogs` |
| P13 | Non-block-time-aligned gaps only warn, don't error | `FuzzVerifyCanAddTimestamp` |

### VerifiedDB Commit/Rewind
| ID | Property | Tested By |
|----|----------|-----------|
| P15 | `Commit(result)` succeeds iff `result.Timestamp == lastTimestamp + 1` (or first commit at any ts) | `FuzzVerifiedDBCommitRewind`, `FuzzVerifiedDBFirstCommit` |
| P16 | After `Rewind(ts)`, `LastTimestamp()` returns `ts - 1` (or uninitialized if all deleted) | `FuzzVerifiedDBCommitRewind` |
| P17 | After `Rewind(ts)`, `Get(t)` errors for all `t >= ts` | `FuzzVerifiedDBCommitRewind` |
| P18 | After `Rewind(ts)`, `Commit(ts)` succeeds (re-commit from rewind point) | `FuzzVerifiedDBCommitRewind`, `FuzzVerifiedDBFirstCommit` |
| P19 | `ErrAlreadyCommitted` and `ErrNonSequential` are correctly distinguished | `FuzzVerifiedDBCommitRewind` |
| P20 | JSON round-trip preserves all `VerifiedResult` fields; data survives close/reopen | `FuzzVerifiedDBCommitRewind`, `FuzzVerifiedDBPersistence` |

### DenyList
| ID | Property | Tested By |
|----|----------|-----------|
| P21 | `Contains(h, hash)` returns true iff `Add(h, hash)` was previously called | `FuzzDenyListAddContains` |
| P22 | `Add` is idempotent — duplicate adds don't increase hash count | `FuzzDenyListAddContains` |
| P23 | Hashes at different heights are isolated from each other | `FuzzDenyListAddContains` |
| P24 | Concatenated 32-byte hash storage handles boundary alignment correctly | `FuzzDenyListAddContains` |

### Engine Controller Rewind
| ID | Property | Tested By |
|----|----------|-----------|
| P25 | Rewind never succeeds when target is before finalized head (`ErrRewindOverFinalizedHead`) | `FuzzRewindToTimestamp`, `FuzzComputeRewindTargets` |
| P26 | After successful rewind, unsafe head == target block (verified via FCU head hash) | `FuzzRewindToTimestamp` |
| P27 | After successful rewind, finalized head is unchanged; `finalized <= safe` always holds | `FuzzRewindToTimestamp`, `FuzzComputeRewindTargets` |

### Interop Orchestration (progressInterop / handleResult / Reset)
| ID | Property | Tested By |
|----|----------|-----------|
| P28 | Timestamps are processed strictly sequentially (no gaps, no repeats) | `FuzzProgressInteropValid` |
| P29 | Valid results are committed; invalid results trigger `invalidateBlock` on correct chains only | `FuzzProgressInteropValid`, `FuzzProgressInteropInvalid` |
| P30 | Empty results (no L2Heads) are no-ops — state is not modified | `FuzzHandleResultEmpty` |
| P31 | After invalidation, the interop loop can resume and commit at the same timestamp | `FuzzProgressInteropInvalid` |
| P32 | Reset correctly rewinds both logsDB and verifiedDB; `currentL1` reset to empty; can resume committing | `FuzzProgressInteropReset` |

### Result Type
| ID | Property | Tested By |
|----|----------|-----------|
| P34 | `Result.IsValid() == (len(InvalidHeads) == 0)` | `FuzzResultProperties` |
| P35 | `ToVerifiedResult()` strips invalid heads, preserves all other fields | `FuzzResultProperties` |
| P36 | Empty results correctly detected by `IsEmpty()` | `FuzzResultProperties` |

### Concurrency
| ID | Property | Tested By |
|----|----------|-----------|
| — | Thread safety: parallel Add/Contains never error or lose writes | `FuzzDenyListConcurrent` |
| — | Read-after-write visibility under concurrency | `FuzzDenyListConcurrent` |

---

## 1. DenyList — `chain_container/fuzz_invalidation_test.go`

### `FuzzDenyListAddContains`
**Properties tested:**
- **P21:** `Contains(h, hash)` returns true iff `Add(h, hash)` was previously called
- **P22:** `Add` is idempotent — duplicate adds don't increase hash count
- **P23:** Hashes at different heights are isolated from each other
- **P24:** Concatenated 32-byte hash storage handles boundary alignment correctly

```bash
go test -fuzz=FuzzDenyListAddContains ./op-supernode/supernode/chain_container/ -fuzztime=60s
```

Results:
```
fuzz: elapsed: 30s, execs: 538358 (19036/sec), new interesting: 1 (total: 91)
fuzz: elapsed: 33s, execs: 584012 (15222/sec), new interesting: 2 (total: 92)
fuzz: elapsed: 7m18s, execs: 9178180 (21450/sec), new interesting: 2 (total: 92)
fuzz: elapsed: 8m27s, execs: 10613926 (20912/sec), new interesting: 2 (total: 92)
```

### `FuzzDenyListConcurrent`
**Properties tested:**
- Thread safety: parallel Add/Contains from multiple goroutines never error or lose writes
- Read-after-write visibility: a hash is always found immediately after Add, even under concurrency

```bash
go test -fuzz=FuzzDenyListConcurrent ./op-supernode/supernode/chain_container/ -fuzztime=60s
```

Results:

```
fuzz: elapsed: 27s, execs: 480112 (17583/sec), new interesting: 21 (total: 154)
fuzz: elapsed: 1m24s, execs: 1472021 (16863/sec), new interesting: 22 (total: 155)
fuzz: elapsed: 4h17m57s, execs: 152163737 (9713/sec), new interesting: 66 (total: 68)
fuzz: elapsed: 4h18m0s, execs: 152192768 (9677/sec), new interesting: 67 (total: 69)
fuzz: elapsed: 5h52m9s, execs: 207193909 (9847/sec), new interesting: 67 (total: 69)
```

---

## 2. Engine Controller Rewind — `chain_container/engine_controller/fuzz_rewind_test.go`

### `FuzzRewindToTimestamp`
**Properties tested:**
- **P25:** Rewind never succeeds when target is before finalized head (`ErrRewindOverFinalizedHead`)
- **P26:** After successful rewind, unsafe head == target block (verified via FCU head hash)
- **P27:** After successful rewind, finalized head is unchanged

```bash
go test -fuzz=FuzzRewindToTimestamp ./op-supernode/supernode/chain_container/engine_controller/ -fuzztime=60s
```

fuzz: elapsed: 3s, execs: 24320 (8105/sec), new interesting: 13 (total: 17)
fuzz: elapsed: 6s, execs: 59028 (11567/sec), new interesting: 14 (total: 18)
fuzz: elapsed: 5m0s, execs: 3464235 (0/sec), new interesting: 14 (total: 18)

### `FuzzComputeRewindTargets`
**Properties tested:**
- **P25:** Returns error when target < finalized
- **P27:** Finalized head is always <= target after clamping; finalized <= safe always holds

```bash
go test -fuzz=FuzzComputeRewindTargets ./op-supernode/supernode/chain_container/engine_controller/ -fuzztime=60s
```


fuzz: elapsed: 3s, execs: 35123 (11704/sec), new interesting: 16 (total: 18)
fuzz: elapsed: 5m0s, execs: 4405223 (0/sec), new interesting: 16 (total: 18)


---

## 3. LogsDB Timestamp Verification — `activity/interop/fuzz_logdb_test.go`

### `FuzzVerifyCanAddTimestamp`
**Properties tested:**
- **P9:** Gap violations are always detected (gap > blockTime triggers error)
- **P13:** Non-block-time-aligned gaps only warn, don't error

```bash
go test -fuzz=FuzzVerifyCanAddTimestamp ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 1m0s, execs: 1753747 (28822/sec), new interesting: 1 (total: 22)


### `FuzzProcessBlockLogs`
**Properties tested:**
- **P11:** First block with empty parent hash is accepted exactly once (virtual parent seal handling)
- **P12:** AddLog called once per log, SealBlock called expected number of times, log indices are sequential

```bash
go test -fuzz=FuzzProcessBlockLogs ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 2m0s, execs: 2833828 (23341/sec), new interesting: 2 (total: 44)
fuzz: elapsed: 2m0s, execs: 2833828 (0/sec), new interesting: 2 (total: 44)

---

## 4. Interop Message Verification — `activity/interop/fuzz_algo_test.go`

### `FuzzVerifyInteropMessagesValid`
**Properties tested:**
- **P1:** Valid cross-chain messages never produce `InvalidHeads`
- **P3:** `Result.IsValid()` ↔ `len(InvalidHeads) == 0`

```bash
go test -fuzz=FuzzVerifyInteropMessagesValid ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 2m0s, execs: 2769528 (0/sec), new interesting: 21 (total: 25)

### `FuzzVerifyInteropMessagesFails`
**Properties tested:**
- **P2:** Every invalidation type is correctly detected (unknown source chain, timestamp violation, expired message, message not found/conflict, block hash mismatch)

```bash
go test -fuzz=FuzzVerifyInteropMessagesFails ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 1m0s, execs: 1431913 (28172/sec), new interesting: 1 (total: 21)

### `FuzzVerifyExpiryBoundary`
**Properties tested:**
- **P4:** `execMsg.Timestamp + ExpiryTime` overflow doesn't cause false positive/negative; exact boundary values (at, one past, one before expiry) are handled correctly

```bash
go test -fuzz=FuzzVerifyExpiryBoundary ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzVerifyExpiryOverflow`
**Properties tested:**
- **P4 (overflow):** Demonstrates that `execMsg.Timestamp + ExpiryTime` uint64 overflow causes false expiration of logically valid messages. When `initTS > MaxUint64 - ExpiryTime`, the addition wraps to a small number, triggering `ErrMessageExpired` even though `execTS - initTS < ExpiryTime`. Asserts the **current buggy behavior** — if the production code is fixed, update assertions to expect valid results.

```bash
go test -fuzz=FuzzVerifyExpiryOverflow ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzVerifyFirstBlockSkipped`
**Properties tested:**
- **P5:** First block (`ErrSkipped` path) correctly handles hash match vs mismatch against `FirstSealedBlock`

```bash
go test -fuzz=FuzzVerifyFirstBlockSkipped ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 5m0s, execs: 5076937 (0/sec), new interesting: 11 (total: 14)


### `FuzzVerifyMultipleInvalidMessages`
**Properties tested:**
- **P6:** Block with multiple invalid executing messages is still marked invalid (first-invalid-short-circuits)

```bash
go test -fuzz=FuzzVerifyMultipleInvalidMessages ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 5m0s, execs: 5044294 (0/sec), new interesting: 19 (total: 22)


### `FuzzVerifyMissingChains`
**Properties tested:**
- **P7:** Chains not present in `logsDBs` are silently excluded from `Result.L2Heads`

```bash
go test -fuzz=FuzzVerifyMissingChains ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 5m0s, execs: 7491473 (26008/sec), new interesting: 24 (total: 26)


### `FuzzResultProperties`
**Properties tested:**
- **P34:** `Result.IsValid() == (len(InvalidHeads) == 0)`
- **P35:** `ToVerifiedResult()` strips invalid heads, preserves all other fields
- **P36:** Empty results correctly detected by `IsEmpty()`

```bash
go test -fuzz=FuzzResultProperties ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 5m0s, execs: 4603574 (15442/sec), new interesting: 10 (total: 13)


---

## 5. Interop Orchestration — `activity/interop/fuzz_interop_test.go`

### `FuzzProgressInteropValid`
**Properties tested:**
- **P28:** Timestamps are processed strictly sequentially (no gaps, no repeats)
- **P29:** Valid verification results are committed to VerifiedDB

```bash
go test -fuzz=FuzzProgressInteropValid ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

fuzz: elapsed: 2m36s, execs: 4290 (27/sec), new interesting: 29 (total: 32)
fuzz: elapsed: 2m39s, execs: 4377 (29/sec), new interesting: 30 (total: 33)


### `FuzzProgressInteropInvalid`
**Properties tested:**
- **P29:** Invalid results trigger block invalidation via `invalidateBlock` on the correct chains only
- **P31:** After invalidation, the interop loop can resume and commit at the same timestamp

```bash
go test -fuzz=FuzzProgressInteropInvalid ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzProgressInteropReset`
**Properties tested:**
- **P32:** Reset correctly rewinds both logsDB and verifiedDB; logsDB rewound to `block - 1`, `currentL1` reset to empty, verifiedDB entries after rewind point deleted, can resume committing

```bash
go test -fuzz=FuzzProgressInteropReset ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzHandleResultEmpty`
**Properties tested:**
- **P30:** Empty results (no L2Heads) are no-ops — state is not modified

```bash
go test -fuzz=FuzzHandleResultEmpty ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

---

## 6. VerifiedDB — `activity/interop/fuzz_verified_db_test.go`

### `FuzzVerifiedDBCommitRewind`
**Properties tested:**
- **P15:** `Commit(result)` succeeds iff `result.Timestamp == lastTimestamp + 1` (or first commit)
- **P16:** After `Rewind(ts)`, `LastTimestamp()` returns `ts - 1` (or uninitialized if all deleted)
- **P17:** After `Rewind(ts)`, `Get(t)` errors for all `t >= ts`
- **P18:** After `Rewind(ts)`, `Commit(ts)` succeeds (re-commit from rewind point)
- **P19:** `ErrAlreadyCommitted` and `ErrNonSequential` are correctly distinguished
- **P20:** JSON round-trip preserves all `VerifiedResult` fields

```bash
go test -fuzz=FuzzVerifiedDBCommitRewind ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzVerifiedDBFirstCommit`
**Properties tested:**
- **P15:** First commit succeeds at any timestamp; subsequent must be sequential
- **P18:** First commit after full rewind succeeds at any timestamp

```bash
go test -fuzz=FuzzVerifiedDBFirstCommit ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

### `FuzzVerifiedDBPersistence`
**Properties tested:**
- **P20:** Data survives close/reopen; all fields preserved after persistence round-trip

```bash
go test -fuzz=FuzzVerifiedDBPersistence ./op-supernode/supernode/activity/interop/ -fuzztime=60s
```

---

## Run All Fuzz Targets (kaas)

### DenyList (chain_container)
```bash
kaas go test -fuzz='FuzzDenyListAddContains,FuzzDenyListConcurrent' ./op-supernode/supernode/chain_container/ --fuzztime=5m
```

### Engine Controller Rewind (engine_controller)
```bash
kaas go test -fuzz='FuzzRewindToTimestamp,FuzzComputeRewindTargets' ./op-supernode/supernode/chain_container/engine_controller/ --fuzztime=5m
```

### Interop — all targets (activity/interop)
```bash
kaas go test -fuzz='FuzzVerifyCanAddTimestamp,FuzzProcessBlockLogs,FuzzVerifyInteropMessagesValid,FuzzVerifyInteropMessagesFails,FuzzVerifyExpiryBoundary,FuzzVerifyExpiryOverflow' ./op-supernode/supernode/activity/interop/ --fuzztime=5m
```
Limited to 5 targets

```bash
kaas go test -fuzz='FuzzVerifyFirstBlockSkipped,FuzzVerifyMultipleInvalidMessages,FuzzVerifyMissingChains,FuzzResultProperties,FuzzProgressInteropValid' ./op-supernode/supernode/activity/interop/ --fuzztime=5m

kaas go test -fuzz='FuzzProgressInteropInvalid,FuzzProgressInteropReset,FuzzHandleResultEmpty,FuzzVerifiedDBCommitRewind,FuzzVerifiedDBFirstCommit' ./op-supernode/supernode/activity/interop/ --fuzztime=5m

kaas go test -fuzz='FuzzVerifiedDBPersistence' ./op-supernode/supernode/activity/interop/ --fuzztime=5m
```
