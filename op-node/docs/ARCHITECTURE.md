# OP Node Architecture Documentation

## Overview
The op-node is the consensus-layer client for Optimism rollups. It derives L2 block inputs from L1 data and drives an external L2 Execution Engine to build an L2 chain.

## Key Components

### Configuration System
The configuration system (`op-node/config`) handles all node configuration including:
- L1 endpoint configuration
- L2 endpoint configuration
- RPC server configuration
- Metrics configuration
- P2P configuration
- Driver configuration

### Node Lifecycle
The node lifecycle (`op-node/node`) manages:
- Initialization
- Starting services
- Graceful shutdown
- Health checks

### Rollup Driver
The rollup driver (`op-node/rollup/driver`) handles:
- Deriving L2 blocks from L1 data
- Sequencing blocks (if sequencer)
- Verifying blocks (if verifier)
- Syncing with L1

### Metrics
The metrics system (`op-node/metrics`) provides:
- Prometheus metrics
- Performance monitoring
- Health status tracking
- Error rate monitoring

### Error Handling
The error handling system (`op-node/errors`) provides:
- Structured error context
- Error logging with context
- Error wrapping utilities

## Architecture Flow

```
L1 Chain → op-node → L2 Execution Engine
     ↓
  P2P Network
     ↓
  Metrics/Monitoring
```

## Configuration Validation
Configuration is validated on startup using the validation utilities in `op-node/config/validation.go`.

## Retry Logic
L1/L2 operations use retry logic with exponential backoff from `op-node/retry`.

