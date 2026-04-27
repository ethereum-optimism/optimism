package core

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/proofs/utils"
	"github.com/ethereum/go-ethereum/common"
)

func TestExecutePayloadSuccess(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()
	sys := utils.NewMixedOpProofPreset(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	opRethELNode := sys.RethWithProofL2ELNode()

	// Wait for the validator's EL head to reach the sequencer's head.
	seqHead, err := sys.L2ELSequencerNode().Escape().L2EthClient().InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		gt.Fatal(err)
	}
	sys.L2ELValidatorNode().WaitForBlockNumber(seqHead.NumberU64())

	plannedTxOption := user.PlanTransfer(user.Address(), eth.OneWei)
	plannedTx := txplan.NewPlannedTx(plannedTxOption)
	signedTx, err := plannedTx.Signed.Eval(ctx)
	if err != nil {
		gt.Fatal(err)
	}

	raw, err := signedTx.MarshalBinary()
	if err != nil {
		gt.Fatal(err)
	}

	lastBlock, err := opRethELNode.Escape().L2EthClient().InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		gt.Fatal(err)
	}

	// Wait for the proofs ExEx store to index the parent block. The ExEx processes
	// ChainCommitted notifications asynchronously, so the EL head can advance before
	// the store is ready. debug_executePayload reads from the ExEx store and will
	// fail with "no state found" if it hasn't caught up yet.
	utils.WaitForProofsStoreBlock(t, opRethELNode.Escape().L2EthClient(), lastBlock.NumberU64())

	blockTime := lastBlock.Time() + 1
	gasLimit := eth.Uint64Quantity(lastBlock.GasLimit())

	var prevRandao eth.Bytes32
	copy(prevRandao[:], lastBlock.MixDigest().Bytes())

	var zero1559 eth.Bytes8
	minBaseFee := uint64(10)

	attrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(blockTime),
		PrevRandao:            prevRandao,
		SuggestedFeeRecipient: lastBlock.Coinbase(),
		Withdrawals:           nil,
		ParentBeaconBlockRoot: lastBlock.ParentBeaconRoot(),
		Transactions:          []eth.Data{eth.Data(raw)},
		NoTxPool:              true,
		GasLimit:              &gasLimit,
		EIP1559Params:         &zero1559,
		MinBaseFee:            &minBaseFee,
	}

	witness, err := opRethELNode.Escape().L2EthClient().PayloadExecutionWitness(ctx, lastBlock.Hash(), attrs)
	if err != nil {
		gt.Fatal(err)
	}
	if witness == nil {
		gt.Fatal("empty witness")
	}
}

func TestExecutePayloadWithInvalidParentHash(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()
	sys := utils.NewMixedOpProofPreset(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	opRethELNode := sys.RethWithProofL2ELNode()

	// Wait for the validator's EL head to reach the sequencer's head.
	seqHead, err := sys.L2ELSequencerNode().Escape().L2EthClient().InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		gt.Fatal(err)
	}
	sys.L2ELValidatorNode().WaitForBlockNumber(seqHead.NumberU64())

	plannedTxOption := user.PlanTransfer(user.Address(), eth.OneWei)
	plannedTx := txplan.NewPlannedTx(plannedTxOption)
	signedTx, err := plannedTx.Signed.Eval(ctx)
	if err != nil {
		gt.Fatal(err)
	}

	raw, err := signedTx.MarshalBinary()
	if err != nil {
		gt.Fatal(err)
	}

	lastBlock, err := opRethELNode.Escape().L2EthClient().InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		gt.Fatal(err)
	}

	// Wait for the proofs ExEx store to index the parent block (same race as TestExecutePayloadSuccess).
	utils.WaitForProofsStoreBlock(t, opRethELNode.Escape().L2EthClient(), lastBlock.NumberU64())

	blockTime := lastBlock.Time() + 1
	gasLimit := eth.Uint64Quantity(lastBlock.GasLimit())

	var prevRandao eth.Bytes32
	copy(prevRandao[:], lastBlock.MixDigest().Bytes())

	var zero1559 eth.Bytes8
	minBaseFee := uint64(10)

	attrs := eth.PayloadAttributes{
		Timestamp:             eth.Uint64Quantity(blockTime),
		PrevRandao:            prevRandao,
		SuggestedFeeRecipient: lastBlock.Coinbase(),
		Withdrawals:           nil,
		ParentBeaconBlockRoot: lastBlock.ParentBeaconRoot(),
		Transactions:          []eth.Data{eth.Data(raw)},
		NoTxPool:              true,
		GasLimit:              &gasLimit,
		EIP1559Params:         &zero1559,
		MinBaseFee:            &minBaseFee,
	}

	_, err = opRethELNode.Escape().L2EthClient().PayloadExecutionWitness(ctx, common.Hash{}, attrs)
	if err == nil {
		gt.Fatal("expected error")
	}
}
