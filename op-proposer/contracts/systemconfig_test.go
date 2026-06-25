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

func TestSystemConfigProposerAddresses(t *testing.T) {
	systemConfigAddr := common.Address{0x11}
	portalAddr := common.Address{0x22}
	dgfAddr := common.Address{0x33}
	asrAddr := common.Address{0x44}
	stubRpc := batchingTest.NewAbiBasedRpc(t, systemConfigAddr, snapshots.LoadSystemConfigABI())
	stubRpc.AddContract(portalAddr, snapshots.LoadOptimismPortal2ABI())
	stubRpc.SetResponse(systemConfigAddr, methodDisputeGameFactory, rpcblock.Latest, nil, []interface{}{dgfAddr})
	stubRpc.SetResponse(systemConfigAddr, methodOptimismPortal, rpcblock.Latest, nil, []interface{}{portalAddr})
	stubRpc.SetResponse(portalAddr, methodAnchorStateRegistry, rpcblock.Latest, nil, []interface{}{asrAddr})

	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	systemConfig := NewSystemConfig(systemConfigAddr, caller, time.Minute)
	dgf, asr, err := systemConfig.ProposerAddresses(context.Background())
	require.NoError(t, err)
	require.Equal(t, dgfAddr, dgf)
	require.Equal(t, asrAddr, asr)
}
