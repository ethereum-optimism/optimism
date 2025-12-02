# `kona-engine`

<a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/kona-engine"><img src="https://img.shields.io/crates/v/kona-engine.svg?label=kona-engine&labelColor=2a2f35" alt="Kona Engine"></a>
<a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://img.shields.io/codecov/c/github/op-rs/kona"><img src="https://img.shields.io/codecov/c/github/op-rs/kona" alt="Codecov"></a>

An extensible implementation of the [OP Stack][op-stack] rollup node engine client.

## Overview

The `kona-engine` crate provides a task-based engine client for interacting with Ethereum execution layers. It implements the Engine API specification and manages the execution layer state through a priority-driven task queue system.

## Key Components

- **[`Engine`](crate::Engine)** - Main task queue processor that executes engine operations atomically
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

- **Automatic Forkchoice Handling**: The [`BuildTask`](crate::BuildTask) automatically performs forkchoice updates during block building, eliminating the need for explicit forkchoice management in user code.
- **Internal Synchronization**: [`SynchronizeTask`](crate::SynchronizeTask) handles internal execution layer synchronization and is primarily used by other tasks rather than directly by users.
- **Priority-Based Execution**: Tasks are executed in priority order to ensure optimal sequencer performance and block processing efficiency.

## Engine API Compatibility

The crate supports multiple Engine API versions with automatic version selection based on the rollup configuration:

- **Engine Forkchoice Updated**: V2, V3
- **Engine New Payload**: V2, V3, V4
- **Engine Get Payload**: V2, V3, V4

Version selection follows Optimism hardfork activation times (Bedrock, Canyon, Delta, Ecotone, Isthmus).

## Features

- `metrics` - Enable Prometheus metrics collection (optional)

<!-- Hyper Links -->

[op-stack]: https://specs.optimism.io
