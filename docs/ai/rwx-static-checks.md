# ShellCheck and Semgrep shadows

The existing optional `optimism-pr-checks-shadow` now includes Main's
`shell-check`, `semgrep-test` and `semgrep-scan-local`. All three occurrences
remain **uncounted** until complete hosted same-SHA original-report comparison
passes. Coverage remains 67/86 (78%). Circle's job names and required gates stay
unchanged.

ShellCheck runs the exact published `circleci/shellcheck@3.2.0` command, retained
with its MIT license. The command's SHA-256 is
`d8f51c02b4a6ce3ebc32024dfaf8a16ec4a4445331a7832e8137d6afc00fb0a4`.
The pinned Circle image contains ShellCheck 0.9.0; native tool preparation installs
that version separately from other consumers of the repository's tool pins.
The original `find` selection includes every `.sh` file, preserves its newline
exclusion, and excludes `packages/contracts-bedrock/lib` and `docs/public-docs`.
The original severity, shell, formatting, external-source and exclude parameters,
as well as `.shellcheckrc`, remain authoritative. Both the independent original
find output and the exact orb's executed file list must agree. Each selected file
is bound to its source hash. The full original aggregate `shellcheck.log` and
execution log stay intact; per-file JUnit is explicitly derived.

Semgrep runs the two original local-rule commands with the original timeout and
failure policy. `--json` retains machine reports; scan `--verbose` retains every
target exclusion and diagnostic. The pinned Circle image resolves to Semgrep
1.178.0, also installed by the native tool producer. These pins do not change the
1.137.0 tool used by the separate contract-fast checks.

The original PR environment sets `SEMGREP_BASELINE_REF=develop`. This applies a
baseline to the original local scan, as the real pinned CLI confirms. Native
tasks fetch that branch before scanning. Both providers retain its exact commit,
merge base and complete changed-file inventory, and reject baseline drift during
execution or comparison. A `develop` run retains the original full-scan behavior.
Missing baseline objects fail before scanning; an original empty selection remains
explicitly empty. [Semgrep's CI documentation](https://semgrep.dev/docs/semgrep-ci/sample-ci-configs)
describes baseline-aware scanning.

Complete rule/fixture/ignore/source hashes, scanned file identities and hashes,
skipped targets with reasons, parser warnings, skipped rules and engine selection
are compared. Rule-test reports retain every expected and reported line, missing
fixture classification and fix-test result. Original runtime profiling remains
retained but is excluded from semantic equality. Absolute workspace prefixes are
normalized only in the comparison view; every original remains unchanged.
Original zero-byte Circle logs can be restored solely from their declared empty
file hashes. Missing nonempty originals, changed settings, unknown source paths,
duplicate targets, stale revisions, changed warnings/exclusions, unexplained
retries and fabricated derived results fail validation.

All verdicts and live helper fixtures execute freshly with `cache: false`.
Native ShellCheck uses 2 CPU / 8 GiB; Semgrep uses 8 CPU / 16 GiB. Protected warming
targets only tool preparation and executes zero scans or helper fixtures.
Real pinned-tool fixtures verify paths containing spaces, both ShellCheck ignore
directories, baseline selection, preservation of unchanged findings, a new Semgrep
finding, a missing baseline, and an original SC2086 failure. Native helper artifacts
retain each intentional initial failure separately. These fixtures and thirteen
comparison regressions pass locally; hosted parity remains required.
