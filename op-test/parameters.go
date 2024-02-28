package op_test

type testParameters struct {
	// TODO CLI flag definitions for most common options; L1 forks, L2 forks, etc.
	// TODO CLI flag to set arbitrary named parameters (string slice)
	// Can use urfave CLI lib on init() time to create test-program flags/env vars
}

func (t *testParameters) Select(name string, options []string) []string {
	// TODO hook up to CLI / env filter
	return options
}

func (t *testParameters) Values(name string) []string {
	// TODO hook up to CLI / env filter
	return []string{"a", "b", "c"}
}

var _ ParameterSelector = (*testParameters)(nil)

type parameterManagerCtxKey struct{}

// BackendKind is a common parameter, used to identify the type of backend to test against.
// TODO: we may want to select backend-kind per type of actor in the tests.
// Composing different backends together can allow us to do more types of tests
// (e.g. running tests in managed form, while hooked up to external op-reth).
type BackendKind string

func (b BackendKind) String() string {
	return string(b)
}

const (
	// Live backends run tests against external services; live networks like the monorepo devnet.
	Live BackendKind = "live"
	// Managed backends run in-process services.
	// This is the main migration target for op-e2e system tests.
	Managed BackendKind = "managed"
	// Instant backends apply state-changes synchronously, and don't run full services.
	// This is the main migration target for op-e2e/actions tests.
	Instant BackendKind = "instant"
)
