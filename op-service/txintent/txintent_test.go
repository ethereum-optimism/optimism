package txintent

import (
	"bytes"
	"context"
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

	chainID := eth.ChainIDFromUInt64(1)

	signer := types.LatestSignerForChainID(chainID.ToBig())
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

func TestTxIntentMultiCall(t *testing.T) {
	rng := rand.New(rand.NewSource(12345))
	ctx := context.Background()

	chainID := eth.ChainIDFromUInt64(1)

	signer := types.LatestSignerForChainID(chainID.ToBig())
	tx := NewIntent[*MultiTrigger, *MulticallOutput]()

	callCnt := 3
	multiTrigger := &MultiTrigger{Emitter: testutils.RandomAddress(rng)}
	randomTxs := []*types.Transaction{}
	for range callCnt {
		randomTx := testutils.RandomAccessListTx(rng, signer)
		randomTxs = append(randomTxs, randomTx)
		// make sure To is not nil
		randomTo := testutils.RandomAddress(rng)
		randomAccessList := testutils.RandomAccessList(rng)
		multiTrigger.Calls = append(multiTrigger.Calls, &call{
			to:         &randomTo,
			data:       randomTx.Data(),
			accessList: randomAccessList,
		})
	}
	tx.Content.Fn(func(ctx context.Context) (*MultiTrigger, error) {
		return multiTrigger, nil
	})
	_, err := tx.Content.Eval(ctx)
	require.NoError(t, err)

	// Evaulate to check that the Content info propagated to PlannedTx
	to, err := tx.PlannedTx.To.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, multiTrigger.Emitter, *to)
	data, err := tx.PlannedTx.Data.Eval(ctx)
	require.NoError(t, err)
	accessList, err := tx.PlannedTx.AccessList.Eval(ctx)
	require.NoError(t, err)

	var stackedAccessList types.AccessList
	for _, call := range multiTrigger.Calls {
		// Check batched tx contains each calldata
		subData, err := call.Data()
		require.NoError(t, err)
		require.True(t, bytes.Contains(data, subData))
		// Check batched tx contains each accesslist
		subAccessList, err := call.AccessList()
		require.NoError(t, err)
		stackedAccessList = append(stackedAccessList, subAccessList...)
	}
	require.Equal(t, stackedAccessList, accessList)

	randomReceipt := testutils.RandomReceipt(rng, signer, randomTxs[0], 0, 0)
	randomBlockRef := testutils.RandomBlockRef(rng)
	randomReceipt.BlockHash = randomBlockRef.Hash

	tx.PlannedTx.Included.Set(randomReceipt)
	tx.PlannedTx.IncludedBlock.Set(randomBlockRef)
	tx.PlannedTx.ChainID.Set(chainID)

	result, err := tx.Result.Eval(ctx)
	require.NoError(t, err)

	// Check that FromReceipt correctly processed desired result
	require.Equal(t, randomBlockRef.Hash, result.receipt.BlockHash)
	require.Equal(t, randomBlockRef.Number, result.includedIn.Number)
	require.Equal(t, chainID, result.chainID)
}
