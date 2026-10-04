# Remaining Rust shadow validation

This stage extends the existing optional `optimism-rust-shadow` in the single
[draft PR #23151](https://github.com/ethereum-optimism/optimism/pull/23151).
Circle keeps all required gates. The completed core Rust stage and its original
reports remain documented in [rwx-rust-parity.md](rwx-rust-parity.md).

## Implementation awaiting hosted validation

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

Hosted execution, original same-SHA comparison, and unchanged-input cache reuse
are still pending. These six jobs do **not** yet increase the verified inventory.
Cannon Docker lint/build/offline execution, the optional Rust gate equivalent,
and Rust E2E remain the next work. No new performance matrix is part of this
stage.
