package contracts

import (
	"context"
	"math/big"
	"testing"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	versZKLatest = "0.0.0"
)

var zkVersions = []contractVersion{
	{
		version:  versZKLatest,
		gameType: gameTypes.OptimisticZKGameType,
		loadAbi:  snapshots.LoadZKDisputeGameABI,
	},
}

func TestZKSimpleGetters(t *testing.T) {
	tests := []struct {
		methodAlias string
		method      string
		args        []interface{}
		result      interface{}
		expected    interface{} // Defaults to expecting the same as result
		call        func(game OptimisticZKDisputeGameContract) (any, error)
		applies     func(version contractVersion) bool
	}{
		{
			methodAlias: "status",
			method:      methodStatus,
			result:      gameTypes.GameStatusChallengerWon,
			call: func(game OptimisticZKDisputeGameContract) (any, error) {
				return game.GetStatus(context.Background())
			},
		},
		{
			methodAlias: "l1Head",
			method:      methodL1Head,
			result:      common.Hash{0xdd, 0xbb},
			call: func(game OptimisticZKDisputeGameContract) (any, error) {
				return game.GetL1Head(context.Background())
			},
		},
		{
			methodAlias: "resolve",
			method:      methodResolve,
			result:      gameTypes.GameStatusInProgress,
			call: func(game OptimisticZKDisputeGameContract) (any, error) {
				return game.CallResolve(context.Background())
			},
		},
		{
			methodAlias: "resolvedAt",
			method:      methodResolvedAt,
			result:      uint64(240402),
			expected:    time.Unix(240402, 0),
			call: func(game OptimisticZKDisputeGameContract) (any, error) {
				return game.GetResolvedAt(context.Background(), rpcblock.Latest)
			},
		},
	}
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			for _, test := range tests {
				test := test
				t.Run(test.methodAlias, func(t *testing.T) {
					if test.applies != nil && !test.applies(version) {
						t.Skip("Skipping for this version")
					}
					stubRpc, game := setupZKDisputeGameTest(t, version)
					stubRpc.SetResponse(fdgAddr, test.method, rpcblock.Latest, nil, []interface{}{test.result})
					status, err := test.call(game)
					require.NoError(t, err)
					expected := test.expected
					if expected == nil {
						expected = test.result
					}
					require.Equal(t, expected, status)
				})
			}
		})
	}
}

func TestZKGetMetadata(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			expectedL1Head := common.Hash{0x0a, 0x0b}
			expectedL2BlockNumber := uint64(123)
			expectedRootClaim := common.Hash{0x01, 0x02}
			expectedStatus := gameTypes.GameStatusChallengerWon
			block := rpcblock.ByNumber(889)
			stubRpc.SetResponse(fdgAddr, methodL1Head, block, nil, []interface{}{expectedL1Head})
			stubRpc.SetResponse(fdgAddr, methodL2SequenceNumber, block, nil, []interface{}{new(big.Int).SetUint64(expectedL2BlockNumber)})
			stubRpc.SetResponse(fdgAddr, methodRootClaim, block, nil, []interface{}{expectedRootClaim})
			stubRpc.SetResponse(fdgAddr, methodStatus, block, nil, []interface{}{expectedStatus})
			actual, err := contract.GetMetadata(context.Background(), block)
			expected := GenericGameMetadata{
				L1Head:        expectedL1Head,
				L2SequenceNum: expectedL2BlockNumber,
				ProposedRoot:  expectedRootClaim,
				Status:        expectedStatus,
			}
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestZKGetGameRange(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			expectedStart := uint64(65)
			expectedEnd := uint64(102)
			stubRpc.SetResponse(fdgAddr, methodStartingBlockNumber, rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(expectedStart)})
			stubRpc.SetResponse(fdgAddr, methodL2SequenceNumber, rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(expectedEnd)})
			start, end, err := contract.GetGameRange(context.Background())
			require.NoError(t, err)
			require.Equal(t, expectedStart, start)
			require.Equal(t, expectedEnd, end)
		})
	}
}

func TestZKResolveTx(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, game := setupZKDisputeGameTest(t, version)
			stubRpc.SetResponse(fdgAddr, methodResolve, rpcblock.Latest, nil, nil)
			tx, err := game.ResolveTx()
			require.NoError(t, err)
			stubRpc.VerifyTxCandidate(tx)
		})
	}
}

func setupZKDisputeGameTest(t *testing.T, version contractVersion) (*batchingTest.AbiBasedRpc, OptimisticZKDisputeGameContract) {
	fdgAbi := version.loadAbi()

	vmAbi := snapshots.LoadMIPSABI()
	oracleAbi := snapshots.LoadPreimageOracleABI()

	stubRpc := batchingTest.NewAbiBasedRpc(t, fdgAddr, fdgAbi)
	stubRpc.AddContract(vmAddr, vmAbi)
	stubRpc.AddContract(oracleAddr, oracleAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)

	stubRpc.SetResponse(fdgAddr, methodGameType, rpcblock.Latest, nil, []interface{}{uint32(version.gameType)})
	stubRpc.SetResponse(fdgAddr, methodVersion, rpcblock.Latest, nil, []interface{}{version.version})
	stubRpc.SetResponse(oracleAddr, methodVersion, rpcblock.Latest, nil, []interface{}{oracleLatest})
	game, err := NewOptimisticZKDisputeGameContract(contractMetrics.NoopContractMetrics, fdgAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
