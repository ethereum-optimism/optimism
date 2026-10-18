# `kona-engine`

Engine API operations and reset traversal for the [OP Stack](https://specs.optimism.io).

`EngineClient` supplies the configured L1/L2 RPC connections. Ordinary functions build,
retrieve, insert, consolidate, finalize and synchronize execution payloads, updating
`EngineState` only for accepted changes. Consolidation reports whether derived attributes
applied, required a deposits-only replacement, or were dropped.

`find_starting_forkchoice` takes an explicit `SyncStartPolicy`: ordinary resets retain the
owner's completed-safe/finalized checkpoint, while initialization alone may use its documented
once-only trusted resume policy. Safe selection returns a validated epoch parent. The
`synchronize` result distinguishes current `VALID` acceptance from a `SYNCING` response
whose target the owner must retain and retry;
historical sync completion does not establish current readiness. Scheduling, bounded admission,
retries, reset epochs and producer acknowledgements belong to the engine actor in
`kona-node-service`.

Engine API versions follow the injected rollup configuration. The `metrics` feature enables
Prometheus metrics; `test-utils` supplies shared engine fixtures and a strict RPC script.
