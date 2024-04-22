package op_e2e

import (
	"context"
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/receipts"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func TestCustomGasTokenLockAndMint(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	l2Client := sys.Clients["sequencer"]

	opts, err := bind.NewKeyedTransactorWithChainID(sys.Cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Deploy WETH
	wethAddress, tx, WETH, err := bindings.DeployWETH(opts, l1Client)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err, "Waiting for deposit tx on L1")

	// Get some WETH
	opts.Value = big.NewInt(params.Ether)
	tx, err = WETH.Deposit(opts)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)
	opts.Value = nil
	wethBalance, err := WETH.BalanceOf(&bind.CallOpts{}, opts.From)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(params.Ether), wethBalance)

	// Approve WETH with the bridge
	tx, err = WETH.Approve(opts, cfg.L1Deployments.OptimismPortal, new(big.Int).SetUint64(math.MaxUint64))
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// TODO activate the custom gas token by redeploying the SystemConfig
	// proxyAdmin, err := bindings.NewProxyAdmin(cfg.L1Deployments.ProxyAdmin, l1Client)

	systemConfig, err := bindings.NewSystemConfig(cfg.L1Deployments.SystemConfig, l1Client)
	require.NoError(t, err)

	owner, err := systemConfig.Owner(&bind.CallOpts{})
	require.NoError(t, err)
	overhead, err := systemConfig.Overhead(&bind.CallOpts{})
	require.NoError(t, err)
	scalar, err := systemConfig.Scalar(&bind.CallOpts{})
	require.NoError(t, err)
	batcherHash, err := systemConfig.BatcherHash(&bind.CallOpts{})
	require.NoError(t, err)
	gasLimit, err := systemConfig.GasLimit(&bind.CallOpts{})
	require.NoError(t, err)
	unsafeBlockSigner, err := systemConfig.UnsafeBlockSigner(&bind.CallOpts{})
	require.NoError(t, err)
	resourceConfig, err := systemConfig.ResourceConfig(&bind.CallOpts{})
	require.NoError(t, err)
	batchInbox, err := systemConfig.BatchInbox(&bind.CallOpts{})
	require.NoError(t, err)
	addresses := bindings.SystemConfigAddresses{}
	addresses.L1CrossDomainMessenger, err = systemConfig.L1CrossDomainMessenger(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.L1ERC721Bridge, err = systemConfig.L1ERC721Bridge(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.L1StandardBridge, err = systemConfig.L1StandardBridge(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.L2OutputOracle, err = systemConfig.L2OutputOracle(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.OptimismPortal, err = systemConfig.OptimismPortal(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.OptimismMintableERC20Factory, err = systemConfig.OptimismMintableERC20Factory(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.GasPayingToken = wethAddress

	newSystemConfigAddr, tx, _, err := bindings.DeploySystemConfig(opts, l1Client)

	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	abi, err := abi.JSON(strings.NewReader(bindings.SystemConfigABI))
	require.NoError(t, err)
	encodedInitializeCall, err := abi.Pack("initialize",
		owner,
		overhead,
		scalar,
		batcherHash,
		gasLimit,
		unsafeBlockSigner,
		resourceConfig,
		batchInbox,
		addresses)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	proxyAdmin, err := bindings.NewProxyAdmin(cfg.L1Deployments.ProxyAdmin, l1Client)
	require.NoError(t, err)

	deployerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Deployer, cfg.L1ChainIDBig())
	require.NoError(t, err)

	proxyAdminOwner, err := proxyAdmin.Owner(&bind.CallOpts{})
	safe, err := bindings.NewSafe(proxyAdminOwner, l1Client)
	require.NoError(t, err)

	safe.ExecTransaction(deployerOpts, cfg.L1Deployments.ProxyAdmin, 0, data)

	require.NoError(t, err)

	require.Equal(t, proxyAdminOwner, ownerOpts.From)

	// TODO looks like we need
	// Account #9: 0xa0ee7a142d267c1f36714e4a8f75612f20a79720 (10000 ETH)
	// Private Key: 0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6

	tx, err = proxyAdmin.UpgradeAndCall(ownerOpts, cfg.L1Deployments.SystemConfigProxy, newSystemConfigAddr, encodedInitializeCall)
	require.NoError(t, err)

	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// Bridge the WETH
	portal, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
	require.NoError(t, err)

	to := opts.From
	mint := big.NewInt(100)
	value := big.NewInt(0)
	isCreation := false
	data := []byte{}

	tx, err = transactions.PadGasEstimate(opts, 1.1, func(opts *bind.TransactOpts) (*types.Transaction, error) {
		return portal.DepositERC20Transaction(opts, to, mint, value, 100000, isCreation, data)
	})
	require.NoError(t, err)
	depositReceipt, err := wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	t.Log("Deposit custom gas token through OptimismPortal", "gas used", depositReceipt.GasUsed)

	// compute the deposit transaction hash + poll for it
	depositEvent, err := receipts.FindLog(depositReceipt.Logs, portal.ParseTransactionDeposited)
	require.NoError(t, err, "Should emit deposit event")

	depositTx, err := derive.UnmarshalDepositLogEvent(&depositEvent.Raw)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l2Client, types.NewTx(depositTx).Hash())
	require.NoError(t, err)

	// Should have balance on L2
	l2Balance, err := l2Client.BalanceAt(context.Background(), opts.From, nil)
	require.NoError(t, err)
	require.Equal(t, l2Balance, big.NewInt(100))
}
