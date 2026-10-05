# Optional native PR gates

Circle continues to own its four required gates. Native Rust and Main aggregates
are implemented. Each remains uncounted until full automatic execution, complete
same-SHA Circle dependency/verdict comparison and terminal checks are verified.

`pr-gates.yml` embeds twelve existing workload definitions in parallel. Each
executes once, with its original source, tools, profiles, resources, caches and
fresh verdicts. Shared formatting and lint work serve both aggregates without a
second run. The coordinator preserves the existing optional workload check names
and adds `optimism-main-gate-shadow`. Standalone CLI modes and protected `develop`
cache warming remain available.

Version 3 of `ops/ci/pr-gates.json` maps Circle's exact 21 Rust and 19 Main terminal
dependency names to the actual native tasks. Main includes every one of the
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

Fourteen real Git/YQ fixtures pass: authoritative selection, exact coordinator
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
Full automatic validation and complete original Circle gate evidence remain
required. The Contracts aggregate remains unimplemented. Main's selector uploader
and flaky-report jobs are outside its original aggregate dependency set and
remain separate coverage work.

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
Complete automatic GitHub status and same-SHA Circle parity remain required.
