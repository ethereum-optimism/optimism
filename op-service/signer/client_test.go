package signer

import (
	"context"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// fakeSignerService serves eth_signTransaction, returning a canned response.
type fakeSignerService struct {
	signed hexutil.Bytes
}

func (f *fakeSignerService) SignTransaction(args TransactionArgs) (hexutil.Bytes, error) {
	return f.signed, nil
}

func newTestSignerClient(t *testing.T, svc *fakeSignerService) *SignerClient {
	server := rpc.NewServer()
	t.Cleanup(server.Stop)
	require.NoError(t, server.RegisterName("eth", svc))
	client := rpc.DialInProc(server)
	t.Cleanup(client.Close)
	return &SignerClient{client: client, logger: testlog.Logger(t, log.LevelDebug)}
}

func testBlobTx(t *testing.T, chainID *big.Int) (unsigned, signed *types.Transaction) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sidecar := &types.BlobTxSidecar{
		Blobs:       []kzg4844.Blob{{}},
		Commitments: []kzg4844.Commitment{{}},
		Proofs:      []kzg4844.Proof{{}},
	}
	unsigned = types.NewTx(&types.BlobTx{
		ChainID:    uint256.MustFromBig(chainID),
		Nonce:      1,
		GasTipCap:  uint256.NewInt(1),
		GasFeeCap:  uint256.NewInt(2),
		Gas:        21000,
		To:         common.Address{0x01},
		BlobFeeCap: uint256.NewInt(3),
		BlobHashes: []common.Hash{{0x01}},
		Sidecar:    sidecar,
	})
	signed, err = types.SignTx(unsigned, types.NewCancunSigner(chainID), key)
	require.NoError(t, err)
	return unsigned, signed
}

func TestSignTransactionReattachesBlobSidecar(t *testing.T) {
	chainID := big.NewInt(901)
	unsigned, signed := testBlobTx(t, chainID)

	// The remote signer is sent the tx without its sidecar and returns it that way.
	signedNoSidecar, err := signed.WithoutBlobTxSidecar().MarshalBinary()
	require.NoError(t, err)
	client := newTestSignerClient(t, &fakeSignerService{signed: signedNoSidecar})

	got, err := client.SignTransaction(context.Background(), chainID, common.Address{0xaa}, unsigned)
	require.NoError(t, err)
	require.EqualValues(t, types.BlobTxType, got.Type())
	require.Equal(t, unsigned.BlobTxSidecar(), got.BlobTxSidecar())
	require.Equal(t, signed.Hash(), got.Hash())
}

func TestSignTransactionNonBlobResponseForBlobTx(t *testing.T) {
	chainID := big.NewInt(901)
	unsigned, _ := testBlobTx(t, chainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	plain, err := types.SignNewTx(key, types.NewLondonSigner(chainID), &types.DynamicFeeTx{
		ChainID:   chainID,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       21000,
		To:        &common.Address{0x01},
	})
	require.NoError(t, err)
	plainRaw, err := plain.MarshalBinary()
	require.NoError(t, err)
	client := newTestSignerClient(t, &fakeSignerService{signed: plainRaw})

	_, err = client.SignTransaction(context.Background(), chainID, common.Address{0xaa}, unsigned)
	require.ErrorContains(t, err, "not a blob tx")
}

func TestSignTransactionPlainTx(t *testing.T) {
	chainID := big.NewInt(901)
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	unsigned := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       21000,
		To:        &common.Address{0x01},
	})
	signed, err := types.SignTx(unsigned, types.NewLondonSigner(chainID), key)
	require.NoError(t, err)
	signedRaw, err := signed.MarshalBinary()
	require.NoError(t, err)
	client := newTestSignerClient(t, &fakeSignerService{signed: signedRaw})

	got, err := client.SignTransaction(context.Background(), chainID, common.Address{0xaa}, unsigned)
	require.NoError(t, err)
	require.Nil(t, got.BlobTxSidecar())
	require.Equal(t, signed.Hash(), got.Hash())
}
