package bridge

import (
	"bytes"
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestMintOnRevertedDeposit(t *testing.T) {
	op_e2e.InitParallel(t)
	cfg := e2esys.DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")

	l1Client := sys.NodeClient("l1")
	l2Verif := sys.NodeClient("sequencer")

	// create signer
	aliceKey := cfg.Secrets.Alice
	opts, err := bind.NewKeyedTransactorWithChainID(aliceKey, cfg.L1ChainIDBig())
	require.NoError(t, err)
	fromAddr := opts.From

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	startBalance, err := l2Verif.BalanceAt(ctx, fromAddr, nil)
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	startNonce, err := l2Verif.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	cancel()

	toAddr := common.Address{0xff, 0xff}
	mintAmount := big.NewInt(9_000_000)
	opts.Value = mintAmount
	helpers.SendDepositTx(t, cfg, l1Client, l2Verif, opts, func(l2Opts *helpers.DepositTxOpts) {
		l2Opts.ToAddr = toAddr
		// trigger a revert by transferring more than we have available
		l2Opts.Value = new(big.Int).Mul(common.Big2, startBalance)
		l2Opts.ExpectedStatus = types.ReceiptStatusFailed
	})

	// Confirm balance
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	endBalance, err := wait.ForBalanceChange(ctx, l2Verif, fromAddr, startBalance)
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	toAddrBalance, err := l2Verif.BalanceAt(ctx, toAddr, nil)
	cancel()
	require.NoError(t, err)

	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change")
	require.Equal(t, common.Big0.Int64(), toAddrBalance.Int64(), "The recipient account balance should be zero")

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	endNonce, err := l2Verif.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	cancel()
	require.Equal(t, startNonce+1, endNonce, "Nonce of deposit sender should increment on L2, even if the deposit fails")
}

var (
	deployPrefixSize = byte(16)
	deployPrefix     = []byte{
		// Copy input data after this prefix into memory starting at address 0x00
		// CODECOPY arg size
		byte(vm.PUSH1), deployPrefixSize,
		byte(vm.CODESIZE),
		byte(vm.SUB),
		// CODECOPY arg offset
		byte(vm.PUSH1), deployPrefixSize,
		// CODECOPY arg destOffset
		byte(vm.PUSH1), 0x00,
		byte(vm.CODECOPY),

		// Return code from memory
		// RETURN arg size
		byte(vm.PUSH1), deployPrefixSize,
		byte(vm.CODESIZE),
		byte(vm.SUB),
		// RETURN arg offset
		byte(vm.PUSH1), 0x00,
		byte(vm.RETURN),
	}
	sstoreContract = []byte{
		// Load first word from call data
		byte(vm.PUSH1), 0x00,
		byte(vm.CALLDATALOAD),

		// Store it to slot 0
		byte(vm.PUSH1), 0x00,
		byte(vm.SSTORE),
	}
)

func TestMintCallToDelegatedAccount(t *testing.T) {
	// This test:
	// 1. Deploys a contract on L2 that stores the first word of the call data to storage
	// 2. Sets the delegation designation for an account on L2 to the new contract with nonzero calldata and ensures this is set in the contract
	// 3. Sends a deposit tx (using the L1 bridge) to the L2 account with the delegation code
	// 4. Ensures the deposit properly calls the contract and stores the first word of the call data to storage
	// 5. Ensures the deposit sender's nonce is incremented on L2
	// 6. Ensures the deposit recipient's balance is incremented on L2

	op_e2e.InitParallel(t)
	cfg := e2esys.DefaultSystemConfig(t)
	cfg.DeployConfig = cfg.DeployConfig.Copy()
	cfg.DeployConfig.ActivateForkAtGenesis(rollup.Isthmus)
	delete(cfg.Nodes, "verifier")
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")

	l1Client := sys.NodeClient("l1")
	l2Seq := sys.NodeClient("sequencer")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Simple constructor that is prefixed to the actual contract code
	// Results in the contract code being returned as the code for the new contract
	deployData := append(deployPrefix, sstoreContract...)
	signer := types.NewIsthmusSigner(cfg.L2ChainIDBig())

	tx := types.MustSignNewTx(cfg.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainIDBig(),
		Nonce:     0,
		To:        nil,
		Value:     big.NewInt(0),
		Gas:       500000,
		GasFeeCap: big.NewInt(5000000000),
		GasTipCap: big.NewInt(2),
		Data:      deployData,
	})

	err = l2Seq.SendTransaction(ctx, tx)
	require.NoError(t, err, "Failed to send contract deployment transaction")

	_, err = wait.ForReceipt(ctx, l2Seq, tx.Hash(), 1)
	require.NoError(t, err, "Failed to get receipt for set code transaction")

	contractAddr := crypto.CreateAddress(cfg.Secrets.Addresses().Alice, 0)

	// validate codeat
	code, err := l2Seq.PendingCodeAt(ctx, contractAddr)
	require.NoError(t, err, "Failed to get code at contract address")
	if !bytes.Equal(code, sstoreContract) {
		t.Fatalf("Code at contract address is incorrect: got %s, want %s", common.Bytes2Hex(code), common.Bytes2Hex(sstoreContract))
	}

	// validate slot 0 is empty
	slot0, err := l2Seq.StorageAt(ctx, contractAddr, common.Hash{0x00}, nil)
	require.NoError(t, err, "Failed to get storage at slot 0")
	require.Equal(t, make([]byte, 32), slot0, "Slot 0 should be empty")

	// set delegation designation for an account on L2 to new contract
	auth1, err := types.SignSetCode(cfg.Secrets.Bob, types.SetCodeAuthorization{
		Address: contractAddr,
		Nonce:   1,
	})

	require.NoError(t, err, "Failed to sign set code authorization")

	// Set slot0 to 0x01 first
	txdata := &types.SetCodeTx{
		ChainID:   uint256.MustFromBig(cfg.L2ChainIDBig()),
		Nonce:     0,
		To:        cfg.Secrets.Addresses().Bob,
		Gas:       500000,
		GasFeeCap: uint256.NewInt(5000000000),
		GasTipCap: uint256.NewInt(2),
		AuthList:  []types.SetCodeAuthorization{auth1},
		Data:      []byte{0x01},
	}

	tx = types.MustSignNewTx(cfg.Secrets.Bob, signer, txdata)

	err = l2Seq.SendTransaction(ctx, tx)
	require.NoError(t, err, "Failed to send set code transaction")

	_, err = wait.ForReceipt(ctx, l2Seq, tx.Hash(), 1)
	require.NoError(t, err, "Failed to get receipt for set code transaction")

	// ensure the delegation was set
	delegation, err := l2Seq.CodeAt(ctx, cfg.Secrets.Addresses().Bob, nil)
	require.NoError(t, err, "Failed to get delegation code")
	want := types.AddressToDelegation(auth1.Address)
	if !bytes.Equal(delegation, want) {
		t.Fatalf("addr1 code incorrect: got %s, want %s", common.Bytes2Hex(delegation), common.Bytes2Hex(want))
	}

	// ensure the code was executed correctly
	slot0, err = l2Seq.StorageAt(ctx, cfg.Secrets.Addresses().Bob, common.Hash{0x00}, nil)
	require.NoError(t, err, "Failed to get storage at slot 0")
	require.Equal(t, byte(0x01), slot0[0], "The first word of the call data should be stored in slot 0")

	aliceKey := cfg.Secrets.Alice
	opts, err := bind.NewKeyedTransactorWithChainID(aliceKey, cfg.L1ChainIDBig())
	require.NoError(t, err)
	fromAddr := opts.From

	startBalance, err := l2Seq.BalanceAt(ctx, cfg.Secrets.Addresses().Bob, nil)
	require.NoError(t, err)

	startNonce, err := l2Seq.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	// send a deposit to bob with the delegation code and setting slot 0 to 0x42
	toAddr := cfg.Secrets.Addresses().Bob
	mintAmount := big.NewInt(9_000_000)
	opts.Value = mintAmount
	helpers.SendDepositTx(t, cfg, l1Client, l2Seq, opts, func(l2Opts *helpers.DepositTxOpts) {
		l2Opts.ToAddr = toAddr
		l2Opts.Data = []byte{0x42}
	})

	endBalance, err := wait.ForBalanceChange(ctx, l2Seq, toAddr, startBalance)
	require.NoError(t, err)

	// Bob balance should have increased by the mint amount
	diff := new(big.Int)
	diff = diff.Sub(endBalance, startBalance)
	require.Equal(t, mintAmount, diff, "Did not get expected balance change")

	// Bob slot 0 should be 0x42
	slot0, err = l2Seq.StorageAt(ctx, cfg.Secrets.Addresses().Bob, common.Hash{0x00}, nil)
	require.NoError(t, err, "Failed to get storage at slot 0")
	require.Equal(t, byte(0x42), slot0[0], "The first word of the call data should be stored in slot 0")

	// From nonce should have increased as a result
	endNonce, err := l2Seq.NonceAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	require.Equal(t, startNonce+1, endNonce, "Nonce of deposit sender should increment on L2, even if the deposit fails")
}

func TestDepositTxCreateContract(t *testing.T) {
	op_e2e.InitParallel(t)
	cfg := e2esys.DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")

	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")

	l1Client := sys.NodeClient("l1")
	l2Client := sys.NodeClient("sequencer")

	opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.NoError(t, err)

	deployData := append(deployPrefix, sstoreContract...)

	l2Receipt := helpers.SendDepositTx(t, cfg, l1Client, l2Client, opts, func(l2Opts *helpers.DepositTxOpts) {
		l2Opts.Data = deployData
		l2Opts.Value = common.Big0
		l2Opts.IsCreation = true
		l2Opts.ToAddr = common.Address{}
		l2Opts.GasLimit = 1_000_000
	})
	require.NotEqual(t, common.Address{}, l2Receipt.ContractAddress, "should not have zero address")
	code, err := l2Client.CodeAt(context.Background(), l2Receipt.ContractAddress, nil)
	require.NoError(t, err, "get deployed contract code")
	require.Equal(t, sstoreContract, code, "should have deployed correct contract code")
}
