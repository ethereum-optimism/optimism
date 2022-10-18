package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/gas-oracle/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/client"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/require"
)

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

	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block, nil)
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
		L2Time: l2GenesisBlock.Time(),
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
	l1Info, err := derive.L1InfoDepositBytes(1, l1Block)
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

// TestFCUVSNewPayloadDiscrepency attempts to exploit a difference in block creation between
// FCU and NewPayload. It specifically looks at the fact that NewPayload uses a shared EVMContext
// for the full block while FCU (and mining code) uses a new EVMContext for each transaction.
func TestFCUVSNewPayloadDiscrepency(t *testing.T) {
	// Setup an L2 EE and create a client connection to the engine.
	// We also need to setup a L1 Genesis to create the rollup genesis.
	log := testlog.Logger(t, log.LvlCrit)
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.FundDevAccounts = false
	cfg.DeployConfig.GasPriceOracleOverhead = 100_000_000_000
	cfg.DeployConfig.GasPriceOracleDecimals = 1
	cfg.DeployConfig.GasPriceOracleScalar = 10

	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig)
	require.Nil(t, err)
	l1Block := l1Genesis.ToBlock()

	l2Addrs := &genesis.L2Addresses{
		ProxyAdmin:                  predeploys.DevProxyAdminAddr,
		L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
		L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
	}
	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block, l2Addrs)
	l2Genesis.Alloc[cfg.Secrets.Addresses().Alice] = core.GenesisAccount{Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))}
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
		L2Time: l2GenesisBlock.Time(),
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
	l2Seq, err := ethclient.Dial(node.HTTPEndpoint())
	require.Nil(t, err)

	// Finally create the engine client
	client, err := sources.NewEngineClient(
		l2Node,
		log,
		nil,
		sources.EngineClientDefaultConfig(&rollup.Config{Genesis: rollupGenesis}),
	)
	require.Nil(t, err)

	// Create the test data
	// L1 Block Info Deposit
	// Tx From GPO Owner (alice) to decrease the GPO params
	// Tx to fund new account (enough ETH for the new account to pay for fees in the new but not old regime)
	// Tx from the new account (should succeed in FCU, but fail in new payload).
	// Note: It would be easy to go the other way (increase GPO params), but that would imply a block that fails
	// in FCU but succeeds in NewPayload. I intend to test it, but it requires fully hardcoded data (to test that
	// the NewPayload succeeds)
	l1Info, err := derive.L1InfoDepositBytes(1, l1Block)
	require.Nil(t, err)

	gpoContract, err := bindings.NewGasPriceOracle(common.HexToAddress(predeploys.GasPriceOracle), l2Seq)
	require.Nil(t, err)
	l2opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L2ChainIDBig())
	require.Nil(t, err)

	setScalarTx, err := gpoContract.SetScalar(l2opts, big.NewInt(0))
	require.Nil(t, err)
	setScalar, err := setScalarTx.MarshalBinary()
	require.Nil(t, err)

	rawFundTx := types.NewTransaction(1, cfg.Secrets.Addresses().Mallory, big.NewInt(50_000*params.GWei), 30000, big.NewInt(5*params.GWei), nil)
	fundTx, err := l2opts.Signer(cfg.Secrets.Addresses().Alice, rawFundTx)
	require.Nil(t, err)
	fund, err := fundTx.MarshalBinary()
	require.Nil(t, err)

	rawInvalidTx := types.NewTransaction(0, cfg.Secrets.Addresses().Bob, big.NewInt(0), 25_000, big.NewInt(1*params.GWei), nil)
	invalidTx, err := types.SignTx(rawInvalidTx, types.LatestSignerForChainID(cfg.L2ChainIDBig()), cfg.Secrets.Mallory)
	require.Nil(t, err)
	invalid, err := invalidTx.MarshalBinary()
	require.Nil(t, err)

	attrs := eth.PayloadAttributes{
		Timestamp:    hexutil.Uint64(l2GenesisBlock.Time() + 2),
		Transactions: []hexutil.Bytes{l1Info, setScalar, fund, invalid},
		NoTxPool:     true,
	}

	// Go through the flow of FCU, GetPayload, NewPayload, FCU
	// FCU should succeed, but NewPayload should fail.
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

	// fc.HeadBlockHash = payload.BlockHash
	// res, err = client.ForkchoiceUpdate(ctx, &fc, nil)
	// require.Nil(t, err)
	// require.Equal(t, eth.ExecutionValid, res.PayloadStatus.Status)

}
