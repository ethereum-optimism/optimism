// Package loadtest contains interop load tests that run against sysgo and sysext networks
// satisfying the SimpleInterop spec.
//
// Test names follow the convention Test<spammer><scheduler>. The spammer name identifies what
// kinds of transactions will be spammed. The scheduler defines how often the spammer is executed
// and when spamming will stop.
//
// There are two schedulers:
//
//   - Steady: spams until the NAT_STEADY_TIMEOUT (see below) before exiting successfully. It
//     attempts to approach but not exceed gas target, simulating benign but heavy
//     load. Budget overdraft is the only fatal error.
//   - Burst: spams as fast as possible until the budget is depleted. All non-overdraft errors are
//     ignored, although they may reduce the spammer's frequency. Burst is intended to simulate a
//     DoS attack.
//
// Configure global test behavior with the following environment variables:
//
//   - NAT_INTEROP_LOADTEST_TARGET (default: 100): the initial number of messages that should be
//     passed per L2 slot in each test.
//   - NAT_INTEROP_LOADTEST_BUDGET (default: 1): the max amount of ETH to spend per L2 in each
//     test. It may be a float. The test will panic if the total overflows a uint256.
//     budget may be used during test setup, e.g., to deploy contracts.
//   - NAT_STEADY_TIMEOUT (default: min(3m, go test timeout)): the amount of time to run the
//     spammer in each Steady test. Also see https://github.com/golang/go/issues/48157.
//
// All errors encountered during test setup are fatal, including go test timeout expiration.
//
// Each test increases the message throughput until some threshold is reached (e.g., the gas
// target). The throughput is decreased if the threshold is exceeded or if errors are encountered
// (e.g., transaction inclusion failures).
//
// Visualizations for client-side metrics are stored in an artifacts directory, categorized by
// test name and timestamp: artifacts/<test-name>_<yyyymmdd-hhmmss>/<metric-name>.png.
//
// Examples:
//
//	NAT_INTEROP_LOADTEST_BUDGET=1.2 go test -v -run TestRelayBurst
//	NAT_STEADY_TIMEOUT=1m NAT_INTEROP_LOADTEST_TARGET=500 go test -v -run TestRelaySteady
package loadtest
