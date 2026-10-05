# NUT pre-fork state regeneration shadow

The optional `optimism-nut-prefork-shadow` implements Main's
`check-nut-prefork-states` through the existing `run-main` routing. Complete same-SHA original-report parity passes at
`fd426d8724c4df4d968db0e986b92d9177b717f4`: both selected forks, karst and lagoon,
execute all six original case identities successfully with no skips or retries.
This occurrence is counted. Current coverage is 76/86 (88%).

Both providers run the original `just _check-nut-prefork-states` loop, including
its final Git diff. Discovery retains every `*_state.json` input and the
original `jovian` exclusion. Selected forks execute sequentially, rebuilding
the test package between forks so its embedded predecessor state stays current.
The original `go test -count=1 -run TestGenerateForkState
./rust/kona/tests/proofs/` selection, environment, default concurrency, timeout
and zero retries are preserved. The CI-only report mode wraps the same Go
invocation with gotestsum and retains complete original JSON, JUnit and per-test
logs. Manual recipe execution keeps the original direct Go command.

Native modules and ci-profile contracts build independently. A minimal
superchain producer regenerates/verifies the bundle from its exact registry
gitlink and committed checksum. Runtime verifies every artifact's source,
toolchain, settings, archive and file hashes before discovery. Full source,
submodules, Git metadata, relative fixtures and pinned Go/Forge remain available.
The initial fresh verdict worker uses 8 CPUs / 16 GiB and an isolated native
Go compiler cache. Verdicts and reports are excluded from reusable outputs.
Protected warming targets only these producers and executes zero tests/helpers.

The comparer rederives the complete fork/package/test selection from original
source and discovery, requires exactly one initial execution per selected fork,
compares every original case/outcome/skip/retry, verifies generated states against
the committed bytes, and rejects changed source or stale/corrupt dependencies.
Native JUnit groups the original cases by fork while retaining each actual
case name and outcome; its original-to-projected grouping is explicitly sealed.
Failed or canceled loops retain partial originals without inventing later fork
verdicts. Just converts its handled SIGTERM to exit 143; the underlying Go
wrapper's original signal termination is retained separately.

Eight Linux fixtures pass using real Go, Forge, the production Just loop,
production superchain sync script and original Git diff. They cover new forks,
new matching tests, the original exclusion, repeated fresh execution,
source-relative contract artifacts, state drift after successful Go tests,
intentional failure, cancellation, and complete-report comparison. Resealed
wrong commands, omissions, stale revisions/settings, false coverage and corrupt
bundles still fail comparison. ShellCheck, RWX lint, and merged/activated Circle
config validation pass. The hosted helper task passes all eight fixtures. Complete hosted execution
and comparison pass; the next pushed definition still needs terminal PR checks.

## Retained hosted evidence

[Circle pipeline 135582, job 5635593](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135582)
and [RWX run e2952b8b](https://cloud.rwx.com/optimism/runs/e2952b8b619146b4ab1355a19e5e2804)
pass the [full comparison](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-nut-prefork-evidence/parity.json), verifying 72
Circle and 79 RWX original file seals, identical complete selection, tools,
commands, source inputs, generated state bytes and every original outcome.
Go 1.26.6 actually executes the full suite. Native reporting displays six
passing cases.

The first native push run passed the actual generation but displayed zero
cases because its XML report selected the Go JSON parser explicitly. The
corrected full execution uses JUnit parser inference. Its source remains the
exact fd42 commit; the CLI uploads only the retained
[corrected definition](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-nut-prefork-evidence/verified-native-definition.yml).
The exact three-line parser-option block removal, both native snapshots, first
original report hashes and complete fixture seals are retained in the
[run and cache index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-nut-prefork-evidence/runs-cache-and-fixtures.json).
The first reporting defect remains visible and contributes no extra coverage.

The [dependency-only warm rehearsal b1076421](https://cloud.rwx.com/optimism/runs/b1076421c2454800a927edb471375298)
reuses all three complete producer outputs, with zero suite verdicts or helper
tests. The exact source and cached-from task identities are retained. Protected
`develop` cache-rebuild execution remains unobserved. No median or speed win is
claimed.

The pushed definition at `6245472e` also succeeded in automatic native run
`8487717c`, with all six tests displayed and zero failures. The
[pushed reporting confirmation](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-nut-prefork-evidence/pushed-reporting-confirmation.json)
retains exact task identity and snapshot seal. This confirms the reporting fix
and adds no extra occurrence.
