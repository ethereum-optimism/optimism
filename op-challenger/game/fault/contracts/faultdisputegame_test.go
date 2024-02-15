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

var (
	fdgAddr    = common.HexToAddress("0x24112842371dFC380576ebb09Ae16Cb6B6caD7CB")
	vmAddr     = common.HexToAddress("0x33332842371dFC380576ebb09Ae16Cb6B6c3333")
	oracleAddr = common.HexToAddress("0x44442842371dFC380576ebb09Ae16Cb6B6ca4444")
)

func TestSimpleGetters(t *testing.T) {
	tests := []struct {
		methodAlias string
		method      string
		args        []interface{}
		result      interface{}
		expected    interface{} // Defaults to expecting the same as result
		call        func(game *FaultDisputeGameContract) (any, error)
	}{
		{
			methodAlias: "status",
			method:      methodStatus,
			result:      types.GameStatusChallengerWon,
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetStatus(context.Background())
			},
		},
		{
			methodAlias: "gameDuration",
			method:      methodGameDuration,
			result:      uint64(5566),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetGameDuration(context.Background())
			},
		},
		{
			methodAlias: "maxGameDepth",
			method:      methodMaxGameDepth,
			result:      big.NewInt(128),
			expected:    faultTypes.Depth(128),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetMaxGameDepth(context.Background())
			},
		},
		{
			methodAlias: "absolutePrestate",
			method:      methodAbsolutePrestate,
			result:      common.Hash{0xab},
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetAbsolutePrestateHash(context.Background())
			},
		},
		{
			methodAlias: "claimCount",
			method:      methodClaimCount,
			result:      big.NewInt(9876),
			expected:    uint64(9876),
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetClaimCount(context.Background())
			},
		},
		{
			methodAlias: "l1Head",
			method:      methodL1Head,
			result:      common.Hash{0xdd, 0xbb},
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.GetL1Head(context.Background())
			},
		},
		{
			methodAlias: "resolve",
			method:      methodResolve,
			result:      types.GameStatusInProgress,
			call: func(game *FaultDisputeGameContract) (any, error) {
				return game.CallResolve(context.Background())
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.methodAlias, func(t *testing.T) {
			stubRpc, game := setupFaultDisputeGameTest(t)
			stubRpc.SetResponse(fdgAddr, test.method, batching.BlockLatest, nil, []interface{}{test.result})
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

func TestClock_EncodingDecoding(t *testing.T) {
	t.Run("DurationAndTimestamp", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000050000000000000002")
		encoded := new(big.Int).SetBytes(by)
		clock := decodeClock(encoded)
		require.Equal(t, uint64(5), clock.Duration)
		require.Equal(t, uint64(2), clock.Timestamp)
		require.Equal(t, encoded, packClock(clock))
	})

	t.Run("ZeroDuration", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000000000000000000002")
		encoded := new(big.Int).SetBytes(by)
		clock := decodeClock(encoded)
		require.Equal(t, uint64(0), clock.Duration)
		require.Equal(t, uint64(2), clock.Timestamp)
		require.Equal(t, encoded, packClock(clock))
	})

	t.Run("ZeroTimestamp", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000050000000000000000")
		encoded := new(big.Int).SetBytes(by)
		clock := decodeClock(encoded)
		require.Equal(t, uint64(5), clock.Duration)
		require.Equal(t, uint64(0), clock.Timestamp)
		require.Equal(t, encoded, packClock(clock))
	})

	t.Run("ZeroClock", func(t *testing.T) {
		by := common.Hex2Bytes("00000000000000000000000000000000")
		encoded := new(big.Int).SetBytes(by)
		clock := decodeClock(encoded)
		require.Equal(t, uint64(0), clock.Duration)
		require.Equal(t, uint64(0), clock.Timestamp)
		require.Equal(t, encoded.Uint64(), packClock(clock).Uint64())
	})
}

func TestGetOracleAddr(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	stubRpc.SetResponse(fdgAddr, methodVM, batching.BlockLatest, nil, []interface{}{vmAddr})
	stubRpc.SetResponse(vmAddr, methodOracle, batching.BlockLatest, nil, []interface{}{oracleAddr})

	actual, err := game.GetOracle(context.Background())
	require.NoError(t, err)
	require.Equal(t, oracleAddr, actual.Addr())
}

func TestGetClaim(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	idx := big.NewInt(2)
	parentIndex := uint32(1)
	counteredBy := common.Address{0x01}
	claimant := common.Address{0x02}
	bond := big.NewInt(5)
	value := common.Hash{0xab}
	position := big.NewInt(2)
	clock := big.NewInt(1234)
	stubRpc.SetResponse(fdgAddr, methodClaim, batching.BlockLatest, []interface{}{idx}, []interface{}{parentIndex, counteredBy, claimant, bond, value, position, clock})
	status, err := game.GetClaim(context.Background(), idx.Uint64())
	require.NoError(t, err)
	require.Equal(t, faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    value,
			Position: faultTypes.NewPositionFromGIndex(position),
			Bond:     bond,
		},
		CounteredBy:         counteredBy,
		Claimant:            claimant,
		Clock:               decodeClock(big.NewInt(1234)),
		ContractIndex:       int(idx.Uint64()),
		ParentContractIndex: 1,
	}, status)
}

func TestGetAllClaims(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	claim0 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xaa},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(1)),
			Bond:     big.NewInt(5),
		},
		CounteredBy:         common.Address{0x01},
		Claimant:            common.Address{0x02},
		Clock:               decodeClock(big.NewInt(1234)),
		ContractIndex:       0,
		ParentContractIndex: math.MaxUint32,
	}
	claim1 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xab},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(2)),
			Bond:     big.NewInt(5),
		},
		CounteredBy:         common.Address{0x02},
		Claimant:            common.Address{0x01},
		Clock:               decodeClock(big.NewInt(4455)),
		ContractIndex:       1,
		ParentContractIndex: 0,
	}
	claim2 := faultTypes.Claim{
		ClaimData: faultTypes.ClaimData{
			Value:    common.Hash{0xbb},
			Position: faultTypes.NewPositionFromGIndex(big.NewInt(6)),
			Bond:     big.NewInt(5),
		},
		Claimant:            common.Address{0x02},
		Clock:               decodeClock(big.NewInt(7777)),
		ContractIndex:       2,
		ParentContractIndex: 1,
	}
	expectedClaims := []faultTypes.Claim{claim0, claim1, claim2}
	stubRpc.SetResponse(fdgAddr, methodClaimCount, batching.BlockLatest, nil, []interface{}{big.NewInt(int64(len(expectedClaims)))})
	for _, claim := range expectedClaims {
		expectGetClaim(stubRpc, claim)
	}
	claims, err := game.GetAllClaims(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedClaims, claims)
}

func TestCallResolveClaim(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	stubRpc.SetResponse(fdgAddr, methodResolveClaim, batching.BlockLatest, []interface{}{big.NewInt(123)}, nil)
	err := game.CallResolveClaim(context.Background(), 123)
	require.NoError(t, err)
}

func TestResolveClaimTxTest(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	stubRpc.SetResponse(fdgAddr, methodResolveClaim, batching.BlockLatest, []interface{}{big.NewInt(123)}, nil)
	tx, err := game.ResolveClaimTx(123)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestResolveTx(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	stubRpc.SetResponse(fdgAddr, methodResolve, batching.BlockLatest, nil, nil)
	tx, err := game.ResolveTx()
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestAttackTx(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	value := common.Hash{0xaa}
	stubRpc.SetResponse(fdgAddr, methodAttack, batching.BlockLatest, []interface{}{big.NewInt(111), value}, nil)
	tx, err := game.AttackTx(111, value)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestDefendTx(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	value := common.Hash{0xaa}
	stubRpc.SetResponse(fdgAddr, methodDefend, batching.BlockLatest, []interface{}{big.NewInt(111), value}, nil)
	tx, err := game.DefendTx(111, value)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func TestStepTx(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	stateData := []byte{1, 2, 3}
	proofData := []byte{4, 5, 6, 7, 8, 9}
	stubRpc.SetResponse(fdgAddr, methodStep, batching.BlockLatest, []interface{}{big.NewInt(111), true, stateData, proofData}, nil)
	tx, err := game.StepTx(111, true, stateData, proofData)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
}

func expectGetClaim(stubRpc *batchingTest.AbiBasedRpc, claim faultTypes.Claim) {
	stubRpc.SetResponse(
		fdgAddr,
		methodClaim,
		batching.BlockLatest,
		[]interface{}{big.NewInt(int64(claim.ContractIndex))},
		[]interface{}{
			uint32(claim.ParentContractIndex),
			claim.CounteredBy,
			claim.Claimant,
			claim.Bond,
			claim.Value,
			claim.Position.ToGIndex(),
			packClock(claim.Clock),
		})
}

func TestGetBlockRange(t *testing.T) {
	stubRpc, contract := setupFaultDisputeGameTest(t)
	expectedStart := uint64(65)
	expectedEnd := uint64(102)
	stubRpc.SetResponse(fdgAddr, methodGenesisBlockNumber, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedStart)})
	stubRpc.SetResponse(fdgAddr, methodL2BlockNumber, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedEnd)})
	start, end, err := contract.GetBlockRange(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedStart, start)
	require.Equal(t, expectedEnd, end)
}

func TestGetSplitDepth(t *testing.T) {
	stubRpc, contract := setupFaultDisputeGameTest(t)
	expectedSplitDepth := faultTypes.Depth(15)
	stubRpc.SetResponse(fdgAddr, methodSplitDepth, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(uint64(expectedSplitDepth))})
	splitDepth, err := contract.GetSplitDepth(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedSplitDepth, splitDepth)
}

func TestGetGameMetadata(t *testing.T) {
	stubRpc, contract := setupFaultDisputeGameTest(t)
	expectedL2BlockNumber := uint64(123)
	expectedGameDuration := uint64(456)
	expectedRootClaim := common.Hash{0x01, 0x02}
	expectedStatus := types.GameStatusChallengerWon
	stubRpc.SetResponse(fdgAddr, methodL2BlockNumber, batching.BlockLatest, nil, []interface{}{new(big.Int).SetUint64(expectedL2BlockNumber)})
	stubRpc.SetResponse(fdgAddr, methodRootClaim, batching.BlockLatest, nil, []interface{}{expectedRootClaim})
	stubRpc.SetResponse(fdgAddr, methodStatus, batching.BlockLatest, nil, []interface{}{expectedStatus})
	stubRpc.SetResponse(fdgAddr, methodGameDuration, batching.BlockLatest, nil, []interface{}{expectedGameDuration})
	l2BlockNumber, rootClaim, status, duration, err := contract.GetGameMetadata(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedL2BlockNumber, l2BlockNumber)
	require.Equal(t, expectedRootClaim, rootClaim)
	require.Equal(t, expectedStatus, status)
	require.Equal(t, expectedGameDuration, duration)
}

func TestGetGenesisOutputRoot(t *testing.T) {
	stubRpc, contract := setupFaultDisputeGameTest(t)
	expectedOutputRoot := common.HexToHash("0x1234")
	stubRpc.SetResponse(fdgAddr, methodGenesisOutputRoot, batching.BlockLatest, nil, []interface{}{expectedOutputRoot})
	genesisOutputRoot, err := contract.GetGenesisOutputRoot(context.Background())
	require.NoError(t, err)
	require.Equal(t, expectedOutputRoot, genesisOutputRoot)
}

func TestFaultDisputeGame_UpdateOracleTx(t *testing.T) {
	t.Run("Local", func(t *testing.T) {
		stubRpc, game := setupFaultDisputeGameTest(t)
		data := faultTypes.NewPreimageOracleData(common.Hash{0x01, 0xbc}.Bytes(), []byte{1, 2, 3, 4, 5, 6, 7}, 16)
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodAddLocalData, batching.BlockLatest, []interface{}{
			data.GetIdent(),
			new(big.Int).SetUint64(claimIdx),
			new(big.Int).SetUint64(uint64(data.OracleOffset)),
		}, nil)
		tx, err := game.UpdateOracleTx(context.Background(), claimIdx, data)
		require.NoError(t, err)
		stubRpc.VerifyTxCandidate(tx)
	})

	t.Run("Global", func(t *testing.T) {
		stubRpc, game := setupFaultDisputeGameTest(t)
		data := faultTypes.NewPreimageOracleData(common.Hash{0x02, 0xbc}.Bytes(), []byte{1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14, 15}, 16)
		claimIdx := uint64(6)
		stubRpc.SetResponse(fdgAddr, methodVM, batching.BlockLatest, nil, []interface{}{vmAddr})
		stubRpc.SetResponse(vmAddr, methodOracle, batching.BlockLatest, nil, []interface{}{oracleAddr})
		stubRpc.SetResponse(oracleAddr, methodLoadKeccak256PreimagePart, batching.BlockLatest, []interface{}{
			new(big.Int).SetUint64(uint64(data.OracleOffset)),
			data.GetPreimageWithoutSize(),
		}, nil)
		tx, err := game.UpdateOracleTx(context.Background(), claimIdx, data)
		require.NoError(t, err)
		stubRpc.VerifyTxCandidate(tx)
	})
}

func TestFaultDisputeGame_GetCredit(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)
	addr := common.Address{0x01}
	expected := big.NewInt(4284)
	stubRpc.SetResponse(fdgAddr, methodCredit, batching.BlockLatest, []interface{}{addr}, []interface{}{expected})

	actual, err := game.GetCredit(context.Background(), addr)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestFaultDisputeGame_GetCredits(t *testing.T) {
	stubRpc, game := setupFaultDisputeGameTest(t)

	block := batching.BlockByNumber(482)

	addrs := []common.Address{{0x01}, {0x02}, {0x03}}
	expected := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(0)}

	for i, addr := range addrs {
		stubRpc.SetResponse(fdgAddr, methodCredit, block, []interface{}{addr}, []interface{}{expected[i]})
	}

	actual, err := game.GetCredits(context.Background(), block, addrs...)
	require.NoError(t, err)
	require.Equal(t, len(expected), len(actual))
	for i := range expected {
		require.Zerof(t, expected[i].Cmp(actual[i]), "expected: %v actual: %v", expected[i], actual[i])
	}
}

func setupFaultDisputeGameTest(t *testing.T) (*batchingTest.AbiBasedRpc, *FaultDisputeGameContract) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	require.NoError(t, err)

	vmAbi, err := bindings.MIPSMetaData.GetAbi()
	require.NoError(t, err)
	oracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	require.NoError(t, err)

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	stubRpc.AddContract(vmAddr, vmAbi)
	stubRpc.AddContract(oracleAddr, oracleAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	game, err := NewFaultDisputeGameContract(fdgAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
