# PR2: Add Unit Tests for Critical Functions

## Summary
This PR adds comprehensive unit tests for critical op-node functions that previously lacked test coverage, improving code reliability and maintainability.

## Changes
- Added unit tests for `NewRollupConfig` function covering:
  - Loading config from known networks
  - Loading config from JSON file
  - Error handling for invalid files
  - Error handling for invalid JSON
  - Network and file conflict scenarios
- Added unit tests for `NewL1EndpointConfig` function
- Added unit tests for `NewL2EndpointConfig` function
- Added unit tests for `NewDriverConfig` function
- Added unit tests for `NewConfigPersistence` function

## Benefits
- Improved test coverage for critical configuration functions
- Better regression prevention
- Easier refactoring with confidence
- Documentation through test cases

## Files Changed
- `op-node/service_test.go` (new file)

