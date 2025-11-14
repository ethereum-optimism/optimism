package core

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/op-rs/op-geth/proofs/utils"
)

func simpleStorageSetValue(t devtest.T, parsedABI *abi.ABI, user *dsl.EOA, contractAddress common.Address, value *big.Int) *types.Receipt {
	ctx := t.Ctx()
	callData, err := parsedABI.Pack("setValue", value)
	if err != nil {
		t.Error("failed to pack set call data: %v", err)
		t.FailNow()
	}

	callTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&contractAddress), txplan.WithData(callData))
	callRes, err := callTx.Included.Eval(ctx)
	if err != nil {
		t.Error("failed to create set tx: %v", err)
		t.FailNow()
	}

	if callRes.Status != types.ReceiptStatusSuccessful {
		t.Error("set transaction failed")
		t.FailNow()
	}
	return callRes
}

func TestStorageProofUsingSimpleStorageContract(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSingleChainMultiNode(t)
	artifactPath := "../contracts/artifacts/SimpleStorage.sol/SimpleStorage.json"
	parsedABI, bin, err := utils.LoadArtifact(artifactPath)
	if err != nil {
		t.Error("failed to load artifact: %v", err)
		t.FailNow()
	}

	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)

	// deploy contract via helper
	contractAddress, blockNum, err := utils.DeployContract(ctx, user, bin)
	if err != nil {
		t.Error("failed to deploy contract: %v", err)
		t.FailNow()
	}
	t.Logf("contract deployed at address %s in L2 block %d", contractAddress.Hex(), blockNum)

	// fetch and verify initial proof (should be zeroed storage)
	utils.FetchAndVerifyProofs(t, sys, contractAddress, []common.Hash{common.HexToHash("0x0")}, blockNum)

	type caseEntry struct {
		Block uint64
		Value *big.Int
	}
	var cases []caseEntry
	for i := 1; i <= 5; i++ {
		writeVal := big.NewInt(int64(i * 10))
		callRes := simpleStorageSetValue(t, &parsedABI, user, contractAddress, writeVal)

		cases = append(cases, caseEntry{
			Block: callRes.BlockNumber.Uint64(),
			Value: writeVal,
		})
		t.Logf("setValue transaction included in L2 block %d", callRes.BlockNumber)
	}

	// test reset storage to zero
	callRes := simpleStorageSetValue(t, &parsedABI, user, contractAddress, big.NewInt(0))
	cases = append(cases, caseEntry{
		Block: callRes.BlockNumber.Uint64(),
		Value: big.NewInt(0),
	})
	t.Logf("reset setValue transaction included in L2 block %d", callRes.BlockNumber)

	// for each case, get proof and verify
	for _, c := range cases {
		utils.FetchAndVerifyProofs(t, sys, contractAddress, []common.Hash{common.HexToHash("0x0")}, c.Block)
	}

	// test with non-existent storage slot
	nonExistentSlot := common.HexToHash("0xdeadbeef")
	utils.FetchAndVerifyProofs(t, sys, contractAddress, []common.Hash{nonExistentSlot}, cases[len(cases)-1].Block)
}

func multiStorageSetValues(t devtest.T, parsedABI *abi.ABI, user *dsl.EOA, contractAddress common.Address, aVal, bVal *big.Int) *types.Receipt {
	ctx := t.Ctx()
	callData, err := parsedABI.Pack("setValues", aVal, bVal)
	if err != nil {
		t.Error("failed to pack set call data: %v", err)
		t.FailNow()
	}

	callTx := txplan.NewPlannedTx(user.Plan(), txplan.WithTo(&contractAddress), txplan.WithData(callData))
	callRes, err := callTx.Included.Eval(ctx)
	if err != nil {
		t.Error("failed to create set tx: %v", err)
		t.FailNow()
	}

	if callRes.Status != types.ReceiptStatusSuccessful {
		t.Error("set transaction failed")
		t.FailNow()
	}
	return callRes
}

func TestStorageProofUsingMultiStorageContract(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSingleChainMultiNode(t)
	artifactPath := "../contracts/artifacts/MultiStorage.sol/MultiStorage.json"
	parsedABI, bin, err := utils.LoadArtifact(artifactPath)
	if err != nil {
		t.Error("failed to load artifact: %v", err)
		t.FailNow()
	}

	user := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)

	// deploy contract via helper
	contractAddress, blockNum, err := utils.DeployContract(ctx, user, bin)
	if err != nil {
		t.Error("failed to deploy contract: %v", err)
		t.FailNow()
	}

	t.Logf("contract deployed at address %s in L2 block %d", contractAddress.Hex(), blockNum)

	// fetch and verify initial proof (should be zeroed storage)
	utils.FetchAndVerifyProofs(t, sys, contractAddress, []common.Hash{common.HexToHash("0x0"), common.HexToHash("0x1")}, blockNum)

	// set multiple storage slots
	type caseEntry struct {
		Block      uint64
		SlotValues map[common.Hash]*big.Int
	}
	var cases []caseEntry

	for i := 1; i <= 5; i++ {
		aVal := big.NewInt(int64(i * 10))
		bVal := big.NewInt(int64(i * 20))
		callRes := multiStorageSetValues(t, &parsedABI, user, contractAddress, aVal, bVal)

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
	callRes := multiStorageSetValues(t, &parsedABI, user, contractAddress, big.NewInt(0), big.NewInt(0))
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

		utils.FetchAndVerifyProofs(t, sys, contractAddress, slots, c.Block)
	}
}

// helper: compute mapping slot = keccak256(pad(key) ++ pad(slotIndex))
func mappingSlot(key common.Address, slotIndex uint64) common.Hash {
	keyBytes := common.LeftPadBytes(key.Bytes(), 32)
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(slotIndex).Bytes(), 32)
	return crypto.Keccak256Hash(append(keyBytes, slotBytes...))
}

// nested mapping: allowances[owner][spender] where `slotIndex` is the storage slot of allowances mapping
// innerSlot = keccak256(pad(owner) ++ pad(slotIndex))
// entrySlot = keccak256(pad(spender) ++ innerSlot)
func nestedMappingSlot(owner, spender common.Address, slotIndex uint64) common.Hash {
	ownerBytes := common.LeftPadBytes(owner.Bytes(), 32)
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(slotIndex).Bytes(), 32)
	inner := crypto.Keccak256(ownerBytes, slotBytes)
	spenderBytes := common.LeftPadBytes(spender.Bytes(), 32)
	return crypto.Keccak256Hash(append(spenderBytes, inner...))
}

// dynamic array element slot: element k stored at Big(keccak256(p)) + k
func arrayIndexSlot(slotIndex uint64, idx uint64) common.Hash {
	slotBytes := common.LeftPadBytes(new(big.Int).SetUint64(slotIndex).Bytes(), 32)
	base := crypto.Keccak256(slotBytes)
	baseInt := new(big.Int).SetBytes(base)
	elem := new(big.Int).Add(baseInt, new(big.Int).SetUint64(idx))
	return common.BigToHash(elem)
}

func TestTokenVaultStorageProofs(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSingleChainMultiNode(t)
	artifactPath := "../contracts/artifacts/TokenVault.sol/TokenVault.json"
	parsedABI, bin, err := utils.LoadArtifact(artifactPath)
	if err != nil {
		t.Errorf("failed to load artifact: %v", err)
		t.FailNow()
	}

	// funder EOA that will deploy / interact
	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.OneEther)

	// deploy contract
	contractAddr, deployBlock, err := utils.DeployContract(ctx, alice, bin)
	if err != nil {
		t.Errorf("deploy failed: %v", err)
		t.FailNow()
	}
	t.Logf("TokenVault deployed at %s block=%d", contractAddr.Hex(), deployBlock)

	userAddr := alice.Address()

	// call deposit (payable)
	depositAmount := eth.OneHundredthEther
	depositCalldata, err := parsedABI.Pack("deposit")
	if err != nil {
		t.Errorf("failed to pack deposit: %v", err)
		t.FailNow()
	}
	depTx := txplan.NewPlannedTx(alice.Plan(), txplan.WithTo(&contractAddr), txplan.WithData(depositCalldata), txplan.WithValue(depositAmount))
	depRes, err := depTx.Included.Eval(ctx)
	if err != nil {
		t.Errorf("deposit tx failed: %v", err)
		t.FailNow()
	}

	if depRes.Status != types.ReceiptStatusSuccessful {
		t.Error("set transaction failed")
		t.FailNow()
	}

	depositBlock := depRes.BlockNumber.Uint64()
	t.Logf("deposit included in block %d", depositBlock)

	// call approve(spender, amount) - use same user as spender for simplicity, or create another funded EOA
	approveAmount := big.NewInt(100)
	spenderAddr := bob.Address()
	approveCalldata, err := parsedABI.Pack("approve", spenderAddr, approveAmount)
	if err != nil {
		t.Errorf("failed to pack approve: %v", err)
		t.FailNow()
	}
	approveTx := txplan.NewPlannedTx(alice.Plan(), txplan.WithTo(&contractAddr), txplan.WithData(approveCalldata))
	approveRes, err := approveTx.Included.Eval(ctx)
	if err != nil {
		t.Errorf("approve tx failed: %v", err)
		t.FailNow()
	}

	if approveRes.Status != types.ReceiptStatusSuccessful {
		t.Error("approve transaction failed")
		t.FailNow()
	}

	approveBlock := approveRes.BlockNumber.Uint64()
	t.Logf("approve included in block %d", approveBlock)

	// call deactivateAllowance(spender)
	deactCalldata, err := parsedABI.Pack("deactivateAllowance", spenderAddr)
	if err != nil {
		t.Errorf("failed to pack deactivateAllowance: %v", err)
		t.FailNow()
	}
	deactTx := txplan.NewPlannedTx(alice.Plan(), txplan.WithTo(&contractAddr), txplan.WithData(deactCalldata))
	deactRes, err := deactTx.Included.Eval(ctx)
	if err != nil {
		t.Errorf("deactivateAllowance tx failed: %v", err)
		t.FailNow()
	}

	if deactRes.Status != types.ReceiptStatusSuccessful {
		t.Error("deactivateAllowance transaction failed")
		t.FailNow()
	}

	deactBlock := deactRes.BlockNumber.Uint64()
	t.Logf("deactivateAllowance included in block %d", deactBlock)

	// --- compute storage slots and verify proofs ---
	const pBalances = 0   // mapping(address => uint256) slot index
	const pAllowances = 1 // mapping(address => mapping(address => uint256)) slot index
	const pDepositors = 2 // dynamic array slot index

	// balance slot for user
	balanceSlot := mappingSlot(userAddr, pBalances)
	// nested allowance slot owner=user, spender=spenderAddr
	allowanceSlot := nestedMappingSlot(userAddr, spenderAddr, pAllowances)
	// depositors[0] element slot
	depositor0Slot := arrayIndexSlot(pDepositors, 0)

	// fetch & verify proofs at appropriate blocks
	// balance after deposit (depositBlock)
	t.Logf("Verifying balance slot %s at deposit block %d", balanceSlot.Hex(), depositBlock)
	utils.FetchAndVerifyProofs(t, sys, contractAddr, []common.Hash{balanceSlot, depositor0Slot}, depositBlock)
	// allowance after approve (approveBlock)
	t.Logf("Verifying allowance slot %s at approve block %d", allowanceSlot.Hex(), approveBlock)
	utils.FetchAndVerifyProofs(t, sys, contractAddr, []common.Hash{allowanceSlot}, approveBlock)
	// after deactivation, allowance should be zero at deactBlock
	t.Logf("Verifying allowance slot %s at deactivate block %d", allowanceSlot.Hex(), deactBlock)
	utils.FetchAndVerifyProofs(t, sys, contractAddr, []common.Hash{allowanceSlot}, deactBlock)
}
