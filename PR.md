# PR1: Flashblocks Runtime Constructor (No Orchestrator Path)

## Summary

This PR makes `presets.NewSingleChainWithFlashblocks(...)` run through a real constructor DAG instead of the legacy orchestrator/system wiring path.

It is the first concrete preset migration where:

- the preset no longer calls `DefaultSingleChainSystemWithFlashblocks()`,
- construction happens via direct hierarchical constructor calls in `sysgo`, and
- acceptance tests still consume the same preset API surface.

## What Changed

### 1) New runtime for flashblocks

Added:

- `op-devstack/sysgo/flashblocks_runtime.go`

This runtime now constructs and boots the flashblocks test target directly:

1. build L1/L2 intent world,
2. start L1 (EL + fake beacon CL),
3. start sequencer EL,
4. start builder EL (`op-rbuilder`),
5. wire EL P2P peering,
6. start rollup-boost,
7. start sequencer CL (`op-node`),
8. start faucet service for L1 and L2.

The runtime exports topology + endpoint data needed by presets (L1/L2 configs, deployment, node RPCs, flashblocks WS URLs, faucet endpoints).

### 2) `NewSingleChainWithFlashblocks` now uses the runtime

Updated:

- `op-devstack/presets/flashblocks.go`

Changes:

- `NewSingleChainWithFlashblocks` now instantiates `sysgo.NewFlashblocksRuntime(...)`.
- It assembles DSL/shim frontends directly from runtime references.
- It no longer routes through orchestrator/system constructor chains.
- It rejects orchestrator options for this preset (`opts` must be empty).

### 3) Removed dead flashblocks orchestrator adapter

Updated:

- `op-devstack/presets/sysgo_runtime.go`

Removed:

- `singleChainWithFlashblocksRuntime` type
- `singleChainWithFlashblocksRuntimeFromOrchestrator(...)`

This deletes the now-unused preset-specific flashblocks orchestrator hydration path.

### 4) Added runtime test-sequencer startup

The runtime now starts an in-process test-sequencer service directly (no orchestrator path), configures L1 + L2 sequencing backends, and exports:

- admin RPC endpoint,
- JWT secret,
- per-chain control RPC endpoints.

The preset wires this into `dsl.TestSequencer` via the existing frontend constructor.

### 5) Flashblocks tests are back to strict test-sequencer usage

Updated:

- `op-acceptance-tests/tests/flashblocks/flashblocks_stream_test.go`

`driveViaTestSequencer(...)` now requires the test-sequencer to exist again (fallback removed), so the test behavior matches the prior deterministic sequencing model.

## Validation

Executed:

- `go test ./op-devstack/sysgo -run '^$'`
- `go test ./op-devstack/presets -run '^$'`
- `go test ./op-acceptance-tests/tests/flashblocks -count=1`

All passed.

## PR2 Proposal

1. Move shared constructor primitives into a dedicated package (e.g. runtime builders for L1/L2/faucet/sequencer services).
2. Migrate next preset(s) to runtime assembly (`minimal`, then `base/conductor` path).
3. Start deleting flashblocks legacy sysgo constructor plumbing in `system.go` once no call-sites remain.
