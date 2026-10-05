# Remaining Rust shadow validation

This stage extends the existing optional `optimism-rust-shadow` in the single
[draft PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151).
Circle keeps all required gates. The completed core Rust stage and its original
reports remain documented in [rwx-rust-parity.md](rwx-rust-parity.md).

## Verified workloads

| Circle job | Shared runner mode | Complete workload |
| --- | --- | --- |
| rust-wasm-unknown | wasm-unknown | Four selected packages, wasm32-unknown-unknown, no default features |
| rust-wasm-wasi | wasm-wasi | Three selected packages, wasm32-wasip1, default features |
| rust-zepter | zepter | Pinned `zepter run check` in the Rust workspace |
| rust-typos | typos | Pinned `typos` in the Rust workspace |
| kona-registry-snapshot-check | registry | Fresh superchain regeneration of all three committed snapshots and Git diff |
| interop-deposits-diff | interop | Full Go/Rust activation deposit dumps and byte comparison |

Both providers call the shared runner. Generic Circle job parameters retain
their existing fallbacks. WASM selection and execution are verified against
cargo-hack's original dry plan and live command indices; every selected library
must exist and have an archive header. Reports retain library hashes, source
revision, target, toolchain, manifests, effective commands and original logs.
Library byte reproducibility between host environments is not claimed.

The registry runner cleans only `kona-registry` before building with
`KONA_SYNC_SUPERCHAIN=true`. This forces the build script even when a restored
Cargo target previously used the same environment setting. The original before
and after versions of `chainList.json`, `configs.json` and `depsets.json` are
retained, hashed and compared, together with the original Git diff verdict.

The Interop script retains each original stdout, stderr and process exit when
`CI_INTEROP_REPORT_DIR` is set. Default local behavior is unchanged. The shared
runner builds the Go superchain bundle and runs the existing full comparison.
Evidence validation also rejects empty dumps, missing activation variants,
missing deposit fields, missing transaction indices and dumper failures.

RWX uses separate compiler/target caches for these jobs and includes run and
attempt identity in every fresh verdict's cache key. WASM standard libraries,
source-check tools and the Interop Go toolchain are reusable preparation layers.
Verdict reports remain artifact outputs, excluded from reusable compiler layers.
Protected develop warming adds only WASM builds and registry regeneration;
it does not execute unit, offline, Interop or documentation tests.

## Validation checkpoint

The new reporting checks and Circle adapter fixtures passed locally. The pinned
Linux fixture exercised the real shared shell runner, archived fresh tests,
original failure evidence and compiler cache preparation. ShellCheck, RWX lint,
merged Circle config validation and activated workflow processing passed.

At benchmark revision `a9df2def8a23e26b3e00bc4044ed009dd72f8cd7`, all six
workloads passed on [native RWX](https://cloud.rwx.com/optimism/runs/a0011b4866814245821bd9bb1d5e9028)
and [Circle pipeline 135548](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135548).
Complete original-report comparison verifies the same 76 workspace packages,
settings, pinned tools, input hashes, commands, working directories and outcomes.
WASM executes exactly four and three selected package builds respectively.
All three regenerated registry snapshots match committed inputs on each provider
and their original file hashes agree between providers. Both original Interop
dumps agree byte for byte across providers, covering both activation variants,
58 deposits and their two gas records. Every fresh native task executed on
attempt one, with zero observed retries.

The [immutable evidence index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-rust-evidence/extra-parity.json) retains complete
coverage, original-file hashes, commands and artifact provenance. Complete
provider reports are also retained under `.ci/rwx-rust-stage-evidence/a9df/`.
Circle omits empty log/stderr artifacts; the index records each explicitly
allowed omission with its original SHA256(empty). No nonempty original may be
missing or corrupt. WASM library archives have different byte hashes between
host environments; this stage compares build inputs, full command coverage and
successful library production and does not claim host archive reproducibility.
Original JUnit reports match byte for byte for all six jobs.

The checked-in comparison helper rejects missing stages/inputs, checksum damage,
stale revisions, mismatched package manifests/settings/commands, differing
coverage and unresolved task retries. Its eight failure-oriented tests pass on
both local and pinned Linux runtimes. Existing helper/config checks remain green.

The [unchanged-input cache repeat](https://cloud.rwx.com/optimism/runs/1f389c2059204932a401ccc1c2e79bbc)
restored the same Rust source fingerprint and executed both tasks freshly with
new report identities. All four WASM builds reused their compiled targets;
registry regeneration still compiled its build script after the crate-only clean.
Both coverage reports remain identical. The [cache evidence index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-rust-evidence/extra-cache-reuse.json)
retains original hashes, task identities and source/cache records. These are
functional cache checks, not comparable pipeline speed measurements.

Inventory coverage is now **35/86 = 41%**, including **18/22** Rust workflow jobs. Cannon Docker
lint/build/offline execution, the optional Rust gate equivalent, and Rust E2E
remain the next work. No new performance matrix is part of this stage.

The Cannon port must inspect the final guest verdict: `cannon run` writes its
final state and can return CLI success even if the guest's exit code is nonzero.
Retain the original offline recipe and additionally record `cannon witness` for
its freshly produced final state, rejecting a guest that did not exit with code
zero. This is required for honest verdict evidence, not a replacement workload.
