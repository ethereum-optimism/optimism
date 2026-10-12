# `kona-engine`

An extensible implementation of the [OP Stack][op-stack] rollup node engine client.

## Overview

The `kona-engine` crate provides an engine client for interacting with Ethereum execution layers. It implements the Engine API specification and manages the execution layer state by performing engine operations one at a time.

## Key Components

- **[`Engine`](crate::Engine)** - Performs engine operations atomically and publishes the engine state:
  - [`insert_unsafe`](crate::Engine::insert_unsafe) - Insert an unsafe payload from gossip or the admin API
  - [`build`](crate::Engine::build) and [`seal`](crate::Engine::seal) - Build a block and make it the unsafe head
  - [`consolidate`](crate::Engine::consolidate) - Advance the safe chain, reorging the unsafe chain if needed
  - [`finalize`](crate::Engine::finalize) - Finalize a safe block on L1 confirmation
  - [`reset`](crate::Engine::reset) - Find a sync starting point and reorg the execution layer to it
- **[`EngineClient`](crate::EngineClient)** - HTTP client for Engine API communication with JWT authentication
- **[`EngineState`](crate::EngineState)** - Tracks the current state of the execution layer
- **Error types** - One per operation, each with an [`EngineTaskErrorSeverity`](crate::EngineTaskErrorSeverity) that tells the caller how to recover

## Architecture

The engine handles forkchoice synchronization automatically:

- **Automatic Forkchoice Handling**: Each operation performs the forkchoice updates it needs, eliminating the need for explicit forkchoice management in user code.
- **Sequential Execution**: Operations run one at a time, in the order the caller performs them, with exclusive access to the engine state.

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
