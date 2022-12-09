package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
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
