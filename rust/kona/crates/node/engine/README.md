# `kona-engine`

An extensible implementation of the [OP Stack][op-stack] rollup node engine client.

## Overview

The `kona-engine` crate provides the building blocks a rollup node uses to drive an execution layer through the Engine API. The engine actor in `kona-node-actors` performs the engine operations with them, through the `EngineApi` in `kona-sources`.

## Key Components

- **[`find_starting_forkchoice`]** - Finds a plausible forkchoice to sync from
- **[`AttributesMatch`]** - Checks whether a block matches derived payload attributes
- **[`ImportedBlockSink`]** - Receives every block the engine makes canonical

## Metrics

Prometheus metrics instrumentation is always enabled.

<!-- Hyper Links -->

[op-stack]: https://specs.optimism.io
