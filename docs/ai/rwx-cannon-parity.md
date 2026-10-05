# Cannon Rust shadow

The three remaining Cannon workloads use the shared
`ops/ci/rust-cannon.sh` adapter on both Circle and RWX:
`kona-lint-cannon`, `kona-build-fpvm-cannon-client`, and
`kona-host-client-offline-cannon`. All three pass complete same-SHA hosted parity
and are checked in the [inventory](rwx-parity-todos.md), bringing implementation
coverage to **38/86 = 44%**, including **21/22 Rust workflow jobs**.

The original Just recipes remain authoritative. Variant discovery retains all
configured clients; lint retains its MIPS target, nightly toolchain and feature
flags; release compilation retains the `release-client-lto` profile. Offline
execution retains the complete pinned OP Sepolia witness and all six original
block/chain/output-root inputs in `ops/ci/cannon-witness.json`.

RWX prepares the reproducible Docker environment, witness archive and native
Cannon binaries as separate dependencies. It preserves Docker/BuildKit data
through `docker: preserve-data` and keeps isolated host Cargo, Go and sccache
compiler caches. Verdict tasks include run and attempt identities; runtime
states, witness extraction directories, logs and reports are excluded from
reusable filesystem outputs. Protected cache warming builds dependencies and
ELFs without executing the offline guest.

Each dependency records its source revision, complete input hashes, toolchain,
variant inventory, image/base digests and binary hashes. Consumers reject stale,
corrupt, incomplete or mismatched dependencies. The shared adapter removes old
runtime states before replay and checks the original final `cannon witness`
output: the VM must have exited with code zero, with the expected state version,
a nonempty execution and the original output-root validation log. A successful
Cannon CLI exit alone is insufficient to establish a successful guest.

Original stage commands, working directories, exits, logs, MIPS ELFs, compressed
final state and JUnit are retained after success, failure and cancellation.
Offline JUnit reports one executed guest; compiling the Interop client does not
claim an Interop replay.

The Cargo source fingerprint now covers `op-core/nuts/bundles`, which Kona
embeds from outside the Rust workspace. Changed contents receive fresh mtimes
before native builds and Docker COPY; unchanged inputs restore stable mtimes.
Real Cargo fixtures reproduce stale embedded data, verify rebuilding changed
inputs and verify subsequent unchanged-target reuse. Helper fixtures also cover
missing witness data, failed discovery, missing/corrupt binaries, mismatched
producer provenance, a failing VM behind a successful CLI exit, original failure
collection and process cancellation.

At benchmark revision `2c68e86760cbf76fa9cfb78f58aee5e130ef45a7`, all three pass on
[native RWX](https://cloud.rwx.com/optimism/runs/2e7238e8499d43eda927b1924b579415)
and [Circle pipeline 135550](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135550).
Circle jobs are lint 5633371, build 5633369 and offline replay 5633368. Complete
original reports agree on source/input hashes, settings, toolchains, variant
discovery, effective commands, working directories, outcomes and JUnit. Both
image and runtime MIPS binaries match byte for byte for every selected variant.
The offline guest exits successfully after **1,402,053,869 steps**. Its complete
original witness, witness hash and full final state match between providers;
the compressed final-state hashes also agree. Native Go binaries retain both
host hash sets; host debug paths may differ. Docker image IDs differ, while base
digests, toolchains, inputs and ELF outputs agree.

The [immutable parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-rust-evidence/cannon-parity.json) retains complete
original-file checksums, commands, producer bindings and guest-state evidence.
Original archives are retained under `.ci/rwx-cannon-stage-evidence/2c68/` and in
the provider artifacts. Circle omits only its zero-byte `guest.log`; its original
manifest declares SHA256(empty). Missing nonempty originals fail comparison.
All verdicts execute on attempt one, with zero observed retries.

Thirteen Cannon helper tests, eight target-cache fixtures and ten damaged/stale
provider-comparison tests pass locally and on pinned Linux runtimes. Tests cover
the actual shell adapter, fresh state replacement, original parameters, failing
guest detection behind successful CLI exits, report collection and cancellation.
The cache fixtures execute real Cargo rebuilds and unchanged-target reuse.
Routing/Circle fixtures, ShellCheck, RWX lint and merged/activated Circle
validation pass. No speed advantage is claimed.

An [isolated same-source CLI run](https://cloud.rwx.com/optimism/runs/0e52b2257bb545e992ea227728c12864)
repeated all three workloads freshly, then replayed an intentionally wrong L2
claim. The unchanged-input repeat passes complete parity against the same Circle
originals, including both ELF inventories and the full final guest state. Every
source fingerprint reports unchanged inputs. Original logs show reusable Docker
layers and retained Cargo targets; the host build finished in 3.08 seconds. The
repeat recorded 5 seconds for environment preparation, 11 for lint, 9 for client
build and 72 for offline execution. Initial executions were 159, 18, 103 and 212
seconds respectively. These are task execution observations, excluding queueing,
setup and transfers. No full-pipeline or provider speed advantage is claimed.
Sccache reported three uncacheable requests on the repeat; this proves Cargo and
Docker reuse rather than attributing the improvement to sccache hits.

The wrong-claim guest exited with code one after **1,402,060,522 steps**, while
both the original Cannon run command and witness command returned zero. The
shared final-state guard rejected it, emitted failing JUnit, retained original
commands/logs/state and made the diagnostic task and CLI run fail as expected.
This uses a separate diagnostic definition; it does not replace a passing push
verdict or modify tests. The [cache and failure index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-rust-evidence/cannon-cache-and-failure.json)
retains both successful fresh repeats and the original failure, including the
explicit wrong-claim override. Runtime dependency-order advisories from the
benchmark are retained; the push definition now declares ancestor inputs before
their consumers, as RWX advised. Rust E2E is the next workload stage; final
aggregate checks and optional gate equivalence remain separate requirements.
