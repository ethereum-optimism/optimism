package derive

import (
	"context"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// TestPlasmaDataSource verifies that commitments are correctly read from l1 and then
// forwarded to the Plasma DA to return the correct inputs in the iterator.
func TestPlasmaDataSource(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	ctx := context.Background()

	rng := rand.New(rand.NewSource(1234))

	l1F := &testutils.MockL1Source{}

	storage := plasma.NewMockDAClient(logger)

	da := plasma.NewPlasmaDAWithStorage(logger, storage)

	// Create rollup genesis and config
	l1Time := uint64(2)
	refA := testutils.RandomBlockRef(rng)
	refA.Number = 1
	l1Refs := []eth.L1BlockRef{refA}
	refA0 := eth.L2BlockRef{
		Hash:           testutils.RandomHash(rng),
		Number:         0,
		ParentHash:     common.Hash{},
		Time:           refA.Time,
		L1Origin:       refA.ID(),
		SequenceNumber: 0,
	}
	batcherPriv := testutils.RandomKey()
	batcherAddr := crypto.PubkeyToAddress(batcherPriv.PublicKey)
	batcherInbox := common.Address{42}
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     refA.ID(),
			L2:     refA0.ID(),
			L2Time: refA0.Time,
		},
		BlockTime:          1,
		SeqWindowSize:      20,
		BatchInboxAddress:  batcherInbox,
		DAChallengeAddress: common.Address{43},
	}
	// keep track of random input data to validate against
	var inputs [][]byte

	signer := cfg.L1Signer()

	factory := NewDataSourceFactory(logger, cfg, l1F, nil, da)

	for i := uint64(0); i <= 18; i++ {
		parent := l1Refs[len(l1Refs)-1]
		// create a new mock l1 ref
		ref := eth.L1BlockRef{
			Hash:       testutils.RandomHash(rng),
			Number:     parent.Number + 1,
			ParentHash: parent.Hash,
			Time:       parent.Time + l1Time,
		}
		l1Refs = append(l1Refs, ref)
		logger.Info("new l1 block", "ref", ref)

		// pick a random number of commitments to include in the l1 block
		c := rng.Intn(4)
		var txs []*types.Transaction

		for j := 0; j < c; j++ {
			// mock input commitments in l1 transactions
			input := testutils.RandomData(rng, 2000)
			comm, _ := storage.SetInput(ctx, input)
			inputs = append(inputs, input)

			tx, err := types.SignNewTx(batcherPriv, signer, &types.DynamicFeeTx{
				ChainID:   signer.ChainID(),
				Nonce:     0,
				GasTipCap: big.NewInt(2 * params.GWei),
				GasFeeCap: big.NewInt(30 * params.GWei),
				Gas:       100_000,
				To:        &batcherInbox,
				Value:     big.NewInt(int64(0)),
				Data:      comm,
			})
			require.NoError(t, err)

			txs = append(txs, tx)
		}
		logger.Info("included commitments", "count", c)
		l1F.ExpectInfoAndTxsByHash(ref.Hash, testutils.RandomBlockInfo(rng), txs, nil)

		// create a new data source for each block
		src, err := factory.OpenData(ctx, ref, batcherAddr)
		require.NoError(t, err)
		for j := 0; j < c; j++ {
			data, err := src.Next(ctx)
			// check that each commitment is resolved
			require.NoError(t, err)
			require.Equal(t, hexutil.Bytes(inputs[len(inputs)-(c-j)]), data)
		}
		// returns EOF once done
		_, err = src.Next(ctx)
		require.ErrorIs(t, err, io.EOF)
	}
}
