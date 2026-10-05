# Optional native PR gates

Circle continues to own its four required gates. The native Rust aggregate has
complete same-SHA original gate parity and successful terminal GitHub checks at
`68f13331`. The complete Main aggregate now passes original gate parity and
successful terminal GitHub checks at `4ae28fd9`. Both optional aggregates are
counted; neither changes required-gate ownership.

`pr-gates.yml` embeds sixteen existing workload definitions in parallel. Each
executes once, with its original source, tools, profiles, resources, caches and
fresh verdicts. Shared formatting and lint work serve both aggregates without a
second run. The coordinator preserves the existing optional workload check names
and adds `optimism-main-gate-shadow` and `optimism-contracts-gate-shadow`. Standalone CLI modes and protected `develop`
cache warming remain available.

Version 3 of `ops/ci/pr-gates.json` maps Circle's exact 21 Rust, 19 Main and
21 Contracts terminal dependency names to the actual native tasks. Main includes every one of the
twelve Go verdicts and both eight-shard acceptance variants, alongside the actual
static checks, generated mocks, locks, provenance, Cannon, fetcher, SP1 and Kontrol
workloads. Discovery checks the original Circle configuration with pinned YQ and
rejects changed dependencies, omitted feature partitions, duplicate assignments,
unassigned embedded runs and always-successful gates. It verifies source/branch/tag
forwarding, selected shard counts, fresh mode, existing check names and actual
engine-bound receipt conditions. Main-only workloads remain outside Rust's gate.
The chosen Go configuration stays at twelve shards and parallelism eight. The
24-shard Go and alternate acceptance CLI modes retain their existing behavior;
their aggregate receipts only apply to the coordinator's verified shard counts.

Each group receipt waits for every selected task to finish or be skipped. Native
engine states supply its environment. Selected failures, cancellations or skipped
tasks fail the receipt; an unselected workload is a safe skip only when all its
tasks were skipped. Receipts execute freshly and retain source/settings, complete
task states and file hashes. They perform zero tests and are absent from warming.

The aggregate waits for the embedded runs to reach terminal states, then evaluates
their actual scoped receipts. It validates every original sealed report against its
own exact source SHA, branch, complete inputs, native run identity, task attempt
and selection. Missing, extra, duplicate, corrupt, resealed stale or foreign
originals cannot pass. It records exactly one result for each original Circle
dependency. A failed or never-started receipt uses a separate failure observer
that retains the available engine states without requiring a nonexistent artifact.

An enclosing pilot run can fail on a Main-only job while its Rust formatting
receipt passes. The Rust aggregate follows that receipt, preserving the original
Rust gate's scope. A successful enclosing run with all workload tasks skipped
cannot stand in for an executed selected workload. Native embedded runs provide
these dependencies directly; the gate requires no external status polling,
installation token or extra credential.

Seventeen real Git/YQ fixtures pass: authoritative selection, exact coordinator
bindings, selected failure/skip handling, complete original aggregation, resealed
provenance, missing/corrupt/duplicate/extra and incorrectly typed originals,
failed/never-started receipt collection, genuine zero-test safe skips, all Main
dependencies and shards, changed shard modes, duplicate execution and an omitted
Go shard. Rust verdicts remain fresh through engine-provided run/attempt cache
keys, preserving their compiler tool caches. Other verdicts use `cache: false`.
The [Main implementation preflight](rwx-pr-gates-evidence/main-preflight.json)
retains local validation and clearly excludes unverified hosted coverage.
Producer artifact names and paths must match their consumer bindings.
One final status task waits for both mutually exclusive observers and executes
on every terminal outcome. It succeeds only when the aggregate actually passed
and the failure observer was correctly skipped. Actual aggregate failure,
cancellation, selected skip, contradictory observers or invalid engine values
fail this final verdict. Its complete original states and provenance are retained.
The [native embedded-run probes](rwx-pr-gates-evidence/embedded-preflight.json)
exercise actual passes, intentional failures, skips, a Main-only failure with a
passing Rust receipt, artifact mounting and matching parent/child run identities.
Their runs correctly fail while their observers succeed. These fixtures and
probes add no workload coverage.

The earlier API-based design failed before execution in automatic runs
[66765a69](rwx-pr-gates-evidence/first-hosted-failure.json) and
[44bdd307](rwx-pr-gates-evidence/embedded-preflight.json). CLI probes exposed a
GitHub token, but the automatic runs did not expose the `github` expression
context. The original failures and CLI probes remain retained as diagnostic
evidence. The native coordinator replaces that implementation and its wait helper.
The corrected Rust and Main aggregates now have full automatic validation and
complete original Circle gate evidence. The Contracts aggregate is implemented and awaits full hosted original parity.
Main's selector uploader and flaky-report jobs are outside its original aggregate
dependency set; both now have their own complete same-SHA parity evidence.

The first full embedded run at `eff84abed4` passed all three actual workload
groups and their receipts, then rejected an aggregate reference to `report`
instead of the producer's `receipt` artifact before execution. The original
failed run is retained in the preflight index. The corrected binding and a
regression fixture now validate the producer declaration as well as the
consumer expression. This failed aggregate adds no coverage.

The corrected aggregate at `94481c20a3` executed successfully despite the
enclosing pilot's Main-only lint failure. Its original GitHub check still failed:
the custom check included both mutually exclusive observers. RWX
[reports a skipped custom-check task as failure](https://www.rwx.com/docs/status-checks),
so that binding could never pass. The check now follows the single executed
final verdict. Native probes `b4c598b5` and `64144557` preserve a passing Rust
verdict through an actual Main-only failure and correctly fail an actual Rust
failure. Both enclosing probe runs intentionally fail; neither adds coverage.
The corrected automatic run
[`581722c0`](https://cloud.rwx.com/optimism/runs/581722c0a37849df9ef50c68789a9637)
at `68f13331fb6494b94e5c05401c69931d51a5b0fb` passed the complete native Rust
aggregate and its one final verdict. Circle pipeline 135601, workflow
`2a1c5cde-6157-4f66-85e3-0a46b462951d`, original
[job 5636909](https://circleci.com/gh/ethereum-optimism/optimism/5636909) passed
the genuine verifier for every one of its 21 dependencies. Both GitHub gate
statuses finished successfully on that SHA. The
[complete original comparison](rwx-pr-gates-evidence/rust-parity.json) retains
every source/API/config/log/receipt hash and both run and GitHub observations.
This closes one Rust occurrence and adds zero tests.

The full automatic coordinator run
[`25348ccd`](https://cloud.rwx.com/optimism/runs/25348ccd212542e2b25379ab8667a3d6)
at `4ae28fd902ad968ac38c835f0409b62cd035d6a0` passed every actual Main workload,
all ten scoped receipts, the genuine aggregate and its one final status. Circle
pipeline 135603, workflow `05d7f444-5b78-4111-b847-97c7739a323f`, original
[job 5637015](https://circleci.com/gh/ethereum-optimism/optimism/5637015) passed
the genuine verifier for all 19 exact dependencies. All four required Circle
gates, dependency review and optional native checks finished successfully on
that head. The [complete original Main comparison](rwx-pr-gates-evidence/main-parity.json)
retains all source/API/config/log/receipt hashes, actual engine states and
terminal GitHub checks. Original aggregate and final-status archives were
downloaded using the existing signed-in account after the CLI identity was
unable to unlock the RPC vault. No vault permissions changed. This closes one
Main occurrence and adds zero tests.

`ops/ci/compare-pr-gates.py` validates the actual source revision's manifest and
complete committed inputs, including symlinks and gitlinks. It accepts historical
version 2 and current version 3 evidence without substituting the latest working
tree's selection. The full original Circle dependency pages must be complete and
unique, all required jobs must pass, and the original executed verifier must
report exactly those jobs. Compiled always-successful gates, truncated or
incomplete logs, failed/skipped prerequisites, stale source/settings, corrupt or
unsealed files, cached native verdicts and foreign GitHub checks fail comparison.
Every native receipt and the final status must execute freshly. Seven additional
real Git/YQ comparison fixtures pass, including source link modes, an unrelated
Main-only failure, and deliberately invalid provider evidence. They add no
coverage. The automatic coordinator runs all 21 gate and comparison fixtures.


The final Contracts aggregate maps all four standard variants, four modified-file
heavy-fuzz variants, four coverage variants, seven L1 chain/feature upgrades,
OP Mainnet L2 fork and the fast checks to their 21 actual native verdicts.
Its five fresh receipts retain every engine state; the fast-check receipt reuses
the existing PR-check child. Each contract child retains its CLI mode and protected
compiler-only warming, while the coordinator owns its single automatic run.
Coverage keeps the original `project` checkout depth; receipt discovery validates
that directory against the clone producer and artifact path. No extra test run,
status polling or credential is introduced by the aggregate itself.
Local real Git/YQ tests cover complete Contracts authority, omission/failure of
the L2 verdict, safe skips, checkout paths, independent Rust/Main gate scope and
the complete original provider comparison. These fixtures add zero coverage.

The [Contracts implementation preflight](rwx-pr-gates-evidence/contracts-preflight.json)
retains all changed input hashes and local validation. Hosted coverage remains
pending until the complete original gate and L2 comparisons pass.
