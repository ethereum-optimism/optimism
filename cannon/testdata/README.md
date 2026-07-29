# Cannon testdata

The `testdata` directory name (special Go exception) prevents tools like `go mod tidy`
that run from the monorepo root from picking up on the test data,
preventing noisy dependabot PRs.

`diff-hello/` holds a minimal MIPS guest program used only by `just diff-cannon`
and `op-challenger` cache tests. It is built with Go 1.24 so the guest binary
does not depend on Go 1.26 runtime syscalls that production Cannon no longer
implements. Legacy Go FPVM guest program fixtures were removed because
production fault proofs use the kona client instead.
