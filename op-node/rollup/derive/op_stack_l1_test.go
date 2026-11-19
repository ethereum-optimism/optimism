package derive_test

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// opStackL1Block returns an L1 block shaped like an OP Stack block, as the L1 of a chain whose L1
// is an OP Stack chain sees it, and the batcher data that derivation must read from it, in order.
// It holds the L1-info deposit, user transactions and deposits to and from the batch inbox and
// batcher addresses, and a trailing post-exec transaction.
func opStackL1Block(t *testing.T, cfg *rollup.Config, batcherKey *ecdsa.PrivateKey) (testutils.RPCBlock, []eth.Data) {
	rng := rand.New(rand.NewSource(1234))
	signer := cfg.L1Signer()
	batcherAddr := crypto.PubkeyToAddress(batcherKey.PublicKey)
	otherKey := testutils.InsecureRandomKey(rng)
	otherAddr := testutils.RandomAddress(rng)

	userTx := func(key *ecdsa.PrivateKey, nonce uint64, to common.Address, data []byte) *types.Transaction {
		return types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   cfg.L1ChainID,
			Nonce:     nonce,
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(100),
			Gas:       1_000_000,
			To:        &to,
			Data:      data,
		})
	}
	deposit := func(source byte, from common.Address, to common.Address, data []byte) *optypes.DepositTx {
		return &optypes.DepositTx{
			SourceHash: common.Hash{source},
			From:       from,
			To:         &to,
			Mint:       big.NewInt(0),
			Value:      big.NewInt(0),
			Gas:        1_000_000,
			Data:       data,
		}
	}

	l1Info := deposit(0x01, derive.L1InfoDepositerAddress, predeploys.L1BlockAddr, testutils.RandomData(rng, 260))
	l1Info.IsSystemTransaction = false
	userBatch := userTx(batcherKey, 0, cfg.BatchInboxAddress, testutils.RandomData(rng, 300))
	userOther := userTx(batcherKey, 1, otherAddr, testutils.RandomData(rng, 100))
	userUnauthorized := userTx(otherKey, 0, cfg.BatchInboxAddress, testutils.RandomData(rng, 120))
	depositBatch := deposit(0x02, batcherAddr, cfg.BatchInboxAddress, testutils.RandomData(rng, 400))
	depositUnauthorized := deposit(0x03, otherAddr, cfg.BatchInboxAddress, testutils.RandomData(rng, 150))
	depositToOther := deposit(0x04, batcherAddr, otherAddr, testutils.RandomData(rng, 90))
	legacyBatch := types.MustSignNewTx(batcherKey, signer, &types.LegacyTx{
		Nonce:    2,
		GasPrice: big.NewInt(100),
		Gas:      1_000_000,
		To:       &cfg.BatchInboxAddress,
		Data:     testutils.RandomData(rng, 200),
	})
	postExec := &optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}

	txs := []testutils.RPCTx{l1Info, userBatch, userOther, userUnauthorized, depositBatch, depositUnauthorized, depositToOther, legacyBatch, postExec}
	var receipts optypes.Receipts
	for i, tx := range txs {
		raw, err := tx.MarshalBinary()
		require.NoError(t, err)
		r := &optypes.Receipt{Receipt: types.Receipt{
			Status:            types.ReceiptStatusSuccessful,
			CumulativeGasUsed: uint64(i+1) * 50_000,
			Logs:              []*types.Log{},
		}}
		switch {
		case optypes.IsDepositTxBytes(raw):
			r.Type = optypes.DepositTxType
			r.DepositNonce = ptr.New(uint64(i))
			r.DepositReceiptVersion = ptr.New(uint64(1))
		case optypes.IsPostExecTxBytes(raw):
			r.Type = optypes.PostExecTxType
		default:
			r.Type = raw[0]
			if raw[0] >= 0x80 {
				r.Type = types.LegacyTxType
			}
		}
		receipts = append(receipts, r)
	}
	block := testutils.NewRPCBlock(t, 100, txs, receipts)
	return block, []eth.Data{userBatch.Data(), depositBatch.Data, legacyBatch.Data()}
}

// noBlobs is an L1BlobsFetcher for an L1 without blobs, which an OP Stack chain is.
type noBlobs struct{}

func (noBlobs) GetBlobsByHash(context.Context, uint64, []common.Hash) ([]*eth.Blob, error) {
	return nil, errors.New("an OP Stack L1 has no blobs")
}

// TestDataSourceOPStackL1 reads an OP Stack-shaped L1 block through the sources L1 client and the
// calldata and blob data sources, and checks that they yield the batcher data of its user and
// deposit-type batcher transactions in block order.
func TestDataSourceOPStackL1(t *testing.T) {
	batcherKey := testutils.InsecureRandomKey(rand.New(rand.NewSource(42)))
	batcherAddr := crypto.PubkeyToAddress(batcherKey.PublicKey)

	for _, tc := range []struct {
		name        string
		ecotoneTime *uint64
	}{
		{"calldata", nil},
		{"blobs", ptr.New(uint64(0))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := testlog.Logger(t, log.LevelInfo)
			cfg := &rollup.Config{
				L1ChainID:         big.NewInt(901),
				BatchInboxAddress: common.HexToAddress("0xff00000000000000000000000000000000000901"),
				EcotoneTime:       tc.ecotoneTime,
			}
			block, want := opStackL1Block(t, cfg, batcherKey)
			l1, err := sources.NewL1Client(testutils.NewBlockRPC(block), logger, nil,
				sources.L1ClientSimpleConfig(false, sources.RPCKindStandard, 10))
			require.NoError(t, err)

			src, err := derive.NewDataSourceFactory(logger, cfg, l1, noBlobs{}, nil).OpenData(t.Context(), block.Ref, batcherAddr)
			require.NoError(t, err)
			var got []eth.Data
			for {
				data, err := src.Next(t.Context())
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				got = append(got, data)
			}
			require.Equal(t, want, got)
		})
	}
}
