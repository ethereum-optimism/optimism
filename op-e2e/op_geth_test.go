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
	"github.com/ethereum/go-ethereum/core/vm"
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

func TestPreregolith(t *testing.T) {
	futureTimestamp := hexutil.Uint64(4)
	tests := []struct {
		name         string
		regolithTime *hexutil.Uint64
	}{
		{name: "RegolithNotScheduled"},
		{name: "RegolithNotYetActive", regolithTime: &futureTimestamp},
	}
	for _, test := range tests {
		test := test
		t.Run("GasUsed_"+test.name, func(t *testing.T) {
			// Setup an L2 EE and create a client connection to the engine.
			// We also need to setup a L1 Genesis to create the rollup genesis.
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			fromAddr := cfg.Secrets.Addresses().Alice

			oldBalance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
			require.NoError(t, err)

			// Simple transfer deposit tx
			depositTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &fromAddr, // send it to ourselves
				Value:               big.NewInt(params.Ether),
				Gas:                 25000,
				IsSystemTransaction: false,
			})

			block, err := opGeth.AddL2Block(ctx, depositTx)
			require.NoError(t, err)

			// L1Info tx should report 0 gas used
			infoTx, err := opGeth.L2Client.TransactionInBlock(ctx, block.BlockHash, 0)
			require.NoError(t, err)
			infoRcpt, err := opGeth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
			require.NoError(t, err)
			require.Zero(t, infoRcpt.GasUsed, "should use 0 gas for system tx")

			// Deposit tx should report all gas used
			receipt, err := opGeth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
			require.NoError(t, err)
			require.Equal(t, depositTx.Gas(), receipt.GasUsed, "should report all gas used")

			// Should not refund ETH for unused gas
			newBalance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
			require.NoError(t, err)
			require.Equal(t, oldBalance, newBalance, "should not repay sender for unused gas")
		})

		t.Run("DepositNonce_"+test.name, func(t *testing.T) {
			// Setup an L2 EE and create a client connection to the engine.
			// We also need to setup a L1 Genesis to create the rollup genesis.
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			fromAddr := cfg.Secrets.Addresses().Alice
			// Include a tx just to ensure Alice's nonce isn't 0
			incrementNonceTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &fromAddr,
				Value:               big.NewInt(0),
				Gas:                 21_000,
				IsSystemTransaction: false,
			})

			// Contract creation deposit tx
			contractCreateTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				Value:               big.NewInt(params.Ether),
				Gas:                 1000001,
				Data:                []byte{},
				IsSystemTransaction: false,
			})

			_, err = opGeth.AddL2Block(ctx, incrementNonceTx, contractCreateTx)
			require.NoError(t, err)

			expectedNonce := uint64(1)
			incorrectContractAddress := crypto.CreateAddress(fromAddr, uint64(0))
			correctContractAddress := crypto.CreateAddress(fromAddr, expectedNonce)
			createRcpt, err := opGeth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status, "create should succeed")
			require.Nil(t, createRcpt.DepositNonce, "should not report deposit nonce")
			require.Equal(t, incorrectContractAddress, createRcpt.ContractAddress, "should report correct contract address")

			contractBalance, err := opGeth.L2Client.BalanceAt(ctx, incorrectContractAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(0), contractBalance.Uint64(), "balance unchanged on incorrect contract address")

			contractBalance, err = opGeth.L2Client.BalanceAt(ctx, correctContractAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(params.Ether), contractBalance.Uint64(), "balance changed on correct contract address")

			// Check the actual transaction nonce is reported correctly when retrieving the tx from the API.
			tx, _, err := opGeth.L2Client.TransactionByHash(ctx, contractCreateTx.Hash())
			require.NoError(t, err)
			require.Zero(t, *tx.EffectiveNonce(), "should report 0 as tx nonce")
		})

		t.Run("UnusedGasConsumed_"+test.name, func(t *testing.T) {
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			fromAddr := cfg.Secrets.Addresses().Alice

			// Deposit TX with a high gas limit but using very little actual gas
			depositTx := types.NewTx(&types.DepositTx{
				From:  fromAddr,
				To:    &fromAddr, // send it to ourselves
				Value: big.NewInt(params.Ether),
				// SystemTx is assigned 1M gas limit
				Gas:                 uint64(cfg.DeployConfig.L2GenesisBlockGasLimit) - 1_000_000,
				IsSystemTransaction: false,
			})

			signer := types.LatestSigner(opGeth.L2ChainConfig)
			// Second tx with a gas limit that will fit in regolith but not bedrock
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

			_, err = opGeth.AddL2Block(ctx, depositTx, tx)
			// Geth checks the gas limit usage of transactions as part of validating the payload attributes and refuses to even start building the block
			require.ErrorContains(t, err, "Invalid payload attributes", "block should be invalid due to using too much gas")
		})

		t.Run("AllowSystemTx_"+test.name, func(t *testing.T) {
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			systemTx, err := derive.L1InfoDeposit(1, opGeth.L1Head, opGeth.SystemConfig, false)
			systemTx.IsSystemTransaction = true
			require.NoError(t, err)

			_, err = opGeth.AddL2Block(ctx, types.NewTx(systemTx))
			require.NoError(t, err, "should allow blocks containing system tx")
		})
	}
}

func TestRegolith(t *testing.T) {
	tests := []struct {
		name             string
		regolithTime     hexutil.Uint64
		activateRegolith func(ctx context.Context, opGeth *OpGeth)
	}{
		{name: "ActivateAtGenesis", regolithTime: 0, activateRegolith: func(ctx context.Context, opGeth *OpGeth) {}},
		{name: "ActivateAfterGenesis", regolithTime: 2, activateRegolith: func(ctx context.Context, opGeth *OpGeth) {
			_, err := opGeth.AddL2Block(ctx)
			require.NoError(t, err)
		}},
	}
	for _, test := range tests {
		test := test
		t.Run("GasUsedIsAccurate_"+test.name, func(t *testing.T) {
			// Setup an L2 EE and create a client connection to the engine.
			// We also need to setup a L1 Genesis to create the rollup genesis.
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = &test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			test.activateRegolith(ctx, opGeth)

			fromAddr := cfg.Secrets.Addresses().Alice

			oldBalance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
			require.NoError(t, err)

			// Simple transfer deposit tx
			depositTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &fromAddr, // send it to ourselves
				Value:               big.NewInt(params.Ether),
				Gas:                 25000,
				IsSystemTransaction: false,
			})

			block, err := opGeth.AddL2Block(ctx, depositTx)
			require.NoError(t, err)

			// L1Info tx should report actual gas used, not 0 or the tx gas limit
			infoTx, err := opGeth.L2Client.TransactionInBlock(ctx, block.BlockHash, 0)
			require.NoError(t, err)
			infoRcpt, err := opGeth.L2Client.TransactionReceipt(ctx, infoTx.Hash())
			require.NoError(t, err)
			require.NotZero(t, infoRcpt.GasUsed)
			require.NotEqual(t, infoTx.Gas(), infoRcpt.GasUsed)

			// Deposit tx should report actual gas used (21,000 for a normal transfer)
			receipt, err := opGeth.L2Client.TransactionReceipt(ctx, depositTx.Hash())
			require.NoError(t, err)
			require.Equal(t, uint64(21_000), receipt.GasUsed, "should report actual gas used")

			// Should not refund ETH for unused gas
			newBalance, err := opGeth.L2Client.BalanceAt(ctx, fromAddr, nil)
			require.NoError(t, err)
			require.Equal(t, oldBalance, newBalance, "should not repay sender for unused gas")
		})

		t.Run("DepositNonceCorrect_"+test.name, func(t *testing.T) {
			// Setup an L2 EE and create a client connection to the engine.
			// We also need to setup a L1 Genesis to create the rollup genesis.
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = &test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			test.activateRegolith(ctx, opGeth)

			fromAddr := cfg.Secrets.Addresses().Alice
			// Include a tx just to ensure Alice's nonce isn't 0
			incrementNonceTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &fromAddr,
				Value:               big.NewInt(0),
				Gas:                 21_000,
				IsSystemTransaction: false,
			})

			// Contract creation deposit tx
			contractCreateTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				Value:               big.NewInt(params.Ether),
				Gas:                 1000001,
				Data:                []byte{},
				IsSystemTransaction: false,
			})

			_, err = opGeth.AddL2Block(ctx, incrementNonceTx, contractCreateTx)
			require.NoError(t, err)

			expectedNonce := uint64(1)
			correctContractAddress := crypto.CreateAddress(fromAddr, expectedNonce)
			createRcpt, err := opGeth.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status, "create should succeed")
			require.Equal(t, &expectedNonce, createRcpt.DepositNonce, "should report correct deposit nonce")
			require.Equal(t, correctContractAddress, createRcpt.ContractAddress, "should report correct contract address")

			contractBalance, err := opGeth.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
			require.NoError(t, err)
			require.Equal(t, uint64(params.Ether), contractBalance.Uint64(), "balance changed on correct contract address")

			// Check the actual transaction nonce is reported correctly when retrieving the tx from the API.
			tx, _, err := opGeth.L2Client.TransactionByHash(ctx, contractCreateTx.Hash())
			require.NoError(t, err)
			require.Equal(t, expectedNonce, *tx.EffectiveNonce(), "should report actual tx nonce")
		})

		t.Run("ReturnUnusedGasToPool_"+test.name, func(t *testing.T) {
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = &test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			test.activateRegolith(ctx, opGeth)

			fromAddr := cfg.Secrets.Addresses().Alice

			// Deposit TX with a high gas limit but using very little actual gas
			depositTx := types.NewTx(&types.DepositTx{
				From:  fromAddr,
				To:    &fromAddr, // send it to ourselves
				Value: big.NewInt(params.Ether),
				// SystemTx is assigned 1M gas limit
				Gas:                 uint64(cfg.DeployConfig.L2GenesisBlockGasLimit) - 1_000_000,
				IsSystemTransaction: false,
			})

			signer := types.LatestSigner(opGeth.L2ChainConfig)
			// Second tx with a gas limit that will fit in regolith but not bedrock
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

			_, err = opGeth.AddL2Block(ctx, depositTx, tx)
			require.NoError(t, err, "block should be valid as cumulativeGasUsed only tracks actual usage now")
		})

		t.Run("RejectSystemTx_"+test.name, func(t *testing.T) {
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = &test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			test.activateRegolith(ctx, opGeth)

			systemTx, err := derive.L1InfoDeposit(1, opGeth.L1Head, opGeth.SystemConfig, false)
			systemTx.IsSystemTransaction = true
			require.NoError(t, err)

			_, err = opGeth.AddL2Block(ctx, types.NewTx(systemTx))
			require.ErrorIs(t, err, ErrNewPayloadNotValid, "should reject blocks containing system tx")
		})

		t.Run("IncludeGasRefunds_"+test.name, func(t *testing.T) {
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

			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.L2GenesisRegolithTimeOffset = &test.regolithTime

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			opGeth, err := NewOpGeth(t, ctx, &cfg)
			require.NoError(t, err)
			defer opGeth.Close()

			test.activateRegolith(ctx, opGeth)
			fromAddr := cfg.Secrets.Addresses().Alice
			storeContractAddr := crypto.CreateAddress(fromAddr, 0)

			// Deposit TX to deploy a contract that lets us store an arbitrary value
			deployTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				Value:               common.Big0,
				Data:                deployData,
				Gas:                 1_000_000,
				IsSystemTransaction: false,
			})

			// Store a non-zero value
			storeTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &storeContractAddr,
				Value:               common.Big0,
				Data:                []byte{0x06},
				Gas:                 1_000_000,
				IsSystemTransaction: false,
			})

			// Store a non-zero value
			zeroTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &storeContractAddr,
				Value:               common.Big0,
				Data:                []byte{0x00},
				Gas:                 1_000_000,
				IsSystemTransaction: false,
			})

			// Store a non-zero value again
			// Has same gas cost as zeroTx, except the first tx gets a gas refund for clearing the storage slot
			rezeroTx := types.NewTx(&types.DepositTx{
				From:                fromAddr,
				To:                  &storeContractAddr,
				Value:               common.Big0,
				Data:                []byte{0x00},
				Gas:                 1_000_001,
				IsSystemTransaction: false,
			})

			_, err = opGeth.AddL2Block(ctx, deployTx, storeTx, zeroTx, rezeroTx)
			require.NoError(t, err)

			// Sanity check the contract code deployed correctly
			code, err := opGeth.L2Client.CodeAt(ctx, storeContractAddr, nil)
			require.NoError(t, err)
			require.Equal(t, sstoreContract, code, "should create contract with expected code")

			deployReceipt, err := opGeth.L2Client.TransactionReceipt(ctx, deployTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, deployReceipt.Status)
			require.Equal(t, storeContractAddr, deployReceipt.ContractAddress, "should create contract at expected address")

			storeReceipt, err := opGeth.L2Client.TransactionReceipt(ctx, storeTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, storeReceipt.Status, "setting storage value should succeed")

			zeroReceipt, err := opGeth.L2Client.TransactionReceipt(ctx, zeroTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, zeroReceipt.Status, "zeroing storage value should succeed")

			rezeroReceipt, err := opGeth.L2Client.TransactionReceipt(ctx, rezeroTx.Hash())
			require.NoError(t, err)
			require.Equal(t, types.ReceiptStatusSuccessful, rezeroReceipt.Status, "rezeroing storage value should succeed")

			require.Greater(t, rezeroReceipt.GasUsed, zeroReceipt.GasUsed, "rezero should use more gas due to not getting gas refund for clearing slot")
		})
	}
}
