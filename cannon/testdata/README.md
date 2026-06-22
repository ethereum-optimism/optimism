# Cannon testdata

The `testdata` directory name (special Go exception) prevents tools like `go mod tidy`
that run from the monorepo root from picking up on the test data,
preventing noisy dependabot PRs.

`diff-hello/` holds a minimal MIPS guest program used only by `just diff-cannon`
to compare Cannon releases. Legacy Go FPVM guest program fixtures were removed
because production fault proofs use the kona client instead.
