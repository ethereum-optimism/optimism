package msg

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestInitExecMsg tests basic interop messaging
func TestInitExecMsg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

	eventLoggerAddress := alice.DeployEventLogger()
	// Trigger random init message at chain A
	initIntent, _ := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(30)))
	// Make sure supervisor indexes block which includes init message
	sys.Supervisor.AdvancedUnsafeHead(alice.ChainID(), 2)
	// Single event in tx so index is 0
	bob.SendExecMessage(initIntent, 0)
}
