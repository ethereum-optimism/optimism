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

func TestCreateMoveTxZKChallenge(t *testing.T) {
	gameAddr := common.Address{0xaa}
	bond := big.NewInt(1234)
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.ZKDisputeGameType)})
	stubRPC.SetResponse(gameAddr, "challengerBond", rpcblock.Latest, nil, []interface{}{bond})
	stubRPC.SetResponse(gameAddr, "challenge", rpcblock.Latest, nil, nil)
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	tx, err := createMoveTx(context.Background(), caller, gameAddr, true, 0, common.Hash{})
	require.NoError(t, err)
	require.Equal(t, bond, tx.Value)
	stubRPC.VerifyTxCandidate(tx)
}

func TestCreateMoveTxRejectsZKDefense(t *testing.T) {
	gameAddr := common.Address{0xaa}
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadZKDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.ZKDisputeGameType)})
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	_, err := createMoveTx(context.Background(), caller, gameAddr, false, 0, common.Hash{})
	require.EqualError(t, err, "zk dispute games do not support defense moves")
}

func TestCreateMoveTxFaultGameAttack(t *testing.T) {
	gameAddr := common.Address{0xbb}
	parentClaim := common.Hash{0x11}
	attackClaim := common.Hash{0x22}
	bond := big.NewInt(5678)
	stubRPC := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadFaultDisputeGameABI())
	stubRPC.SetResponse(gameAddr, "gameType", rpcblock.Latest, nil, []interface{}{uint32(gameTypes.CannonGameType)})
	stubRPC.SetResponse(gameAddr, "version", rpcblock.Latest, nil, []interface{}{"1.4.0"})
	stubRPC.SetResponse(gameAddr, "claimData", rpcblock.Latest, []interface{}{big.NewInt(3)}, []interface{}{
		uint32(0), common.Address{}, common.Address{}, big.NewInt(0), parentClaim, big.NewInt(1), big.NewInt(0),
	})
	stubRPC.SetResponse(gameAddr, "getRequiredBond", rpcblock.Latest, []interface{}{big.NewInt(2)}, []interface{}{bond})
	stubRPC.SetResponse(gameAddr, "attack", rpcblock.Latest, []interface{}{parentClaim, big.NewInt(3), attackClaim}, nil)
	caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)

	tx, err := createMoveTx(context.Background(), caller, gameAddr, true, 3, attackClaim)
	require.NoError(t, err)
	require.Equal(t, bond, tx.Value)
	stubRPC.VerifyTxCandidate(tx)
}
