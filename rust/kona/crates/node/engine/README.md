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

The [`EngineClient`](crate::EngineClient) supports multiple Engine API versions:

- **Engine Forkchoice Updated**: V2 and V3, selected by the payload attributes' timestamp (V3 from Ecotone; V3 without attributes)
- **Engine Get Payload**: V2 to V5, selected by the payload's timestamp (V3 from Ecotone, V4 from Isthmus, V5 from Karst)
- **Engine New Payload**: V1 to V4, selected by the payload envelope's version

## Metrics

Prometheus metrics instrumentation is always enabled.

<!-- Hyper Links -->

[op-stack]: https://specs.optimism.io
