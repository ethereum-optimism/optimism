# `kona-metrics`

<a href="https://github.com/ethereum-optimism/optimism/actions/workflows/rust_ci.yaml"><img src="https://github.com/ethereum-optimism/optimism/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://github.com/ethereum-optimism/optimism/blob/develop/LICENSE-MIT"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="MIT License"></a>

The chain dimension for kona's metrics.

kona resolves every metric through the process-wide `metrics` recorder, so a metric's chain is not
knowable at the emit site. This crate carries the chain in an ambient scope and adds it as a label
in a wrapping `Recorder`, so no emit site needs to know about it.

- `scoped` and `sync_scoped` bind a chain to the work that emits its metrics.
- `ChainLabelRecorder` wraps the real recorder and reads that binding on every emit.

The binding is a `tokio::task_local`, so it survives `.await` and thread migration but is not
inherited by a spawned task. Every task that emits kona metrics needs its own scope.
