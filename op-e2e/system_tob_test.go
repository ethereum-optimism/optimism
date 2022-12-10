package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestGasPriceOracleFeeUpdates checks that the gas price oracle cannot be locked by mis-configuring parameters.
func TestGasPriceOracleFeeUpdates(t *testing.T) {
	parallel(t)
	// Define our values to set in the GasPriceOracle (we set them high to see if it can lock L2 or stop bindings
	// from updating the prices once again.
	overheadValue := abi.MaxUint256
	scalarValue := abi.MaxUint256
	var cancel context.CancelFunc

	// Setup our logger handler
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	// Create our system configuration for L1/L2 and start it
	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	// l2Verif := sys.Clients["verifier"]
	ethPrivKey := cfg.Secrets.SysCfgOwner

	// Bind to the SystemConfig & GasPriceOracle contracts
	sysconfig, err := bindings.NewSystemConfig(predeploys.DevSystemConfigAddr, l1Client)
	require.Nil(t, err)
	gpoContract, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, l2Seq)
	require.Nil(t, err)

	// Obtain our signer.
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.Nil(t, err)

	// Define our L1 transaction timeout duration.
	txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

	// Update the gas config, wait for it to show up on L2, & verify that it was set as intended
	opts.Context, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
	tx, err := sysconfig.SetGasConfig(opts, overheadValue, scalarValue)
	cancel()
	require.Nil(t, err, "sending overhead update tx")

	receipt, err := waitForTransaction(tx.Hash(), l1Client, txTimeoutDuration)
	require.Nil(t, err, "waiting for sysconfig set gas config update tx")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	_, err = waitForL1OriginOnL2(receipt.BlockNumber.Uint64(), l2Seq, txTimeoutDuration)
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

	receipt, err = waitForTransaction(tx.Hash(), l1Client, txTimeoutDuration)
	require.Nil(t, err, "waiting for sysconfig set gas config update tx")
	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

	_, err = waitForL1OriginOnL2(receipt.BlockNumber.Uint64(), l2Seq, txTimeoutDuration)
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
	parallel(t)
	// Setup our logger handler
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	// Create our system configuration for L1/L2 and start it
	cfg := DefaultSystemConfig(t)
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	txSigningKey := sys.cfg.Secrets.Alice
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
func startConfigWithTestAccounts(cfg *SystemConfig, accountsToGenerate int) (*System, []*TestAccount, error) {
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
	sys, err := cfg.Start()
	if err != nil {
		return sys, nil, err
	}

	// Return our results.
	return sys, testAccounts, err
}

// TestMixedDepositValidity makes a number of deposit transactions, some which will succeed in transferring value,
// while others do not. It ensures that the expected nonces/balances match after several interactions.
func TestMixedDepositValidity(t *testing.T) {
	parallel(t)
	// Define how many deposit txs we'll make. Each deposit mints a fixed amount and transfers up to 1/3 of the user's
	// balance. As such, this number cannot be too high or else the test will always fail due to lack of balance in L1.
	const depositTxCount = 15

	// Define how many accounts we'll use to deposit funds
	const accountUsedToDeposit = 5

	// Setup our logger handler
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	// Create our system configuration, funding all accounts we created for L1/L2, and start it
	cfg := DefaultSystemConfig(t)
	sys, testAccounts, err := startConfigWithTestAccounts(&cfg, accountUsedToDeposit)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	// Obtain our sequencer, verifier, and transactor keypair.
	l1Client := sys.Clients["l1"]
	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]
	require.NoError(t, err)

	// Define our L1 transaction timeout duration.
	txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

	// Bind to the deposit contract
	depositContract, err := bindings.NewOptimismPortal(predeploys.DevOptimismPortalAddr, l1Client)
	require.NoError(t, err)

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
	randomProvider := rand.New(rand.NewSource(time.Now().Unix()))

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
		tx, err := depositContract.DepositTransaction(transactor.Account.L1Opts, toAddr, transferValue, 100_000, false, nil)
		require.Nil(t, err, "with deposit tx")

		// Wait for the deposit tx to appear in L1.
		receipt, err := waitForTransaction(tx.Hash(), l1Client, txTimeoutDuration)
		require.Nil(t, err, "Waiting for deposit tx on L1")
		require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

		// Reconstruct the L2 tx hash to wait for the deposit in L2.
		reconstructedDep, err := derive.UnmarshalDepositLogEvent(receipt.Logs[0])
		require.NoError(t, err, "Could not reconstruct L2 Deposit")
		tx = types.NewTx(reconstructedDep)
		receipt, err = waitForTransaction(tx.Hash(), l2Verif, txTimeoutDuration)
		require.NoError(t, err)

		// Verify the result of the L2 tx receipt. Based on how much we transferred it should be successful/failed.
		if validTransfer {
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "Transaction should have succeeded")
		} else {
			require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "Transaction should have failed")
		}

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
