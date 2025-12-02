# Cannon testdata

These example Go programs are used in tests,
and encapsulated as their own Go modules.

## Testdata

The `testdata` directory name (special Go exception) prevents tools like `go mod tidy`
that run from the monorepo root from picking up on the test data,
preventing noisy dependabot PRs.

