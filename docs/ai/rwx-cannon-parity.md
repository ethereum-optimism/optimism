# Cannon Rust shadow

The three remaining Cannon workloads use the shared
`ops/ci/rust-cannon.sh` adapter on both Circle and RWX:
`kona-lint-cannon`, `kona-build-fpvm-cannon-client`, and
`kona-host-client-offline-cannon`. Hosted parity is pending; these jobs remain
unchecked in the [inventory](rwx-parity-todos.md).

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

Next validation: execute all three native jobs and Circle adapters on the same
SHA, compare complete original reports and guest state, and retain a fresh cache
repeat with unchanged input coverage. No speed claim or verified coverage
increment is made before those runs complete.
