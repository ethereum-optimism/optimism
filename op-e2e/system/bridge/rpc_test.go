package bridge

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestL2SequencerRPCDepositTx checks that the L2 sequencer will not accept DepositTx type transactions.
// The acceptance of these transactions would allow for arbitrary minting of ETH in L2.
func TestL2SequencerRPCDepositTx(t *testing.T) {
	op_e2e.InitParallel(t)

	// Create our system configuration for L1/L2 and start it
	cfg := e2esys.DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	// Obtain our sequencer, verifier, and transactor keypair.
	l2Seq := sys.NodeClient("sequencer")
	l2Verif := sys.NodeClient("verifier")
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
