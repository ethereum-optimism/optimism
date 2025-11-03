# PR7: Context and Timeout Management

## Summary
This PR improves context and timeout management throughout op-node for better resource management.

## Changes
- Added `context` package with timeout utilities
- Implemented `TimeoutConfig` for configurable timeouts
- Added `WithL1Timeout` for L1 operations
- Added `WithL2Timeout` for L2 operations
- Added `WithConfigReloadTimeout` for config reload
- Added automatic timeout logging

## Benefits
- Better resource management
- Prevention of resource leaks
- Better timeout handling
- Improved reliability

## Files Changed
- `op-node/context/timeout_utils.go` (new file)

