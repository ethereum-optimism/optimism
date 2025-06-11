package bridge

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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

	// Simple constructor that is prefixed to the actual contract code
	// Results in the contract code being returned as the code for the new contract
	deployPrefixSize := byte(16)
	deployPrefix := []byte{
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
	// Stores the first word from call data code to storage slot 0
	sstoreContract := []byte{
		// Load first word from call data
		byte(vm.PUSH1), 0x00,
		byte(vm.CALLDATALOAD),

		// Store it to slot 0
		byte(vm.PUSH1), 0x00,
		byte(vm.SSTORE),
	}

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
