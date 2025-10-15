package base

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

var dummyLogs = []string{
	"rpc error: code = DeadlineExceeded desc = context deadline exceeded while waiting for L2 block",
	"assertion failed: expected balance to increase after funding, but it did not",
	"unexpected revert: contract call failed with error 'insufficient funds for gas * price + value'",
}

// This test only exists to be flaky, and is used to test the flake-shake system.
func TestDummyFlakyTest(gt *testing.T) {
	t := devtest.SerialT(gt)

	t.Log("This test is flaky to test the flake-shake system")

	if rand.Float64() < 0.05 {
		// provide a dummy log, from a pool of three messages
		t.Log(dummyLogs[rand.Intn(len(dummyLogs))])
		t.Fail()
	}
}
