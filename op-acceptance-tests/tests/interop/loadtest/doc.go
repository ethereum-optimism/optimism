// Package loadtest contains interop load tests that run against sysgo and sysext networks
// satisfying the SimpleInterop spec.
//
// Configure global test behavior with the following environment variables:
//
//   - NAT_INTEROP_LOADTEST_TARGET (default: 100): the initial number of messages that should be
//     passed per L2 slot in each test.
//   - NAT_INTEROP_LOADTEST_BUDGET (default: 1): the max amount of ETH to spend per L2 in each
//     test.
//
// Individual tests may define their own environment variables of the form NAT_<test>_<name>. See
// their go doc comments for details.
//
// Budget depletion and the go test timeout can end any test. They are interpreted as failures
// unless noted otherwise.
//
// Each test increases the message throughput until some threshold is reached (e.g., the gas
// target). The throughput is decreased if the threshold is exceeded or if errors are encountered
// (e.g., transaction inclusion failures).
//
// Visualizations for client-side metrics are stored in an artifacts directory, categorized by
// test name and timestamp: <metric-name>_<YYYYMMDD-HHMMSS>.png.
//
// Examples:
//
//	NAT_INTEROP_LOADTEST_BUDGET=2 go test -v -run Burst
//	NAT_INTEROP_LOADTEST_TARGET=500 go test -v -timeout 5m -run Steady
package loadtest
