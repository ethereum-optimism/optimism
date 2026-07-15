package contracts

import (
	"context"
	"errors"
	"math/big"
	"testing"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

var (
	superPermissionedGameAddr = common.HexToAddress("0x1234567890123456789012345678901234567890")
	superPermissionedASRAddr  = common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
)

func TestSuperPermissionedGame_BondCapabilities(t *testing.T) {
	_, game := setupSuperPermissionedDisputeGameTest(t)
	recipient := common.Address{0xaa}

	require.False(t, game.HasBondsToClaim())

	credit, status, err := game.GetCredit(context.Background(), recipient)
	require.NoError(t, err)
	require.Zero(t, credit.Sign())
	require.Equal(t, gameTypes.GameStatusDefenderWon, status)

	candidate, err := game.ClaimCreditTx(context.Background(), recipient)
	require.Same(t, ErrClaimCreditNotSupported, err)
	require.Equal(t, txmgr.TxCandidate{}, candidate)
}

func TestSuperPermissionedGame_IsClosed(t *testing.T) {
	gameSequence := new(big.Int).Lsh(big.NewInt(1), 200)
	tests := []struct {
		name           string
		anchorSequence *big.Int
		want           bool
	}{
		{
			name:           "AnchorBehindGame",
			anchorSequence: new(big.Int).Sub(new(big.Int).Set(gameSequence), big.NewInt(1)),
			want:           false,
		},
		{
			name:           "AnchorEqualsGame",
			anchorSequence: new(big.Int).Set(gameSequence),
			want:           true,
		},
		{
			name:           "AnchorAheadOfGame",
			anchorSequence: new(big.Int).Add(new(big.Int).Set(gameSequence), big.NewInt(1)),
			want:           true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stubRpc, game := setupSuperPermissionedDisputeGameTest(t)
			stubRpc.SetResponse(superPermissionedGameAddr, methodSuperPermissionedAnchorStateRegistry, rpcblock.Latest, nil, []interface{}{superPermissionedASRAddr})
			stubRpc.SetResponse(superPermissionedGameAddr, methodSuperPermissionedL2SequenceNumber, rpcblock.Latest, nil, []interface{}{gameSequence})
			stubRpc.SetResponse(superPermissionedASRAddr, methodGetAnchorRoot, rpcblock.Latest, nil, []interface{}{common.Hash{0xab}, test.anchorSequence})

			closed, err := game.IsClosed(context.Background())

			require.NoError(t, err)
			require.Equal(t, test.want, closed)
		})
	}
}

func TestSuperPermissionedGame_CloseGameTx(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		stubRpc, game := setupSuperPermissionedDisputeGameTest(t)
		stubRpc.SetResponse(superPermissionedGameAddr, methodSuperPermissionedAnchorStateRegistry, rpcblock.Latest, nil, []interface{}{superPermissionedASRAddr})
		stubRpc.SetResponse(superPermissionedASRAddr, methodSetAnchorState, rpcblock.Latest, []interface{}{superPermissionedGameAddr}, nil)

		candidate, err := game.CloseGameTx(context.Background())

		require.NoError(t, err)
		require.Equal(t, &superPermissionedASRAddr, candidate.To)
		stubRpc.VerifyTxCandidate(candidate)
	})

	t.Run("AnchorStateRegistryLookupFails", func(t *testing.T) {
		lookupErr := errors.New("anchor state registry unavailable")
		caller := batching.NewMultiCaller(&erroringRPC{err: lookupErr}, batching.DefaultBatchSize)
		game := NewSuperPermissionedDisputeGameContract(contractMetrics.NoopContractMetrics, superPermissionedGameAddr, caller)

		candidate, err := game.CloseGameTx(context.Background())

		require.ErrorIs(t, err, lookupErr)
		require.NotErrorIs(t, err, ErrSimulationFailed)
		require.Equal(t, txmgr.TxCandidate{}, candidate)
	})

	t.Run("AnchorStateSimulationFails", func(t *testing.T) {
		stubRpc, game := setupSuperPermissionedDisputeGameTest(t)
		stubRpc.SetResponse(superPermissionedGameAddr, methodSuperPermissionedAnchorStateRegistry, rpcblock.Latest, nil, []interface{}{superPermissionedASRAddr})
		simulationErr := errors.New("game is not finalized")
		stubRpc.SetError(superPermissionedASRAddr, methodSetAnchorState, rpcblock.Latest, []interface{}{superPermissionedGameAddr}, simulationErr)

		candidate, err := game.CloseGameTx(context.Background())

		require.ErrorIs(t, err, ErrSimulationFailed)
		require.ErrorIs(t, err, simulationErr)
		require.Equal(t, txmgr.TxCandidate{}, candidate)
	})
}

func setupSuperPermissionedDisputeGameTest(t *testing.T) (*batchingTest.AbiBasedRpc, *SuperPermissionedDisputeGameContract) {
	t.Helper()
	stubRpc := batchingTest.NewAbiBasedRpc(t, superPermissionedGameAddr, snapshots.LoadSuperPermissionedDisputeGameABI())
	stubRpc.AddContract(superPermissionedASRAddr, snapshots.LoadAnchorStateRegistryABI())
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	game := NewSuperPermissionedDisputeGameContract(contractMetrics.NoopContractMetrics, superPermissionedGameAddr, caller)
	return stubRpc, game
}

type erroringRPC struct {
	err error
}

func (r *erroringRPC) CallContext(context.Context, interface{}, string, ...interface{}) error {
	return r.err
}

func (r *erroringRPC) BatchCallContext(context.Context, []rpc.BatchElem) error {
	return r.err
}
