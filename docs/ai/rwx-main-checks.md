# Main validator shadows

Seven additional Main occurrences use the existing optional
`optimism-pr-checks-shadow` and shared `run-main` route. They remain uncounted
until their complete hosted originals pass same-revision comparison.

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

Every Main verdict has `cache: false` and records run/attempt identity. Go
compilation caches are isolated by validator; shared modules are downloaded and
verified by the existing producer. Compiler state is reusable, while reports
and verdicts are excluded from filesystem outputs. Protected warming prepares
tools/modules and executes zero Main verdicts. An observed CLI rehearsal does
not establish a protected `develop` cache-rebuild event.

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
