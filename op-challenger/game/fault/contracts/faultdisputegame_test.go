package contracts

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSimpleGetters(t *testing.T) {
	tests := []struct {
		method   string
		args     []interface{}
		result   interface{}
		expected interface{} // Defaults to expecting the same as result
		call     func(game *FaultDisputeGameContract) (any, error)
	}{
		{
			method: methodStatus,
			result: types.GameStatusChallengerWon,
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetStatus(context.Background())
			},
		},
		{
			method: methodGameDuration,
			result: uint64(5566),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetGameDuration(context.Background())
			},
		},
		{
			method:   methodMaxGameDepth,
			result:   big.NewInt(128),
			expected: uint64(128),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetMaxGameDepth(context.Background())
			},
		},
		{
			method: methodAbsolutePrestate,
			result: common.Hash{0xab},
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetAbsolutePrestateHash(context.Background())
			},
		},
		{
			method:   methodClaimCount,
			result:   big.NewInt(9876),
			expected: uint64(9876),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetClaimCount(context.Background())
			},
		},
		{
			method: methodL1Head,
			result: common.Hash{0xdd, 0xbb},
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetL1Head(context.Background())
			},
		},
		{
			method: methodResolve,
			result: types.GameStatusInProgress,
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.CallResolve(context.Background())
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.method, func(t *testing.T) {
			stubRpc, game := setup(t)
			stubRpc.SetResponse(test.method, batching.BlockLatest, nil, []interface{}{test.result})
			status, err := test.call(game)
			require.NoError(t, err)
			expected := test.expected
			if expected == nil {
				expected = test.result
			}
			require.Equal(t, expected, status)
		})
	}
}

func TestGetProposals(t *testing.T) {
	stubRpc, game := setup(t)
	agreedIndex := big.NewInt(5)
	agreedBlockNum := big.NewInt(6)
	agreedRoot := common.Hash{0xaa}
	disputedIndex := big.NewInt(7)
	disputedBlockNum := big.NewInt(8)
	disputedRoot := common.Hash{0xdd}
	agreed := Proposal{
		Index:         agreedIndex,
		L2BlockNumber: agreedBlockNum,
		OutputRoot:    agreedRoot,
	}
	disputed := Proposal{
		Index:         disputedIndex,
		L2BlockNumber: disputedBlockNum,
		OutputRoot:    disputedRoot,
	}
	stubRpc.SetResponse(methodProposals, batching.BlockLatest, []interface{}{}, []interface{}{
		agreed, disputed,
	})
	actualAgreed, actualDisputed, err := game.GetProposals(context.Background())
	require.NoError(t, err)
	require.Equal(t, agreed, actualAgreed)
	require.Equal(t, disputed, actualDisputed)
}

func TestGetClaim(t *testing.T) {
	stubRpc, game := setup(t)
	idx := big.NewInt(2)
	parentIndex := uint32(1)
	countered := true
	value := common.Hash{0xab}
	position := big.NewInt(2)
	clock := big.NewInt(1234)
	stubRpc.SetResponse(methodClaim, batching.BlockLatest, []interface{}{idx}, []interface{}{parentIndex, countered, value, position, clock})
	status, err := game.GetClaim(context.Background(), idx.Uint64())
	require.NoError(t, err)
	require.Equal(t, faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    value,
			Position: faultTypes.NewPositionFromGIndex(position),
		},
		Countered:           true,
		Clock:               1234,
		ContractIndex:       int(idx.Uint64()),
		ParentContractIndex: 1,
	}, status)
}

func TestGetAllClaims(t *testing.T) {
	stubRpc, game := setup(t)
	claim0 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xaa},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(1)),
		},
		Countered:           true,
		Clock:               1234,
		ContractIndex:       0,
		ParentContractIndex: math.MaxUint32,
	}
	claim1 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xab},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(2)),
		},
		Countered:           true,
		Clock:               4455,
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	claim2 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xbb},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(6)),
		},
		Countered:           false,
		Clock:               7777,
		ContractIndex:       2,
		ParentContractIndex: 1,
	}
	expectedClaims := []faultTypes.Claim{claim0, claim1, claim2}
	stubRpc.SetResponse(methodClaimCount, batching.BlockLatest, nil, []interface{}{big.NewInt(int64(len(expectedClaims)))})
	for _, claim := range expectedClaims {
		expectGetClaim(stubRpc, claim)
	}
	claims, err := game.GetAllClaims(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedClaims, claims)
}

func TestCallResolveClaim(t *testing.T) {
	stubRpc, game := setup(t)
	stubRpc.SetResponse(methodResolveClaim, batching.BlockLatest, []interface{}{big.NewInt(123)}, nil)
	err := game.CallResolveClaim(context.Background(), 123)
	require.NoError(t, err)
}

func TestResolveClaimTx(t *testing.T) {
	stubRpc, game := setup(t)
	stubRpc.SetResponse(methodResolveClaim, batching.BlockLatest, []interface{}{big.NewInt(123)}, nil)
	tx, err := game.ResolveClaimTx(123)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestResolveTx(t *testing.T) {
	stubRpc, game := setup(t)
	stubRpc.SetResponse(methodResolve, batching.BlockLatest, nil, nil)
	tx, err := game.ResolveTx()
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestAttackTx(t *testing.T) {
	stubRpc, game := setup(t)
	value := common.Hash{0xaa}
	stubRpc.SetResponse(methodAttack, batching.BlockLatest, []interface{}{big.NewInt(111), value}, nil)
	tx, err := game.AttackTx(111, value)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestDefendTx(t *testing.T) {
	stubRpc, game := setup(t)
	value := common.Hash{0xaa}
	stubRpc.SetResponse(methodDefend, batching.BlockLatest, []interface{}{big.NewInt(111), value}, nil)
	tx, err := game.DefendTx(111, value)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestStepTx(t *testing.T) {
	stubRpc, game := setup(t)
	stateData := []byte{1, 2, 3}
	proofData := []byte{4, 5, 6, 7, 8, 9}
	stubRpc.SetResponse(methodStep, batching.BlockLatest, []interface{}{big.NewInt(111), true, stateData, proofData}, nil)
	tx, err := game.StepTx(111, true, stateData, proofData)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func expectGetClaim(stubRpc *batchingTest.AbiBasedRpc, claim faultTypes.Claim) {
	stubRpc.SetResponse(
		methodClaim,
		batching.BlockLatest,
		[]interface{}{big.NewInt(int64(claim.ContractIndex))},
		[]interface{}{
			uint32(claim.ParentContractIndex),
			claim.Countered,
			claim.Value,
			claim.Position.ToGIndex(),
			big.NewInt(int64(claim.Clock)),
		})
}

func setup(t *testing.T) (*batchingTest.AbiBasedRpc, *FaultDisputeGameContract) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)
	address := common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAbi, address)
	caller := batching.NewMultiCaller(stubRpc, 100)
	game, err := NewFaultDisputeGameContract(address, caller)
	require.NoError(t, err)
	return stubRpc, game
}
