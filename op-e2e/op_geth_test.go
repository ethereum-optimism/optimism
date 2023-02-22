package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestMissingGasLimit tests that op-geth cannot build a block without gas limit while optimism is active in the chain config.
func TestMissingGasLimit(t *testing.T) {
	// Setup an L2 EE and create a client connection to the engine.
	// We also need to setup a L1 Genesis to create the rollup genesis.
	log := testlog.Logger(t, log.LvlCrit)
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false

	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig)
	require.Nil(t, err)
	l1Block := l1Genesis.ToBlock()

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block)
	require.Nil(t, err)
	l2GenesisBlock := l2Genesis.ToBlock()

	rollupGenesis := rollup.Genesis{
		L1: eth.BlockID{
			Hash:   l1Block.Hash(),
			Number: l1Block.NumberU64(),
		},
		L2: eth.BlockID{
			Hash:   l2GenesisBlock.Hash(),
			Number: l2GenesisBlock.NumberU64(),
		},
		L2Time:       l2GenesisBlock.Time(),
		SystemConfig: e2eutils.SystemConfigFromDeployConfig(cfg.DeployConfig),
	}

	node, _, err := initL2Geth("l2", big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, writeDefaultJWT(t))
	require.Nil(t, err)
	require.Nil(t, node.Start())
	defer node.Close()

	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(testingJWTSecret))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	l2Node, err := client.NewRPC(ctx, log, node.WSAuthEndpoint(), auth)
	require.Nil(t, err)

	// Finally create the engine client
	client, err := sources.NewEngineClient(
		l2Node,
		log,
		nil,
		sources.EngineClientDefaultConfig(&rollup.Config{Genesis: rollupGenesis}),
	)
	require.Nil(t, err)

	attrs := eth.PayloadAttributes{
		Timestamp:    hexutil.Uint64(l2GenesisBlock.Time() + 2),
		Transactions: []hexutil.Bytes{},
		NoTxPool:     true,
		GasLimit:     nil, // no gas limit
	}

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fc := eth.ForkchoiceState{
		HeadBlockHash: l2GenesisBlock.Hash(),
		SafeBlockHash: l2GenesisBlock.Hash(),
	}
	res, err := client.ForkchoiceUpdate(ctx, &fc, &attrs)
	require.ErrorIs(t, err, eth.InputError{})
	require.Equal(t, eth.InvalidPayloadAttributes, err.(eth.InputError).Code)
	require.Nil(t, res)
}

// TestInvalidDepositInFCU runs an invalid deposit through a FCU/GetPayload/NewPayload/FCU set of calls.
// This tests that deposits must always allow the block to be built even if they are invalid.
func TestInvalidDepositInFCU(t *testing.T) {
	// Setup an L2 EE and create a client connection to the engine.
	// We also need to setup a L1 Genesis to create the rollup genesis.
	log := testlog.Logger(t, log.LvlCrit)
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false

	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig)
	require.Nil(t, err)
	l1Block := l1Genesis.ToBlock()

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block)
	require.Nil(t, err)
	l2GenesisBlock := l2Genesis.ToBlock()

	rollupGenesis := rollup.Genesis{
		L1: eth.BlockID{
			Hash:   l1Block.Hash(),
			Number: l1Block.NumberU64(),
		},
		L2: eth.BlockID{
			Hash:   l2GenesisBlock.Hash(),
			Number: l2GenesisBlock.NumberU64(),
		},
		L2Time:       l2GenesisBlock.Time(),
		SystemConfig: e2eutils.SystemConfigFromDeployConfig(cfg.DeployConfig),
	}

	node, _, err := initL2Geth("l2", big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, writeDefaultJWT(t))
	require.Nil(t, err)
	require.Nil(t, node.Start())
	defer node.Close()

	auth := rpc.WithHTTPAuth(gn.NewJWTAuth(testingJWTSecret))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	l2Node, err := client.NewRPC(ctx, log, node.WSAuthEndpoint(), auth)
	require.Nil(t, err)

	// Finally create the engine client
	client, err := sources.NewEngineClient(
		l2Node,
		log,
		nil,
		sources.EngineClientDefaultConfig(&rollup.Config{Genesis: rollupGenesis}),
	)
	require.Nil(t, err)

	// Create the test data (L1 Info Tx and then always failing deposit)
	l1Info, err := derive.L1InfoDepositBytes(1, l1Block, rollupGenesis.SystemConfig)
	require.Nil(t, err)

	// Create a deposit from alice that will always fail (not enough funds)
	fromAddr := cfg.Secrets.Addresses().Alice
	l2Client, err := ethclient.Dial(node.HTTPEndpoint())
	require.Nil(t, err)
	balance, err := l2Client.BalanceAt(ctx, fromAddr, nil)
	require.Nil(t, err)
	require.Equal(t, 0, balance.Cmp(common.Big0))

	badDepositTx := types.NewTx(&types.DepositTx{
		// TODO: Source Hash
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})
	badDeposit, err := badDepositTx.MarshalBinary()
	require.Nil(t, err)

	attrs := eth.PayloadAttributes{
		Timestamp:    hexutil.Uint64(l2GenesisBlock.Time() + 2),
		Transactions: []hexutil.Bytes{l1Info, badDeposit},
		NoTxPool:     true,
		GasLimit:     (*eth.Uint64Quantity)(&rollupGenesis.SystemConfig.GasLimit),
	}

	// Go through the flow of FCU, GetPayload, NewPayload, FCU
	// We are inserting a block with an invalid deposit.
	// The invalid deposit should still remain in the block.
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fc := eth.ForkchoiceState{
		HeadBlockHash: l2GenesisBlock.Hash(),
		SafeBlockHash: l2GenesisBlock.Hash(),
	}
	res, err := client.ForkchoiceUpdate(ctx, &fc, &attrs)
	require.Nil(t, err)
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
	require.NotNil(t, res.PayloadID)

	payload, err := client.GetPayload(ctx, *res.PayloadID)
	require.Nil(t, err)
	require.NotNil(t, payload)
	require.Equal(t, payload.Transactions, attrs.Transactions) // Ensure we don't drop the transactions

	status, err := client.NewPayload(ctx, payload)
	require.Nil(t, err)
	require.Equal(t, eth.ExecutionValid, status.Status)

	fc.HeadBlockHash = payload.BlockHash
	res, err = client.ForkchoiceUpdate(ctx, &fc, nil)
	require.Nil(t, err)
	require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)
}

// TestActivateRegolithAtGenesis runs deposit transactions on a chain with Regolith enabled at genesis
func TestActivateRegolithAtGenesis(t *testing.T) {
	// Setup an L2 EE and create a client connection to the engine.
	// We also need to setup a L1 Genesis to create the rollup genesis.
	cfg := DefaultSystemConfig(t)
	regolithTime := uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = (*hexutil.Uint64)(&regolithTime)

	devnet, err := NewDevnet(t, cfg.DeployConfig)
	require.NoError(t, err)
	defer devnet.Close()

	fromAddr := cfg.Secrets.Addresses().Alice

	// Simple transfer deposit tx
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx := types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000000,
		Data:                []byte{},
		IsSystemTransaction: false,
	})

	// Add a new block with these transactions
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload, err := devnet.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err := devnet.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.NotEqual(t, depositTx.Gas(), receipt.GasUsed)
	require.Equal(t, uint64(0), *receipt.DepositNonce)

	infoTx, err := devnet.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	infoRcpt, err := devnet.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.NotZero(t, infoRcpt.GasUsed)

	expectedContractAddress := crypto.CreateAddress(fromAddr, uint64(1))
	createRcpt, err := devnet.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)
	require.Equal(t, uint64(1), *createRcpt.DepositNonce)
	contractBalance, err := devnet.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, contractCreateTx.Value(), contractBalance)
}

// TestActivateRegolithAtGenesis runs deposit transactions on a chain with Regolith enabled from block 2
func TestActivateRegolithAfterGenesis(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	regolithTime := uint64(4)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = (*hexutil.Uint64)(&regolithTime)

	devnet, err := NewDevnet(t, cfg.DeployConfig)
	require.NoError(t, err)
	defer devnet.Close()

	fromAddr := cfg.Secrets.Addresses().Alice

	// Simple transfer deposit tx
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25000,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx := types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000000,
		Data:                []byte{},
		IsSystemTransaction: false,
	})

	// Add a new block with these transactions
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First block is still in bedrock
	payload, err := devnet.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err := devnet.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.Equal(t, depositTx.Gas(), receipt.GasUsed)

	infoTx, err := devnet.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	infoRcpt, err := devnet.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.Zero(t, infoRcpt.GasUsed)

	expectedContractAddress := crypto.CreateAddress(fromAddr, uint64(0)) // Expected to be wrong
	createRcpt, err := devnet.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)

	contractBalance, err := devnet.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(0), contractBalance.Uint64())

	// Second block is in regolith
	// Simple transfer deposit tx
	depositTx = types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		To:                  &fromAddr, // send it to ourselves
		Value:               big.NewInt(params.Ether),
		Gas:                 25001,
		IsSystemTransaction: false,
	})

	// Contract creation deposit tx
	contractCreateTx = types.NewTx(&types.DepositTx{
		SourceHash:          devnet.L1Head.Hash(),
		From:                fromAddr,
		Value:               big.NewInt(params.Ether),
		Gas:                 1000001,
		Data:                []byte{},
		IsSystemTransaction: false,
	})
	payload, err = devnet.AddL2Block(ctx, depositTx, contractCreateTx)
	require.NoError(t, err)

	// Check the deposit tx show actual gas used, not gas limit
	receipt, err = devnet.L2Client.TransactionReceipt(ctx, depositTx.Hash())
	require.NoError(t, err)
	require.NotEqual(t, depositTx.Gas(), receipt.GasUsed)

	infoTx, err = devnet.L2Client.TransactionInBlock(ctx, payload.BlockHash, 0)
	require.NoError(t, err)
	infoRcpt, err = devnet.L2Client.TransactionReceipt(ctx, infoTx.Hash())
	require.NoError(t, err)
	require.NotZero(t, infoRcpt.GasUsed)

	expectedContractAddress = crypto.CreateAddress(fromAddr, uint64(3))
	createRcpt, err = devnet.L2Client.TransactionReceipt(ctx, contractCreateTx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, createRcpt.Status)
	require.Equal(t, expectedContractAddress, createRcpt.ContractAddress)

	contractBalance, err = devnet.L2Client.BalanceAt(ctx, createRcpt.ContractAddress, nil)
	require.NoError(t, err)
	require.Equal(t, contractCreateTx.Value(), contractBalance)
}

// TestRegolithDepositTxUnusedGas checks that unused gas from deposit transactions is returned to the block gas pool
// Also checks the user is not refunded for unused gas.
func TestRegolithDepositTxUnusedGas(t *testing.T) {
	cfg := DefaultSystemConfig(t)
	regolithTime := uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = (*hexutil.Uint64)(&regolithTime)

	devnet, err := NewDevnet(t, cfg.DeployConfig)
	require.NoError(t, err)
	defer devnet.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fromAddr := cfg.Secrets.Addresses().Alice

	aliceBalance, err := devnet.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.NoError(t, err)

	// Deposit TX with a high gas limit but using very little actual gas
	depositTx := types.NewTx(&types.DepositTx{
		SourceHash: devnet.L1Head.Hash(),
		From:       fromAddr,
		To:         &fromAddr, // send it to ourselves
		Value:      big.NewInt(params.Ether),
		// SystemTx is assigned 1M gas limit
		Gas:                 uint64(cfg.DeployConfig.L2GenesisBlockGasLimit) - 1_000_000,
		IsSystemTransaction: false,
	})

	signer := types.LatestSigner(devnet.L2ChainConfig)
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

	_, err = devnet.AddL2Block(ctx, depositTx, tx)
	require.NoError(t, err)

	newAliceBalance, err := devnet.L2Client.BalanceAt(ctx, fromAddr, nil)
	require.Equal(t, aliceBalance, newAliceBalance, "should not refund fee for unused gas")
}
