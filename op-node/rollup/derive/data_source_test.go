package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func encodeDeposit(t *testing.T, from common.Address, to *common.Address, data []byte) hexutil.Bytes {
	t.Helper()
	raw, err := (&optypes.DepositTx{
		SourceHash: common.Hash{0x01},
		From:       from,
		To:         to,
		Value:      big.NewInt(0),
		Gas:        1_000_000,
		Data:       data,
	}).MarshalBinary()
	require.NoError(t, err)
	return raw
}

// TestBatcherTxsOPStackL1 checks the batcher transactions read from the transactions of an L1
// block that is an OP Stack block: a deposit is a batcher transaction when it is sent to the batch
// inbox from the batcher address, and a post-exec transaction never is.
func TestBatcherTxsOPStackL1(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	batcherKey := testutils.InsecureRandomKey(rng)
	batcherAddr := crypto.PubkeyToAddress(batcherKey.PublicKey)
	inbox := testutils.RandomAddress(rng)
	other := testutils.RandomAddress(rng)
	signer := types.NewPragueSigner(big.NewInt(901))
	cfg := DataSourceConfig{l1Signer: signer, batchInboxAddress: inbox}
	logger := testlog.Logger(t, log.LevelCrit)

	userBatch := types.MustSignNewTx(batcherKey, signer, &types.DynamicFeeTx{
		ChainID: signer.ChainID(), Gas: 100_000, GasFeeCap: big.NewInt(10), To: &inbox,
		Data: testutils.RandomData(rng, 100),
	})
	userRaw, err := userBatch.MarshalBinary()
	require.NoError(t, err)
	blobBatch := types.MustSignNewTx(batcherKey, signer, &types.BlobTx{
		ChainID: uint256.NewInt(901), Gas: 100_000, To: inbox,
		Data:       testutils.RandomData(rng, 10),
		BlobHashes: []common.Hash{testutils.RandomHash(rng)},
	})
	blobRaw, err := blobBatch.MarshalBinary()
	require.NoError(t, err)
	postExec, err := (&optypes.PostExecTx{Data: []byte{0xc2, 0x80, 0x80}}).MarshalBinary()
	require.NoError(t, err)

	depositData := testutils.RandomData(rng, 120)
	depositBatch := encodeDeposit(t, batcherAddr, &inbox, depositData)
	txs := []hexutil.Bytes{
		encodeDeposit(t, other, &other, testutils.RandomData(rng, 50)), // L1-info-like deposit
		depositBatch,
		encodeDeposit(t, other, &inbox, testutils.RandomData(rng, 60)),       // unauthorized sender
		encodeDeposit(t, batcherAddr, &other, testutils.RandomData(rng, 70)), // other recipient
		encodeDeposit(t, batcherAddr, nil, testutils.RandomData(rng, 80)),    // contract creation
		userRaw,
		blobRaw,
		postExec,
	}

	t.Run("calldata", func(t *testing.T) {
		out, err := DataFromEVMTransactions(cfg, batcherAddr, txs, logger)
		require.NoError(t, err)
		require.Equal(t, []eth.Data{depositData, userBatch.Data(), blobBatch.Data()}, out)
	})

	t.Run("blobs", func(t *testing.T) {
		data, hashes, err := dataAndHashesFromTxs(txs, &cfg, batcherAddr, logger)
		require.NoError(t, err)
		require.Equal(t, blobBatch.BlobHashes(), hashes)
		require.Len(t, data, 3)
		require.Equal(t, eth.Data(depositData), *data[0].calldata)
		require.Equal(t, userBatch.Data(), []byte(*data[1].calldata))
		require.Nil(t, data[2].calldata)
	})
}

// TestBatcherTxsUndecodable checks that a transaction that does not decode fails the whole block,
// whatever its type.
func TestBatcherTxsUndecodable(t *testing.T) {
	cfg := DataSourceConfig{l1Signer: types.NewPragueSigner(big.NewInt(901))}
	logger := testlog.Logger(t, log.LevelCrit)
	for _, tc := range []struct {
		name string
		raw  hexutil.Bytes
	}{
		{"empty", hexutil.Bytes{}},
		{"deposit", hexutil.Bytes{optypes.DepositTxType, 0xc1}},
		{"post-exec without payload", hexutil.Bytes{optypes.PostExecTxType}},
		{"unknown type", hexutil.Bytes{0x05, 0xc0}},
		{"dynamic fee", hexutil.Bytes{types.DynamicFeeTxType, 0xc0}},
		{"legacy", hexutil.Bytes{0xc1, 0x80}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DataFromEVMTransactions(cfg, common.Address{}, []hexutil.Bytes{tc.raw}, logger)
			require.Error(t, err)
			_, _, err = dataAndHashesFromTxs([]hexutil.Bytes{tc.raw}, &cfg, common.Address{}, logger)
			require.Error(t, err)
		})
	}
}

// TestDataSourceUndecodableTx checks that the calldata and blob data sources fail with a temporary
// error on a block holding a transaction that does not decode.
func TestDataSourceUndecodableTx(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	ref := testutils.RandomBlockRef(rng)
	cfg := DataSourceConfig{l1Signer: types.NewPragueSigner(big.NewInt(901))}
	logger := testlog.Logger(t, log.LevelCrit)
	txs := []hexutil.Bytes{{optypes.DepositTxType, 0xc1}}

	t.Run("calldata", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectInfoAndRawTxsByHash(ref.Hash, testutils.RandomBlockInfo(rng), txs, nil)
		l1F.ExpectInfoAndRawTxsByHash(ref.Hash, testutils.RandomBlockInfo(rng), txs, nil)
		src := NewCalldataSource(t.Context(), logger, cfg, l1F, ref, common.Address{})
		_, err := src.Next(t.Context())
		require.ErrorIs(t, err, ErrTemporary)
	})

	t.Run("blobs", func(t *testing.T) {
		l1F := &testutils.MockL1Source{}
		defer l1F.AssertExpectations(t)
		l1F.ExpectInfoAndRawTxsByHash(ref.Hash, testutils.RandomBlockInfo(rng), txs, nil)
		src := NewBlobDataSource(t.Context(), logger, cfg, l1F, nil, ref, common.Address{})
		_, err := src.Next(t.Context())
		require.ErrorIs(t, err, ErrTemporary)
	})
}
