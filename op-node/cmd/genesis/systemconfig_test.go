package genesis

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSystemConfigContract_StartBlock(t *testing.T) {
	addr := common.Address{0xaa}
	sysCfgAbi := snapshots.LoadSystemConfigABI()
	stubRpc := batchingTest.NewAbiBasedRpc(t, addr, sysCfgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	sysCfg := NewSystemConfigContract(caller, addr)
	expected := big.NewInt(56)
	stubRpc.SetResponse(addr, methodStartBlock, rpcblock.Latest, nil, []interface{}{expected})

	result, err := sysCfg.StartBlock(context.Background())
	require.NoError(t, err)
	require.Truef(t, result.Cmp(expected) == 0, "expected %v, got %v", expected, result)
}
