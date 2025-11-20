package core

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/op-rs/op-geth/proofs/utils"
)

func TestStorageProofUsingSimpleStorageContract(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSingleChainMultiNode(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)

	// deploy contract via helper
	contract, receipt := utils.DeploySimpleStorage(t, user)
	t.Logf("contract deployed at address %s in L2 block %d", contract.Address().Hex(), receipt.BlockNumber.Uint64())

	// fetch and verify initial proof (should be zeroed storage)
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{common.HexToHash("0x0")}, receipt.BlockNumber.Uint64())

	type caseEntry struct {
		Block uint64
		Value *big.Int
	}
	var cases []caseEntry
	for i := 1; i <= 5; i++ {
		writeVal := big.NewInt(int64(i * 10))
		callRes := contract.SetValue(user, writeVal)

		cases = append(cases, caseEntry{
			Block: callRes.BlockNumber.Uint64(),
			Value: writeVal,
		})
		t.Logf("setValue transaction included in L2 block %d", callRes.BlockNumber)
	}

	// test reset storage to zero
	callRes := contract.SetValue(user, big.NewInt(0))
	cases = append(cases, caseEntry{
		Block: callRes.BlockNumber.Uint64(),
		Value: big.NewInt(0),
	})
	t.Logf("reset setValue transaction included in L2 block %d", callRes.BlockNumber)

	// for each case, get proof and verify
	for _, c := range cases {
		utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{common.HexToHash("0x0")}, c.Block)
	}

	// test with non-existent storage slot
	nonExistentSlot := common.HexToHash("0xdeadbeef")
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{nonExistentSlot}, cases[len(cases)-1].Block)
}

func TestStorageProofUsingMultiStorageContract(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSingleChainMultiNode(t)
	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)

	// deploy contract via helper
	contract, receipt := utils.DeployMultiStorage(t, user)
	t.Logf("contract deployed at address %s in L2 block %d", contract.Address().Hex(), receipt.BlockNumber.Uint64())

	// fetch and verify initial proof (should be zeroed storage)
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}, receipt.BlockNumber.Uint64())

	// set multiple storage slots
	type caseEntry struct {
		Block      uint64
		SlotValues map[common.Hash]*big.Int
	}
	var cases []caseEntry

	for i := 1; i <= 5; i++ {
		aVal := big.NewInt(int64(i * 10))
		bVal := big.NewInt(int64(i * 20))
		callRes := contract.SetValues(user, aVal, bVal)

		cases = append(cases, caseEntry{
			Block: callRes.BlockNumber.Uint64(),
			SlotValues: map[common.Hash]*big.Int{
				common.HexToHash("0x0"): aVal,
				common.HexToHash("0x1"): bVal,
			},
		})
		t.Logf("setValues transaction included in L2 block %d", callRes.BlockNumber)
	}

	// test reset storage slots to zero
	callRes := contract.SetValues(user, big.NewInt(0), big.NewInt(0))
	cases = append(cases, caseEntry{
		Block: callRes.BlockNumber.Uint64(),
		SlotValues: map[common.Hash]*big.Int{
			common.HexToHash("0x0"): big.NewInt(0),
			common.HexToHash("0x1"): big.NewInt(0),
		},
	})
	t.Logf("reset setValues transaction included in L2 block %d", callRes.BlockNumber)

	// for each case, get proof and verify
	for _, c := range cases {
		var slots []common.Hash
		for slot := range c.SlotValues {
			slots = append(slots, slot)
		}

		utils.FetchAndVerifyProofs(t, sys, contract.Address(), slots, c.Block)
	}
}

func TestTokenVaultStorageProofs(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSingleChainMultiNode(t)
	// funder EOA that will deploy / interact
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.OneEther)

	// deploy contract
	contract, deployBlock := utils.DeployTokenVault(t, alice)
	t.Logf("TokenVault deployed at %s block=%d", contract.Address().Hex(), deployBlock.BlockNumber.Uint64())

	userAddr := alice.Address()

	// call deposit (payable)
	depositAmount := eth.OneHundredthEther
	depRes := contract.Deposit(alice, depositAmount)
	depositBlock := depRes.BlockNumber.Uint64()
	t.Logf("deposit included in block %d", depositBlock)

	// call approve(spender, amount) - use same user as spender for simplicity, or create another funded EOA
	approveAmount := big.NewInt(100)
	spenderAddr := bob.Address()
	approveRes := contract.Approve(alice, spenderAddr, approveAmount)
	approveBlock := approveRes.BlockNumber.Uint64()
	t.Logf("approve included in block %d", approveBlock)

	// call deactivateAllowance(spender)
	deactRes := contract.DeactivateAllowance(alice, spenderAddr)
	deactBlock := deactRes.BlockNumber.Uint64()
	t.Logf("deactivateAllowance included in block %d", deactBlock)

	// balance slot for user
	balanceSlot := contract.GetBalanceSlot(userAddr)
	// nested allowance slot owner=user, spender=spenderAddr
	allowanceSlot := contract.GetAllowanceSlot(userAddr, spenderAddr)
	// depositors[0] element slot
	depositor0Slot := contract.GetDepositorSlot(0)

	// fetch & verify proofs at appropriate blocks
	// balance after deposit (depositBlock)
	t.Logf("Verifying balance slot %s at deposit block %d", balanceSlot.Hex(), depositBlock)
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{balanceSlot, depositor0Slot}, depositBlock)
	// allowance after approve (approveBlock)
	t.Logf("Verifying allowance slot %s at approve block %d", allowanceSlot.Hex(), approveBlock)
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{allowanceSlot}, approveBlock)
	// after deactivation, allowance should be zero at deactBlock
	t.Logf("Verifying allowance slot %s at deactivate block %d", allowanceSlot.Hex(), deactBlock)
	utils.FetchAndVerifyProofs(t, sys, contract.Address(), []common.Hash{allowanceSlot}, deactBlock)
}
