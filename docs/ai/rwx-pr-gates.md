# Optional native PR gates

Circle continues to own its four required gates. The first native aggregate,
`optimism-rust-gate-shadow`, is implemented but remains uncounted until its full
automatic execution and same-SHA original Circle comparison are verified.

`ops/ci/pr-gates.json` maps every original terminal dependency from
`required-rust-ci` to its actual native workload tasks. Discovery checks the
original Circle configuration with pinned YQ, requires all 21 names in their
original order, and rejects changed dependencies, missing feature partitions,
duplicate assignments or an always-successful gate. The existing native shadows
retain their check names and publish additional receipts for Rust formatting,
the complete Rust workspace, and the three op-reth Rust workloads. Main-only
workloads remain outside Rust's dependency set.

Each receipt waits for every selected task to finish or be skipped. Actual
RWX task states supply its environment. Selected failed, canceled or skipped
tasks fail the receipt; an unselected workload is a safe skip only when all its
tasks were skipped. Receipts execute freshly, retain complete source/settings,
task states and hashes, and perform zero tests. Warm-only runs do not select them.
The original optional checks continue to report their complete workloads.

The aggregate fetches the exact commit's complete original GitHub status pages,
retains every response byte and request observation, and rechecks the first page
to detect concurrent pagination shifts. It accepts only the newest status for
each required receipt from the verified RWX integration actor and the Optimism
native run URL. An older success cannot hide a newer pending/failing status;
missing, foreign, malformed and timed-out statuses cannot pass. A final pass
requires every original dependency's native group to pass.

The GitHub installation token is available only to the waiting runtime tasks on
the pilot branch or `develop`; the API client makes repository-specific GET
requests. It never records authorization headers. Four fresh workers can wait
40 minutes each, with sealed continuation reports between them and a final
deadline. This limits individual worker/token lifetimes without dropping earlier
observations. No compiler cache, test-result cache or credential is exported.

Eight fixtures exercise actual Git/YQ authority discovery and HTTP servers:
complete pagination, a changing page boundary, latest statuses, engine-state
receipt validation, pending continuation, stale/resealed or corrupt inputs,
original failures and never-started prerequisites. All pass locally. An isolated
native probe has also exercised genuine successful, failed and skipped tasks;
the run correctly fails while its terminal observer retains their engine states.
These fixtures and probes add no workload coverage. Hosted gate validation,
complete original Circle dependency/orb verdict evidence, and final PR checks
remain required. Main and Contracts aggregates are still unimplemented.
