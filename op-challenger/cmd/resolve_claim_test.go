package main

import (
	"context"
	"math/big"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCreateResolveClaimTxRejectsZKGame(t *testing.T) {
	gameAddr := common.Address{0xaa}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.ZKDisputeGameType)})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	_, err := createResolveClaimTx(context.Background(), caller, gameAddr, 0)
	require.EqualError(t, err, "zk dispute game 0xaa00000000000000000000000000000000000000 has no claims to resolve, use the resolve command")
}

func TestCreateResolveClaimTxFaultGame(t *testing.T) {
	gameAddr := common.Address{0xbb}
	claimIdx := uint64(3)
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadFaultDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.CannonGameType)})
	stubRPC.SetResponse(gameAddr, "version", rpcblock.Latest, nil, []interface{}{"1.4.0"})
	stubRPC.SetResponse(gameAddr, "resolveClaim", rpcblock.Latest, []interface{}{new(big.Int).SetUint64(claimIdx), big.NewInt(512)}, nil)
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	tx, err := createResolveClaimTx(context.Background(), caller, gameAddr, claimIdx)
	require.NoError(t, err)
	stubRPC.VerifyTxCandidate(tx)
}
