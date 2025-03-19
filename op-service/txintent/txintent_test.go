package txintent

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type call struct {
	to         *common.Address
	data       []byte
	accessList types.AccessList
}

func (c *call) To() (*common.Address, error)          { return c.to, nil }
func (c *call) Data() ([]byte, error)                 { return c.data, nil }
func (c *call) AccessList() (types.AccessList, error) { return c.accessList, nil }

type result struct {
	blockHash   common.Hash
	blockNumber uint64
}

func (r *result) FromReceipt(ctx context.Context, rec *types.Receipt, includedIn eth.BlockRef, chainID eth.ChainID) error {
	r.blockHash = rec.BlockHash
	r.blockNumber = includedIn.Number
	return nil
}
func (r *result) Init() Result { return &result{} }

func TestTxIntent(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	ctx := context.Background()

	signer := types.LatestSignerForChainID(big.NewInt(1))
	randomTx := testutils.RandomAccessListTx(rng, signer)
	tx := NewIntent[*call, *result]()

	randomCall := &call{
		to:         randomTx.To(),
		data:       randomTx.Data(),
		accessList: randomTx.AccessList(),
	}
	tx.Content.Fn(func(ctx context.Context) (*call, error) {
		return randomCall, nil
	})
	_, err := tx.Content.Eval(ctx)
	require.NoError(t, err)

	// Evaulate to check that the Content info propagated to PlannedTx
	to, err := tx.PlannedTx.To.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, randomCall.to, to)
	data, err := tx.PlannedTx.Data.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, randomCall.data, []byte(data))
	accessList, err := tx.PlannedTx.AccessList.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, randomCall.accessList, accessList)

	randomReceipt := testutils.RandomReceipt(rng, signer, randomTx, 0, 0)
	randomBlockRef := testutils.RandomBlockRef(rng)
	randomReceipt.BlockHash = randomBlockRef.Hash

	tx.PlannedTx.Included.Set(randomReceipt)
	tx.PlannedTx.IncludedBlock.Set(randomBlockRef)
	tx.PlannedTx.ChainID.Set(eth.ChainIDFromUInt64(1))

	result, err := tx.Result.Eval(ctx)
	require.NoError(t, err)

	// Check that FromReceipt correctly processed desired result
	require.Equal(t, randomBlockRef.Hash, result.blockHash)
	require.Equal(t, randomBlockRef.Number, result.blockNumber)
}
