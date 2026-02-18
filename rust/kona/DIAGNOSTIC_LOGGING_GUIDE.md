# Diagnostic Logging Guide - Safe Head Stall

## What Was Added

This build contains **verbose diagnostic logging ONLY** - no functional changes. It will help confirm the infinite retry bug in production.

## Changes Made

### 1. ConsolidateTask Block Fetch Logging
**File**: `engine/src/task_queue/tasks/consolidate/task.rs`

**Added logging for:**
- Block fetch attempts
- Successful fetches (with duration)
- Failed fetches (with error details)
- Missing blocks

### 2. Engine Drain Retry Logging
**File**: `engine/src/task_queue/core.rs`

**Added logging for:**
- Task execution attempts
- Successful task completions (task removed from queue)
- **Failed tasks that will retry** (task NOT removed from queue)
- Error severity classification

### 3. **Queue Inspection Logging** (NEW)
**File**: `engine/src/task_queue/core.rs`

**Added detailed queue state logging:**
- **Queue snapshot at drain start**: Shows first 5 tasks in queue
- **Queue contents on retry**: Shows front 3 tasks when task fails
- **Task enqueue events**: Logs when tasks are added to queue
- **High queue warning**: Alert when queue length > 5 (tasks backing up)

This will show:
- ✅ What task is stuck at position 0
- ✅ If the SAME task appears repeatedly
- ✅ If other tasks are piling up behind it
- ✅ Queue growth over time

### 4. Error Severity Comment
**File**: `engine/src/task_queue/tasks/consolidate/error.rs`

**Added comment** explaining that `FailedToFetchUnsafeL2Block` returns `Temporary` severity, causing infinite retries.

## What to Look For in Logs

### Confirming the Infinite Retry Bug

When the stall occurs, you should see this pattern:

```
DEBUG engine::drain::queue queue_len=1 queue_contents=["Consolidate(ConsolidateTask { ... block_num: 42222304 ... })"] "DIAGNOSTIC: Queue state at drain start"

DEBUG engine::consolidate block_num=42222304 "Attempting to fetch L2 block for consolidation"
WARN engine::consolidate block_num=42222304 error=... "Failed to fetch unsafe l2 block for consolidation (DIAGNOSTIC: This will cause retry)"

WARN engine::drain task_type="Consolidate" severity="Temporary" queue_len=1 queue_front_tasks=["Consolidate(ConsolidateTask { ... block_num: 42222304 ... })"] "Task failed - will be retried on next drain (DIAGNOSTIC: Task NOT removed from queue)"

[2 seconds later - SAME TASK AGAIN...]

DEBUG engine::drain::queue queue_len=1 queue_contents=["Consolidate(ConsolidateTask { ... block_num: 42222304 ... })"] "DIAGNOSTIC: Queue state at drain start"

DEBUG engine::consolidate block_num=42222304 "Attempting to fetch L2 block for consolidation"
WARN engine::consolidate block_num=42222304 "Failed to fetch unsafe l2 block for consolidation (DIAGNOSTIC: This will cause retry)"

WARN engine::drain queue_front_tasks=["Consolidate(ConsolidateTask { ... block_num: 42222304 ... })"] "Task failed - will be retried on next drain (DIAGNOSTIC: Task NOT removed from queue)"

[Repeats infinitely with SAME task...]
```

**Key observation**: The `queue_contents` and `queue_front_tasks` arrays show the **SAME ConsolidateTask** with the **SAME block_num** every time!

### Key Indicators

**✅ Bug Confirmed If You See:**

1. **Same block number** logged repeatedly (100s or 1000s of times)
   ```
   grep "block_num=42222304" kona.log | wc -l
   # Should be very high (100+, 1000+)
   ```

2. **"Task NOT removed from queue"** appearing repeatedly
   ```
   grep "Task NOT removed from queue" kona.log | tail -20
   ```

3. **`queue_len=1`** consistently (task stuck in queue)
   ```
   grep "queue_len=1" kona.log | wc -l
   ```

4. **`severity="Temporary"`** on every failure
   ```
   grep 'severity="Temporary"' kona.log | tail -10
   ```

5. **No "Task executed successfully"** for the stuck block
   ```
   grep -A 1 "block_num=42222304" kona.log | grep "successfully"
   # Should be empty
   ```

## Quick Diagnostic Commands

### 1. Show Queue Contents Over Time
```bash
grep "queue_contents" kona.log | tail -20
```

**Expected**: SAME task appearing repeatedly in position 0

### 2. Count Retry Attempts for Stuck Block
```bash
STUCK_BLOCK=42222304  # Replace with your stuck block
grep "block_num=$STUCK_BLOCK" kona.log | wc -l
```

**Expected**: 10+ (proof of retries)

### 3. Extract Unique Tasks in Queue
```bash
grep "queue_front_tasks" kona.log | sed 's/.*queue_front_tasks=\[\(.*\)\].*/\1/' | sort | uniq -c
```

**Expected**: One task with very high count (the stuck task)

### 4. Show Retry Pattern
```bash
grep "DIAGNOSTIC" kona.log | tail -30
```

**Expected**: Alternating fetch attempt → failure → retry messages

### 5. Check Task Queue State
```bash
grep "queue_len=" kona.log | tail -20
```

**Expected**: Consistently showing `queue_len=1`

### 6. Monitor Queue Growth
```bash
grep "queue_len=" kona.log | awk '{print $NF}' | tail -50
```

**Expected**: If tasks are backing up, you'll see: 1, 1, 1, 2, 2, 3, 4 (growing)

### 7. Check for High Queue Warnings
```bash
grep "Queue length is high" kona.log
```

**Expected**: Warnings if other tasks pile up behind stuck task

### 4. Verify Block Exists in EL
```bash
cast block $STUCK_BLOCK --rpc-url <your-l2-rpc>
```

**Expected**: Block EXISTS (proving it's not an EL sync issue)

### 5. Monitor Real-Time
```bash
tail -f kona.log | grep --line-buffered "DIAGNOSTIC\|consolidate"
```

**Expected**: Continuous stream of retry attempts for same block

## Success Criteria

The bug is **definitively confirmed** if:

1. ✅ Logs show 50+ retry attempts for the same block
2. ✅ Block exists in EL but fetch keeps failing
3. ✅ Task stays at `queue_len=1` throughout
4. ✅ All failures show `severity="Temporary"`
5. ✅ Safe head is not advancing (check via RPC)
6. ✅ **Queue snapshots show SAME ConsolidateTask repeatedly**
7. ✅ **`queue_front_tasks` array unchanged across retries**

### The Smoking Gun: Queue Evidence

The queue logs provide **irrefutable proof**:

```bash
# Extract queue snapshots
grep "queue_contents\|queue_front_tasks" kona.log > queue-snapshots.log

# You should see the EXACT SAME task at position 0, over and over:
# Consolidate(ConsolidateTask { input: BlockInfo(L2BlockInfo { block_info: { number: 42222304 ... } }) })
# Consolidate(ConsolidateTask { input: BlockInfo(L2BlockInfo { block_info: { number: 42222304 ... } }) })
# Consolidate(ConsolidateTask { input: BlockInfo(L2BlockInfo { block_info: { number: 42222304 ... } }) })
# ... (repeated 100s or 1000s of times)
```

**This proves:**
- ❌ Task is NEVER removed from queue (stuck at position 0)
- ❌ Same block_num appears in every snapshot
- ❌ Queue processing is completely blocked
- ✅ Temporary error severity prevents task removal
- ✅ No retry limit exists (would be capped at 10 with the fix)

## Evidence Collection

Save these logs for the fix PR:

```bash
# Extract stall evidence
STUCK_BLOCK=<your-stuck-block>
grep -A 2 -B 2 "block_num=$STUCK_BLOCK" kona.log > stall-evidence.log

# Count retries
echo "Total retry attempts: $(grep "block_num=$STUCK_BLOCK" kona.log | wc -l)" >> stall-evidence.log

# Add RPC proof
echo "\nBlock exists in EL:" >> stall-evidence.log
cast block $STUCK_BLOCK --rpc-url <your-l2-rpc> >> stall-evidence.log
```

## After Confirmation

Once you confirm the bug with these logs:

1. **Restart the node** to clear the stuck task
2. **Deploy the fix** (with retry limit) to prevent future occurrences
3. **Monitor for** "Task exceeded max retries" (shows fix working)

## Comparing Before/After

**Before Fix (This Build):**
- Infinite retries
- Manual restart required
- Safe head stuck for hours

**After Fix:**
- 10 retries → automatic reset
- No manual intervention
- Recovery in ~30 seconds

---

**Log Levels**: Set `RUST_LOG=debug,engine::drain=warn,engine::consolidate=warn` for optimal verbosity.
