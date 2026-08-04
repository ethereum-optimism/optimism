package contracts

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	versZKLatest = "0.0.0"
)

var (
	zkGameAddr = common.Address{0x45, 0x44, 0x43}
	zkWethAddr = common.Address{0x57, 0x45, 0x54, 0x48}
)

var zkVersions = []contractVersion{
	{
		version:  versZKLatest,
		gameType: gameTypes.ZKDisputeGameType,
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
		call        func(game ZKDisputeGameContract) (any, error)
		applies     func(version contractVersion) bool
	}{
		{
			methodAlias: "status",
			method:      methodStatus,
			result:      gameTypes.GameStatusChallengerWon,
			call: func(game ZKDisputeGameContract) (any, error) {
				return game.GetStatus(context.Background())
			},
		},
		{
			methodAlias: "l1Head",
			method:      methodL1Head,
			result:      common.Hash{0xdd, 0xbb},
			call: func(game ZKDisputeGameContract) (any, error) {
				return game.GetL1Head(context.Background())
			},
		},
		{
			methodAlias: "resolve",
			method:      methodResolve,
			result:      gameTypes.GameStatusInProgress,
			call: func(game ZKDisputeGameContract) (any, error) {
				return game.CallResolve(context.Background())
			},
		},
		{
			methodAlias: "resolvedAt",
			method:      methodResolvedAt,
			result:      uint64(240402),
			expected:    time.Unix(240402, 0),
			call: func(game ZKDisputeGameContract) (any, error) {
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

func TestProposalStatusFromUint8(t *testing.T) {
	for value := uint8(ProposalStatusUnchallenged); value <= uint8(ProposalStatusResolved); value++ {
		status, err := ProposalStatusFromUint8(value)
		require.NoError(t, err)
		require.Equal(t, ProposalStatus(value), status)
	}
	status, err := ProposalStatusFromUint8(uint8(ProposalStatusResolved) + 1)
	require.ErrorContains(t, err, "invalid proposal status")
	require.Equal(t, ProposalStatusUnchallenged, status)
}

func TestZKGetAnchorStateRegistryAtPinnedBlock(t *testing.T) {
	for _, version := range zkVersions {
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			block := rpcblock.ByHash(common.Hash{0x88, 0x99})
			expected := common.Address{0xab}
			stubRpc.SetResponse(zkGameAddr, methodAnchorStateRegistry, block, nil, []interface{}{expected})
			actual, err := contract.GetAnchorStateRegistry(context.Background(), block)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestZKGetBondMetadataAtPinnedBlock(t *testing.T) {
	for _, version := range zkVersions {
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			block := rpcblock.ByHash(common.Hash{0x91})
			expected := ZKBondMetadata{
				GameCreator:    common.Address{0x01},
				TotalBonds:     big.NewInt(23),
				ChallengerBond: big.NewInt(7),
			}
			stubRpc.SetResponse(zkGameAddr, methodGameCreator, block, nil, []interface{}{expected.GameCreator})
			stubRpc.SetResponse(zkGameAddr, methodTotalBonds, block, nil, []interface{}{expected.TotalBonds})
			stubRpc.SetResponse(zkGameAddr, methodChallengerBond, block, nil, []interface{}{expected.ChallengerBond})

			actual, err := contract.GetBondMetadata(t.Context(), block)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestZKGetCreditsAtPinnedBlock(t *testing.T) {
	for _, version := range zkVersions {
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			block := rpcblock.ByNumber(492)
			recipients := []common.Address{{0x01}, {0x02}, {0x03}}
			expected := []*big.Int{big.NewInt(3), big.NewInt(5), big.NewInt(8)}
			for i, recipient := range recipients {
				stubRpc.SetResponse(zkGameAddr, methodCredit, block, []interface{}{recipient}, []interface{}{expected[i]})
			}

			actual, err := contract.GetCredits(t.Context(), block, recipients...)
			require.NoError(t, err)
			require.Equal(t, expected, actual)

			empty, err := contract.GetCredits(t.Context(), block)
			require.NoError(t, err)
			require.Empty(t, empty)
			require.NotNil(t, empty)
		})
	}
}

func TestZKGetWithdrawalsAtPinnedBlock(t *testing.T) {
	for _, version := range zkVersions {
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			stubRpc.AddContract(zkWethAddr, snapshots.LoadDelayedWETHABI())
			block := rpcblock.ByHash(common.Hash{0x92})
			recipients := []common.Address{{0x01}, {0x02}}
			expected := []*WithdrawalRequest{
				{Amount: big.NewInt(11), Timestamp: big.NewInt(101)},
				{Amount: big.NewInt(13), Timestamp: big.NewInt(103)},
			}
			stubRpc.SetResponse(zkGameAddr, methodWETH, block, nil, []interface{}{zkWethAddr})
			for i, recipient := range recipients {
				stubRpc.SetResponse(zkWethAddr, methodWithdrawals, block, []interface{}{zkGameAddr, recipient}, []interface{}{expected[i].Amount, expected[i].Timestamp})
			}

			actual, err := contract.GetWithdrawals(t.Context(), block, recipients...)
			require.NoError(t, err)
			require.Equal(t, expected, actual)

			_, emptyContract := setupZKDisputeGameTest(t, version)
			empty, err := emptyContract.GetWithdrawals(t.Context(), block)
			require.NoError(t, err)
			require.Empty(t, empty)
			require.NotNil(t, empty)
		})
	}
}

func TestZKGetBalanceAndDelayAtPinnedBlock(t *testing.T) {
	for _, version := range zkVersions {
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			stubRpc.AddContract(zkWethAddr, snapshots.LoadDelayedWETHABI())
			block := rpcblock.ByNumber(493)
			balance := big.NewInt(144)
			delaySeconds := int64(77)
			stubRpc.SetResponse(zkGameAddr, methodWETH, block, nil, []interface{}{zkWethAddr})
			stubRpc.AddExpectedCall(batchingTest.NewGetBalanceCall(zkWethAddr, block, balance))
			stubRpc.SetResponse(zkWethAddr, methodDelay, block, nil, []interface{}{big.NewInt(delaySeconds)})

			actualBalance, actualDelay, actualAddr, err := contract.GetBalanceAndDelay(t.Context(), block)
			require.NoError(t, err)
			require.Equal(t, balance, actualBalance)
			require.Equal(t, time.Duration(delaySeconds)*time.Second, actualDelay)
			require.Equal(t, zkWethAddr, actualAddr)
		})
	}
}

func TestValidateZKResultCount(t *testing.T) {
	require.NoError(t, validateZKResultCount(2, 2))
	require.ErrorContains(t, validateZKResultCount(2, 1), "expected 2 results but got 1")
	require.ErrorContains(t, validateZKResultCount(2, 3), "expected 2 results but got 3")
}

func TestZKGetGameRange(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			expectedStart := uint64(65)
			expectedEnd := uint64(102)
			stubRpc.SetResponse(zkGameAddr, methodStartingSequenceNumber, rpcblock.Latest, nil, []interface{}{new(big.Int).SetUint64(expectedStart)})
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

func TestZKGetChallengerMetadata(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, contract := setupZKDisputeGameTest(t, version)
			expectedParentIndex := uint32(525)
			expectedProposalStatus := ProposalStatusChallengedAndValidProofProvided
			challenger := common.Address{0xad}
			prover := common.Address{0xac}
			expectedL2BlockNumber := uint64(123)
			expectedRootClaim := common.Hash{0x01, 0x02}
			expectedDeadline := time.Unix(84928429020, 0)
			block := rpcblock.ByNumber(889)
			stubRpc.SetResponse(zkGameAddr, methodClaimData, block, nil, []interface{}{
				expectedParentIndex, expectedProposalStatus, challenger, prover, uint64(expectedDeadline.Unix()), expectedRootClaim,
			})
			stubRpc.SetResponse(zkGameAddr, methodL2SequenceNumber, block, nil, []interface{}{new(big.Int).SetUint64(expectedL2BlockNumber)})
			actual, err := contract.GetChallengerMetadata(context.Background(), block)
			expected := ChallengerMetadata{
				ParentIndex:      expectedParentIndex,
				ProposalStatus:   expectedProposalStatus,
				Challenger:       challenger,
				Prover:           prover,
				ProposedRoot:     expectedRootClaim,
				L2SequenceNumber: expectedL2BlockNumber,
				Deadline:         expectedDeadline,
			}
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}
}

func TestZKGetChallengerMetadataAllowsUnknownProposalStatus(t *testing.T) {
	stubRpc, contract := setupZKDisputeGameTest(t, zkVersions[0])
	block := rpcblock.ByNumber(889)
	unknown := ProposalStatus(255)
	stubRpc.SetResponse(zkGameAddr, methodClaimData, block, nil, []interface{}{
		uint32(0), unknown, common.Address{}, common.Address{}, uint64(0), common.Hash{},
	})
	stubRpc.SetResponse(zkGameAddr, methodL2SequenceNumber, block, nil, []interface{}{new(big.Int)})

	metadata, err := contract.GetChallengerMetadata(context.Background(), block)
	require.NoError(t, err)
	require.Equal(t, unknown, metadata.ProposalStatus)
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
			require.Equal(t, bigs.Uint64Strict(l2SequenceNumber), actualSeqNum)
		})
	}
}

func TestZKGame_GetCredit(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, game := setupZKDisputeGameTest(t, version)
			addr := common.Address{0x01}
			expectedCredit := big.NewInt(4284)
			expectedStatus := gameTypes.GameStatusChallengerWon
			stubRpc.SetResponse(zkGameAddr, methodCredit, rpcblock.Latest, []interface{}{addr}, []interface{}{expectedCredit})
			stubRpc.SetResponse(zkGameAddr, methodStatus, rpcblock.Latest, nil, []interface{}{expectedStatus})

			actualCredit, actualStatus, err := game.GetCredit(context.Background(), addr)
			require.NoError(t, err)
			require.Equal(t, expectedCredit, actualCredit)
			require.Equal(t, expectedStatus, actualStatus)
		})
	}
}

func TestZKGame_ClaimCreditTx(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			t.Run("Success", func(t *testing.T) {
				stubRpc, game := setupZKDisputeGameTest(t, version)
				addr := common.Address{0xaa}

				stubRpc.SetResponse(zkGameAddr, methodClaimCredit, rpcblock.Latest, []interface{}{addr}, nil)
				tx, err := game.ClaimCreditTx(context.Background(), addr)
				require.NoError(t, err)
				stubRpc.VerifyTxCandidate(tx)
			})

			t.Run("SimulationFails", func(t *testing.T) {
				stubRpc, game := setupZKDisputeGameTest(t, version)
				addr := common.Address{0xaa}

				stubRpc.SetError(zkGameAddr, methodClaimCredit, rpcblock.Latest, []interface{}{addr}, errors.New("still locked"))
				tx, err := game.ClaimCreditTx(context.Background(), addr)
				require.ErrorIs(t, err, ErrSimulationFailed)
				require.Equal(t, txmgr.TxCandidate{}, tx)
			})
		})
	}
}

func TestZKGame_GetBondDistributionMode(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			stubRpc, game := setupZKDisputeGameTest(t, version)
			stubRpc.SetResponse(zkGameAddr, methodBondDistributionMode, rpcblock.Latest, nil, []interface{}{uint8(faultTypes.NormalDistributionMode)})

			mode, err := game.GetBondDistributionMode(context.Background(), rpcblock.Latest)
			require.NoError(t, err)
			require.Equal(t, faultTypes.NormalDistributionMode, mode)
		})
	}
}

func TestZKGame_IsClosed(t *testing.T) {
	modes := []struct {
		name   string
		mode   faultTypes.BondDistributionMode
		closed bool
	}{
		{name: "Undecided", mode: faultTypes.UndecidedDistributionMode, closed: false},
		{name: "Normal", mode: faultTypes.NormalDistributionMode, closed: true},
		{name: "Refund", mode: faultTypes.RefundDistributionMode, closed: true},
		{name: "Legacy", mode: faultTypes.LegacyDistributionMode, closed: true},
	}
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			for _, test := range modes {
				test := test
				t.Run(test.name, func(t *testing.T) {
					stubRpc, game := setupZKDisputeGameTest(t, version)
					stubRpc.SetResponse(zkGameAddr, methodBondDistributionMode, rpcblock.Latest, nil, []interface{}{uint8(test.mode)})

					closed, err := game.IsClosed(context.Background())

					require.NoError(t, err)
					require.Equal(t, test.closed, closed)
				})
			}
		})
	}
}

func TestZKGame_CloseGameTx(t *testing.T) {
	for _, version := range zkVersions {
		version := version
		t.Run(version.String(), func(t *testing.T) {
			t.Run("Success", func(t *testing.T) {
				stubRpc, game := setupZKDisputeGameTest(t, version)
				stubRpc.SetResponse(zkGameAddr, methodCloseGame, rpcblock.Latest, nil, nil)

				tx, err := game.CloseGameTx(context.Background())
				require.NoError(t, err)
				stubRpc.VerifyTxCandidate(tx)
			})

			t.Run("SimulationFails", func(t *testing.T) {
				stubRpc, game := setupZKDisputeGameTest(t, version)
				stubRpc.SetError(zkGameAddr, methodCloseGame, rpcblock.Latest, nil, errors.New("game not ready"))

				tx, err := game.CloseGameTx(context.Background())
				require.ErrorIs(t, err, ErrSimulationFailed)
				require.Equal(t, txmgr.TxCandidate{}, tx)
			})
		})
	}
}

func setupZKDisputeGameTest(t *testing.T, version contractVersion) (*batchingTest.AbiBasedRpc, ZKDisputeGameContract) {
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
	game, err := NewZKDisputeGameContract(contractMetrics.NoopContractMetrics, zkGameAddr, caller)
	require.NoError(t, err)
	return stubRpc, game
}
