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

var (
	zkGameAddr = common.Address{0x45, 0x44, 0x43}
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
					stubRpc.SetResponse(zkGameAddr, test.method, rpcblock.Latest, nil, []interface{}{test.result})
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
			stubRpc.SetResponse(zkGameAddr, methodL1Head, block, nil, []interface{}{expectedL1Head})
			stubRpc.SetResponse(zkGameAddr, methodL2SequenceNumber, block, nil, []interface{}{new(big.Int).SetUint64(expectedL2BlockNumber)})
			stubRpc.SetResponse(zkGameAddr, methodRootClaim, block, nil, []interface{}{expectedRootClaim})
			stubRpc.SetResponse(zkGameAddr, methodStatus, block, nil, []interface{}{expectedStatus})
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
			stubRpc.SetResponse(zkGameAddr, methodStartingBlockNumber, rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(expectedStart)})
			stubRpc.SetResponse(zkGameAddr, methodL2SequenceNumber, rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(expectedEnd)})
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
			stubRpc.SetResponse(zkGameAddr, methodResolve, rpcblock.Latest, nil, nil)
			tx, err := game.ResolveTx()
			require.NoError(t, err)
			stubRpc.VerifyTxCandidate(tx)
		})
	}
}

func TestZKCanChallenge(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			parentIndex := uint32(525)
			claim := common.Hash{0xbb}
			deadline := uint64(42824240)

			tests := []struct {
				name           string
				counteredBy    common.Address
				prover         common.Address
				status         ProposalStatus
				expectedResult bool
			}{
				{
					name:           "Unchallenged",
					status:         ProposalStatusUnchallenged,
					expectedResult: true,
				},
				{
					name:           "Challenged",
					counteredBy:    common.Address{0xaa},
					status:         ProposalStatusChallenged,
					expectedResult: false,
				},
				{
					name:           "UnchallengedAndProven",
					prover:         common.Address{0xaa},
					status:         ProposalStatusUnchallengedAndValidProofProvided,
					expectedResult: false,
				},
				{
					name:           "ChallengedAndProven",
					counteredBy:    common.Address{0xaa},
					prover:         common.Address{0xbb},
					status:         ProposalStatusChallengedAndValidProofProvided,
					expectedResult: false,
				},
				{
					name:           "Resolved",
					status:         ProposalStatusResolved,
					expectedResult: false,
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					stubRpc, game := setupZKDisputeGameTest(t, version)
					stubRpc.SetResponse(zkGameAddr, methodClaimData, rpcblock.Latest, nil, []interface{}{
						parentIndex, test.counteredBy, test.prover, claim, test.status, deadline,
					})
					result, err := game.CanChallenge(context.Background())
					require.NoError(t, err)
					require.Equal(t, test.expectedResult, result)
				})
			}
		})
	}
}

func TestZKChallengeTx(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			bond := big.NewInt(97592472)

			stubRpc, game := setupZKDisputeGameTest(t, version)
			stubRpc.SetResponse(zkGameAddr, methodChallengerBond, rpcblock.Latest, nil, []interface{}{bond})
			stubRpc.SetResponse(zkGameAddr, methodChallenge, rpcblock.Latest, nil, nil)

			tx, err := game.ChallengeTx(context.Background())
			require.NoError(t, err)
			stubRpc.VerifyTxCandidate(tx)
		})
	}
}

func TestZKGetProposal(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			rootClaim := common.Hash{0xaa}
			l2SequenceNumber := big.NewInt(1236)
			stubRpc, game := setupZKDisputeGameTest(t, version)
			stubRpc.SetResponse(zkGameAddr, methodRootClaim, rpcblock.Latest, nil, []interface{}{rootClaim})
			stubRpc.SetResponse(zkGameAddr, methodL2SequenceNumber, rpcblock.Latest, nil, []interface{}{l2SequenceNumber})

			actualClaim, actualSeqNum, err := game.GetProposal(context.Background())
			require.NoError(t, err)
			require.Equal(t, rootClaim, actualClaim)
			require.Equal(t, l2SequenceNumber.Uint64(), actualSeqNum)
		})
	}
}

func setupZKDisputeGameTest(t *testing.T, version contractVersion) (*batchingTest.AbiBasedRpc, OptimisticZKDisputeGameContract) {
	fdgAbi := version.loadAbi()

	vmAbi := snapshots.LoadMIPSABI()
	oracleAbi := snapshots.LoadPreimageOracleABI()

	stubRpc := batchingTest.NewAbiBasedRpc(t, zkGameAddr, fdgAbi)
	stubRpc.AddContract(vmAddr, vmAbi)
	stubRpc.AddContract(oracleAddr, oracleAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)

	stubRpc.SetResponse(zkGameAddr, methodGameType, rpcblock.Latest, nil, []interface{}{uint32(version.gameType)})
	stubRpc.SetResponse(zkGameAddr, methodVersion, rpcblock.Latest, nil, []interface{}{version.version})
	stubRpc.SetResponse(oracleAddr, methodVersion, rpcblock.Latest, nil, []interface{}{oracleLatest})
	game, err := NewOptimisticZKDisputeGameContract(contractMetrics.NoopContractMetrics, zkGameAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
