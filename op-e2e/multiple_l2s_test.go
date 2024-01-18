package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestMultipleL2s(t *testing.T) {
	cfg := DefaultSystemConfigInterop(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()
}

func erc20BridgeDeposit(
	t *testing.T, log log.Logger, rollupCfg *rollup.Config, aliceKey *ecdsa.PrivateKey, l1Client *ethclient.Client,
	l2Client *ethclient.Client, l1ChainID *big.Int, l2ChainID *big.Int, l1Deployments *genesis.L1Deployments,
) {
	log.Info("genesis", "l2", rollupCfg.Genesis.L2, "l1", rollupCfg.Genesis.L1, "l2_time", rollupCfg.Genesis.L2Time)

	opts, err := bind.NewKeyedTransactorWithChainID(aliceKey, l1ChainID)
	require.Nil(t, err)

	// Deploy WETH9
	weth9Address, tx, WETH9, err := bindings.DeployWETH9(opts, l1Client)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err, "Waiting for deposit tx on L1")

	// Get some WETH
	opts.Value = big.NewInt(params.Ether)
	tx, err = WETH9.Fallback(opts, []byte{})
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)
	opts.Value = nil
	wethBalance, err := WETH9.BalanceOf(&bind.CallOpts{}, opts.From)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(params.Ether), wethBalance)

	// Deploy L2 WETH9
	l2Opts, err := bind.NewKeyedTransactorWithChainID(aliceKey, l2ChainID)
	require.NoError(t, err)
	optimismMintableTokenFactory, err := bindings.NewOptimismMintableERC20Factory(predeploys.OptimismMintableERC20FactoryAddr, l2Client)
	require.NoError(t, err)
	tx, err = optimismMintableTokenFactory.CreateOptimismMintableERC20(l2Opts, weth9Address, "L2-WETH", "L2-WETH")
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l2Client, tx.Hash())
	require.NoError(t, err)

	// Get the deployment event to have access to the L2 WETH9 address
	it, err := optimismMintableTokenFactory.FilterOptimismMintableERC20Created(&bind.FilterOpts{Start: 0}, nil, nil)
	require.NoError(t, err)
	var event *bindings.OptimismMintableERC20FactoryOptimismMintableERC20Created
	for it.Next() {
		event = it.Event
	}
	require.NotNil(t, event)

	// Approve WETH9 with the bridge
	tx, err = WETH9.Approve(opts, l1Deployments.L1StandardBridgeProxy, new(big.Int).SetUint64(math.MaxUint64))
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// Bridge the WETH9
	l1StandardBridge, err := bindings.NewL1StandardBridge(l1Deployments.L1StandardBridgeProxy, l1Client)
	require.NoError(t, err)
	tx, err = transactions.PadGasEstimate(opts, 1.1, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return l1StandardBridge.BridgeERC20(opts, weth9Address, event.LocalToken, big.NewInt(100), 100000, []byte{})
	})
	require.NoError(t, err)
	depositReceipt, err := wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	t.Log("Deposit through L1StandardBridge", "gas used", depositReceipt.GasUsed)

	// compute the deposit transaction hash + poll for it
	portal, err := bindings.NewOptimismPortal(l1Deployments.OptimismPortalProxy, l1Client)
	require.NoError(t, err)

	depIt, err := portal.FilterTransactionDeposited(&bind.FilterOpts{Start: 0}, nil, nil, nil)
	require.NoError(t, err)
	var depositEvent *bindings.OptimismPortalTransactionDeposited
	for depIt.Next() {
		depositEvent = depIt.Event
	}
	require.NotNil(t, depositEvent)

	depositTx, err := derive.UnmarshalDepositLogEvent(&depositEvent.Raw)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l2Client, types.NewTx(depositTx).Hash())
	require.NoError(t, err)

	// Ensure that the deposit went through
	optimismMintableToken, err := bindings.NewOptimismMintableERC20(event.LocalToken, l2Client)
	require.NoError(t, err)

	// Should have balance on L2
	l2Balance, err := optimismMintableToken.BalanceOf(&bind.CallOpts{}, opts.From)
	require.NoError(t, err)
	require.Equal(t, l2Balance, big.NewInt(100))

	fmt.Println("foo")
	fmt.Println(l2Balance)
}

func TestERC20BridgeDepositsOnBothL2s(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfigInterop(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LvlInfo)

	erc20BridgeDeposit(
		t, log, sys.RollupConfigs[0], cfg.Secrets.Alice,
		sys.Clients["l1"], sys.Clients["sequencer"], cfg.L1ChainIDBig(),
		cfg.L2ChainIDBig(), cfg.L1Deployments[0],
	)

	erc20BridgeDeposit(
		t, log, sys.RollupConfigs[1], cfg.Secrets.Alice,
		sys.Clients["l1"], sys.Clients["sequencer_2"], cfg.L1ChainIDBig(),
		cfg.L2ChainIDBig_2(), cfg.L1Deployments[1],
	)
}
