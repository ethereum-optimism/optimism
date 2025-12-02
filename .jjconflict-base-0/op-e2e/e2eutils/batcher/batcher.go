package batcher

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type Helper struct {
	t         *testing.T
	privKey   *ecdsa.PrivateKey
	rollupCfg *rollup.Config
	l1Client  *ethclient.Client
}

func NewHelper(t *testing.T, privKey *ecdsa.PrivateKey, rollupCfg *rollup.Config, l1Client *ethclient.Client) *Helper {
	return &Helper{
		t:         t,
		privKey:   privKey,
		rollupCfg: rollupCfg,
		l1Client:  l1Client,
	}
}

func (h *Helper) SendLargeInvalidBatch(ctx context.Context) {
	nonce, err := h.l1Client.PendingNonceAt(ctx, crypto.PubkeyToAddress(h.privKey.PublicKey))
	require.NoError(h.t, err, "Should get next batcher nonce")

	maxTxDataSize := 131072 // As per the Ethereum spec.
	data := testutils.RandomData(rand.New(rand.NewSource(9849248)), maxTxDataSize-200)
	tx := gethTypes.MustSignNewTx(h.privKey, h.rollupCfg.L1Signer(), &gethTypes.DynamicFeeTx{
		ChainID:   h.rollupCfg.L1ChainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(1 * params.GWei),
		GasFeeCap: big.NewInt(10 * params.GWei),
		Gas:       5_000_000,
		To:        &h.rollupCfg.BatchInboxAddress,
		Value:     big.NewInt(0),
		Data:      data,
	})
	err = h.l1Client.SendTransaction(ctx, tx)
	require.NoError(h.t, err, "Should send large batch transaction")
	_, err = wait.ForReceiptOK(ctx, h.l1Client, tx.Hash())
	require.NoError(h.t, err, "Tx should be ok")
}
