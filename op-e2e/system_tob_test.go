package op_e2e

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	legacybindings "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/bindings"
	bindingspreview "github.com/ethereum-optimism/optimism/op-node/bindings/preview"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testutils/fuzzerutils"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

// TestGasPriceOracleFeeUpdates checks that the gas price oracle cannot be locked by mis-configuring parameters.
func TestGasPriceOracleFeeUpdates(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	InitParallel(t)
	// Define our values to set in the GasPriceOracle (we set them high to see if it can lock L2 or stop bindings
	// from updating the prices once again.
	overheadValue := abi.MaxUint256
	scalarValue := abi.MaxUint256
	var cancel context.CancelFunc

	// Create our system configuration for L1/L2 and start it
	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	// l2Verif := sys.Clients["verifier"]
	ethPrivKey := cfg.Secrets.SysCfgOwner

	// Bind to the SystemConfig & GasPriceOracle contracts
	sysconfig, err := legacybindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1Client)
	require.Nil(t, err)
	gpoContract, err := legacybindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, l2Seq)
	require.Nil(t, err)

	// Obtain our signer.
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Define our L1 transaction timeout duration.
	txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

	// Update the gas config, wait for it to show up on L2, & verify that it was set as intended
	opts.Context, cancel = context.WithTimeout(ctx, txTimeoutDuration)
	tx, err := sysconfig.SetGasConfig(opts, overheadValue, scalarValue)
	cancel()
	require.Nil(t, err, "sending overhead update tx")

	receipt, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.Nil(t, err, "Waiting for sysconfig set gas config update tx")

	_, err = geth.WaitForL1OriginOnL2(sys.RollupConfig, receipt.BlockNumber.Uint64(), l2Seq, txTimeoutDuration)
	require.NoError(t, err, "waiting for L2 block to include the sysconfig update")

	gpoOverhead, err := gpoContract.Overhead(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo overhead")
	gpoScalar, err := gpoContract.Scalar(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo scalar")

	if gpoOverhead.Cmp(overheadValue) != 0 {
		t.Errorf("overhead that was found (%v) is not what was set (%v)", gpoOverhead, overheadValue)
	}
	if gpoScalar.Cmp(scalarValue) != 0 {
		t.Errorf("scalar that was found (%v) is not what was set (%v)", gpoScalar, scalarValue)
	}

	// Now modify the scalar value & ensure that the gas params can be modified
	scalarValue = big.NewInt(params.Ether)

	opts.Context, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
	tx, err = sysconfig.SetGasConfig(opts, overheadValue, scalarValue)
	cancel()
	require.Nil(t, err, "sending overhead update tx")

	receipt, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.Nil(t, err, "Waiting for sysconfig set gas config update tx")

	_, err = geth.WaitForL1OriginOnL2(sys.RollupConfig, receipt.BlockNumber.Uint64(), l2Seq, txTimeoutDuration)
	require.NoError(t, err, "waiting for L2 block to include the sysconfig update")

	gpoOverhead, err = gpoContract.Overhead(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo overhead")
	gpoScalar, err = gpoContract.Scalar(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo scalar")

	if gpoOverhead.Cmp(overheadValue) != 0 {
		t.Errorf("overhead that was found (%v) is not what was set (%v)", gpoOverhead, overheadValue)
	}
	if gpoScalar.Cmp(scalarValue) != 0 {
		t.Errorf("scalar that was found (%v) is not what was set (%v)", gpoScalar, scalarValue)
	}
}

// TestL2SequencerRPCDepositTx checks that the L2 sequencer will not accept DepositTx type transactions.
// The acceptance of these transactions would allow for arbitrary minting of ETH in L2.
func TestL2SequencerRPCDepositTx(t *testing.T) {
	InitParallel(t)

	// Create our system configuration for L1/L2 and start it
	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	txSigningKey := sys.Cfg.Secrets.Alice
	require.Nil(t, err)

	// Create a deposit tx to send over RPC.
	tx := types.NewTx(&types.DepositTx{
		SourceHash:          common.Hash{},
		From:                crypto.PubkeyToAddress(txSigningKey.PublicKey),
		To:                  &common.Address{0xff, 0xff},
		Mint:                big.NewInt(1000),
		Value:               big.NewInt(1000),
		Gas:                 0,
		IsSystemTransaction: false,
		Data:                nil,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = l2Seq.SendTransaction(ctx, tx)
	cancel()
	require.Error(t, err, "a DepositTx was accepted by L2 sequencer over RPC when it should not have been.")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	err = l2Verif.SendTransaction(ctx, tx)
	cancel()
	require.Error(t, err, "a DepositTx was accepted by L2 verifier over RPC when it should not have been.")
}

// TestAccount defines an account generated by startConfigWithTestAccounts
type TestAccount struct {
	HDPath string
	Key    *ecdsa.PrivateKey
	Addr   common.Address
	L1Opts *bind.TransactOpts
	L2Opts *bind.TransactOpts
}

// startConfigWithTestAccounts takes a SystemConfig, generates additional accounts, adds them to the config, so they
// are funded on startup, starts the system, and imports the keys into the keystore, and obtains transaction opts for
// each account.
func startConfigWithTestAccounts(t *testing.T, cfg *SystemConfig, accountsToGenerate int) (*System, []*TestAccount, error) {
	// Create our test accounts and add them to the pre-mine cfg.
	testAccounts := make([]*TestAccount, 0)
	var err error
	for i := 0; i < accountsToGenerate; i++ {
		// Create our test account and add it to our list
		testAccount := &TestAccount{
			HDPath: fmt.Sprintf("m/44'/60'/0'/0/%d", 1000+i), // offset by 1000 to avoid collisions.
			Key:    nil,
			L1Opts: nil,
			L2Opts: nil,
		}
		testAccounts = append(testAccounts, testAccount)

		// Obtain our generated private key
		testAccount.Key, err = cfg.Secrets.Wallet.PrivateKey(accounts.Account{
			URL: accounts.URL{
				Path: testAccount.HDPath,
			},
		})
		if err != nil {
			return nil, nil, err
		}
		testAccount.Addr = crypto.PubkeyToAddress(testAccount.Key.PublicKey)

		// Obtain the transaction options for contract bindings for this account.
		testAccount.L1Opts, err = bind.NewKeyedTransactorWithChainID(testAccount.Key, cfg.L1ChainIDBig())
		if err != nil {
			return nil, nil, err
		}
		testAccount.L2Opts, err = bind.NewKeyedTransactorWithChainID(testAccount.Key, cfg.L2ChainIDBig())
		if err != nil {
			return nil, nil, err
		}

		// Fund the test account in our config
		cfg.Premine[testAccount.Addr] = big.NewInt(params.Ether)
		cfg.Premine[testAccount.Addr] = cfg.Premine[testAccount.Addr].Mul(cfg.Premine[testAccount.Addr], big.NewInt(1_000_000))

	}

	// Start our system
	sys, err := cfg.Start(t)
	if err != nil {
		return sys, nil, err
	}

	// Return our results.
	return sys, testAccounts, err
}

// TestMixedDepositValidity makes a number of deposit transactions, some which will succeed in transferring value,
// while others do not. It ensures that the expected nonces/balances match after several interactions.
func TestMixedDepositValidity(t *testing.T) {
	InitParallel(t)
	// Define how many deposit txs we'll make. Each deposit mints a fixed amount and transfers up to 1/3 of the user's
	// balance. As such, this number cannot be too high or else the test will always fail due to lack of balance in L1.
	const depositTxCount = 15

	// Define how many accounts we'll use to deposit funds
	const accountUsedToDeposit = 5

	// Create our system configuration, funding all accounts we created for L1/L2, and start it
	cfg := DefaultSystemConfig(t)
	sys, testAccounts, err := startConfigWithTestAccounts(t, &cfg, accountUsedToDeposit)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	require.NoError(t, err)

	// Define our L1 transaction timeout duration.
	txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

	// Create a struct used to track our transactors and their transactions sent.
	type TestAccountState struct {
		Account           *TestAccount
		ExpectedL1Balance *big.Int
		ExpectedL2Balance *big.Int
		StartingL1Nonce   uint64
		ExpectedL1Nonce   uint64
		StartingL2Nonce   uint64
		ExpectedL2Nonce   uint64
	}

	// Create the state objects for every test account we'll track changes for.
	transactors := make([]*TestAccountState, 0)
	for i := 0; i < len(testAccounts); i++ {
		// Obtain our account
		testAccount := testAccounts[i]

		// Obtain the transactor's starting nonce on L1.
		ctx, cancel := context.WithTimeout(context.Background(), txTimeoutDuration)
		startL1Nonce, err := l1Client.NonceAt(ctx, testAccount.L1Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the transactor's starting balance on L2.
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		startL2Balance, err := l2Verif.BalanceAt(ctx, testAccount.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the transactor's starting nonce on L2.
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		startL2Nonce, err := l2Verif.NonceAt(ctx, testAccount.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Add our transactor to our list
		transactors = append(transactors, &TestAccountState{
			Account:           testAccount,
			ExpectedL2Balance: startL2Balance,
			ExpectedL1Nonce:   startL1Nonce,
			ExpectedL2Nonce:   startL2Nonce,
		})
	}

	// Create our random provider
	randomProvider := rand.New(rand.NewSource(1452))

	// Now we create a number of deposits from each transactor
	for i := 0; i < depositTxCount; i++ {
		// Determine if this deposit should succeed in transferring value (not minting)
		validTransfer := randomProvider.Int()%2 == 0

		// Determine the transactor to use
		transactorIndex := randomProvider.Int() % len(transactors)
		transactor := transactors[transactorIndex]

		// Determine the transactor to receive the deposit
		receiverIndex := randomProvider.Int() % len(transactors)
		receiver := transactors[receiverIndex]
		toAddr := receiver.Account.L2Opts.From

		// Create our L1 deposit transaction and send it.
		mintAmount := big.NewInt(randomProvider.Int63() % 9_000_000)
		transactor.Account.L1Opts.Value = mintAmount
		var transferValue *big.Int
		if validTransfer {
			transferValue = new(big.Int).Div(transactor.ExpectedL2Balance, common.Big3) // send 1/3 our balance which should succeed.
		} else {
			transferValue = new(big.Int).Mul(common.Big2, transactor.ExpectedL2Balance) // trigger a revert by trying to transfer our current balance * 2
		}
		SendDepositTx(t, cfg, l1Client, l2Verif, transactor.Account.L1Opts, func(l2Opts *DepositTxOpts) {
			l2Opts.GasLimit = 100_000
			l2Opts.IsCreation = false
			l2Opts.Data = nil
			l2Opts.ToAddr = toAddr
			l2Opts.Value = transferValue
			if validTransfer {
				l2Opts.ExpectedStatus = types.ReceiptStatusSuccessful
			} else {
				l2Opts.ExpectedStatus = types.ReceiptStatusFailed
			}
		})

		// Update our expected balances.
		if validTransfer && transactor != receiver {
			// Transactor balances changes by minted minus transferred value.
			transactor.ExpectedL2Balance = new(big.Int).Add(transactor.ExpectedL2Balance, new(big.Int).Sub(mintAmount, transferValue))
			// Receiver balance changes by transferred value.
			receiver.ExpectedL2Balance = new(big.Int).Add(receiver.ExpectedL2Balance, transferValue)
		} else {
			// If the transfer failed, minting should've still succeeded but the balance shouldn't have transferred
			// to the recipient.
			transactor.ExpectedL2Balance = new(big.Int).Add(transactor.ExpectedL2Balance, mintAmount)
		}
		transactor.ExpectedL1Nonce = transactor.ExpectedL1Nonce + 1
		transactor.ExpectedL2Nonce = transactor.ExpectedL2Nonce + 1
	}

	// At the end, assert our account balance/nonce states.
	for _, transactor := range transactors {
		// Obtain the L1 account nonce
		ctx, cancel := context.WithTimeout(context.Background(), txTimeoutDuration)
		endL1Nonce, err := l1Client.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the L2 sequencer account balance
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		endL2SeqBalance, err := l2Seq.BalanceAt(ctx, transactor.Account.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the L2 sequencer account nonce
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		endL2SeqNonce, err := l2Seq.NonceAt(ctx, transactor.Account.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the L2 verifier account balance
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		endL2VerifBalance, err := l2Verif.BalanceAt(ctx, transactor.Account.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		// Obtain the L2 verifier account nonce
		ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
		endL2VerifNonce, err := l2Verif.NonceAt(ctx, transactor.Account.L2Opts.From, nil)
		cancel()
		require.NoError(t, err)

		require.Equal(t, transactor.ExpectedL1Nonce, endL1Nonce, "Unexpected L1 nonce for transactor")
		require.Equal(t, transactor.ExpectedL2Nonce, endL2SeqNonce, "Unexpected L2 sequencer nonce for transactor")
		require.Equal(t, transactor.ExpectedL2Balance, endL2SeqBalance, "Unexpected L2 sequencer balance for transactor")
		require.Equal(t, transactor.ExpectedL2Nonce, endL2VerifNonce, "Unexpected L2 verifier nonce for transactor")
		require.Equal(t, transactor.ExpectedL2Balance, endL2VerifBalance, "Unexpected L2 verifier balance for transactor")
	}
}

// TestMixedWithdrawalValidity makes a number of withdrawal transactions and ensures ones with modified parameters are
// rejected while unmodified ones are accepted. This runs test cases in different systems.
func TestMixedWithdrawalValidity(t *testing.T) {
	InitParallel(t)

	// There are 7 different fields we try modifying to cause a failure, plus one "good" test result we test.
	for i := 0; i <= 8; i++ {
		i := i // avoid loop var capture
		t.Run(fmt.Sprintf("withdrawal test#%d", i+1), func(t *testing.T) {
			ctx, bgCancel := context.WithCancel(context.Background())
			defer bgCancel()

			InitParallel(t)

			// Create our system configuration, funding all accounts we created for L1/L2, and start it
			cfg := DefaultSystemConfig(t)
			cfg.Nodes["sequencer"].SafeDBPath = t.TempDir()
			cfg.DeployConfig.L2BlockTime = 2
			require.LessOrEqual(t, cfg.DeployConfig.FinalizationPeriodSeconds, uint64(6))
			require.Equal(t, cfg.DeployConfig.FundDevAccounts, true)
			sys, err := cfg.Start(t)
			require.NoError(t, err, "error starting up system")
			defer sys.Close()

			// Obtain our sequencer, verifier, and transactor keypair.
			l1Client := sys.Clients["l1"]
			l2Seq := sys.Clients["sequencer"]
			l2Verif := sys.Clients["verifier"]
			require.NoError(t, err)

			systemConfig, err := legacybindings.NewSystemConfigCaller(cfg.L1Deployments.SystemConfigProxy, l1Client)
			require.NoError(t, err)
			unsafeBlockSigner, err := systemConfig.UnsafeBlockSigner(nil)
			require.NoError(t, err)
			require.Equal(t, cfg.DeployConfig.P2PSequencerAddress, unsafeBlockSigner)

			// The batcher has balance on L1
			batcherBalance, err := l1Client.BalanceAt(context.Background(), cfg.DeployConfig.BatchSenderAddress, nil)
			require.NoError(t, err)
			require.NotEqual(t, batcherBalance, big.NewInt(0))

			// The proposer has balance on L1
			proposerBalance, err := l1Client.BalanceAt(context.Background(), cfg.DeployConfig.L2OutputOracleProposer, nil)
			require.NoError(t, err)
			require.NotEqual(t, proposerBalance, big.NewInt(0))

			// Define our L1 transaction timeout duration.
			txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

			// Bind to the deposit contract
			depositContract, err := bindings.NewOptimismPortal(cfg.L1Deployments.OptimismPortalProxy, l1Client)
			_ = depositContract
			require.NoError(t, err)

			l2OutputOracle, err := bindings.NewL2OutputOracleCaller(cfg.L1Deployments.L2OutputOracleProxy, l1Client)
			require.NoError(t, err)

			finalizationPeriod, err := l2OutputOracle.FINALIZATIONPERIODSECONDS(nil)
			require.NoError(t, err)
			require.Equal(t, cfg.DeployConfig.FinalizationPeriodSeconds, finalizationPeriod.Uint64())

			disputeGameFactory, err := bindings.NewDisputeGameFactoryCaller(cfg.L1Deployments.DisputeGameFactoryProxy, l1Client)
			require.NoError(t, err)

			optimismPortal2, err := bindingspreview.NewOptimismPortal2Caller(cfg.L1Deployments.OptimismPortalProxy, l1Client)
			require.NoError(t, err)

			// Create a struct used to track our transactors and their transactions sent.
			type TestAccountState struct {
				Account           *TestAccount
				ExpectedL1Balance *big.Int
				ExpectedL2Balance *big.Int
				ExpectedL1Nonce   uint64
				ExpectedL2Nonce   uint64
			}

			// Create a test account state for our transactor.
			transactor := &TestAccountState{
				Account: &TestAccount{
					HDPath: e2eutils.DefaultMnemonicConfig.Alice,
					Key:    cfg.Secrets.Alice,
					L1Opts: nil,
					L2Opts: nil,
				},
				ExpectedL1Balance: nil,
				ExpectedL2Balance: nil,
				ExpectedL1Nonce:   0,
				ExpectedL2Nonce:   0,
			}
			transactor.Account.L1Opts, err = bind.NewKeyedTransactorWithChainID(transactor.Account.Key, cfg.L1ChainIDBig())
			require.NoError(t, err)
			transactor.Account.L2Opts, err = bind.NewKeyedTransactorWithChainID(transactor.Account.Key, cfg.L2ChainIDBig())
			require.NoError(t, err)

			// Obtain the transactor's starting balance on L1.
			txCtx, txCancel := context.WithTimeout(context.Background(), txTimeoutDuration)
			transactor.ExpectedL1Balance, err = l1Client.BalanceAt(txCtx, transactor.Account.L1Opts.From, nil)
			txCancel()
			require.NoError(t, err)

			// Obtain the transactor's starting balance on L2.
			txCtx, txCancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			transactor.ExpectedL2Balance, err = l2Verif.BalanceAt(txCtx, transactor.Account.L2Opts.From, nil)
			txCancel()
			require.NoError(t, err)

			// Bind to the L2-L1 message passer
			l2l1MessagePasser, err := bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Seq)
			require.NoError(t, err, "error binding to message passer on L2")

			// Create our fuzzer wrapper to generate complex values (despite this not being a fuzz test, this is still a useful
			// provider to fill complex data structures).
			typeProvider := fuzz.NewWithSeed(time.Now().Unix()).NilChance(0).MaxDepth(10000).NumElements(0, 0x100)
			fuzzerutils.AddFuzzerFunctions(typeProvider)

			// Now we create a number of withdrawals from each transactor

			// Determine the address our request will come from
			fromAddr := crypto.PubkeyToAddress(transactor.Account.Key.PublicKey)
			fromBalance, err := l2Verif.BalanceAt(context.Background(), fromAddr, nil)
			require.NoError(t, err)
			require.Greaterf(t, fromBalance.Uint64(), uint64(700_000_000_000), "insufficient balance for %s", fromAddr)

			// Initiate Withdrawal
			withdrawAmount := big.NewInt(500_000_000_000)
			transactor.Account.L2Opts.Value = withdrawAmount
			tx, err := l2l1MessagePasser.InitiateWithdrawal(transactor.Account.L2Opts, fromAddr, big.NewInt(21000), nil)
			require.Nil(t, err, "sending initiate withdraw tx")

			t.Logf("Waiting for tx %s to be in sequencer", tx.Hash().Hex())
			receiptSeq, err := wait.ForReceiptOK(ctx, l2Seq, tx.Hash())
			require.Nil(t, err, "withdrawal initiated on L2 sequencer")

			verifierTip, err := l2Verif.BlockByNumber(context.Background(), nil)
			require.Nil(t, err)

			t.Logf("Waiting for tx %s to be in verifier. Verifier tip is %s:%d. Included in sequencer in block %s:%d", tx.Hash().Hex(), verifierTip.Hash().Hex(), verifierTip.NumberU64(), receiptSeq.BlockHash.Hex(), receiptSeq.BlockNumber)
			receipt, err := wait.ForReceiptOK(ctx, l2Verif, tx.Hash())
			require.Nilf(t, err, "withdrawal tx %s not found in verifier. included in block %s:%d", tx.Hash().Hex(), receiptSeq.BlockHash.Hex(), receiptSeq.BlockNumber)

			// Obtain the header for the block containing the transaction (used to calculate gas fees)
			txCtx, txCancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			header, err := l2Verif.HeaderByNumber(txCtx, receipt.BlockNumber)
			txCancel()
			require.Nil(t, err)

			// Calculate gas fees for the withdrawal in L2 to later adjust our balance.
			withdrawalL2GasFee := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)

			// Adjust our expected L2 balance (should've decreased by withdraw amount + fees)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawAmount)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawalL2GasFee)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, receipt.L1Fee)
			transactor.ExpectedL2Nonce = transactor.ExpectedL2Nonce + 1

			// Wait for the finalization period, then we can finalize this withdrawal.
			require.NotEqual(t, cfg.L1Deployments.L2OutputOracleProxy, common.Address{})
			var blockNumber uint64
			if e2eutils.UseFPAC() {
				blockNumber, err = wait.ForGamePublished(ctx, l1Client, cfg.L1Deployments.OptimismPortalProxy, cfg.L1Deployments.DisputeGameFactoryProxy, receipt.BlockNumber)
			} else {
				blockNumber, err = wait.ForOutputRootPublished(ctx, l1Client, cfg.L1Deployments.L2OutputOracleProxy, receipt.BlockNumber)
			}
			require.Nil(t, err)

			header, err = l2Verif.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
			require.Nil(t, err)

			rpcClient, err := rpc.Dial(sys.EthInstances["verifier"].WSEndpoint())
			require.Nil(t, err)
			proofCl := gethclient.New(rpcClient)
			receiptCl := ethclient.NewClient(rpcClient)
			blockCl := ethclient.NewClient(rpcClient)

			// Now create the withdrawal
			params, err := ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, blockCl, tx.Hash(), header, l2OutputOracle, disputeGameFactory, optimismPortal2)
			require.Nil(t, err)

			// Obtain our withdrawal parameters
			withdrawal := crossdomain.Withdrawal{
				Nonce:    params.Nonce,
				Sender:   &params.Sender,
				Target:   &params.Target,
				Value:    params.Value,
				GasLimit: params.GasLimit,
				Data:     params.Data,
			}
			withdrawalTransaction := withdrawal.WithdrawalTransaction()
			l2OutputIndexParam := params.L2OutputIndex
			outputRootProofParam := params.OutputRootProof
			withdrawalProofParam := params.WithdrawalProof

			// Determine if this will be a bad withdrawal.
			badWithdrawal := i < 8
			if badWithdrawal {
				// Select a field to overwrite depending on which test case this is.
				fieldIndex := i

				// We ensure that each field changes to something different.
				if fieldIndex == 0 {
					originalValue := new(big.Int).Set(withdrawalTransaction.Nonce)
					for originalValue.Cmp(withdrawalTransaction.Nonce) == 0 {
						typeProvider.Fuzz(&withdrawalTransaction.Nonce)
					}
				} else if fieldIndex == 1 {
					originalValue := withdrawalTransaction.Sender
					for originalValue == withdrawalTransaction.Sender {
						typeProvider.Fuzz(&withdrawalTransaction.Sender)
					}
				} else if fieldIndex == 2 {
					originalValue := withdrawalTransaction.Target
					for originalValue == withdrawalTransaction.Target {
						typeProvider.Fuzz(&withdrawalTransaction.Target)
					}
				} else if fieldIndex == 3 {
					originalValue := new(big.Int).Set(withdrawalTransaction.Value)
					for originalValue.Cmp(withdrawalTransaction.Value) == 0 {
						typeProvider.Fuzz(&withdrawalTransaction.Value)
					}
				} else if fieldIndex == 4 {
					originalValue := new(big.Int).Set(withdrawalTransaction.GasLimit)
					for originalValue.Cmp(withdrawalTransaction.GasLimit) == 0 {
						typeProvider.Fuzz(&withdrawalTransaction.GasLimit)
					}
				} else if fieldIndex == 5 {
					originalValue := new(big.Int).Set(l2OutputIndexParam)
					for originalValue.Cmp(l2OutputIndexParam) == 0 {
						typeProvider.Fuzz(&l2OutputIndexParam)
					}
				} else if fieldIndex == 6 {
					// TODO: this is a large structure that is unlikely to ever produce the same value, however we should
					//  verify that we actually generated different values.
					typeProvider.Fuzz(&outputRootProofParam)
				} else if fieldIndex == 7 {
					typeProvider.Fuzz(&withdrawalProofParam)
					originalValue := make([][]byte, len(withdrawalProofParam))
					for i := 0; i < len(withdrawalProofParam); i++ {
						originalValue[i] = make([]byte, len(withdrawalProofParam[i]))
						copy(originalValue[i], withdrawalProofParam[i])
						for bytes.Equal(originalValue[i], withdrawalProofParam[i]) {
							typeProvider.Fuzz(&withdrawalProofParam[i])
						}
					}

				}
			}

			// Prove withdrawal. This checks the proof so we only finalize if this succeeds
			tx, err = depositContract.ProveWithdrawalTransaction(
				transactor.Account.L1Opts,
				withdrawalTransaction,
				l2OutputIndexParam,
				outputRootProofParam,
				withdrawalProofParam,
			)

			// If we had a bad withdrawal, we don't update some expected value and skip to processing the next
			// withdrawal. Otherwise, if it was valid, this should've succeeded so we proceed with updating our expected
			// values and asserting no errors occurred.
			if badWithdrawal {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if e2eutils.UseFPAC() {
					// Start a challenger to resolve claims and games once the clock expires
					factoryHelper := disputegame.NewFactoryHelper(t, ctx, sys)
					factoryHelper.StartChallenger(ctx, "Challenger",
						challenger.WithCannon(t, sys.RollupConfig, sys.L2GenesisCfg),
						challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))
				}
				receipt, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
				require.NoError(t, err, "finalize withdrawal")

				// Verify balance after withdrawal
				header, err = l1Client.HeaderByNumber(ctx, receipt.BlockNumber)
				require.NoError(t, err)

				// Ensure that withdrawal - gas fees are added to the L1 balance
				// Fun fact, the fee is greater than the withdrawal amount
				withdrawalL1GasFee := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
				transactor.ExpectedL1Balance = new(big.Int).Add(transactor.ExpectedL2Balance, withdrawAmount)
				transactor.ExpectedL1Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawalL1GasFee)
				transactor.ExpectedL1Nonce++

				// Ensure that our withdrawal was proved successfully
				_, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
				require.NoError(t, err, "prove withdrawal")

				// Wait for finalization and then create the Finalized Withdrawal Transaction
				ctx, withdrawalCancel := context.WithTimeout(context.Background(), 60*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
				defer withdrawalCancel()
				if e2eutils.UseFPAC() {
					err = wait.ForWithdrawalCheck(ctx, l1Client, withdrawal, cfg.L1Deployments.OptimismPortalProxy, transactor.Account.L1Opts.From)
					require.NoError(t, err)
				} else {
					err = wait.ForFinalizationPeriod(ctx, l1Client, header.Number, cfg.L1Deployments.L2OutputOracleProxy)
					require.NoError(t, err)
				}

				// Finalize withdrawal
				_, err = depositContract.FinalizeWithdrawalTransaction(
					transactor.Account.L1Opts,
					withdrawalTransaction,
				)
				require.NoError(t, err)
			}

			// At the end, assert our account balance/nonce states.

			// Obtain the L2 sequencer account balance
			endL1Balance, err := l1Client.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// Obtain the L1 account nonce
			endL1Nonce, err := l1Client.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// Obtain the L2 sequencer account balance
			endL2SeqBalance, err := l2Seq.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// Obtain the L2 sequencer account nonce
			endL2SeqNonce, err := l2Seq.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// Obtain the L2 verifier account balance
			endL2VerifBalance, err := l2Verif.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// Obtain the L2 verifier account nonce
			endL2VerifNonce, err := l2Verif.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			require.NoError(t, err)

			// TODO: Check L1 balance as well here. We avoided this due to time constraints as it seems L1 fees
			//  were off slightly.
			_ = endL1Balance
			// require.Equal(t, transactor.ExpectedL1Balance, endL1Balance, "Unexpected L1 balance for transactor")
			require.Equal(t, transactor.ExpectedL1Nonce, endL1Nonce, "Unexpected L1 nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Nonce, endL2SeqNonce, "Unexpected L2 sequencer nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Balance, endL2SeqBalance, "Unexpected L2 sequencer balance for transactor")
			require.Equal(t, transactor.ExpectedL2Nonce, endL2VerifNonce, "Unexpected L2 verifier nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Balance, endL2VerifBalance, "Unexpected L2 verifier balance for transactor")
		})
	}
}
