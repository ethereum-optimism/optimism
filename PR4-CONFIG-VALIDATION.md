# PR4: Robust Configuration Validation

## Summary
This PR adds comprehensive configuration validation with descriptive error messages to prevent invalid configurations from causing runtime errors.

## Changes
- Added `validation.go` with `ValidateConfig` function
- Implemented `ValidationError` struct for structured error reporting
- Added validation for L1 endpoint configuration
- Added validation for L2 endpoint configuration
- Added validation for RPC configuration
- Added validation for metrics configuration
- All validations provide descriptive error messages with field names and values

## Benefits
- Prevents runtime errors from invalid configurations
- Better error messages for debugging configuration issues
- Early detection of configuration problems
- Improved developer experience

## Files Changed
- `op-node/config/validation.go` (new file)

