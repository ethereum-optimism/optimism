# Optional native PR gates

Circle continues to own its four required gates. The native Rust aggregate is
implemented but remains uncounted until full automatic execution, complete
same-SHA Circle dependency/verdict comparison and terminal checks are verified.

`rust-gate.yml` embeds the existing pilot, Rust workspace and op-reth definitions
in parallel. Each executes once, with its original source, tools, profiles,
resources, caches and fresh verdicts. The coordinator preserves their existing
optional check names. Their standalone CLI modes and protected `develop` cache
warming remain available.

`ops/ci/pr-gates.json` maps Circle's exact 21 terminal dependency names to the
actual native tasks. Discovery checks the original Circle configuration with
pinned YQ and rejects changed dependencies, omitted feature partitions, duplicate
assignments and always-successful gates. It also verifies each embedded call,
source/branch/tag forwarding, fresh mode, existing check name and actual
engine-bound receipt condition. Main-only workloads remain outside Rust's gate.

Each group receipt waits for every selected task to finish or be skipped. Native
engine states supply its environment. Selected failures, cancellations or skipped
tasks fail the receipt; an unselected workload is a safe skip only when all its
tasks were skipped. Receipts execute freshly and retain source/settings, complete
task states and file hashes. They perform zero tests and are absent from warming.

The aggregate waits for the embedded runs to reach terminal states, then evaluates
their actual Rust receipts. It validates every original sealed report against its
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

Nine real Git/YQ fixtures pass: authoritative selection, exact coordinator
bindings, selected failure/skip handling, complete original aggregation, resealed
provenance, missing/corrupt/duplicate/extra and incorrectly typed originals,
failed/never-started receipt collection, and genuine zero-test safe skips.
Producer artifact names and paths must match their consumer bindings.
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
required. Main and Contracts aggregates are still unimplemented.

The first full embedded run at `eff84abed4` passed all three actual workload
groups and their receipts, then rejected an aggregate reference to `report`
instead of the producer's `receipt` artifact before execution. The original
failed run is retained in the preflight index. The corrected binding and a
regression fixture now validate the producer declaration as well as the
consumer expression. This failed aggregate adds no coverage.
