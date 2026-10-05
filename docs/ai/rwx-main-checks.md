# Main validator shadows

Seven additional Main occurrences now pass complete original-report parity at
`c2d2b810ad429a0c8f62e6f8d76b077b50b8a631`, using the existing optional
`optimism-pr-checks-shadow` and shared `run-main` route. Total implementation
coverage is 63/86 (73%); 23 occurrences remain.

[Complete parity evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-main-checks-evidence/parity.json) retains every
selection, effective setting, source input and original file hash. Circle pipeline
[135565](https://app.circleci.com/pipelines/github/ethereum-optimism/optimism/135565)
and native run [a883f061](https://cloud.rwx.com/optimism/runs/a883f061b373443196e5e91902d97ccf)
passed all seven original validators, with one initial native attempt each.

| Occurrence | Original command | Native resources |
| --- | --- | --- |
| todo-issues-check | `./ops/scripts/todo-checker.sh --verbose --strict` | 2 CPU / 8 GiB |
| l2-chains-sync-check | `bash .circleci/scripts/check-l2-chains-sync.sh` | 2 CPU / 8 GiB |
| op-deployer-forge-version | `just check-forge-version`, in `op-deployer` | 2 CPU / 8 GiB |
| check-op-geth-version | `just check-op-geth-version` | 4 CPU / 8 GiB |
| check-nut-locks | `go run ./ops/scripts/check-nut-locks` | 4 CPU / 8 GiB |
| check-generated-mocks-op-node | `just generate-mocks-op-node && git diff --exit-code` | 8 CPU / 16 GiB |
| check-generated-mocks-op-service | `just generate-mocks-op-service && git diff --exit-code` | 8 CPU / 16 GiB |

The shared runner retains every original command, exit/signal, log, complete
selection, effective Go/tool settings and tracked source/link/submodule-pointer
hashes before and after execution. JUnit reports each validator as one command verdict; it does not invent
individual tests. The original PR TODO setting remains `check_closed: false`;
scheduled closed-issue checks keep their original command. No GitHub issue
credential is needed by the PR validator.

Mock discovery uses Go's `generate` build tag, includes ordinary and test files,
retains ignored files and every directive, and rejects package/dependency errors.
Original Go source bytes and the dry generator plan remain available for
independent selection verification. Fresh execution regenerates the mocks and
runs the original complete Git diff assertion. NUT discovery retains the lock,
every configured bundle/state file and the fetched protected `develop` revision
used by the original ancestry validator. Version and L2 matrix checks retain
their complete module/configuration inputs.

The initial hosted observation at `7c547ece` passes complete original-report
parity for the five non-generator validators. Both mock discoveries fail at the
missing gitignored superchain ZIP, before their original commands. The preceding
Circle mock jobs passed at `58a81fbd`; this is a new discovery dependency gap.
[First-failure evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-main-checks-evidence/first-failure.json) retains all
fourteen original reports and both diagnostics. Those first observations added
zero occurrences. The corrected stage closes all seven comparisons and retains
these failures without replacing their evidence.

A minimal native producer now builds the bundle from the exact registry gitlink
and verifies its committed checksum. Both adapters verify the bundle before
discovery and retain complete original bytes. Runtime also validates the native
producer's source, settings and file hashes before consuming it. Fixtures use
the production sync script and Just submodule recipe with a local fixture
registry, reproduce cold missing-embed discovery, and reject a corrupt reused
ZIP before generation. The producer emits no JUnit and records zero tests.

Every Main verdict has `cache: false` and records run/attempt identity. Go
compilation caches are isolated by validator; shared modules are downloaded and
verified by the existing producer. Compiler state is reusable, while reports
and verdicts are excluded from filesystem outputs. Protected warming prepares
tools/modules/bundle and executes zero Main verdicts. CLI rehearsal
[4e241bd3](https://cloud.rwx.com/optimism/runs/4e241bd36cd546fa9b5e4e2abf53d99f)
passed preparation of the exact benchmark SHA, with zero test executions and
zero Main verdict tasks. Its bundle producer binds the registry revision, ZIP
checksum, complete inputs and tools. This does not establish a protected
`develop` cache-rebuild event.

Fixtures execute real Go/Mockery generation twice, including a test-file
directive, then commit a changed interface and verify the stale mock fails the
original Git diff. Real shell/Just fixtures verify strict invalid TODO detection,
complete L2 matrices and Forge pin mismatch with the original nested working
directory. Cancellation retains the process signal and partial output. Piped
parent stdin can make ripgrep read an empty pipe instead of the checkout; all
adapter subprocesses explicitly use closed stdin. Comparison rejects matching
wrong commands, omitted directives/bundles/chains, corrupt source/originals,
stale revision/tools, reused verdicts, unexplained retries and target drift.

Circle remains the required provider. Production publishers, rulesets and the
single PR's draft state are unchanged.

All four required Circle gates, dependency review and all nine optional RWX
checks pass at the benchmark revision: 146 successes, one neutral, zero
unfinished or failed checks. Later observations stay bound to their own SHA.
