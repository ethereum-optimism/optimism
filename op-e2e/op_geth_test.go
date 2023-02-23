package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)

	attrs, err := l2Geth.CreatePayloadAttributes()
	require.NoError(t, err)
	// Remove the GasLimit from the otherwise valid attributes
	attrs.GasLimit = nil

	res, err := l2Geth.StartBlockBuilding(ctx, attrs)
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
	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)

	// Create a deposit from alice that will always fail (not enough funds)
	fromAddr := cfg.Secrets.Addresses().Alice
	balance, err := l2Geth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))

	badDepositTx := types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// We are inserting a block with an invalid deposit.
	// The invalid deposit should still remain in the block.
	_, err = l2Geth.AddL2Block(ctx, badDepositTx)
	require.NoError(t, err)

	// Deposit tx was included, but Alice still shouldn't have any ETH
	balance, err = l2Geth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))
}

// TestActivateRegolithAtGenesis runs deposit transactions on a chain with Regolith enabled at genesis
func TestActivateRegolithAtGenesis(t *testing.T) {
	// Setup an L2 EE and create a client connection to the engine.
	// We also need to setup a L1 Genesis to create the rollup genesis.
	cfg := DefaultSystemConfig(t)
	regolithTime := hexutil.Uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &regolithTime

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)
	defer l2Geth.Close()

	fromAddr := cfg.Secrets.Addresses().Alice

	// Simple transfer deposit tx
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx := types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000000,
		Data:                []byte{},
		IsSystemTransaction: false,
	})

	payload, err := l2Geth.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err := l2Geth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.NotEqual(t, depositTx.Gas(), receipt.GasUsed)
	require.Equal(t, uint64(0), *receipt.DepositNonce)

	infoTx, err := l2Geth.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	infoRcpt, err := l2Geth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.NotZero(t, infoRcpt.GasUsed)

	expectedContractAddress := crypto.CreateAddress(fromAddr, uint64(1))
	createRcpt, err := l2Geth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)
	require.Equal(t, uint64(1), *createRcpt.DepositNonce)
	contractBalance, err := l2Geth.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, contractCreateTx.Value(), contractBalance)
}

// TestActivateRegolithAtGenesis runs deposit transactions on a chain with Regolith enabled from block 2
func TestActivateRegolithAfterGenesis(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	regolithTime := hexutil.Uint64(4)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &regolithTime

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)
	defer l2Geth.Close()

	fromAddr := cfg.Secrets.Addresses().Alice

	// Simple transfer deposit tx
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx := types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000000,
		Data:                []byte{},
		IsSystemTransaction: false,
	})

	// First block is still in bedrock
	payload, err := l2Geth.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err := l2Geth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.Equal(t, depositTx.Gas(), receipt.GasUsed)

	infoTx, err := l2Geth.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	require.True(t, infoTx.IsSystemTx(), "should use system tx in bedrock")
	infoRcpt, err := l2Geth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.Zero(t, infoRcpt.GasUsed)

	expectedContractAddress := crypto.CreateAddress(fromAddr, uint64(0)) // Expected to be wrong
	createRcpt, err := l2Geth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)

	contractBalance, err := l2Geth.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(0), contractBalance.Uint64())

	// Second block is in regolith
	// Simple transfer deposit tx
	depositTx = types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25001,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx = types.NewTx(&types.DepositTx{
		SourceHash:          l2Geth.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000001,
		Data:                []byte{},
		IsSystemTransaction: false,
	})
	payload, err = l2Geth.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err = l2Geth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.NotEqual(t, depositTx.Gas(), receipt.GasUsed)

	infoTx, err = l2Geth.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	infoRcpt, err = l2Geth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.NotZero(t, infoRcpt.GasUsed)

	expectedContractAddress = crypto.CreateAddress(fromAddr, uint64(3))
	createRcpt, err = l2Geth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)

	contractBalance, err = l2Geth.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, contractCreateTx.Value(), contractBalance)

	tx, _, err := l2Geth.L2Client.TransactionByHash(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, uint64(3), tx.Nonce())
}

// TestRegolithDepositTxUnusedGas checks that unused gas from deposit transactions is returned to the block gas pool
// Also checks the user is not refunded for unused gas.
func TestRegolithDepositTxUnusedGas(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	regolithTime := hexutil.Uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &regolithTime

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)
	defer l2Geth.Close()

	fromAddr := cfg.Secrets.Addresses().Alice

	aliceBalance, err := l2Geth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	// Deposit TX with a high gas limit but using very little actual gas
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash: l2Geth.L1Head.Hash(),
		From:       fromAddr,
		To:         &fromAddr, // send it to ourselves
		Value:      big.NewInt(params.Ether),
		// SystemTx is assigned 1M gas limit
		Gas:                 uint64(cfg.DeployConfig.L2GenesisBlockGasLimit) - 1_000_000,
		IsSystemTransaction: false,
	})

	signer := types.LatestSigner(l2Geth.L2ChainConfig)
	// Second deposit tx with a gas limit that will fit in regolith but not bedrock
	tx := types.MustSignNewTx(cfg.Secrets.Bob, signer, &types.DynamicFeeTx{
		ChainID:   big.NewInt(int64(cfg.DeployConfig.L2ChainID)),
		Nonce:     0,
		GasTipCap: big.NewInt(100),
		GasFeeCap: big.NewInt(100000),
		Gas:       1_000_001,
		To:        &cfg.Secrets.Addresses().Alice,
		Value:     big.NewInt(0),
		Data:      nil,
	})

	_, err = l2Geth.AddL2Block(ctx, depositTx, tx)
	require.NoError(t, err)

	newAliceBalance, err := l2Geth.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Equal(t, aliceBalance, newAliceBalance, "should not refund fee for unused gas")
}

// TestRegolithRejectsSystemTx checks that IsSystemTx must be false after Regolith
func TestRegolithRejectsSystemTx(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	regolithTime := hexutil.Uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &regolithTime

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	l2Geth, err := NewL2Geth(t, ctx, &cfg)
	require.NoError(t, err)
	defer l2Geth.Close()

	systemTx, err := derive.L1InfoDeposit(1, l2Geth.L1Head, l2Geth.SystemConfig)
	systemTx.IsSystemTransaction = true
	require.NoError(t, err)

	_, err = l2Geth.AddL2Block(ctx, types.NewTx(systemTx))
	require.ErrorIs(t, err, ErrNewPayloadNotValid)
}

// TODO: Need a test to check that gas refunds (eg for selfdestruct) are included in the receipt gasUsed
