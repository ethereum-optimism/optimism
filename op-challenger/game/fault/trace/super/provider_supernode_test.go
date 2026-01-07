package super

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	types2 "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	interopTypes "github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSuperNodeProvider_Get(t *testing.T) {
	t.Run("AtPostState", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		expectedSuper := eth.NewSuperV1(poststateTimestamp, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromUInt64(1),
			Output:  eth.Bytes32{0xbb},
		})
		response := eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: l1Head,
				Super:              expectedSuper,
				SuperRoot:          eth.SuperRoot(expectedSuper),
			},
		}
		stubSuperNode.Add(response)
		claim, err := provider.Get(context.Background(), types.RootPosition)
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedSuper)), claim)
	})

	t.Run("AtNewTimestamp", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		expectedSuper := eth.NewSuperV1(prestateTimestamp+1, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromUInt64(1),
			Output:  eth.Bytes32{0xbb},
		})
		response := eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: l1Head,
				Super:              expectedSuper,
				SuperRoot:          eth.SuperRoot(expectedSuper),
			},
		}
		stubSuperNode.Add(response)
		claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(StepsPerTimestamp-1)))
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedSuper)), claim)
	})

	t.Run("ValidTransitionBetweenFirstTwoSuperRoots", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		stubSuperNode.Add(prev)
		stubSuperNode.Add(next)

		expectSuperNodeValidTransition(t, provider, prev, next)
	})

	t.Run("Step0SuperRootIsSafeBeforeGameL1Head", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		expectedSuper := eth.NewSuperV1(poststateTimestamp, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromUInt64(1),
			Output:  eth.Bytes32{0xbb},
		})
		response := eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: eth.BlockID{Number: l1Head.Number - 10, Hash: common.Hash{0xcc}},
				Super:              expectedSuper,
				SuperRoot:          eth.SuperRoot(expectedSuper),
			},
		}
		stubSuperNode.Add(response)
		claim, err := provider.Get(context.Background(), types.RootPosition)
		require.NoError(t, err)
		require.Equal(t, common.Hash(eth.SuperRoot(expectedSuper)), claim)
	})

	t.Run("Step0SuperRootNotSafeAtGameL1Head", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		expectedSuper := eth.NewSuperV1(poststateTimestamp, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromUInt64(1),
			Output:  eth.Bytes32{0xbb},
		})
		response := eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xcc}},
				Super:              expectedSuper,
				SuperRoot:          eth.SuperRoot(expectedSuper),
			},
		}
		stubSuperNode.Add(response)
		claim, err := provider.Get(context.Background(), types.RootPosition)
		require.NoError(t, err)
		require.Equal(t, InvalidTransitionHash, claim)
	})

	t.Run("NextSuperRootSafeBeforeGameL1Head", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		// Make super roots be safe earlier
		prev.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number - 10, Hash: common.Hash{0xaa}}
		next.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number - 5, Hash: common.Hash{0xbb}}
		stubSuperNode.Add(prev)
		stubSuperNode.Add(next)
		expectSuperNodeValidTransition(t, provider, prev, next)
	})

	t.Run("PreviousSuperRootNotSafeAtGameL1Head", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		// Make super roots be safe only after L1 head
		prev.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xaa}}
		next.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number + 2, Hash: common.Hash{0xbb}}
		stubSuperNode.Add(prev)
		stubSuperNode.Add(next)

		// All steps should be the invalid transition hash.
		for i := int64(0); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("FirstChainUnsafe", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		// Make super roots be safe only after L1 head
		prev.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number, Hash: common.Hash{0xaa}}
		next.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xbb}}
		next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(1)] = eth.OutputWithRequiredL1{
			Output: &eth.OutputResponse{
				OutputRoot:            eth.Bytes32{0xad},
				BlockRef:              eth.L2BlockRef{Hash: common.Hash{0xcd}},
				WithdrawalStorageRoot: common.Hash{0xde},
				StateRoot:             common.Hash{0xdf},
			},
			RequiredL1: eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xbb}},
		}
		stubSuperNode.Add(prev)
		stubSuperNode.Add(next)

		// All steps should be the invalid transition hash.
		for i := int64(0); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("SecondChainUnsafe", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		// Make super roots be safe only after L1 head
		prev.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number, Hash: common.Hash{0xaa}}
		next.Data.VerifiedRequiredL1 = eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xbb}}
		next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(2)] = eth.OutputWithRequiredL1{
			Output: &eth.OutputResponse{
				OutputRoot:            eth.Bytes32{0xad},
				BlockRef:              eth.L2BlockRef{Hash: common.Hash{0xcd}},
				WithdrawalStorageRoot: common.Hash{0xde},
				StateRoot:             common.Hash{0xdf},
			},
			RequiredL1: eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xbb}},
		}
		stubSuperNode.Add(prev)
		stubSuperNode.Add(next)

		// First step should be valid because we can reach the required block on chain 1
		claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(0)))
		require.NoError(t, err)
		require.NotEqual(t, InvalidTransitionHash, claim, "incorrect claim at index 0")

		// Remaining steps should be the invalid transition hash.
		for i := int64(1); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("Step0ForTimestampBeyondChainHead", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		stubSuperNode.AddAtTimestamp(poststateTimestamp, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data:      nil,
		})

		claim, err := provider.Get(context.Background(), types.RootPosition)
		require.NoError(t, err)
		require.Equal(t, InvalidTransitionHash, claim)
	})

	t.Run("NextSuperRootTimestampBeyondAllChainHeads", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, _ := createValidSuperNodeSuperRoots(l1Head)
		stubSuperNode.Add(prev)
		stubSuperNode.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  prev.ChainIDs,
			Data:      nil,
		})

		// All steps should be the invalid transition hash as there are no chains with optimistic blocks
		for i := int64(0); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("NextSuperRootTimestampBeyondFirstChainHead", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		stubSuperNode.Add(prev)
		stubSuperNode.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  prev.ChainIDs,
			OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
				eth.ChainIDFromUInt64(2): next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(2)],
			},
			Data: nil,
		})
		// All steps should be the invalid transition hash because the first chain is invalid.
		for i := int64(0); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("NextSuperRootTimestampBeyondSecondChainHead", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, next := createValidSuperNodeSuperRoots(l1Head)
		stubSuperNode.Add(prev)
		stubSuperNode.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  next.ChainIDs,
			OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
				eth.ChainIDFromUInt64(1): next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(1)],
			},
			Data: nil,
		})
		// First step should be valid because we can reach the required block on chain 1
		claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(0)))
		require.NoError(t, err)
		require.NotEqual(t, InvalidTransitionHash, claim, "incorrect claim at index 0")

		// All remaining steps should be the invalid transition hash because the second chain is invalid.
		for i := int64(1); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("PreviousSuperRootTimestampBeyondChainHead", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		stubSuperNode.AddAtTimestamp(prestateTimestamp, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data:      nil,
		})
		stubSuperNode.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: l1Head,
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data:      nil,
		})

		// All steps should be the invalid transition hash.
		for i := int64(0); i < StepsPerTimestamp+1; i++ {
			claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			require.Equalf(t, InvalidTransitionHash, claim, "incorrect claim at index %d", i)
		}
	})

	t.Run("Step0NotInSync", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		expectedSuper := eth.NewSuperV1(poststateTimestamp, eth.ChainIDAndOutput{
			ChainID: eth.ChainIDFromUInt64(1),
			Output:  eth.Bytes32{0xbb},
		})
		response := eth.SuperRootAtTimestampResponse{
			CurrentL1: eth.BlockID{Number: l1Head.Number - 1, Hash: common.Hash{0xaa}},
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
			Data: &eth.SuperRootResponseData{
				VerifiedRequiredL1: eth.BlockID{Number: l1Head.Number + 1, Hash: common.Hash{0xcc}},
				Super:              expectedSuper,
				SuperRoot:          eth.SuperRoot(expectedSuper),
			},
		}
		stubSuperNode.Add(response)
		_, err := provider.Get(context.Background(), types.RootPosition)
		require.ErrorIs(t, err, types2.ErrNotInSync)
	})

	t.Run("PreviousSuperRootNotInSync", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		stubSuperNode.AddAtTimestamp(prestateTimestamp, eth.SuperRootAtTimestampResponse{
			CurrentL1: eth.BlockID{Number: l1Head.Number - 1, Hash: common.Hash{0xaa}},
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
		})
		_, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(1)))
		require.ErrorIs(t, err, types2.ErrNotInSync)
	})

	t.Run("NextSuperRootNotInSync", func(t *testing.T) {
		provider, stubSuperNode, l1Head := createSuperNodeProvider(t)
		prev, _ := createValidSuperNodeSuperRoots(l1Head)
		// Previous gives an in sync response
		stubSuperNode.Add(prev)
		// But next gives an out of sync response
		stubSuperNode.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: eth.BlockID{Number: l1Head.Number - 1, Hash: common.Hash{0xaa}},
			ChainIDs:  []eth.ChainID{eth.ChainIDFromUInt64(1), eth.ChainIDFromUInt64(2)},
		})
		_, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(1)))
		require.ErrorIs(t, err, types2.ErrNotInSync)
	})
}

func TestSuperNodeProvider_ComputeStep(t *testing.T) {
	t.Run("ErrorWhenTraceIndexTooBig", func(t *testing.T) {
		// Uses a big game depth so the trace index doesn't fit in uint64
		provider := NewSuperNodeTraceProvider(testlog.Logger(t, log.LvlInfo), nil, &stubSuperNodeRootProvider{}, eth.BlockID{}, 65, prestateTimestamp, poststateTimestamp)
		// Left-most position in top game
		_, _, err := provider.ComputeStep(types.RootPosition)
		require.ErrorIs(t, err, ErrIndexTooBig)
	})

	t.Run("FirstTimestampSteps", func(t *testing.T) {
		provider, _, _ := createSuperNodeProvider(t)
		for i := int64(0); i < StepsPerTimestamp-1; i++ {
			timestamp, step, err := provider.ComputeStep(types.NewPosition(gameDepth, big.NewInt(i)))
			require.NoError(t, err)
			// The prestate must be a super root and is on the timestamp boundary.
			// So the first step has the same timestamp and increments step from 0 to 1.
			require.Equalf(t, prestateTimestamp, timestamp, "Incorrect timestamp at trace index %d", i)
			require.Equalf(t, uint64(i+1), step, "Incorrect step at trace index %d", i)
		}
	})

	t.Run("SecondTimestampSteps", func(t *testing.T) {
		provider, _, _ := createSuperNodeProvider(t)
		for i := int64(-1); i < StepsPerTimestamp-1; i++ {
			traceIndex := StepsPerTimestamp + i
			timestamp, step, err := provider.ComputeStep(types.NewPosition(gameDepth, big.NewInt(traceIndex)))
			require.NoError(t, err)
			// We should now be iterating through the steps of the second timestamp - 1s after the prestate
			require.Equalf(t, prestateTimestamp+1, timestamp, "Incorrect timestamp at trace index %d", traceIndex)
			require.Equalf(t, uint64(i+1), step, "Incorrect step at trace index %d", traceIndex)
		}
	})

	t.Run("LimitToPoststateTimestamp", func(t *testing.T) {
		provider, _, _ := createSuperNodeProvider(t)
		timestamp, step, err := provider.ComputeStep(types.RootPosition)
		require.NoError(t, err)
		require.Equal(t, poststateTimestamp, timestamp, "Incorrect timestamp at root position")
		require.Equal(t, uint64(0), step, "Incorrect step at trace index at root position")
	})

	t.Run("StepShouldLoopBackToZero", func(t *testing.T) {
		provider, _, _ := createSuperNodeProvider(t)
		prevTimestamp := prestateTimestamp
		prevStep := uint64(0) // Absolute prestate is always on a timestamp boundary, so step 0
		for traceIndex := int64(0); traceIndex < 5*StepsPerTimestamp; traceIndex++ {
			timestamp, step, err := provider.ComputeStep(types.NewPosition(gameDepth, big.NewInt(traceIndex)))
			require.NoError(t, err)
			if timestamp == prevTimestamp {
				require.Equal(t, prevStep+1, step, "Incorrect step at trace index %d", traceIndex)
			} else {
				require.Equal(t, prevTimestamp+1, timestamp, "Incorrect timestamp at trace index %d", traceIndex)
				require.Zero(t, step, "Incorrect step at trace index %d", traceIndex)
				require.Equal(t, uint64(StepsPerTimestamp-1), prevStep, "Should only loop back to step 0 after the consolidation step")
			}
			prevTimestamp = timestamp
			prevStep = step
		}
	})
}

func TestSuperNodeProvider_GetStepDataReturnsError(t *testing.T) {
	provider, _, _ := createSuperNodeProvider(t)
	_, _, _, err := provider.GetStepData(context.Background(), types.RootPosition)
	require.ErrorIs(t, err, ErrGetStepData)
}

func TestSuperNodeProvider_GetL2BlockNumberChallengeReturnsError(t *testing.T) {
	provider, _, _ := createSuperNodeProvider(t)
	_, err := provider.GetL2BlockNumberChallenge(context.Background())
	require.ErrorIs(t, err, types.ErrL2BlockNumberValid)
}

func createSuperNodeProvider(t *testing.T) (*SuperNodeTraceProvider, *stubSuperNodeRootProvider, eth.BlockID) {
	logger := testlog.Logger(t, log.LvlInfo)
	l1Head := eth.BlockID{Number: 23542, Hash: common.Hash{0xab, 0xcd}}
	stubSuperNode := &stubSuperNodeRootProvider{
		rootsByTimestamp: make(map[uint64]eth.SuperRootAtTimestampResponse),
	}
	provider := NewSuperNodeTraceProvider(logger, nil, stubSuperNode, l1Head, gameDepth, prestateTimestamp, poststateTimestamp)
	return provider, stubSuperNode, l1Head
}

func toOutputResponse(output *eth.OutputV0) *eth.OutputResponse {
	return &eth.OutputResponse{
		Version:    output.Version(),
		OutputRoot: eth.OutputRoot(output),
		BlockRef: eth.L2BlockRef{
			Hash: output.BlockHash,
		},
		WithdrawalStorageRoot: common.Hash(output.MessagePasserStorageRoot),
		StateRoot:             common.Hash(output.StateRoot),
	}
}

func createValidSuperNodeSuperRoots(l1Head eth.BlockID) (eth.SuperRootAtTimestampResponse, eth.SuperRootAtTimestampResponse) {
	rng := rand.New(rand.NewSource(1))
	outputA1 := testutils.RandomOutputV0(rng)
	outputA2 := testutils.RandomOutputV0(rng)
	outputB1 := testutils.RandomOutputV0(rng)
	outputB2 := testutils.RandomOutputV0(rng)
	chainID1 := eth.ChainIDFromUInt64(1)
	chainID2 := eth.ChainIDFromUInt64(2)
	prevSuper := eth.NewSuperV1(
		prestateTimestamp,
		eth.ChainIDAndOutput{ChainID: chainID1, Output: eth.OutputRoot(outputA1)},
		eth.ChainIDAndOutput{ChainID: chainID2, Output: eth.OutputRoot(outputB1)})
	nextSuper := eth.NewSuperV1(prestateTimestamp+1,
		eth.ChainIDAndOutput{ChainID: chainID1, Output: eth.OutputRoot(outputA2)},
		eth.ChainIDAndOutput{ChainID: chainID2, Output: eth.OutputRoot(outputB2)})

	prevResponse := eth.SuperRootAtTimestampResponse{
		CurrentL1: l1Head,
		ChainIDs:  []eth.ChainID{chainID1, chainID2},
		OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
			chainID1: {
				Output:     toOutputResponse(outputA1),
				RequiredL1: l1Head,
			},
			chainID2: {
				Output:     toOutputResponse(outputB1),
				RequiredL1: l1Head,
			},
		},
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: l1Head,
			Super:              prevSuper,
			SuperRoot:          eth.SuperRoot(prevSuper),
		},
	}
	nextResponse := eth.SuperRootAtTimestampResponse{
		CurrentL1: l1Head,
		ChainIDs:  []eth.ChainID{chainID1, chainID2},
		OptimisticAtTimestamp: map[eth.ChainID]eth.OutputWithRequiredL1{
			chainID1: {
				Output:     toOutputResponse(outputA2),
				RequiredL1: l1Head,
			},
			chainID2: {
				Output:     toOutputResponse(outputB2),
				RequiredL1: l1Head,
			},
		},
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: l1Head,
			Super:              nextSuper,
			SuperRoot:          eth.SuperRoot(nextSuper),
		},
	}
	return prevResponse, nextResponse
}

func expectSuperNodeValidTransition(t *testing.T, provider *SuperNodeTraceProvider, prev eth.SuperRootAtTimestampResponse, next eth.SuperRootAtTimestampResponse) {
	chain1OptimisticBlock := interopTypes.OptimisticBlock{
		BlockHash:  next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(1)].Output.BlockRef.Hash,
		OutputRoot: next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(1)].Output.OutputRoot,
	}
	chain2OptimisticBlock := interopTypes.OptimisticBlock{
		BlockHash:  next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(2)].Output.BlockRef.Hash,
		OutputRoot: next.OptimisticAtTimestamp[eth.ChainIDFromUInt64(2)].Output.OutputRoot,
	}
	expectedFirstStep := &interopTypes.TransitionState{
		SuperRoot:       prev.Data.Super.Marshal(),
		PendingProgress: []interopTypes.OptimisticBlock{chain1OptimisticBlock},
		Step:            1,
	}
	claim, err := provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(0)))
	require.NoError(t, err)
	require.Equal(t, expectedFirstStep.Hash(), claim)

	expectedSecondStep := &interopTypes.TransitionState{
		SuperRoot:       prev.Data.Super.Marshal(),
		PendingProgress: []interopTypes.OptimisticBlock{chain1OptimisticBlock, chain2OptimisticBlock},
		Step:            2,
	}
	claim, err = provider.Get(context.Background(), types.NewPosition(gameDepth, big.NewInt(1)))
	require.NoError(t, err)
	require.Equal(t, expectedSecondStep.Hash(), claim)

	for step := uint64(3); step < StepsPerTimestamp; step++ {
		expectedPaddingStep := &interopTypes.TransitionState{
			SuperRoot:       prev.Data.Super.Marshal(),
			PendingProgress: []interopTypes.OptimisticBlock{chain1OptimisticBlock, chain2OptimisticBlock},
			Step:            step,
		}
		claim, err = provider.Get(context.Background(), types.NewPosition(gameDepth, new(big.Int).SetUint64(step-1)))
		require.NoError(t, err)
		require.Equalf(t, expectedPaddingStep.Hash(), claim, "incorrect hash at step %v", step)
	}
}

type stubSuperNodeRootProvider struct {
	rootsByTimestamp map[uint64]eth.SuperRootAtTimestampResponse
}

func (s *stubSuperNodeRootProvider) Add(root eth.SuperRootAtTimestampResponse) {
	superV1 := root.Data.Super.(*eth.SuperV1)
	s.AddAtTimestamp(superV1.Timestamp, root)
}

func (s *stubSuperNodeRootProvider) AddAtTimestamp(timestamp uint64, root eth.SuperRootAtTimestampResponse) {
	if s.rootsByTimestamp == nil {
		s.rootsByTimestamp = make(map[uint64]eth.SuperRootAtTimestampResponse)
	}
	s.rootsByTimestamp[timestamp] = root
}

func (s *stubSuperNodeRootProvider) SuperRootAtTimestamp(_ context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	root, ok := s.rootsByTimestamp[timestamp]
	if !ok {
		// This is not the not found response - the test just didn't configure a response, so return a generic error
		return eth.SuperRootAtTimestampResponse{}, fmt.Errorf("wowsers, now response for timestamp %v", timestamp)
	}
	return root, nil
}
