package extract

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestOutputAgreementEnricher(t *testing.T) {
	t.Parallel()

	t.Run("ErrorWhenNoRollupClient", func(t *testing.T) {
		validator, _, _ := setupOutputValidatorTest(t)
		validator.clients = nil
		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:            200,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrRollupRpcRequired)
	})

	t.Run("SkipNonOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{4, 5, 7, 8, 10, 49812}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupOutputValidatorTest(t)
				validator.clients = nil // Should not error even though there's no rollup client
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:     200,
					L2BlockNumber: 0,
					RootClaim:     mockRootClaim,
				}
				err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
				require.NoError(t, err)
				require.Zero(t, metrics.fetchTime)
			})
		}
	})

	t.Run("FetchAllOutputRootGameTypes", func(t *testing.T) {
		gameTypes := []uint32{0, 1, 2, 3, 6, 254, 255, 1337}
		for _, gameType := range gameTypes {
			gameType := gameType
			t.Run(fmt.Sprintf("GameType_%d", gameType), func(t *testing.T) {
				validator, _, metrics := setupOutputValidatorTest(t)
				game := &types.EnrichedGameData{
					GameMetadata: challengerTypes.GameMetadata{
						GameType: gameType,
					},
					L1HeadNum:     200,
					L2BlockNumber: 0,
					RootClaim:     mockRootClaim,
				}
				err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
				require.NoError(t, err)
				require.NotZero(t, metrics.fetchTime, "should have fetched output root")
			})
		}
	})

	t.Run("AllNodesReturnError", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		for _, client := range clients {
			client.outputErr = errors.New("boom")
		}
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAllNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllNodesReturnNotFound", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		for _, client := range clients {
			client.outputErr = mockNotFoundRPCError()
		}
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("AllNodesOutOfSync", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].currentL1 = 99
		clients[1].currentL1 = 100 // Out of sync because it is only equal to the game L1 head
		clients[2].currentL1 = 0
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrAllNodesUnavailable)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("SomeNodesOutOfSync", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].currentL1 = 99
		// Would disagree but will be ignored because node is not in sync
		clients[0].outputRoot = common.Hash{0xaa, 0xbb, 0xcc, 0xdd}
		game := &types.EnrichedGameData{
			L1HeadNum:     100,
			L2BlockNumber: 0,
			RootClaim:     mockRootClaim,
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim) // Agree with the claim because all in-sync nodes returned the same result
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("SomeNodesReturnNotFound", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].outputErr = mockNotFoundRPCError()
		clients[1].outputErr = nil
		clients[2].outputErr = nil
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesMatchClaimAndSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 4)
		clients[0].outputErr = mockNotFoundRPCError()
		clients[1].outputErr = mockNotFoundRPCError()
		clients[2].outputRoot = mockRootClaim
		clients[2].safeHeadNum = 100
		clients[3].outputRoot = mockRootClaim
		clients[3].safeHeadNum = 100
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        50,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("MixedResponses_FoundNodesDontMatchClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		differentRoot := common.HexToHash("0x9999")
		clients[0].outputErr = mockNotFoundRPCError()
		clients[1].outputRoot = differentRoot
		clients[2].outputRoot = differentRoot
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        50,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("NodesDiverged", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		divergedRoot := common.HexToHash("0x5678")
		clients[0].outputRoot = mockRootClaim
		clients[1].outputRoot = divergedRoot
		clients[2].outputRoot = divergedRoot
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadNum = 100
		clients[1].safeHeadNum = 99
		clients[2].safeHeadNum = 101
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("SafeHeadError", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadErr = errors.New("boom")
		clients[1].safeHeadErr = nil
		clients[2].safeHeadErr = nil
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        0,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
		require.True(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("OutputMatches_NotSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)
		clients[0].safeHeadNum = 50
		clients[1].safeHeadNum = 60
		clients[2].safeHeadNum = 70
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        80,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_OutputMatchesClaim_NoneReportSafe", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)

		for _, client := range clients {
			client.outputRoot = mockRootClaim
			client.safeHeadNum = 40
		}

		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        50, // Higher than all safe heads
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim, "Should set ExpectedRootClaim to empty hash when not safe")
		require.False(t, game.AgreeWithClaim, "Should disagree because none report it as safe")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("AllNodesAgree_OutputDifferentFromClaim", func(t *testing.T) {
		validator, clients, metrics := setupMultiNodeTest(t, 3)

		differentRoot := common.HexToHash("0xdifferent")
		for _, client := range clients {
			client.outputRoot = differentRoot
			// Safe head numbers don't matter here since the output doesn't match the claim
		}

		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        50,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}

		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, differentRoot, game.ExpectedRootClaim, "Should set ExpectedRootClaim to the agreed output")
		require.False(t, game.AgreeWithClaim, "Should disagree because output doesn't match claim")
		require.NotZero(t, metrics.fetchTime)
	})

	t.Run("BlockNumberLargerThanInt64", func(t *testing.T) {
		validator, rollup, metrics := setupOutputValidatorTest(t)
		// RPC block numbers must be a int64 to be valid. Anything bigger than that should be treated as invalid
		// without even making a request to the node.
		rollup.outputErr = errors.New("should not have even requested the output root")
		game := &types.EnrichedGameData{
			L1HeadNum:            100,
			L2BlockNumber:        uint64(math.MaxInt64) + 1,
			RootClaim:            mockRootClaim,
			RollupEndpointErrors: make(map[string]bool),
		}
		err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.Equal(t, common.Hash{}, game.ExpectedRootClaim)
		require.False(t, game.AgreeWithClaim)
		require.Zero(t, metrics.fetchTime)
	})

	t.Run("RecordEndpointErrors", func(t *testing.T) {
		t.Run("SingleNodeError", func(t *testing.T) {
			validator, client, _ := setupOutputValidatorTest(t)
			client.outputErr = errors.New("connection failed")
			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:            200,
				L2BlockNumber:        100,
				RootClaim:            mockRootClaim,
				RollupEndpointErrors: make(map[string]bool),
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.ErrorIs(t, err, ErrAllNodesUnavailable)
			require.NotNil(t, game.RollupEndpointErrors)
			require.Contains(t, game.RollupEndpointErrors, "client-0")
		})

		t.Run("MultiNodeErrors", func(t *testing.T) {
			validator, clients, _ := setupMultiNodeTest(t, 3)
			clients[0].outputErr = errors.New("connection timeout")
			clients[2].outputErr = errors.New("server error")
			// clients[1] will succeed

			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:            200,
				L2BlockNumber:        100,
				RootClaim:            mockRootClaim,
				RollupEndpointErrors: make(map[string]bool),
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.NoError(t, err)
			require.NotNil(t, game.RollupEndpointErrors)
			require.Contains(t, game.RollupEndpointErrors, "client-0")
			require.Contains(t, game.RollupEndpointErrors, "client-2")
			require.NotContains(t, game.RollupEndpointErrors, "client-1")
			require.Len(t, game.RollupEndpointErrors, 2)
		})

		t.Run("NotFoundErrorsNotRecorded", func(t *testing.T) {
			validator, client, _ := setupOutputValidatorTest(t)
			client.outputErr = mockNotFoundRPCError()
			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:            200,
				L2BlockNumber:        100,
				RootClaim:            mockRootClaim,
				RollupEndpointErrors: make(map[string]bool),
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.NoError(t, err)
			require.NotNil(t, game.RollupEndpointErrors)
			require.Empty(t, game.RollupEndpointErrors)
		})

	})

	t.Run("RecordEndpointErrorCounts", func(t *testing.T) {
		t.Run("SingleNodeErrorCount", func(t *testing.T) {
			validator, client, _ := setupOutputValidatorTest(t)
			client.outputErr = errors.New("connection failed")
			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:                200,
				L2BlockNumber:            100,
				RootClaim:                mockRootClaim,
				RollupEndpointErrors:     make(map[string]bool),
				RollupEndpointErrorCount: 0,
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.ErrorIs(t, err, ErrAllNodesUnavailable)
			require.Equal(t, 1, game.RollupEndpointErrorCount)
		})

		t.Run("MultiNodeErrorCount", func(t *testing.T) {
			validator, clients, _ := setupMultiNodeTest(t, 4)
			clients[0].outputErr = errors.New("connection timeout")
			clients[1].outputErr = errors.New("server error")
			clients[2].outputErr = errors.New("another error")
			// clients[3] will succeed

			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:                200,
				L2BlockNumber:            100,
				RootClaim:                mockRootClaim,
				RollupEndpointErrors:     make(map[string]bool),
				RollupEndpointErrorCount: 0,
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.NoError(t, err)
			require.Equal(t, 3, game.RollupEndpointErrorCount)
		})

		t.Run("NotFoundErrorsNotCounted", func(t *testing.T) {
			validator, clients, _ := setupMultiNodeTest(t, 3)
			clients[0].outputErr = mockNotFoundRPCError()
			clients[1].outputErr = mockNotFoundRPCError()
			// clients[2] will succeed

			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:                200,
				L2BlockNumber:            100,
				RootClaim:                mockRootClaim,
				RollupEndpointErrors:     make(map[string]bool),
				RollupEndpointErrorCount: 0,
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.NoError(t, err)
			require.Equal(t, 0, game.RollupEndpointErrorCount)
		})

		t.Run("MixedErrorTypes", func(t *testing.T) {
			validator, clients, _ := setupMultiNodeTest(t, 4)
			clients[0].outputErr = mockNotFoundRPCError()         // Should not be counted
			clients[1].outputErr = errors.New("connection error") // Should be counted
			clients[2].outputErr = errors.New("server error")     // Should be counted
			// clients[3] will succeed

			game := &types.EnrichedGameData{
				GameMetadata: challengerTypes.GameMetadata{
					GameType: 0,
				},
				L1HeadNum:                200,
				L2BlockNumber:            100,
				RootClaim:                mockRootClaim,
				RollupEndpointErrors:     make(map[string]bool),
				RollupEndpointErrorCount: 0,
			}
			err := validator.Enrich(context.Background(), rpcblock.Latest, nil, game)
			require.NoError(t, err)
			require.Equal(t, 2, game.RollupEndpointErrorCount)
		})
	})
}

// mockNotFoundRPCError creates a minimal rpc.Error that reports a "not found" message
// to exercise the JSON-RPC application error path in the enricher.
func mockNotFoundRPCError() rpc.Error { return testRPCError{msg: "not found", code: -32000} }

type testRPCError struct {
	msg  string
	code int
}

func (e testRPCError) Error() string  { return e.msg }
func (e testRPCError) ErrorCode() int { return e.code }

func setupOutputValidatorTest(t *testing.T) (*OutputAgreementEnricher, *stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	client := &stubRollupClient{
		currentL1:   math.MaxUint64,
		safeHeadNum: 99999999999,
	}
	metrics := &stubOutputMetrics{}
	validator := NewOutputAgreementEnricher(logger, metrics, []OutputRollupClient{client}, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, client, metrics
}

func setupMultiNodeTest(t *testing.T, numNodes int) (*OutputAgreementEnricher, []*stubRollupClient, *stubOutputMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	clients := make([]*stubRollupClient, numNodes)
	rollupClients := make([]OutputRollupClient, numNodes)
	for i := range clients {
		clients[i] = &stubRollupClient{
			currentL1:   math.MaxUint64,
			safeHeadNum: 99999999999,
			outputRoot:  mockRootClaim,
		}
		rollupClients[i] = clients[i]
	}
	metrics := &stubOutputMetrics{}
	validator := NewOutputAgreementEnricher(logger, metrics, rollupClients, clock.NewDeterministicClock(time.Unix(9824924, 499)))
	return validator, clients, metrics
}

type stubOutputMetrics struct {
	fetchTime float64
}

func (s *stubOutputMetrics) RecordOutputFetchTime(fetchTime float64) {
	s.fetchTime = fetchTime
}

type stubRollupClient struct {
	blockNum      uint64
	outputErr     error
	safeHeadErr   error
	safeHeadNum   uint64
	outputRoot    common.Hash
	currentL1     uint64
	syncStatusErr error
}

func (s *stubRollupClient) SyncStatus(_ context.Context) (*eth.SyncStatus, error) {
	if s.syncStatusErr != nil {
		return nil, s.syncStatusErr
	}
	return &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: s.currentL1}}, nil
}

func (s *stubRollupClient) OutputAtBlock(_ context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	s.blockNum = blockNum
	if s.outputErr != nil {
		return nil, s.outputErr
	}
	return &eth.OutputResponse{OutputRoot: eth.Bytes32(s.outputRoot)}, nil
}

func (s *stubRollupClient) SafeHeadAtL1Block(_ context.Context, _ uint64) (*eth.SafeHeadResponse, error) {
	if s.safeHeadErr != nil {
		return nil, s.safeHeadErr
	}
	return &eth.SafeHeadResponse{
		SafeHead: eth.BlockID{
			Number: s.safeHeadNum,
		},
	}, nil
}

func TestOutputAgreementEnricher_SafetyCounting(t *testing.T) {
	t.Parallel()

	t.Run("CountsSafetyWhenOutputRootMatchesRootClaim", func(t *testing.T) {
		rootClaim := common.HexToHash("0xabcd")
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// Client 0: safe (safeHeadNum >= l2BlockNumber)
		clients[0].outputRoot = rootClaim
		clients[0].safeHeadNum = 100

		// Client 1: unsafe (safeHeadNum < l2BlockNumber)
		clients[1].outputRoot = rootClaim
		clients[1].safeHeadNum = 50

		// Client 2: safe
		clients[2].outputRoot = rootClaim
		clients[2].safeHeadNum = 150

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                 200,
			L2BlockNumber:             75,
			RootClaim:                 rootClaim,
			RollupEndpointErrors:      make(map[string]bool),
			RollupEndpointSafeCount:   0,
			RollupEndpointUnsafeCount: 0,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 2, game.RollupEndpointSafeCount, "Should count 2 safe endpoints")
		require.Equal(t, 1, game.RollupEndpointUnsafeCount, "Should count 1 unsafe endpoint")
		require.True(t, game.HasMixedSafety(), "Should have mixed safety")
	})

	t.Run("DoesNotCountSafetyWhenOutputRootDiffersFromRootClaim", func(t *testing.T) {
		rootClaim := common.HexToHash("0xabcd")
		differentRoot := common.HexToHash("0xdiff")
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// All clients return different root but have varying safety
		for _, client := range clients {
			client.outputRoot = differentRoot
			client.safeHeadNum = 100 // All would be safe if checked
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                 200,
			L2BlockNumber:             75,
			RootClaim:                 rootClaim,
			RollupEndpointErrors:      make(map[string]bool),
			RollupEndpointSafeCount:   0,
			RollupEndpointUnsafeCount: 0,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 0, game.RollupEndpointSafeCount, "Should not count safety when output root differs")
		require.Equal(t, 0, game.RollupEndpointUnsafeCount, "Should not count safety when output root differs")
		require.False(t, game.HasMixedSafety(), "Should not have mixed safety")
	})

	t.Run("DoesNotCountSafetyForNotFoundResults", func(t *testing.T) {
		rootClaim := common.HexToHash("0xabcd")
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// Client 0: found and safe
		clients[0].outputRoot = rootClaim
		clients[0].safeHeadNum = 100

		// Client 1: not found
		clients[1].outputErr = errors.New("not found")

		// Client 2: found and unsafe
		clients[2].outputRoot = rootClaim
		clients[2].safeHeadNum = 50

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                 200,
			L2BlockNumber:             75,
			RootClaim:                 rootClaim,
			RollupEndpointErrors:      make(map[string]bool),
			RollupEndpointSafeCount:   0,
			RollupEndpointUnsafeCount: 0,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 1, game.RollupEndpointSafeCount, "Should count only found safe endpoints")
		require.Equal(t, 1, game.RollupEndpointUnsafeCount, "Should count only found unsafe endpoints")
		require.True(t, game.HasMixedSafety(), "Should have mixed safety")
	})

	t.Run("AllEndpointsSafeNoMixedSafety", func(t *testing.T) {
		rootClaim := common.HexToHash("0xabcd")
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// All clients safe
		for _, client := range clients {
			client.outputRoot = rootClaim
			client.safeHeadNum = 100
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                 200,
			L2BlockNumber:             75,
			RootClaim:                 rootClaim,
			RollupEndpointErrors:      make(map[string]bool),
			RollupEndpointSafeCount:   0,
			RollupEndpointUnsafeCount: 0,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)

		require.Equal(t, 3, game.RollupEndpointSafeCount, "Should count all safe endpoints")
		require.Equal(t, 0, game.RollupEndpointUnsafeCount, "Should count no unsafe endpoints")
		require.False(t, game.HasMixedSafety(), "Should not have mixed safety")
	})

	t.Run("TracksDifferentOutputRootsWhenNodesDiverge", func(t *testing.T) {
		enricher, clients, _ := setupMultiNodeTest(t, 3)
		divergedRoot := common.HexToHash("0x5678")

		// Set up different output roots
		clients[0].outputRoot = mockRootClaim
		clients[1].outputRoot = divergedRoot
		clients[2].outputRoot = divergedRoot

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                          100,
			L2BlockNumber:                      0,
			RootClaim:                          mockRootClaim,
			RollupEndpointErrors:               make(map[string]bool),
			RollupEndpointDifferentOutputRoots: false,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.True(t, game.RollupEndpointDifferentOutputRoots, "Should track different output roots")
	})

	t.Run("DoesNotTrackDifferentOutputRootsWhenNodesAgree", func(t *testing.T) {
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// All clients return the same output root
		for _, client := range clients {
			client.outputRoot = mockRootClaim
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                          100,
			L2BlockNumber:                      0,
			RootClaim:                          mockRootClaim,
			RollupEndpointErrors:               make(map[string]bool),
			RollupEndpointDifferentOutputRoots: false,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.False(t, game.RollupEndpointDifferentOutputRoots, "Should not track different output roots when nodes agree")
	})

	t.Run("DoesNotTrackDifferentOutputRootsForMixedAvailability", func(t *testing.T) {
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// Set up mixed availability: some found, some not found
		clients[0].outputRoot = mockRootClaim
		clients[1].outputRoot = mockRootClaim
		clients[2].outputErr = mockNotFoundRPCError() // This client returns "not found"

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                          100,
			L2BlockNumber:                      0,
			RootClaim:                          mockRootClaim,
			RollupEndpointErrors:               make(map[string]bool),
			RollupEndpointDifferentOutputRoots: false,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.False(t, game.RollupEndpointDifferentOutputRoots, "Should not track different output roots for mixed availability")
		require.True(t, game.HasMixedAvailability(), "Should have mixed availability")
	})

	t.Run("TracksDifferentOutputRootsWithSingleDisagreeingNode", func(t *testing.T) {
		enricher, clients, _ := setupMultiNodeTest(t, 3)
		divergedRoot := common.HexToHash("0x9999")

		// Two nodes agree, one disagrees
		clients[0].outputRoot = mockRootClaim
		clients[1].outputRoot = mockRootClaim
		clients[2].outputRoot = divergedRoot

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                          100,
			L2BlockNumber:                      0,
			RootClaim:                          mockRootClaim,
			RollupEndpointErrors:               make(map[string]bool),
			RollupEndpointDifferentOutputRoots: false,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.NoError(t, err)
		require.True(t, game.RollupEndpointDifferentOutputRoots, "Should track different output roots even with single disagreeing node")
	})

	t.Run("DoesNotTrackDifferentOutputRootsWithOnlyErrors", func(t *testing.T) {
		enricher, clients, _ := setupMultiNodeTest(t, 3)

		// All clients return errors (no successful results to compare)
		for _, client := range clients {
			client.outputErr = errors.New("rpc error")
		}

		game := &types.EnrichedGameData{
			GameMetadata: challengerTypes.GameMetadata{
				GameType: 0,
			},
			L1HeadNum:                          100,
			L2BlockNumber:                      0,
			RootClaim:                          mockRootClaim,
			RollupEndpointErrors:               make(map[string]bool),
			RollupEndpointDifferentOutputRoots: false,
		}

		err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
		require.ErrorIs(t, err, ErrAllNodesUnavailable)
		require.False(t, game.RollupEndpointDifferentOutputRoots, "Should not track different output roots when all nodes error")
	})
}
