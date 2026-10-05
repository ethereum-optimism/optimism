# RWX local package contracts

The pilot retains one draft PR and the chosen full Go configuration: twelve
shards, parallelism eight, 16 CPU / 32 GiB compilation and 8 CPU / 16 GiB verdicts.
Acceptance keeps eight shards per variant and its existing per-variant resources.
Circle owns the required gates. Workload selection and parity coverage are unchanged.

## Orchestration and dependencies

The five Go, acceptance, standard/modified contract, coverage and upgrade
definitions contain explicit calls to family-specific compile and verdict
packages. Each repeated body has one implementation under `.rwx/packages`.
The experimental 24-shard Go tasks and 24-shard CLI configuration are removed; the
narrower rollup CLI mode remains available.

Go compilation depends on source, Go tools and discovery after the Go/contract
producers. It does not depend on Rust or prestates. Each verdict uses only its
own compiler leaf and the required runtime producers. Acceptance discovery
and verdicts remain separate. Contract variants retain their own Foundry
outputs, prepared signatures, profiles, feature selection and runtime behavior.

Local package tasks explicitly use `package.use`. Callers use producer leaves
such as `compile-0.build`, so a consumer does not acquire a barrier across all
shards. See [RWX local packages](https://www.rwx.com/docs/local-packages).

## Input and cache boundaries

Package parameters carry scalar settings and relative paths. Native execution
rejected artifact mount expressions in `with`, despite lint accepting them.
Archive bytes instead arrive through filtered `package.use` filesystem inputs.
Go and acceptance verdicts exclude `.ci/go-cache`, `.ci/rust-cache` and
`rust/target` from their incoming snapshots. Their existing artifact restorer
then checks revision, toolchain pins, settings, archive hashes and file hashes
before restoring runtime files. Compiler metadata and binary validation retain
their existing owners.

Common tool preparation imports only the pinned configuration, install script
and named CI helper files. Bootstrap output filtering excludes the checkout and
Git history. Go, Foundry and Rust setup remain separate layers. Existing native
Go, Foundry, Cargo target, sccache and Docker/BuildKit cache keys and output paths
remain isolated. Moving a task into a package can invalidate its native task
cache; this refactor does not claim that every prior task cache entry is reused.

Contract consumers already require their producer's Foundry filesystem. The
producer now includes its sealed preparation metadata in that snapshot.
Coverage/upgrades also import their sealed preflight inputs. RPC values remain
only in availability checks and executing runtime verdicts, outside package
arguments, bootstrap setup and compilation.

Verdicts remain fresh. Reusable outputs contain runtime compiler state, not test
results. Original reports and native test reporting remain task artifacts.
Protected `develop` warming retains its compiler-only targets and executes zero
tests. New generated evidence stays outside Git.

## Gate and execution verification

The existing caller task names, receipts, gate manifest and optional check names
remain stable. `pr-gate.py` resolves each package to exactly one unconditional
executable verdict and checks freshness there. A call to a compiler, a cached
verdict, a no-op, a conditional leaf or a multi-task verdict package cannot pass.
Receipts continue to use actual engine-bound caller states, including failures,
cancellations, retries and safe skips.

The CLI-only `.rwx/local-package-fixture.yml` proves filtered input transfer,
compiler-cache exclusion, nested values/artifacts and caller terminal states.
Its normal mode succeeds, `intentional-failure=true` fails with retained exit
code 17, and `cache-warm=true` skips every verdict. It publishes no PR status.
The coordinator runs the gate fixtures and package graph regression tests.

Native verification retained outside Git:

- [Successful filtered transfer](https://cloud.rwx.com/optimism/runs/2f3e90605eda4e7d94384c687840402d): the verdict executes, compiler state is absent, and the nested report reaches its consumer.
- [Intentional failure](https://cloud.rwx.com/optimism/runs/72fa847783914af9956902dbe803a0fa): the executable leaf and caller both fail; the observer runs and the retained report records exit code 17.
- [Warm-only run](https://cloud.rwx.com/optimism/runs/63c5909d32be4c198bf509c44c34248f): compilation is reused, verdicts/report consumers are skipped, and the observer confirms the skip.

The five main definitions shrink from 5,380 to 2,122 lines. Including all 638
lines of new workload, shared toolchain and probe packages, that is a net
reduction of 2,620 lines (49%). Local validation passes 161 tests with four live
tool tests skipped, plus ShellCheck using RWX's `bash -e -o pipefail` context and
RWX lint across all 49 definitions/packages. Final-head hosted checks provide
the live workload verification and are recorded in the PR.

Validate the completed refactor as one batch: relevant helper tests, ShellCheck
in RWX's shell context, RWX lint, then one combined hosted PR run. Additional
runs follow actual failures or material fixes. This stage does not add Circle
benchmark replays or reopen shard tuning.
