# `kona-engine`

An extensible implementation of the [OP Stack][op-stack] rollup node engine client.

## Overview

The `kona-engine` crate provides a task-based engine client for interacting with Ethereum execution layers. It implements the Engine API specification and manages the execution layer state by executing engine tasks one at a time.

## Key Components

- **[`Engine`](crate::Engine)** - Executes engine tasks atomically and publishes the engine state
- **[`EngineClient`](crate::EngineClient)** - HTTP client for Engine API communication with JWT authentication
- **[`EngineState`](crate::EngineState)** - Tracks the current state of the execution layer
- **Task Types** - Specialized tasks for different engine operations:
  - [`InsertTask`](crate::InsertTask) - Insert new payloads into the execution engine
  - [`BuildTask`](crate::BuildTask) - Build new payloads with automatic forkchoice synchronization
  - [`ConsolidateTask`](crate::ConsolidateTask) - Consolidate unsafe payloads to advance the safe chain
  - [`FinalizeTask`](crate::FinalizeTask) - Finalize safe payloads on L1 confirmation
  - [`SynchronizeTask`](crate::SynchronizeTask) - Internal task for execution layer forkchoice synchronization

## Architecture

The engine implements a task-driven architecture where forkchoice synchronization is handled automatically:

- **Automatic Forkchoice Handling**: The [`BuildTask`] automatically performs forkchoice updates during block building, eliminating the need for explicit forkchoice management in user code.
- **Internal Synchronization**: [`SynchronizeTask`] handles internal execution layer synchronization and is primarily used by other tasks rather than directly by users.
- **Sequential Execution**: Tasks execute one at a time, in the order the caller submits them, with exclusive access to the engine state.

## Engine API Compatibility

The crate supports multiple Engine API versions with automatic version selection based on the rollup configuration:

- **Engine Forkchoice Updated**: V2, V3
- **Engine New Payload**: V2, V3, V4
- **Engine Get Payload**: V2, V3, V4

Version selection follows Optimism hardfork activation times (Bedrock, Canyon, Delta, Ecotone, Isthmus).

## Metrics

Prometheus metrics instrumentation is always enabled.

<!-- Hyper Links -->

[op-stack]: https://specs.optimism.io
