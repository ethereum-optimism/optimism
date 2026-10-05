# NUT pre-fork state regeneration shadow

The optional `optimism-nut-prefork-shadow` implements Main's
`check-nut-prefork-states` through the existing `run-main` routing. Hosted
same-SHA execution and comparison remain pending, so this occurrence is
not counted in verified coverage. Current coverage is 75/86 (87%).

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
config validation pass. Full hosted execution and comparison remain pending.
