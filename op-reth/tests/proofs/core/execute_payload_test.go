package core

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/op-rs/op-geth/proofs/utils"
)

func TestExecutePayloadSuccess(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()
	sys := presets.NewSingleChainMultiNode(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	opRethELNode, _ := utils.IdentifyELNodes(sys.L2EL, sys.L2ELB)

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

	blockTime := lastBlock.Time() + 1
	gasLimit := eth.Uint64Quantity(lastBlock.GasLimit())

	var prevRandao eth.Bytes32
	copy(prevRandao[:], lastBlock.MixDigest().Bytes())

	var zero1559 eth.Bytes8

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
		MinBaseFee:            nil,
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
	sys := presets.NewSingleChainMultiNode(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	opRethELNode, _ := utils.IdentifyELNodes(sys.L2EL, sys.L2ELB)

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

	blockTime := lastBlock.Time() + 1
	gasLimit := eth.Uint64Quantity(lastBlock.GasLimit())

	var prevRandao eth.Bytes32
	copy(prevRandao[:], lastBlock.MixDigest().Bytes())

	var zero1559 eth.Bytes8

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
		MinBaseFee:            nil,
	}

	_, err = opRethELNode.Escape().L2EthClient().PayloadExecutionWitness(ctx, common.Hash{}, attrs)
	if err == nil {
		gt.Fatal("expected error")
	}
}
