package contracts

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAnchorStateRegistryRespectedGameType(t *testing.T) {
	asrAddr := common.Address{0x11, 0x22}
	stubRpc := batchingTest.NewAbiBasedRpc(t, asrAddr, snapshots.LoadAnchorStateRegistryABI())
	stubRpc.SetResponse(asrAddr, methodRespectedGameType, rpcblock.Latest, nil, []interface{}{uint32(9)})

	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	asr := NewAnchorStateRegistry(asrAddr, caller, time.Minute)
	gameType, err := asr.RespectedGameType(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint32(9), gameType)
}
