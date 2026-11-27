package batching

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
)

func newArrayStub(t *testing.T, count *big.Int) (*test.AbiBasedRpc, *BoundContract) {
	addr := common.Address{0xaa}
	erc20Abi, err := test.ERC20MetaData.GetAbi()
	require.NoError(t, err)
	stub := test.NewAbiBasedRpc(t, addr, erc20Abi)
	stub.SetResponse(addr, "totalSupply", rpcblock.Latest, nil, []any{count})
	return stub, NewBoundContract(erc20Abi, addr)
}

func TestReadArrayPages(t *testing.T) {
	count := uint64(2*readArrayPageSize + 3)
	stub, contract := newArrayStub(t, new(big.Int).SetUint64(count))
	stub.SetResponse(common.Address{0xaa}, "balanceOf", rpcblock.Latest, []any{common.Address{}}, []any{big.NewInt(7)})

	var indices []uint64
	results, err := ReadArray(context.Background(), NewMultiCaller(stub, DefaultBatchSize), rpcblock.Latest,
		contract.Call("totalSupply"), func(i *big.Int) *ContractCall {
			indices = append(indices, bigs.Uint64Strict(i))
			return contract.Call("balanceOf", common.Address{})
		})
	require.NoError(t, err)
	require.Len(t, results, int(count))
	for i, idx := range indices {
		require.EqualValues(t, i, idx)
	}
	require.Len(t, indices, int(count))
}

// failingBatchRpc answers single calls from the stub and fails every batch call.
type failingBatchRpc struct{ *test.AbiBasedRpc }

func (failingBatchRpc) BatchCallContext(context.Context, []rpc.BatchElem) error {
	return errors.New("unavailable")
}

func TestReadArrayHugeLength(t *testing.T) {
	stub, contract := newArrayStub(t, new(big.Int).SetUint64(math.MaxUint64))

	var created int
	_, err := ReadArray(context.Background(), NewMultiCaller(failingBatchRpc{stub}, DefaultBatchSize), rpcblock.Latest,
		contract.Call("totalSupply"), func(*big.Int) *ContractCall {
			created++
			return contract.Call("balanceOf", common.Address{})
		})
	require.ErrorContains(t, err, "failed to fetch array data")
	require.Equal(t, readArrayPageSize, created)
}
