package extract

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func setupSuperPermissionedGameCallerTest(t *testing.T) (*batchingTest.AbiBasedRpc, common.Address, *SuperPermissionedGameCaller) {
	gameAddr := common.Address{0xaa}
	rpc := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadSuperPermissionedDisputeGameABI())
	caller := batching.NewMultiCaller(rpc, batching.DefaultBatchSize)
	game := NewSuperPermissionedGameCaller(contractMetrics.NoopContractMetrics, gameAddr, caller)
	return rpc, gameAddr, game
}

func TestSuperPermissionedGameCaller_GetExtendedMetadata(t *testing.T) {
	rpc, gameAddr, game := setupSuperPermissionedGameCallerTest(t)
	block := rpcblock.Latest
	l1Head := common.Hash{0xcc}
	rpc.SetResponse(gameAddr, methodL1Head, block, nil, []any{l1Head})
	rpc.SetResponse(gameAddr, methodL2SequenceNumber, block, nil, []any{big.NewInt(1234)})
	rpc.SetResponse(gameAddr, methodRootClaim, block, nil, []any{mockRootClaim})
	rpc.SetResponse(gameAddr, methodStatus, block, nil, []any{uint8(gameTypes.GameStatusDefenderWon)})

	meta, err := game.GetExtendedMetadata(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, l1Head, meta.L1Head)
	require.Equal(t, uint64(1234), meta.L2SequenceNum)
	require.Equal(t, mockRootClaim, meta.RootClaim)
	require.Equal(t, gameTypes.GameStatusDefenderWon, meta.Status)
}

func TestSuperPermissionedGameCaller_GetExtendedMetadata_L2SequenceNumberOverflow(t *testing.T) {
	rpc, gameAddr, game := setupSuperPermissionedGameCallerTest(t)
	block := rpcblock.Latest
	tooBig := new(big.Int).Add(new(big.Int).SetUint64(math.MaxUint64), big.NewInt(1))
	rpc.SetResponse(gameAddr, methodL1Head, block, nil, []any{common.Hash{}})
	rpc.SetResponse(gameAddr, methodL2SequenceNumber, block, nil, []any{tooBig})
	rpc.SetResponse(gameAddr, methodRootClaim, block, nil, []any{mockRootClaim})
	rpc.SetResponse(gameAddr, methodStatus, block, nil, []any{uint8(gameTypes.GameStatusInProgress)})

	meta, err := game.GetExtendedMetadata(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, uint64(math.MaxUint64), meta.L2SequenceNum)
}

func TestSuperPermissionedGameCaller_GetAnchorStateRegistry(t *testing.T) {
	rpc, gameAddr, game := setupSuperPermissionedGameCallerTest(t)
	block := rpcblock.Latest
	expected := common.HexToAddress("0x0123456789abcDEF0123456789abCDef01234567")
	rpc.SetResponse(gameAddr, methodAnchorStateRegistry, block, nil, []any{expected})

	actual, err := game.GetAnchorStateRegistry(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestSuperPermissionedGameCaller_GetAllClaims(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	claims, err := game.GetAllClaims(context.Background(), rpcblock.Latest)
	require.NoError(t, err)
	require.Nil(t, claims)
}

func TestSuperPermissionedGameCaller_IsResolved(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	resolved, err := game.IsResolved(context.Background(), rpcblock.Latest, faultTypes.Claim{}, faultTypes.Claim{})
	require.NoError(t, err)
	require.Equal(t, []bool{false, false}, resolved)
}

func TestSuperPermissionedGameCaller_GetWithdrawals(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	recipients := []common.Address{{0x01}, {0x02}}
	withdrawals, err := game.GetWithdrawals(context.Background(), rpcblock.Latest, recipients...)
	require.NoError(t, err)
	require.Len(t, withdrawals, len(recipients))
	for _, w := range withdrawals {
		require.Equal(t, big.NewInt(0), w.Amount)
		require.Equal(t, big.NewInt(0), w.Timestamp)
	}
}

func TestSuperPermissionedGameCaller_GetCredits(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	recipients := []common.Address{{0x01}, {0x02}, {0x03}}
	credits, err := game.GetCredits(context.Background(), rpcblock.Latest, recipients...)
	require.NoError(t, err)
	require.Len(t, credits, len(recipients))
	for _, c := range credits {
		require.Equal(t, big.NewInt(0), c)
	}
}

func TestSuperPermissionedGameCaller_GetBondDistributionMode(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	mode, err := game.GetBondDistributionMode(context.Background(), rpcblock.Latest)
	require.NoError(t, err)
	require.Equal(t, faultTypes.UndecidedDistributionMode, mode)
}

func TestSuperPermissionedGameCaller_GetBalanceAndDelay(t *testing.T) {
	_, _, game := setupSuperPermissionedGameCallerTest(t)
	balance, delay, addr, err := game.GetBalanceAndDelay(context.Background(), rpcblock.Latest)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(0), balance)
	require.Equal(t, time.Duration(0), delay)
	require.Equal(t, common.Address{}, addr)
}
