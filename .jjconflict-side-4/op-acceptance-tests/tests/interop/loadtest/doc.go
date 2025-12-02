// Package loadtest contains interop load tests that run against sysgo and sysext networks
// satisfying the SimpleInterop spec.
//
// Configure test behavior with the following environment variables:
//
//   - NAT_INTEROP_LOADTEST_TARGET (default: 100): the initial number of messages that should be
//     passed per L2 slot in each test.
//   - NAT_INTEROP_LOADTEST_BUDGET (default: 1): the max amount of ETH to spend per L2 per test. It
//     may be a float. The test will panic if it overflows a uint256. Funds may be used during test
//     setup to, e.g., deploy contracts.
//   - NAT_INTEROP_LOADTEST_TIMEOUT (default: min(3m, go test timeout)): the amount of time to run
//     each test. Also see https://github.com/golang/go/issues/48157.
//
// Test names follow the convention Test<spammer><scheduler>. The spammer name identifies what
// kinds of transactions will be spammed. The scheduler defines how often the spammer is executed.
// There are two schedulers:
//
//   - Steady: spams up to the gas target, simulating benign but heavy load.
//   - Burst: spams as much as possible, simulating a DoS attack.
//
// Both schedulers decrease throughput if errors are encountered. They exit successfully upon
// timeout or budget depletion, whichever comes first.
//
// All errors encountered during test setup are fatal, including timeouts and budget depletion.
//
// Visualizations for client-side metrics are stored in an artifacts directory, categorized by
// test name and timestamp: artifacts/<test-name>_<yyyymmdd-hhmmss>/<metric-name>.png.
//
// Examples:
//
//	NAT_INTEROP_LOADTEST_BUDGET=1.2 go test -v -run TestRelayBurst
//	NAT_INTEROP_LOADTEST_TIMEOUT=1m NAT_INTEROP_LOADTEST_TARGET=500 go test -v -run TestRelaySteady
package loadtest
