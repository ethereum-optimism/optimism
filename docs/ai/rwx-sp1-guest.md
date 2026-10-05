# Complete SP1 guest shadow

The optional `optimism-sp1-guest-shadow` ports Main's `kona-build-sp1-elfs`
through the shared `run-main` routing. It is implemented and has passed the full
Linux preflight; hosted native execution and resolved same-SHA Circle comparison
remain pending. This occurrence remains uncounted, with verified coverage 77/86.

Both providers retain the canonical SP1 pin checks, native Succinct toolchain,
`just check-sp1-guest-lock` before the build, and `just build-elfs-native` for every
program selected by the original Just loop. The complete artifact includes both
64-bit RISC-V ELFs and `vkeys.toml`, their exact source markers, compiler identity,
file hashes and independently recomputed CPU verification keys. The consumer's
`super-aggregation` key remains validated. Runtime checks execute the original
`just test-sp1-guest`, `just lint-sp1-guest` and `just check-range-vkeys`, with
`RUSTFLAGS=-Dwarnings`, complete Cargo dependency graphs, target discovery,
original logs and actual case-level JUnit. Packages and targets with zero tests
remain included; new programs and test cases are discovered from original inputs.
There are no test retries in this Circle workload.

Native preparation caches the pinned tools and Succinct installation separately.
The ELF producer uses 16 CPUs / 32 GiB; fresh checks use 8 CPUs / 16 GiB. Separate
native ELF and runtime Cargo targets and sccache directories prevent incompatible
compiler outputs from sharing a target. Cargo source fingerprints and stable
content timestamps retain correctness across restored checkouts. Successful
builds commit the target fingerprint; failures retain partial original reports.
Verdicts and logs are excluded from reusable outputs. An explicit
`target-cache-mode=sccache-only` CLI probe empties targets while retaining compiler
cache data. Protected `develop` warming targets only the ELF producer and runs
zero tests and zero helper fixtures.

SP1 6.8.1 reports its CLI version as a Git descriptor rather than its numeric SDK
version. The adapter verifies that the executable comes from the canonical mise
installation and retains its actual binary hash and original descriptor. The
pinned [SDK build implementation](https://github.com/succinctlabs/sp1/tree/v6.8.1/crates/build/src)
uses `riscv64im-succinct-zkvm-elf` and places native compilation beneath the Cargo
metadata target directory's `elf-compilation` subdirectory; cache paths preserve
that behavior.

The comparer validates every original seal, retained pin-check source, exact
command, revision, toolchain, branch, full dependency graph, case outcome, skip and
retry history. Both complete ELF inventories and verification keys must match
byte for byte. Only physical checkout, Cargo home and compiler-output locations
are normalized. Missing, extra, corrupt, stale, resealed false or mismatched
inputs fail validation.

The [complete preflight originals](rwx-sp1-guest-evidence/first-full-preflight.json),
[latest manifest-discovery run](rwx-sp1-guest-evidence/latest-full-preflight.json)
and [all eight real-tool fixtures](rwx-sp1-guest-evidence/local-fixtures.json)
retain their full original file hashes and unpublished source revisions.

The full local preflight executes four super-aggregation unit tests and two
range-vkeys unit tests, retaining the super-range and doc-test targets with zero
cases. Real-tool fixtures exercise new packages, scoped duplicate case names,
ignored tests, actual repeated failures, cancellation and production artifact
validation. The comparer fixtures use complete real originals and deliberately
resealed false case, dependency and compiler reports. Provider aliases in those
isolated fixtures are verifier tests and add no hosted coverage.

The existing Circle job keeps its name, resource class, cache preparation,
artifact/workspace publication and dependency ownership. Its original commands
now use the shared reporting adapter. Circle continues to own the four required
gates; production publishers and GitHub rulesets remain unchanged.
