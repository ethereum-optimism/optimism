# PR8: Health Check and Graceful Shutdown

## Summary
This PR implements health check utilities and improved graceful shutdown for op-node.

## Changes
- Added `health` package with health check utilities
- Implemented `HealthRegistry` for managing component health
- Added `HealthChecker` interface for component health checks
- Added `GracefulShutdown` for graceful shutdown management
- Added health status tracking (healthy, unhealthy, degraded)

## Benefits
- Better monitoring of component health
- Improved graceful shutdown handling
- Better observability
- Production-ready health checking

## Files Changed
- `op-node/health/health_check.go` (new file)

