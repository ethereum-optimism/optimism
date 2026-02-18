# Safe Head Stall Fix

## Problem

When the `ConsolidateTask` fails to fetch an L2 block from the execution layer, it returns `FailedToFetchUnsafeL2Block` with `Temporary` severity. The engine's drain logic keeps temporary-errored tasks in the queue for retry, causing an **infinite retry loop** that stalls safe head progress.

### Root Cause

1. Derivation derives safe head from L1 → enqueues `ConsolidateTask`
2. `ConsolidateTask` tries to fetch L2 block → fetch fails (block doesn't exist / RPC issue)
3. Returns `FailedToFetchUnsafeL2Block` (Temporary severity)
4. Engine drain does NOT pop task → retries infinitely
5. Safe head never advances → **stall**

## Test Case

See `task_queue/tests/consolidate_stall_test.rs`:

```rust
#[tokio::test]
async fn test_consolidate_stall_on_missing_block() {
    let mock_client = Arc::new(MockEngineClient::new_with_failing_block_fetch());
    let mut engine = Engine::new(/* ... */);

    // Enqueue ConsolidateTask
    engine.enqueue(consolidate_task);

    // Drain 10+ times - task stays in queue
    for _ in 0..10 {
        assert!(engine.drain().await.is_err());
        assert_eq!(queue_length, 1); // ❌ Still in queue!
    }
}
```

## Solution

Add retry limit to the engine drain logic:

### Changes

**1. Track retry count** (`engine/src/task_queue/core.rs`):
```rust
pub struct Engine<EngineClient_: EngineClient> {
    // ...
    task_retry_count: usize,  // Tracks consecutive retries for front task
}

const MAX_TASK_RETRIES: usize = 10;
```

**2. Escalate after max retries** (`engine/src/task_queue/core.rs:drain()`):
```rust
match task.execute(&mut self.state).await {
    Ok(_) => {
        self.task_retry_count = 0;  // Reset on success
        self.tasks.pop();
    }
    Err(e) if e.severity() == Temporary => {
        self.task_retry_count += 1;

        if self.task_retry_count >= MAX_TASK_RETRIES {
            warn!("Task exceeded max retries, triggering reset");
            self.task_retry_count = 0;
            self.tasks.pop();  // ✅ Remove stuck task
            return Err(MaxRetriesExceeded);  // Trigger engine reset
        }

        return Err(e);  // Retry
    }
}
```

**3. Add new error variant** (`task_queue/tasks/synchronize/error.rs`):
```rust
pub enum SynchronizeTaskError {
    // ...
    #[error("Task exceeded max retries ({retry_count}): {original_error}")]
    MaxRetriesExceeded {
        original_error: String,
        retry_count: usize,
    },
}

impl EngineTaskError for SynchronizeTaskError {
    fn severity(&self) -> EngineTaskErrorSeverity {
        match self {
            Self::MaxRetriesExceeded { .. } => EngineTaskErrorSeverity::Reset,
            // ...
        }
    }
}
```

## Behavior After Fix

1. `ConsolidateTask` fails to fetch block
2. Retries up to 10 times (configurable via `MAX_TASK_RETRIES`)
3. After 10 retries:
   - Task is **popped from queue** (no longer blocking)
   - `MaxRetriesExceeded` error returned with `Reset` severity
   - Engine reset triggered → finds new sync starting point
   - Node recovers and continues syncing

## Comparison with op-node

| Aspect | op-node (Go) | Kona (before fix) | Kona (after fix) |
|--------|-------------|-------------------|------------------|
| Block fetch | No proactive fetch | Fetches before FCU | Fetches before FCU |
| FCU on missing block | Issues FCU → EL returns SYNCING | Fails with temporary error | Fails with temporary error |
| Retry behavior | Event-driven, no explicit retries | Infinite retries | Max 10 retries → reset |
| Stall prevention | N/A | ❌ None | ✅ Retry limit |

## Testing

```bash
# Run the test
cd rust/kona
cargo nextest run --package kona-engine --test consolidate_stall_test

# Or build and check
just build-native
just lint-native
```

## Metrics to Monitor

After deploying this fix, monitor:
- `kona_engine_reset_count` - Should increase when retry limit hit
- `kona_engine_task_queue_length` - Should not stay at 1+ indefinitely
- Safe head progress - Should advance consistently

## Alternative Approaches Considered

1. **Skip block fetch, just issue FCU** (like op-node)
   - Simpler, but loses validation of block existence
   - Would require larger architectural change

2. **Make `FailedToFetchUnsafeL2Block` a Reset error immediately**
   - Too aggressive - genuine transient RPC errors would trigger unnecessary resets

3. **Exponential backoff between retries**
   - Doesn't solve the fundamental problem of infinite retries
   - Only delays the stall

## Files Modified

- `engine/src/task_queue/core.rs` - Add retry tracking to `Engine`
- `engine/src/task_queue/tasks/synchronize/error.rs` - Add `MaxRetriesExceeded` variant
- `engine/src/task_queue/tests/consolidate_stall_test.rs` - Test case (new file)
