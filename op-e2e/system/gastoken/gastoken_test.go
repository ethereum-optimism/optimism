package gastoken

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"

	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/receipts"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	op_e2e.RunMain(m)
}

func TestCustomGasToken(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.SkipOnFaultProofs) // Custom Gas Token feature is not yet compatible with fault proofs

	cfg := e2esys.DefaultSystemConfig(t)
	offset := hexutil.Uint64(0)
	cfg.DeployConfig.L2GenesisRegolithTimeOffset = &offset
	cfg.DeployConfig.L1CancunTimeOffset = &offset
	cfg.DeployConfig.L2GenesisCanyonTimeOffset = &offset
	cfg.DeployConfig.L2GenesisDeltaTimeOffset = &offset
	cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &offset

	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")

	l1Client := sys.NodeClient("l1")
	l2Client := sys.NodeClient("sequencer")

	aliceOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Deploy WETH9, we'll use this as our custom gas token for the purpose of the test
	weth9Address, tx, weth9, err := bindings.DeployWETH9(aliceOpts, l1Client)
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// setup expectations using custom gas token
	type Expectations struct {
		tokenAddress  common.Address
		tokenName     string
		tokenSymbol   string
		tokenDecimals uint8
	}
	disabledExpectations := Expectations{
		common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"),
		"Ether",
		"ETH",
		uint8(18),
	}
	enabledExpectations := Expectations{}
	enabledExpectations.tokenAddress = weth9Address
	enabledExpectations.tokenName, err = weth9.Name(&bind.CallOpts{})
	require.NoError(t, err)
	enabledExpectations.tokenSymbol, err = weth9.Symbol(&bind.CallOpts{})
	require.NoError(t, err)
	enabledExpectations.tokenDecimals, err = weth9.Decimals(&bind.CallOpts{})
	require.NoError(t, err)

	// Get some WETH
	aliceOpts.Value = big.NewInt(10_000_000)
	tx, err = weth9.Deposit(aliceOpts)
	waitForTx(t, tx, err, l1Client)
	aliceOpts.Value = nil
	newBalance, err := weth9.BalanceOf(&bind.CallOpts{}, aliceOpts.From)
	require.NoError(t, err)
	require.Equal(t, newBalance, big.NewInt(10_000_000))

	// Function to prepare and make call to depositERC20Transaction and make
	// appropriate assertions dependent on whether custom gas tokens have been enabled or not.
	checkDeposit := func(t *testing.T, enabled bool) {
		// Set amount of WETH9 to bridge to the recipient on L2
		amountToBridge := big.NewInt(10)
		recipient := common.HexToAddress("0xbeefdead")

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
		if enabled {
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

			require.EventuallyWithT(t, func(t *assert.CollectT) {
				// check for balance increase on L2
				newL2Balance, err := l2Client.BalanceAt(context.Background(), recipient, nil)
				require.NoError(t, err)
				l2BalanceIncrease := big.NewInt(0).Sub(newL2Balance, previousL2Balance)
				require.Equal(t, amountToBridge, l2BalanceIncrease)
			}, 10*time.Second, 1*time.Second)
		} else {
			require.Error(t, err)
		}
	}

	// Function to prepare and execute withdrawal flow for CGTs
	// and assert token balance is increased on L1.
	checkWithdrawal := func(t *testing.T) {
		l2Seq := l2Client
		l2Verif := sys.NodeClient("verifier")
		fromAddr := aliceOpts.From
		ethPrivKey := cfg.Secrets.Alice

		// Start L2 balance for withdrawal
		startBalanceBeforeWithdrawal, err := l2Seq.BalanceAt(context.Background(), fromAddr, nil)
		require.NoError(t, err)

		withdrawAmount := big.NewInt(5)
		tx, receipt := helpers.SendWithdrawal(t, cfg, l2Seq, cfg.Secrets.Alice, func(opts *helpers.WithdrawalTxOpts) {
			opts.Value = withdrawAmount
			opts.VerifyOnClients(l2Verif)
		})

		// Verify L2 balance after withdrawal
		header, err := l2Verif.HeaderByNumber(context.Background(), receipt.BlockNumber)
		require.NoError(t, err)

		endBalanceAfterWithdrawal, err := wait.ForBalanceChange(context.Background(), l2Seq, fromAddr, startBalanceBeforeWithdrawal)
		require.NoError(t, err)

		// Take fee into account
		diff := new(big.Int).Sub(startBalanceBeforeWithdrawal, endBalanceAfterWithdrawal)
		fees := helpers.CalcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
		fees = fees.Add(fees, receipt.L1Fee)
		diff = diff.Sub(diff, fees)
		require.Equal(t, withdrawAmount, diff)

		// Take start token balance on L1
		startTokenBalanceBeforeFinalize, err := weth9.BalanceOf(&bind.CallOpts{}, fromAddr)
		require.NoError(t, err)

		startETHBalanceBeforeFinalize, err := l1Client.BalanceAt(context.Background(), fromAddr, nil)
		require.NoError(t, err)

		proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt := helpers.ProveAndFinalizeWithdrawal(t, cfg, sys, "verifier", ethPrivKey, receipt)

		// Verify L1 ETH balance change
		proveFee := new(big.Int).Mul(new(big.Int).SetUint64(proveReceipt.GasUsed), proveReceipt.EffectiveGasPrice)
		finalizeFee := new(big.Int).Mul(new(big.Int).SetUint64(finalizeReceipt.GasUsed), finalizeReceipt.EffectiveGasPrice)
		fees = new(big.Int).Add(proveFee, finalizeFee)
		if e2eutils.UseFaultProofs() {
			resolveClaimFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveClaimReceipt.GasUsed), resolveClaimReceipt.EffectiveGasPrice)
			resolveFee := new(big.Int).Mul(new(big.Int).SetUint64(resolveReceipt.GasUsed), resolveReceipt.EffectiveGasPrice)
			fees = new(big.Int).Add(fees, resolveClaimFee)
			fees = new(big.Int).Add(fees, resolveFee)
		}

		// Verify L1ETHBalance after withdrawal
		// On CGT chains, the only change in ETH balance from a withdrawal
		// is a decrease to pay for gas
		endETHBalanceAfterFinalize, err := l1Client.BalanceAt(context.Background(), fromAddr, nil)
		require.NoError(t, err)
		diff = new(big.Int).Sub(endETHBalanceAfterFinalize, startETHBalanceBeforeFinalize)
		require.Equal(t, new(big.Int).Sub(big.NewInt(0), fees), diff)

		// Verify token balance after withdrawal
		// L1 Fees are paid in ETH, and
		// withdrawal is of a Custom Gas Token, so we do not subtract l1 fees from expected balance change
		// as we would if ETH was the gas paying token
		endTokenBalanceAfterFinalize, err := weth9.BalanceOf(&bind.CallOpts{}, fromAddr)
		require.NoError(t, err)
		diff = new(big.Int).Sub(endTokenBalanceAfterFinalize, startTokenBalanceBeforeFinalize)
		require.Equal(t, withdrawAmount, diff)
	}

	// checkFeeWithdrawal ensures that the FeeVault can be withdrawn from
	checkFeeWithdrawal := func(t *testing.T, enabled bool) {
		feeVault, err := bindings.NewSequencerFeeVault(predeploys.SequencerFeeVaultAddr, l2Client)
		require.NoError(t, err)

		// Alice will be sending transactions
		aliceOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L2ChainIDBig())
		require.NoError(t, err)

		// Get the recipient of the funds
		recipient, err := feeVault.RECIPIENT(&bind.CallOpts{})
		require.NoError(t, err)

		// This test depends on the withdrawal network being L1 which is represented
		// by 0 in the enum.
		withdrawalNetwork, err := feeVault.WITHDRAWALNETWORK(&bind.CallOpts{})
		require.NoError(t, err)
		require.Equal(t, withdrawalNetwork, uint8(0))

		// Get the balance of the recipient on L1
		var recipientBalanceBefore *big.Int
		if enabled {
			recipientBalanceBefore, err = weth9.BalanceOf(&bind.CallOpts{}, recipient)
		} else {
			recipientBalanceBefore, err = l1Client.BalanceAt(context.Background(), recipient, nil)
		}
		require.NoError(t, err)

		// Get the min withdrawal amount for the FeeVault
		amount, err := feeVault.MINWITHDRAWALAMOUNT(&bind.CallOpts{})
		require.NoError(t, err)

		l1opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Alice, cfg.L1ChainIDBig())
		require.NoError(t, err)

		optimismPortal, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
		require.NoError(t, err)

		depositAmount := new(big.Int).Mul(amount, big.NewInt(14))
		l1opts.Value = depositAmount

		var receipt *types.Receipt

		// Alice deposits funds
		if enabled {
			// approve + transferFrom flow
			// Cannot use `transfer` because of the tracking of balance in the OptimismPortal
			dep, err := weth9.Deposit(l1opts)
			waitForTx(t, dep, err, l1Client)

			l1opts.Value = nil
			tx, err := weth9.Approve(l1opts, cfg.L1Deployments.OptimismPortalProxy, depositAmount)
			waitForTx(t, tx, err, l1Client)

			require.NoError(t, err)
			deposit, err := optimismPortal.DepositERC20Transaction(l1opts, cfg.Secrets.Addresses().Alice, depositAmount, depositAmount, 500_000, false, []byte{})
			waitForTx(t, deposit, err, l1Client)

			receipt, err = wait.ForReceiptOK(context.Background(), l1Client, deposit.Hash())
			require.NoError(t, err)
		} else {
			// send ether to the portal directly, alice already has funds on L2
			tx, err := optimismPortal.DepositTransaction(l1opts, cfg.Secrets.Addresses().Alice, depositAmount, 500_000, false, []byte{})
			waitForTx(t, tx, err, l1Client)

			receipt, err = wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
			require.NoError(t, err)
		}

		// Compute the deposit transaction hash + poll for it
		depositEvent, err := receipts.FindLog(receipt.Logs, optimismPortal.ParseTransactionDeposited)
		require.NoError(t, err, "Should emit deposit event")
		depositTx, err := derive.UnmarshalDepositLogEvent(&depositEvent.Raw)
		require.NoError(t, err)
		_, err = wait.ForReceiptOK(context.Background(), l2Client, types.NewTx(depositTx).Hash())
		require.NoError(t, err)

		// Get Alice's balance on L2
		aliceBalance, err := l2Client.BalanceAt(context.Background(), cfg.Secrets.Addresses().Alice, nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, aliceBalance.Uint64(), amount.Uint64())

		// Send funds to the FeeVault so its balance is above the min withdrawal amount
		aliceOpts.Value = amount
		feeVaultTx, err := feeVault.Receive(aliceOpts)
		waitForTx(t, feeVaultTx, err, l2Client)

		// Ensure that the balance of the vault is large enough to withdraw
		vaultBalance, err := l2Client.BalanceAt(context.Background(), predeploys.SequencerFeeVaultAddr, nil)
		require.NoError(t, err)
		require.GreaterOrEqual(t, vaultBalance.Uint64(), amount.Uint64())

		// Ensure there is code at the vault address
		code, err := l2Client.CodeAt(context.Background(), predeploys.SequencerFeeVaultAddr, nil)
		require.NoError(t, err)
		require.NotEmpty(t, code)

		// Poke the fee vault to withdraw
		l2Opts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Bob, cfg.L2ChainIDBig())
		require.NoError(t, err)
		withdrawalTx, err := feeVault.Withdraw(l2Opts)
		waitForTx(t, withdrawalTx, err, l2Client)

		// Get the receipt and the amount withdrawn
		receipt, err = l2Client.TransactionReceipt(context.Background(), withdrawalTx.Hash())
		require.NoError(t, err)

		inclusionHeight := receipt.BlockNumber.Uint64()
		it, err := feeVault.FilterWithdrawal(&bind.FilterOpts{
			Start: inclusionHeight,
			End:   &inclusionHeight,
		})
		require.NoError(t, err)
		require.True(t, it.Next())

		withdrawnAmount := it.Event.Value

		// Finalize the withdrawal
		proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt := helpers.ProveAndFinalizeWithdrawal(t, cfg, sys, "verifier", cfg.Secrets.Alice, receipt)
		require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)
		require.Equal(t, types.ReceiptStatusSuccessful, finalizeReceipt.Status)
		if e2eutils.UseFaultProofs() {
			require.Equal(t, types.ReceiptStatusSuccessful, resolveClaimReceipt.Status)
			require.Equal(t, types.ReceiptStatusSuccessful, resolveReceipt.Status)
		}

		// Assert that the recipient's balance did increase
		var recipientBalanceAfter *big.Int
		if enabled {
			recipientBalanceAfter, err = weth9.BalanceOf(&bind.CallOpts{}, recipient)
		} else {
			recipientBalanceAfter, err = l1Client.BalanceAt(context.Background(), recipient, nil)
		}
		require.NoError(t, err)

		require.Equal(t, recipientBalanceAfter, new(big.Int).Add(recipientBalanceBefore, withdrawnAmount))
	}

	checkL1TokenNameAndSymbol := func(t *testing.T, enabled bool) {
		systemConfig, err := bindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1Client)
		require.NoError(t, err)

		token, err := systemConfig.GasPayingToken(&bind.CallOpts{})
		require.NoError(t, err)

		name, err := systemConfig.GasPayingTokenName(&bind.CallOpts{})
		require.NoError(t, err)

		symbol, err := systemConfig.GasPayingTokenSymbol(&bind.CallOpts{})
		require.NoError(t, err)

		if enabled {
			require.Equal(t, enabledExpectations.tokenAddress, token.Addr)
			require.Equal(t, enabledExpectations.tokenDecimals, token.Decimals)
			require.Equal(t, enabledExpectations.tokenName, name)
			require.Equal(t, enabledExpectations.tokenSymbol, symbol)
		} else {
			require.Equal(t, disabledExpectations.tokenAddress, token.Addr)
			require.Equal(t, disabledExpectations.tokenDecimals, token.Decimals)
			require.Equal(t, disabledExpectations.tokenName, name)
			require.Equal(t, disabledExpectations.tokenSymbol, symbol)
		}
	}

	checkL2TokenNameAndSymbol := func(t *testing.T, enabled bool) {
		l1Block, err := bindings.NewL1Block(predeploys.L1BlockAddr, l2Client)
		require.NoError(t, err)

		token, err := l1Block.GasPayingToken(&bind.CallOpts{})
		require.NoError(t, err)

		name, err := l1Block.GasPayingTokenName(&bind.CallOpts{})
		require.NoError(t, err)

		symbol, err := l1Block.GasPayingTokenSymbol(&bind.CallOpts{})
		require.NoError(t, err)

		if enabled {
			require.Equal(t, enabledExpectations.tokenAddress, token.Addr)
			require.Equal(t, enabledExpectations.tokenDecimals, token.Decimals)
			require.Equal(t, enabledExpectations.tokenName, name)
			require.Equal(t, enabledExpectations.tokenSymbol, symbol)
		} else {
			require.Equal(t, disabledExpectations.tokenAddress, token.Addr)
			require.Equal(t, disabledExpectations.tokenDecimals, token.Decimals)
			require.Equal(t, disabledExpectations.tokenName, name)
			require.Equal(t, disabledExpectations.tokenSymbol, symbol)
		}
	}

	checkWETHTokenNameAndSymbol := func(t *testing.T, enabled bool) {
		// Check name and symbol in WETH predeploy
		weth, err := bindings.NewWETH(predeploys.WETHAddr, l2Client)
		require.NoError(t, err)

		name, err := weth.Name(&bind.CallOpts{})
		require.NoError(t, err)

		symbol, err := weth.Symbol(&bind.CallOpts{})
		require.NoError(t, err)

		if enabled {
			require.Equal(t, "Wrapped "+enabledExpectations.tokenName, name)
			require.Equal(t, "W"+enabledExpectations.tokenSymbol, symbol)
		} else {
			require.Equal(t, "Wrapped "+disabledExpectations.tokenName, name)
			require.Equal(t, "W"+disabledExpectations.tokenSymbol, symbol)
		}
	}

	// Begin by testing behaviour when CGT feature is not enabled
	enabled := false
	checkDeposit(t, enabled)
	checkL1TokenNameAndSymbol(t, enabled)
	checkL2TokenNameAndSymbol(t, enabled)
	checkWETHTokenNameAndSymbol(t, enabled)
	checkFeeWithdrawal(t, enabled)

	// Activate custom gas token feature (devnet does not have this activated at genesis)
	setCustomGasToken(t, cfg, sys, weth9Address)

	// Now test behaviour given CGT feature is enabled
	enabled = true
	checkDeposit(t, enabled)
	checkWithdrawal(t)
	checkL1TokenNameAndSymbol(t, enabled)
	checkL2TokenNameAndSymbol(t, enabled)
	checkWETHTokenNameAndSymbol(t, enabled)
	checkFeeWithdrawal(t, enabled)
}

// callViaSafe will use the Safe smart account at safeAddress to send a transaction to target using the provided data. The transaction signature is constructed from
// the supplied opts.
func callViaSafe(opts *bind.TransactOpts, client *ethclient.Client, safeAddress common.Address, target common.Address, data []byte) (*types.Transaction, error) {
	signature := [65]byte{}
	copy(signature[12:], opts.From[:])
	signature[64] = uint8(1)

	safe, err := bindings.NewSafe(safeAddress, client)
	if err != nil {
		return nil, err
	}

	owners, err := safe.GetOwners(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}

	isOwner, err := safe.IsOwner(&bind.CallOpts{}, opts.From)
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, fmt.Errorf("address %s is not in owners list %s", opts.From, owners)
	}

	return safe.ExecTransaction(opts, target, big.NewInt(0), data, 0, big.NewInt(0), big.NewInt(0), big.NewInt(0), common.Address{}, common.Address{}, signature[:])
}

// setCustomGasToeken enables the Custom Gas Token feature on a chain where it wasn't enabled at genesis.
// It reads existing parameters from the SystemConfig contract, inserts the supplied cgtAddress and reinitializes that contract.
// To do this it uses the ProxyAdmin and StorageSetter from the supplied cfg.
func setCustomGasToken(t *testing.T, cfg e2esys.SystemConfig, sys *e2esys.System, cgtAddress common.Address) {
	l1Client := sys.NodeClient("l1")
	deployerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Deployer, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Bind a SystemConfig at the SystemConfigProxy address
	systemConfig, err := bindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1Client)
	require.NoError(t, err)

	// Get existing parameters from SystemConfigProxy contract
	owner, err := systemConfig.Owner(&bind.CallOpts{})
	require.NoError(t, err)
	basefeeScalar, err := systemConfig.BasefeeScalar(&bind.CallOpts{})
	require.NoError(t, err)
	blobbasefeeScalar, err := systemConfig.BlobbasefeeScalar(&bind.CallOpts{})
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
	addresses.DisputeGameFactory, err = systemConfig.DisputeGameFactory(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.OptimismPortal, err = systemConfig.OptimismPortal(&bind.CallOpts{})
	require.NoError(t, err)
	addresses.OptimismMintableERC20Factory, err = systemConfig.OptimismMintableERC20Factory(&bind.CallOpts{})
	require.NoError(t, err)

	// Queue up custom gas token address ready for reinitialization
	addresses.GasPayingToken = cgtAddress

	// Bind a ProxyAdmin to the ProxyAdmin address
	proxyAdmin, err := bindings.NewProxyAdmin(cfg.L1Deployments.ProxyAdmin, l1Client)
	require.NoError(t, err)

	// Compute Proxy Admin Owner (this is a SAFE with 1 owner)
	proxyAdminOwner, err := proxyAdmin.Owner(&bind.CallOpts{})
	require.NoError(t, err)

	// Deploy a new StorageSetter contract
	storageSetterAddr, tx, _, err := bindings.DeployStorageSetter(deployerOpts, l1Client)
	waitForTx(t, tx, err, l1Client)

	// Set up a signer which controls the Proxy Admin Owner SAFE
	safeOwnerOpts, err := bind.NewKeyedTransactorWithChainID(cfg.Secrets.Deployer, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Encode calldata for upgrading SystemConfigProxy to the StorageSetter implementation
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	require.NoError(t, err)
	encodedUpgradeCall, err := proxyAdminABI.Pack("upgrade",
		cfg.L1Deployments.SystemConfigProxy, storageSetterAddr)
	require.NoError(t, err)

	// Execute the upgrade SystemConfigProxy -> StorageSetter
	tx, err = callViaSafe(safeOwnerOpts, l1Client, proxyAdminOwner, cfg.L1Deployments.ProxyAdmin, encodedUpgradeCall)
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
	tx, err = callViaSafe(safeOwnerOpts, l1Client, proxyAdminOwner, cfg.L1Deployments.ProxyAdmin, encodedUpgradeCall)
	waitForTx(t, tx, err, l1Client)

	// Reinitialise with existing initializer values but with custom gas token set
	tx, err = systemConfig.Initialize(deployerOpts, owner,
		basefeeScalar,
		blobbasefeeScalar,
		batcherHash,
		gasLimit,
		unsafeBlockSigner,
		resourceConfig,
		batchInbox,
		addresses)
	require.NoError(t, err)
	receipt, err := wait.ForReceiptOK(context.Background(), l1Client, tx.Hash())
	require.NoError(t, err)

	// Read Custom Gas Token and check it has been set properly
	gpt, err := systemConfig.GasPayingToken(&bind.CallOpts{})
	require.NoError(t, err)
	require.Equal(t, cgtAddress, gpt.Addr)

	optimismPortal, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
	require.NoError(t, err)

	depositEvent, err := receipts.FindLog(receipt.Logs, optimismPortal.ParseTransactionDeposited)
	require.NoError(t, err, "Should emit deposit event")
	depositTx, err := derive.UnmarshalDepositLogEvent(&depositEvent.Raw)

	require.NoError(t, err)
	l2Client := sys.NodeClient("sequencer")
	receipt, err = wait.ForReceiptOK(context.Background(), l2Client, types.NewTx(depositTx).Hash())
	require.NoError(t, err)

	l1Block, err := bindings.NewL1Block(predeploys.L1BlockAddr, l2Client)
	require.NoError(t, err)
	_, err = receipts.FindLog(receipt.Logs, l1Block.ParseGasPayingTokenSet)
	require.NoError(t, err)
}

// waitForTx is a thing wrapper around wait.ForReceiptOK which asserts on there being no errors.
func waitForTx(t *testing.T, tx *types.Transaction, err error, client *ethclient.Client) {
	require.NoError(t, err)
	_, err = wait.ForReceiptOK(context.Background(), client, tx.Hash())
	require.NoError(t, err)
}
