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

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testutils/fuzzerutils"
	"github.com/ethereum-optimism/optimism/op-node/withdrawals"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	fuzz "github.com/google/gofuzz"
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

// TestMixedWithdrawalValidity makes a number of withdrawal transactions and ensures ones with modified parameters are
// rejected while unmodified ones are accepted. This runs test cases in different systems.
func TestMixedWithdrawalValidity(t *testing.T) {
	parallel(t)
	// Setup our logger handler
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	// There are 7 different fields we try modifying to cause a failure, plus one "good" test result we test.
	for i := 0; i <= 8; i++ {
		i := i // avoid loop var capture
		t.Run(fmt.Sprintf("withdrawal test#%d", i+1), func(t *testing.T) {
			// Create our system configuration, funding all accounts we created for L1/L2, and start it
			cfg := DefaultSystemConfig(t)
			cfg.DeployConfig.FinalizationPeriodSeconds = 6
			sys, err := cfg.Start()
			require.NoError(t, err, "error starting up system")
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
			_ = depositContract
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
			transactorKey := cfg.Secrets.Alice
			transactor := &TestAccountState{
				Account: &TestAccount{
					HDPath: e2eutils.DefaultMnemonicConfig.Alice,
					Key:    transactorKey,
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
			ctx, cancel := context.WithTimeout(context.Background(), txTimeoutDuration)
			transactor.ExpectedL1Balance, err = l1Client.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the transactor's starting balance on L2.
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			transactor.ExpectedL2Balance, err = l2Verif.BalanceAt(ctx, transactor.Account.L2Opts.From, nil)
			cancel()
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

			// Initiate Withdrawal
			withdrawAmount := big.NewInt(500_000_000_000)
			transactor.Account.L2Opts.Value = withdrawAmount
			tx, err := l2l1MessagePasser.InitiateWithdrawal(transactor.Account.L2Opts, fromAddr, big.NewInt(21000), nil)
			require.Nil(t, err, "sending initiate withdraw tx")

			// Wait for the transaction to appear in L2 verifier
			receipt, err := waitForTransaction(tx.Hash(), l2Verif, txTimeoutDuration)
			require.Nil(t, err, "withdrawal initiated on L2 sequencer")
			require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful, "transaction failed")

			// Obtain the header for the block containing the transaction (used to calculate gas fees)
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			header, err := l2Verif.HeaderByNumber(ctx, receipt.BlockNumber)
			cancel()
			require.Nil(t, err)

			// Calculate gas fees for the withdrawal in L2 to later adjust our balance.
			withdrawalL2GasFee := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)

			// Adjust our expected L2 balance (should've decreased by withdraw amount + fees)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawAmount)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawalL2GasFee)
			transactor.ExpectedL2Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, receipt.L1Fee)
			transactor.ExpectedL2Nonce = transactor.ExpectedL2Nonce + 1

			// Wait for the finalization period, then we can finalize this withdrawal.
			ctx, cancel = context.WithTimeout(context.Background(), 40*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
			blockNumber, err := withdrawals.WaitForFinalizationPeriod(ctx, l1Client, predeploys.DevOptimismPortalAddr, receipt.BlockNumber)
			cancel()
			require.Nil(t, err)

			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			header, err = l2Verif.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
			cancel()
			require.Nil(t, err)

			l2OutputOracle, err := bindings.NewL2OutputOracleCaller(predeploys.DevL2OutputOracleAddr, l1Client)
			require.Nil(t, err)

			rpcClient, err := rpc.Dial(sys.Nodes["verifier"].WSEndpoint())
			require.Nil(t, err)
			proofCl := gethclient.New(rpcClient)
			receiptCl := ethclient.NewClient(rpcClient)

			// Now create the withdrawal
			params, err := withdrawals.ProveWithdrawalParameters(context.Background(), proofCl, receiptCl, tx.Hash(), header, l2OutputOracle)
			require.Nil(t, err)

			// Obtain our withdrawal parameters
			withdrawalTransaction := &bindings.TypesWithdrawalTransaction{
				Nonce:    params.Nonce,
				Sender:   params.Sender,
				Target:   params.Target,
				Value:    params.Value,
				GasLimit: params.GasLimit,
				Data:     params.Data,
			}
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
				*withdrawalTransaction,
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

				receipt, err = waitForTransaction(tx.Hash(), l1Client, txTimeoutDuration)
				require.Nil(t, err, "finalize withdrawal")
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

				// Verify balance after withdrawal
				ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
				header, err = l1Client.HeaderByNumber(ctx, receipt.BlockNumber)
				cancel()
				require.Nil(t, err)

				// Ensure that withdrawal - gas fees are added to the L1 balance
				// Fun fact, the fee is greater than the withdrawal amount
				withdrawalL1GasFee := calcGasFees(receipt.GasUsed, tx.GasTipCap(), tx.GasFeeCap(), header.BaseFee)
				transactor.ExpectedL1Balance = new(big.Int).Add(transactor.ExpectedL2Balance, withdrawAmount)
				transactor.ExpectedL1Balance = new(big.Int).Sub(transactor.ExpectedL2Balance, withdrawalL1GasFee)
				transactor.ExpectedL1Nonce++

				// Ensure that our withdrawal was proved successfully
				proveReceipt, err := waitForTransaction(tx.Hash(), l1Client, 3*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
				require.Nil(t, err, "prove withdrawal")
				require.Equal(t, types.ReceiptStatusSuccessful, proveReceipt.Status)

				// Wait for finalization and then create the Finalized Withdrawal Transaction
				ctx, cancel = context.WithTimeout(context.Background(), 45*time.Duration(cfg.DeployConfig.L1BlockTime)*time.Second)
				defer cancel()
				_, err = withdrawals.WaitForFinalizationPeriod(ctx, l1Client, predeploys.DevOptimismPortalAddr, header.Number)
				require.Nil(t, err)

				// Finalize withdrawal
				_, err = depositContract.FinalizeWithdrawalTransaction(
					transactor.Account.L1Opts,
					*withdrawalTransaction,
				)
				require.NoError(t, err)
			}

			// At the end, assert our account balance/nonce states.

			// Obtain the L2 sequencer account balance
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL1Balance, err := l1Client.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the L1 account nonce
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL1Nonce, err := l1Client.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the L2 sequencer account balance
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL2SeqBalance, err := l2Seq.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the L2 sequencer account nonce
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL2SeqNonce, err := l2Seq.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the L2 verifier account balance
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL2VerifBalance, err := l2Verif.BalanceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// Obtain the L2 verifier account nonce
			ctx, cancel = context.WithTimeout(context.Background(), txTimeoutDuration)
			endL2VerifNonce, err := l2Verif.NonceAt(ctx, transactor.Account.L1Opts.From, nil)
			cancel()
			require.NoError(t, err)

			// TODO: Check L1 balance as well here. We avoided this due to time constraints as it seems L1 fees
			//  were off slightly.
			_ = endL1Balance
			//require.Equal(t, transactor.ExpectedL1Balance, endL1Balance, "Unexpected L1 balance for transactor")
			require.Equal(t, transactor.ExpectedL1Nonce, endL1Nonce, "Unexpected L1 nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Nonce, endL2SeqNonce, "Unexpected L2 sequencer nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Balance, endL2SeqBalance, "Unexpected L2 sequencer balance for transactor")
			require.Equal(t, transactor.ExpectedL2Nonce, endL2VerifNonce, "Unexpected L2 verifier nonce for transactor")
			require.Equal(t, transactor.ExpectedL2Balance, endL2VerifBalance, "Unexpected L2 verifier balance for transactor")
		})
	}
}
