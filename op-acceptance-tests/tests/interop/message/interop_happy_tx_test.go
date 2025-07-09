package msg

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestInteropHappyTx is testing that a valid init message, followed by a valid exec message are correctly
// included in two L2 chains and that the cross-safe ref for both of them progresses as expected beyond
// the block number where the messages were included
func TestInteropHappyTx(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)

	// two EOAs for triggering the init and exec interop txs
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// wait for chain B to catch up to chain A if necessary
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	// send initiating message on chain A
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	initTx, initReceipt := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(3), rng.Intn(10)))

	// at least one block between the init tx on chain A and the exec tx on chain B
	sys.L2ChainB.WaitForBlock()

	// send executing message on chain B
	_, execReceipt := bob.SendExecMessage(initTx, 0)

	// confirm that the cross-safe safety passed init and exec receipts and that blocks were not reorged
	dsl.CheckAll(t,
		sys.L2CLA.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: initReceipt.BlockNumber.Uint64(),
			Hash:   initReceipt.BlockHash,
		}, 500),
		sys.L2CLB.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: execReceipt.BlockNumber.Uint64(),
			Hash:   execReceipt.BlockHash,
		}, 500),
	)
}
