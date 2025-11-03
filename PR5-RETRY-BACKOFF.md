# PR5: Retry System with Exponential Backoff

## Summary
This PR implements a retry system with exponential backoff for L1/L2 operations to improve resilience and handle transient failures.

## Changes
- Added `retry` package with exponential backoff implementation
- Implemented `RetryWithBackoff` function for retrying operations
- Added `RetryL1Operation` with specific retry configuration for L1
- Added `RetryL2Operation` with specific retry configuration for L2
- Configurable retry parameters (max attempts, delays, multiplier)

## Benefits
- Better handling of transient network failures
- Improved resilience for L1/L2 operations
- Configurable retry behavior
- Better error recovery

## Files Changed
- `op-node/retry/retry_l1_l2.go` (new file)

