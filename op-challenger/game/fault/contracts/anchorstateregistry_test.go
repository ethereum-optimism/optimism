package contracts

import (
	"context"
	"math/big"
	"testing"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAnchorStateRegistry_GetAnchorRoot(t *testing.T) {
	asrAddr := common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
	asrAbi := snapshots.LoadAnchorStateRegistryABI()
	stubRpc := batchingTest.NewAbiBasedRpc(t, asrAddr, asrAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	asr := NewAnchorStateRegistryContract(contractMetrics.NoopContractMetrics, asrAddr, caller)

	block := rpcblock.ByNumber(482)
	expectedRoot := common.Hash{0xab, 0xcd, 0xef}
	expectedSeq := big.NewInt(1234567)
	stubRpc.SetResponse(asrAddr, methodGetAnchorRoot, block, nil, []interface{}{expectedRoot, expectedSeq})

	root, seq, err := asr.GetAnchorRoot(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, expectedRoot, root)
	require.Zerof(t, expectedSeq.Cmp(seq), "expected: %v actual: %v", expectedSeq, seq)
}
