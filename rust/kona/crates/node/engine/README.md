# `kona-engine`

An extensible implementation of the [OP Stack][op-stack] rollup node engine client.

## Overview

The `kona-engine` crate provides the building blocks a rollup node uses to drive an execution layer through the Engine API. The engine actor in `kona-node-actors` performs the engine operations with them.

## Key Components

- **[`EngineClient`](crate::EngineClient)** - HTTP client for Engine API communication with JWT authentication
- **[`EngineState`](crate::EngineState)** - Tracks the current state of the execution layer
- **[`find_starting_forkchoice`](crate::find_starting_forkchoice)** - Finds a plausible forkchoice to sync from
- **[`AttributesMatch`](crate::AttributesMatch)** - Checks whether a block matches derived payload attributes
- **[`ImportedBlockSink`](crate::ImportedBlockSink)** - Receives every block the engine makes canonical

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
