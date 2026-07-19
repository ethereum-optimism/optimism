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

func TestValidatorCatchesUpAfterBlocksStreamReconnect(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBlocksStreamSystem(t)
	sys.Batcher.Stop()
	sys.L1CL.Stop()
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)

	sys.PauseBlocksFeed()
	pausedAt := sys.ValidatorEL.BlockRefByLabel(eth.Unsafe)

	alice := sys.Funder.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.ValidatorEL)
	amount := eth.OneHundredthEther
	transfer := alice.Transfer(bob.Address(), amount)
	sourceReceipt := transfer.Included.Value()
	t.Require().Greater(bigs.Uint64Strict(sourceReceipt.BlockNumber), pausedAt.Number, "sequencer must produce the transaction while the blocks feed is paused")
	t.Require().Less(sys.ValidatorEL.BlockRefByLabel(eth.Unsafe).Number, bigs.Uint64Strict(sourceReceipt.BlockNumber), "validator must lag while the blocks feed is paused")

	sys.ResumeBlocksFeed()
	validatorReceipt := sys.ValidatorEL.WaitForReceipt(sourceReceipt.TxHash)
	t.Require().Equal(sourceReceipt.BlockHash, validatorReceipt.BlockHash, "validator must replay the canonical transaction block after reconnecting")
	bob.WaitForBalance(amount)
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)
}

func TestValidatorFollowsSequencerBlocksStreamReorg(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newBlocksStreamReorgSystem(t)
	sys.Batcher.Stop()
	sys.L1CL.Stop()
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)

	alice := sys.Funder.NewFundedEOA(eth.ThreeHundredthsEther)
	bob := sys.Wallet.NewEOA(sys.ValidatorEL)
	amount := eth.OneHundredthEther
	transfer := alice.Transfer(bob.Address(), amount)
	originalReceipt := transfer.Included.Value()
	validatorReceipt := sys.ValidatorEL.WaitForReceipt(originalReceipt.TxHash)
	t.Require().Equal(originalReceipt.BlockHash, validatorReceipt.BlockHash, "validator must import the original canonical transaction block")
	bob.WaitForBalance(amount)

	original := sys.SourceEL.BlockRefByNumber(bigs.Uint64Strict(originalReceipt.BlockNumber))
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)
	sys.ValidatorEL.InSync(sys.SourceEL, safety.CrossSafe, syncAttempts)
	safeBefore := sys.ValidatorEL.BlockRefByLabel(eth.Safe)
	finalizedBefore := sys.ValidatorEL.BlockRefByLabel(eth.Finalized)

	replacement := sys.ReplaceUnsafeBlock(original)
	sys.ValidatorEL.ReorgExact(original, syncAttempts)
	sys.ValidatorEL.InSync(sys.SourceEL, safety.LocalUnsafe, syncAttempts)
	t.Require().Equal(replacement.Hash, sys.ValidatorEL.BlockRefByNumber(original.Number).Hash, "validator must import the sequencer's replacement canonical block")
	bob.WaitForBalance(eth.ZeroWei)
	sys.ValidatorEL.WaitForProofsStoreBlock(replacement.Number + 1)
	t.Require().Equal(safeBefore, sys.ValidatorEL.BlockRefByLabel(eth.Safe), "unsafe blocks-feed reorg must preserve the validator safe head")
	t.Require().Equal(finalizedBefore, sys.ValidatorEL.BlockRefByLabel(eth.Finalized), "unsafe blocks-feed reorg must preserve the validator finalized head")
}
