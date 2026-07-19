package blocks_stream

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const syncAttempts = 30

func TestValidatorFollowsSequencerBlocksStream(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBlocksStreamSystem(t)
	sys.Batcher.Stop()

	alice := sys.Funder.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.ValidatorEL)
	amount := eth.OneHundredthEther
	transfer := alice.Transfer(bob.Address(), amount)
	sourceReceipt := transfer.Included.Value()

	validatorReceipt := sys.ValidatorEL.WaitForReceipt(sourceReceipt.TxHash)
	t.Require().Equal(sourceReceipt.BlockHash, validatorReceipt.BlockHash, "validator must import the sequencer's canonical transaction block")
	bob.WaitForBalance(amount)
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)
}

func TestValidatorCatchesUpFromSequencerBlocksStream(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBlocksStreamSystem(t)
	sys.Batcher.Stop()
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)

	stoppedAt := sys.ValidatorEL.BlockRefByLabel(eth.Unsafe)
	sys.ValidatorCL.Stop()

	alice := sys.Funder.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.ValidatorEL)
	amount := eth.OneHundredthEther
	transfer := alice.Transfer(bob.Address(), amount)
	sourceReceipt := transfer.Included.Value()
	t.Require().Greater(bigs.Uint64Strict(sourceReceipt.BlockNumber), stoppedAt.Number, "sequencer must produce the transaction after the validator stops")

	sys.ValidatorCL.Start()
	validatorReceipt := sys.ValidatorEL.WaitForReceipt(sourceReceipt.TxHash)
	t.Require().Equal(sourceReceipt.BlockHash, validatorReceipt.BlockHash, "validator must replay the sequencer's canonical transaction block")
	bob.WaitForBalance(amount)
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)
}
