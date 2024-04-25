package op_e2e

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/receipts"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
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

	aliceOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Deploy WETH9, we'll use this as our custom gas token for the purpose of the test
	weth9Address, tx, weth9, err := bindings.DeployWETH9(aliceOpts, l1Client)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// activate custom gas token feature (devnet does not have this activated at genesis)
	setCustomGasToken(t, cfg, sys, weth9Address)

	// Set amount of WETH9 to bridge to the recipient on L2
	amountToBridge := big.NewInt(10)
	recipient := common.HexToAddress("0xbeefdead")

	// Get some WETH
	aliceOpts.Value = big.NewInt(10_000_000)
	tx, err = weth9.Deposit(aliceOpts)
	waitForTx(t, tx, err, l1Client)
	aliceOpts.Value = nil
	newBalance, err := weth9.BalanceOf(&bind.CallOpts{}, aliceOpts.From)
	require.NoError(t, err)
	require.Equal(t, newBalance, big.NewInt(10_000_000))

	// Approve OptimismPortal
	tx, err = weth9.Approve(aliceOpts, cfg.L1Deployments.OptimismPortalProxy, amountToBridge)
	waitForTx(t, tx, err, l1Client)

	// Get recipient L2 balance before bridging
	previousL2Balance, err := l2Client.BalanceAt(context.Background(), recipient, nil)
	require.NoError(t, err)

	// Bridge the tokens
	optimismPortal, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
	require.NoError(t, err)
	tx, err = optimismPortal.DepositERC20Transaction(aliceOpts,
		recipient,
		amountToBridge,
		amountToBridge,
		50_0000, // _gasLimit
		false,
		[]byte{},
	)
	require.NoError(t, err)
	receipt, err := wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// compute the deposit transaction hash + poll for it
	depositEvent, err := receipts.FindLog(receipt.Logs, optimismPortal.ParseTransactionDeposited)
	require.NoError(t, err, "Should emit deposit event")
	depositTx, err := derive.UnmarshalDepositLogEvent(&depositEvent.Raw)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l2Client, types.NewTx(depositTx).Hash())
	require.NoError(t, err)

	// check for balance increase on L2
	newL2Balance, err := l2Client.BalanceAt(context.Background(), recipient, nil)
	require.NoError(t, err)
	l2BalanceIncrease := big.NewInt(0).Sub(newL2Balance, previousL2Balance)
	require.Equal(t, amountToBridge, l2BalanceIncrease)
}

func callViaSafe(t *testing.T, opts *bind.TransactOpts, client *ethclient.Client, safeAddress common.Address, target common.Address, data []byte) (*types.Transaction, error) {
	signature := [65]byte{}
	copy(signature[12:], opts.From[:])
	signature[64] = uint8(1)

	safe, err := bindings.NewSafe(safeAddress, client)
	require.NoError(t, err)

	isOwner, err := safe.IsOwner(&bind.CallOpts{}, opts.From)
	require.NoError(t, err)
	require.True(t, isOwner)

	return safe.ExecTransaction(opts, target, big.NewInt(0), data, 0, big.NewInt(0), big.NewInt(0), big.NewInt(0), common.Address{}, common.Address{}, signature[:])
}

func setCustomGasToken(t *testing.T, cfg SystemConfig, sys *System, cgtAddress common.Address) {
	l1Client := sys.Clients["l1"]
	deployerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Deployer, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Bind a SystemConfig at the SystemConfigProxy address
	systemConfig, err := bindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1Client)
	require.NoError(t, err)

	// Get existing parameters from SystemConfigProxy contract
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

	// minGasLimit, err := systemConfig.MinimumGasLimit(&bind.CallOpts{})
	require.NoError(t, err)

	// Queue up custom gas token address ready for reinitialization
	addresses.GasPayingToken = cgtAddress

	// Bind a ProxyAdmin to the ProxyAdmin address (why not ProxyAdminProxy?)
	proxyAdmin, err := bindings.NewProxyAdmin(cfg.L1Deployments.ProxyAdmin, l1Client)
	require.NoError(t, err)

	// Compute Proxy Admin Owner (this is a SAFE with 1 owner)
	proxyAdminOwner, err := proxyAdmin.Owner(&bind.CallOpts{})
	require.NoError(t, err)

	// Deploy a new StorageSetter contract
	storageSetterAddr, tx, _, err := bindings.DeployStorageSetter(deployerOpts, l1Client)
	waitForTx(t, tx, err, l1Client)

	// Set up a signer which controls the Proxy Admin Owner SAFE
	cliqueSignerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.CliqueSigner, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Encode calldata for upgrading SystemConfigProxy to the StorageSetter implementation
	proxyAdminABI, err := abi.JSON(strings.NewReader(bindings.ProxyAdminABI))
	require.NoError(t, err)
	encodedUpgradeCall, err := proxyAdminABI.Pack("upgrade",
		cfg.L1Deployments.SystemConfigProxy, storageSetterAddr)
	require.NoError(t, err)

	// Execute the upgrade SystemConfigProxy -> StorageSetter
	tx, err = callViaSafe(t, cliqueSignerOpts, l1Client, proxyAdminOwner, cfg.L1Deployments.ProxyAdmin, encodedUpgradeCall)
	waitForTx(t, tx, err, l1Client)

	// Bind a StorageSetter to the SystemConfigProxy address
	storageSetter, err := bindings.NewStorageSetter(cfg.L1Deployments.SystemConfigProxy, l1Client)
	require.NoError(t, err)

	// Use StorageSetter to clear out "initialize" slot
	tx, err = storageSetter.SetBytes320(deployerOpts, [32]byte{0}, [32]byte{0})
	waitForTx(t, tx, err, l1Client)

	// Sanity check previous step worked
	currentSlotValue, err := storageSetter.GetBytes32(&bind.CallOpts{}, [32]byte{0})
	require.NoError(t, err)
	require.Equal(t, currentSlotValue, [32]byte{0})

	// Prepare calldata for SystemConfigProxy -> SystemConfig upgrade
	encodedUpgradeCall, err = proxyAdminABI.Pack("upgrade",
		cfg.L1Deployments.SystemConfigProxy, cfg.L1Deployments.SystemConfig)
	require.NoError(t, err)

	// Execute SystemConfigProxy -> SystemConfig upgrade
	tx, err = callViaSafe(t, cliqueSignerOpts, l1Client, proxyAdminOwner, cfg.L1Deployments.ProxyAdmin, encodedUpgradeCall)
	waitForTx(t, tx, err, l1Client)

	version, err := systemConfig.Version(&bind.CallOpts{})
	require.NoError(t, err)
	t.Log("version: ", version)
	// Reinitialise with existing initializer values but with custom gas token set
	tx, err = systemConfig.Initialize(deployerOpts, owner,
		overhead,
		scalar,
		batcherHash,
		gasLimit,
		unsafeBlockSigner,
		resourceConfig,
		batchInbox,
		addresses)
	waitForTx(t, tx, err, l1Client)

	// Read Custom Gas Token and check it has been set properly
	gpt, err := systemConfig.GasPayingToken(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, cgtAddress, gpt.Addr)
}

func TestSetCustomGasToken(t *testing.T) {
	InitParallel(t)

	cfg := DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.Clients["l1"]
	deployerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Deployer, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Deploy WETH, we'll use this as our custom gas token for the purpose of the test
	wethAddress, tx, _, err := bindings.DeployWETH9(deployerOpts, l1Client)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	setCustomGasToken(t, cfg, sys, wethAddress)

}

func waitForTx(t *testing.T, tx *types.Transaction, err error, client *ethclient.Client) {
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), client, tx.Hash())
	require.NoError(t, err)
}
