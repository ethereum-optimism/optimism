package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestMissingGasLimit tests that op-geth cannot build a block without gas limit while optimism is active in the chain config.
func TestMissingGasLimit(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	attrs, err := opGeth.CreatePayloadAttributes()
	require.NoError(t, err)
	// Remove the GasLimit from the otherwise valid attributes
	attrs.GasLimit = nil

	res, err := opGeth.StartBlockBuilding(ctx, attrs)
	require.ErrorIs(t, err, eth.InputError{})
	require.Equal(t, eth.InvalidPayloadAttributes, err.(eth.InputError).Code)
	require.Nil(t, res)
}

// TestInvalidDepositInFCU runs an invalid deposit through a FCU/GetPayload/NewPayload/FCU set of calls.
// This tests that deposits must always allow the block to be built even if they are invalid.
func TestInvalidDepositInFCU(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	// Create a deposit from alice that will always fail (not enough funds)
	fromAddr := cfg.Secrets.Addresses().Alice
	balance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))

	badDepositTx := types.NewTx(&types.DepositTx{
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// We are inserting a block with an invalid deposit.
	// The invalid deposit should still remain in the block.
	_, err = opGeth.AddL2Block(ctx, badDepositTx)
	require.NoError(t, err)

	// Deposit tx was included, but Alice still shouldn't have any ETH
	balance, err = opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))
}

func TestBedrockSystemTxUsesZeroGas(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	block, err := opGeth.AddL2Block(ctx)
	require.NoError(t, err)
	infoTx, err := opGeth.L2Client.TransactionInBlock(ctx, block.BlockHash, 0)
	require.NoError(t, err)
	require.True(t, infoTx.IsSystemTx())
	receipt, err := opGeth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.Zero(t, receipt.GasUsed)
}

func TestBedrockDepositTx(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	aliceAddr := cfg.Secrets.Addresses().Alice

	// Deposit TX with a higher gas limit than required
	depositTx := types.NewTx(&types.DepositTx{
		From:                aliceAddr,
		To:                  &aliceAddr,
		Value:               big.NewInt(0),
		Gas:                 50_000, // Simple transfer only requires 21,000
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx := types.NewTx(&types.DepositTx{
		From:                aliceAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000001,
		Data:                []byte{},
		IsSystemTransaction: false,
	})

	_, err = opGeth.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)
	receipt, err := opGeth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx should succeed")
	require.Equal(t, depositTx.Gas(), receipt.GasUsed, "should use all gas")

	incorrectContractAddress := crypto.CreateAddress(aliceAddr, uint64(0)) // Expected to be wrong
	correctContractAddress := crypto.CreateAddress(aliceAddr, uint64(1))
	createRcpt, err := opGeth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status, "create should succeed")
	require.Equal(t, incorrectContractAddress, createRcpt.ContractAddress, "should report incorrect contract address")

	contractBalance, err := opGeth.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(0), contractBalance.Uint64(), "balance unchanged on incorrect contract address")

	contractBalance, err = opGeth.L2Client.BalanceAt(ctx, correctContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(params.Ether), contractBalance.Uint64(), "balance changed on correct contract address")
}

func TestBedrockShouldNotRefundDepositTxUnusedGas(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opGeth, err := NewOpGeth(t, ctx, &cfg)
	require.NoError(t, err)
	defer opGeth.Close()

	aliceAddr := cfg.Secrets.Addresses().Alice
	origBalance, err := opGeth.L2Client.BalanceAt(ctx, aliceAddr, nil)
	require.NoError(t, err)

	// Deposit TX with a higher gas limit than required
	depositTx := types.NewTx(&types.DepositTx{
		From:                aliceAddr,
		To:                  &aliceAddr,
		Value:               big.NewInt(0),
		Gas:                 50_000, // Simple transfer only requires 21,000
		IsSystemTransaction: false,
	})

	_, err = opGeth.AddL2Block(ctx, depositTx)
	require.NoError(t, err)
	receipt, err := opGeth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "tx should succeed")

	newBalance, err := opGeth.L2Client.BalanceAt(ctx, aliceAddr, nil)
	require.NoError(t, err)
	require.Equal(t, origBalance, newBalance, "should not refund cost of unused gas")
}
