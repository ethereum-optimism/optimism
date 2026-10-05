# ShellCheck and Semgrep shadows

The existing optional `optimism-pr-checks-shadow` now includes Main's
`shell-check`, `semgrep-test` and `semgrep-scan-local`. All three occurrences
pass complete hosted same-SHA original-report comparison and count toward
74/86 (86%) verified job coverage. Circle's job names and required gates stay
unchanged. The corrected native definition passed an isolated hosted rehearsal.
The pushed PATH fix also passed `optimism-pr-checks-shadow` on
`d8e7d3ecab7e931032bb513819504f9be224b98d` in
[native run 650e5e0d](https://cloud.rwx.com/optimism/runs/650e5e0d67e5441f801a6e1b3114b651).
All checks on that pushed head reached successful terminal states: 154 successes
and one neutral result, including all four Circle gates and dependency review.
Every subsequent final PR head still needs every required and optional check
to finish.

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
comparison regressions pass both locally and in the corrected native rehearsal.

## Hosted closeout

Complete parity passes at `ffefdb34638470dd1126cbf2aaebb4644400b6fb` on
[Circle pipeline 135572](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135572)
and [native rehearsal a656a664](https://cloud.rwx.com/optimism/runs/a656a664a1704612b3020c70fb18627c).
The latter checks out that exact clean source while using the corrected
`pr-checks.yml` definition. Its definition SHA-256 is
`c0201500bc50b6d3f3a2dd916d99b3228ea6cfbec4068498a5bbd1865cd44235`.
It passed all three fresh verdicts and the live helper fixtures. It establishes
same-code parity, not a successful replacement of the failed pushed-head check.

Both providers select all 82 original ShellCheck files. Semgrep rule tests pass
all 25 checks with identical expected/reported lines. The original missing-fixture
classification for `.semgrep/rules/go-acceptance-test-flakes.yaml` remains an
explicit derived skip. The baseline scan selects 193 targets, reports zero
findings/errors, and retains four original size-limit exclusions. The exact
baseline commit is `c8e4ba855d79ca56463909ef5a2c5830a1189401`; merge base,
changed-file inventory, all source/rule/fixture hashes and complete diagnostics
agree. All 54 original files verify against their seals.
The [parity index](rwx-static-checks-evidence/parity.json) retains complete
selections, settings, report hashes and provider identities. Complete originals
remain in `.ci/rwx-static-checks-evidence/ffef/` and the provider artifacts.

The first pushed native run
[18374caf](https://cloud.rwx.com/optimism/runs/18374cafeb294daabd48d158a1b45577)
failed ShellCheck and its real-tool fixture because the producer appended an
assumed `bin` directory to ShellCheck's installation path. The original error is
`No such file or directory: 'shellcheck'`. The producer now exports Mise's actual
resolved PATH. Commands, versions, selection and assertions remain unchanged.
The [failure index](rwx-static-checks-evidence/first-failures.json) retains its
original failed report/task logs and the corrected rehearsal's three intentional
failure fixtures: SC2086, a new Semgrep finding and a missing baseline.
