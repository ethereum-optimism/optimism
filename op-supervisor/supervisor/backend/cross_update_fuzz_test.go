package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Missing:
// 10 - AnchorEvent
// 12 - RewindL1Event

// Done:
// 1 - UpdateCrossUnsafeRequestEvent
// 2 - CrossUnsafeUpdateEvent
// 3 - UpdateCrossSafeRequestEvent
// 4 - LocalSafeUpdateEvent
// 5 - CrossSafeUpdateEvent
// 6 - LocalUnsafeUpdateEvent
// 7 - ChainProcessEvent
// 8 - LocalUnsafeReceivedEvent
// 9 - FinalizedL1UpdateEvent
// 10 - FinalizedL2UpdateEvent
// 11 - InvalidateLocalSafeEvent
// 12 - ChainRewoundEvent
// 13 - UpdateLocalSafeFailedEvent
// 14 - LocalDerivedEvent
// 15 - LocalDerivedOriginUpdateEvent
// 16 - ReplaceBlockEvent
// 17 - FinalizedL1RequestEvent

type SafetyHeads struct {
	// These are block numbers on the chain
	localSafe   types.DerivedIDPair
	localUnsafe eth.BlockID
	crossSafe   types.DerivedIDPair
	crossUnsafe eth.BlockID
}

type State struct {
	chainHeads map[eth.ChainID]*SafetyHeads
}

var chainParams = RandomChainParams{
	chainCount: 3,
	minLength:  10,
	maxLength:  30,

	sameTimestampFrequency: 80,
	dependencyChance:       50,
}

func FuzzUpdateCrossUnsafeSucceeds(f *testing.F) {

	f.Add(int64(-62))

	f.Fuzz(func(t *testing.T, seed int64) {
		t.Logf("Seed %d", seed)
		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)

		t.Run("UpdateCrossUnsafeRequestEvent Success", func(t *testing.T) {
			ChainsInit(t, b, ex, randomChain)

			// Ensure the invariants hold in the intiial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			// Enqueue the UpdateCrossUnsafeRequestEvent
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.UpdateCrossUnsafeRequestEvent{},
				EmitPriority: event.High,
			})

			// Drain the event until it is processed
			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.UpdateCrossUnsafeRequestEvent{}
				}, false))
			t.Log("UpdateCrossUnsafeRequestEvent processed")

			t.Logf("Final State with Seed %d", seed)
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that the state has changed as expected - Liveness property
			AssertCrossUnsafeHeadUpdate(t, randomChain, preState, posState, eth.ChainIDFromUInt64(0))
		})

		t.Run("Cross-unsafe reaches local-unsafe", func(t *testing.T) {
			// Ensure the invariants hold in the intiial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			// Drain the event until it is processed
			require.NoError(t, ex.Drain())
			t.Log("All Events processed")

			t.Logf("Final State with Seed %d", seed)
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that all cross-unsafe heads got equal to respective local-unsafe heads - Liveness property
			for _, chain := range randomChain.chainIDs {
				preLocalUnsafe := preState.chainHeads[chain].localUnsafe
				posCrossUnsafe := posState.chainHeads[chain].crossUnsafe

				require.Equal(t, posCrossUnsafe, preLocalUnsafe, "Cross Unsafe head for chain %d did not reach local-unsafe", chain)
			}
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzUpdateCrossUnsafeFails(f *testing.F) {

	f.Add(int64(63))

	f.Fuzz(func(t *testing.T, seed int64) {
		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)

		t.Run("UpdateCrossUnsafeRequestEvent Fails", func(t *testing.T) {
			// Invalidate a block
			crossUnsafeCandidate := GetCrossUnsafeCandidate(randomChain)
			if crossUnsafeCandidate == nil {
				t.Skip()
			}
			InvalidateBlock(t, &randomChain, crossUnsafeCandidate)
			ChainsInit(t, b, ex, randomChain)

			// Ensure the invariants hold in the intiial state
			t.Logf("Initial State with Seed %d", seed)
			preState := AssertInvariants(t, b, randomChain)

			// Enqueue the UpdateCrossUnsafeRequestEvent
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.UpdateCrossUnsafeRequestEvent{},
				EmitPriority: event.High,
			})

			// Drain the event until it is processed
			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.UpdateCrossUnsafeRequestEvent{}
				}, false))
			t.Log("UpdateCrossUnsafeRequestEvent processed")

			t.Logf("Final State with Seed %d", seed)
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that the state has changed as expected - Liveness property
			AssertCrossUnsafeHeadUpdate(t, randomChain, preState, posState, crossUnsafeCandidate.chain)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})

}

func FuzzUpdateCrossSafeSucceeds(f *testing.F) {

	f.Add(int64(-832))

	f.Fuzz(func(t *testing.T, seed int64) {
		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)

		t.Run("UpdateCrossSafeRequestEvent Succeeds", func(t *testing.T) {
			ChainsInit(t, b, ex, randomChain)

			// Ensure the invariants hold in the intiial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.UpdateCrossSafeRequestEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.UpdateCrossSafeRequestEvent{}
				}, false))

			t.Log("UpdateCrossSafeRequestEvent processed")

			t.Logf("Final State with Seed %d", seed)
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that the state has changed as expected - Liveness property
			AssertCrossSafeHeadUpdate(t, randomChain, preState, posState, eth.ChainIDFromUInt64(0))
		})

		t.Run("Cross-Safe reaches Local-Safe", func(t *testing.T) {
			// Ensure the invariants hold in the intiial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			// Drain the event until it is processed
			require.NoError(t, ex.Drain())
			t.Log("All Events processed")

			t.Logf("Final State with Seed %d", seed)
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that all cross-unsafe heads got equal to respective local-unsafe heads - Liveness property
			for _, chain := range randomChain.chainIDs {
				preLocalSafe := preState.chainHeads[chain].localSafe
				posCrossSafe := posState.chainHeads[chain].crossSafe

				require.Equal(t, preLocalSafe, posCrossSafe, "Cross Safe head for chain %d did not reach Local Safe", chain)
			}
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzUpdateCrossSafeFails(f *testing.F) {

	f.Add(int64(1052))

	f.Fuzz(func(t *testing.T, seed int64) {
		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)

		t.Run("UpdateCrossSafeRequestEvent Fails", func(t *testing.T) {
			crossSafeCandidate := GetCrossSafeCandidate(randomChain)
			if crossSafeCandidate == nil {
				t.Skip()
			}
			InvalidateBlock(t, &randomChain, crossSafeCandidate)
			ChainsInit(t, b, ex, randomChain)

			// Ensure the invariants hold in the intiial state
			t.Logf("Initial State with seed %d", seed)
			preState := AssertInvariants(t, b, randomChain)

			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.UpdateCrossSafeRequestEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.UpdateCrossSafeRequestEvent{}
				}, false))

			t.Log("UpdateCrossSafeRequestEvent processed")

			t.Log("Final State")
			// Assert the invariants hold after handling the event - Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Check that the state has changed as expected - Liveness property
			AssertCrossSafeHeadUpdate(t, randomChain, preState, posState, crossSafeCandidate.chain)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzUpdateLocalSafeInvariants(f *testing.F) {

	f.Add(int64(94)) // Add initial values for fuzzing

	f.Fuzz(func(t *testing.T, seed int64) {

		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)
		ChainsInit(t, b, ex, randomChain)

		t.Run("LocalSafeUpdateEvent Event - LocalSafe equal to unsafe chain", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			chainA := randomChain.chainIDs[0]
			localSafeHead := preState.chainHeads[chainA].localSafe

			newBlock := randomChain.chainBlocks[chainA][localSafeHead.Derived.Number]
			newSource := types.BlockSealFromRef(randomChain.l1SourceMap[ChainBlock{chain: chainA, block: newBlock}])
			newLocalSafe := types.DerivedBlockSealPair{
				Derived: types.BlockSealFromRef(newBlock.BlockRef()),
				Source:  newSource,
			}

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.LocalSafeUpdateEvent{
					ChainID:      chainA,
					NewLocalSafe: newLocalSafe,
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalSafeUpdateEvent{
						ChainID:      chainA,
						NewLocalSafe: newLocalSafe,
					}
				}, false))
			t.Log("LocalSafeUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("LocalSafeUpdateEvent Event - LocalSafe not equal to unsafe chain", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Logf("Initial State with seed %d", seed)
			preState := AssertInvariants(t, b, randomChain)

			chainA := randomChain.chainIDs[0]
			localSafeHead := preState.chainHeads[chainA].localSafe

			// WARN: We assume the first block to be always equal to the unsafe-chain
			if localSafeHead.Derived.Number == 0 {
				t.Skip()
			}

			newBlock := randomChain.chainBlocks[chainA][localSafeHead.Derived.Number]
			newSource := types.BlockSealFromRef(randomChain.l1SourceMap[ChainBlock{chain: chainA, block: newBlock}])
			hashDerived := testutils.RandomHash(randomChain.randomGenerator)
			// Ensure the hash is different from the unsafe chain
			if hashDerived == newBlock.Hash {
				t.Skip()
			}
			newBlock.Hash = hashDerived
			newLocalSafe := types.DerivedBlockSealPair{
				Derived: types.BlockSealFromRef(newBlock.BlockRef()),
				Source:  newSource,
			}

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.LocalSafeUpdateEvent{
					ChainID:      chainA,
					NewLocalSafe: newLocalSafe,
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalSafeUpdateEvent{
						ChainID:      chainA,
						NewLocalSafe: newLocalSafe,
					}
				}, false))
			t.Log("LocalSafeUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			// Liveness property
			require.Equal(t, posState.chainHeads[chainA].localSafe.Derived.Number, posState.chainHeads[chainA].localUnsafe.Number+1, "Local safe head should be equal to local unsafe head + 1")
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzLocalDerivedEventInvariants(f *testing.F) {

	f.Add(int64(-383))

	f.Fuzz(func(t *testing.T, seed int64) {

		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)
		ChainsInit(t, b, ex, randomChain)

		t.Run("LocalDerivedEvent Event Succeeds", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			chainA := randomChain.chainIDs[0]
			chainLength := len(randomChain.chainBlocks[chainA])
			preLocalSafeHead := preState.chainHeads[chainA].localSafe
			localSafetoUpdate := preLocalSafeHead.Derived.Number + 1

			var derived types.DerivedBlockRefPair

			if localSafetoUpdate < uint64(chainLength) {
				nextSafeBlock := randomChain.chainBlocks[chainA][localSafetoUpdate]
				nextSafeBlockSource := randomChain.l1SourceMap[ChainBlock{chain: chainA, block: nextSafeBlock}]
				if preLocalSafeHead.Source.Number < nextSafeBlockSource.Number {
					derived = types.DerivedBlockRefPair{
						Derived: randomChain.chainBlocks[chainA][preLocalSafeHead.Derived.Number].BlockRef(),
						Source:  randomChain.l1Source[preLocalSafeHead.Source.Number+1],
					}
				} else {
					derived = types.DerivedBlockRefPair{
						Derived: nextSafeBlock.BlockRef(),
						Source:  nextSafeBlockSource,
					}
				}
			} else {
				r := randomChain.randomGenerator
				hashDerived := testutils.RandomHash(r)
				source := randomChain.l1Source[preLocalSafeHead.Source.Number]
				derived = types.DerivedBlockRefPair{
					Derived: eth.BlockRef{
						Hash:       hashDerived,
						Number:     localSafetoUpdate,
						ParentHash: preLocalSafeHead.Derived.Hash,
						Time:       uint64(time.Now().Unix()),
					},
					Source: source,
				}
			}

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.LocalDerivedEvent{
					ChainID: chainA,
					Derived: derived,
					NodeID:  "test-node",
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalDerivedEvent{
						ChainID: chainA,
						Derived: derived,
						NodeID:  "test-node",
					}
				}, false))
			t.Log("LocalDerivedEvent processed")

			t.Logf("Final State with seed %d", seed)
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)

			// Liveness property
			posLocalSafeHead := posState.chainHeads[chainA].localSafe
			if preLocalSafeHead.Derived.Number < posLocalSafeHead.Derived.Number {
				require.Equal(t, localSafetoUpdate, posLocalSafeHead.Derived.Number)
			} else {
				require.Equal(t, preLocalSafeHead.Derived.Number, posLocalSafeHead.Derived.Number)
				require.Equal(t, posLocalSafeHead.Source.Number, preLocalSafeHead.Source.Number+1)
			}
		})

		t.Run("LocalDerivedEvent Event Fails", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			chainA := randomChain.chainIDs[0]
			chainLength := len(randomChain.chainBlocks[chainA])
			preLocalSafeHead := preState.chainHeads[chainA].localSafe
			nextLocalSafe := preLocalSafeHead.Derived.Number + 1

			localSafeToUpdate := uint64(randomChain.randomGenerator.Int63n(int64(chainLength + 1)))
			var derived types.DerivedBlockRefPair

			if localSafeToUpdate < uint64(chainLength) {
				nextSafeBlock := randomChain.chainBlocks[chainA][localSafeToUpdate]
				nextSafeBlockSource := randomChain.l1SourceMap[ChainBlock{chain: chainA, block: nextSafeBlock}]
				if localSafeToUpdate == nextLocalSafe && preLocalSafeHead.Source.Number == nextSafeBlockSource.Number {
					t.Skip("Local safe to update is the same as next safe block, skipping")
				}
				derived = types.DerivedBlockRefPair{
					Derived: nextSafeBlock.BlockRef(),
					Source:  nextSafeBlockSource,
				}
			} else {
				r := randomChain.randomGenerator
				hashDerived := testutils.RandomHash(r)
				source := randomChain.l1Source[preLocalSafeHead.Source.Number+1]
				derived = types.DerivedBlockRefPair{
					Derived: eth.BlockRef{
						Hash:       hashDerived,
						Number:     localSafeToUpdate,
						ParentHash: preLocalSafeHead.Derived.Hash,
						Time:       uint64(time.Now().Unix()),
					},
					Source: source,
				}
			}

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.LocalDerivedEvent{
					ChainID: chainA,
					Derived: derived,
					NodeID:  "test-node",
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalDerivedEvent{
						ChainID: chainA,
						Derived: derived,
						NodeID:  "test-node",
					}
				}, false))
			t.Log("LocalDerivedEvent processed")

			t.Logf("Final State with seed %d", seed)
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzReplaceBlockEventInvariants(f *testing.F) {

	f.Add(int64(536))

	f.Fuzz(func(t *testing.T, seed int64) {

		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)
		ChainsInit(t, b, ex, randomChain)

		t.Run("ReplaceBlockEvent Event", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)

			chainA := randomChain.chainIDs[0]
			localSafe := preState.chainHeads[chainA].localSafe.Derived.Number
			crossSafe := preState.chainHeads[chainA].crossSafe
			genesisSource := randomChain.l1SourceMap[ChainBlock{chain: chainA, block: randomChain.chainBlocks[chainA][0]}]
			if crossSafe.Derived.Number == localSafe || crossSafe.Source.Number == genesisSource.Number {
				t.Skip()
			}
			crossSafeHeadCandidate := crossSafe.Derived.Number + 1
			block := randomChain.chainBlocks[chainA][crossSafeHeadCandidate]
			source := randomChain.l1SourceMap[ChainBlock{chain: chainA, block: block}]

			invalidated := types.DerivedBlockRefPair{
				Derived: block.BlockRef(),
				Source:  source,
			}
			b.chainDBs.InvalidateLocalSafe(chainA, invalidated)

			t.Logf("State after Chain %d Block Number %d invalidation", chainA, crossSafeHeadCandidate)
			AssertInvariants(t, b, randomChain)

			_, err := b.LocalSafe(context.Background(), chainA)
			require.Equal(t, err, types.ErrAwaitReplacementBlock)

			newHash := testutils.RandomHash(randomChain.randomGenerator)
			replacementBlock := eth.BlockRef{
				Hash:       newHash,
				Number:     invalidated.Derived.Number,
				ParentHash: invalidated.Derived.ParentHash,
				Time:       uint64(time.Now().Unix()),
			}
			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.ReplaceBlockEvent{
					ChainID: chainA,
					Replacement: types.BlockReplacement{
						Replacement: replacementBlock,
						Invalidated: block.Hash,
					},
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.ReplaceBlockEvent{
						ChainID: chainA,
						Replacement: types.BlockReplacement{
							Replacement: replacementBlock,
							Invalidated: block.Hash,
						},
					}
				}, false))

			t.Log("ReplaceBlockEvent processed")

			t.Logf("Final State with seed %d", seed)
			AssertInvariants(t, b, randomChain)

			newLocalSafe, _ := b.LocalSafe(context.Background(), chainA)
			require.Equal(t, crossSafeHeadCandidate, newLocalSafe.Derived.Number)
			require.Equal(t, newLocalSafe.Derived.Hash, newHash)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})
}

func FuzzChainProcessEventInvariants(f *testing.F) {

	f.Add(int64(28))

	f.Fuzz(func(t *testing.T, seed int64) {

		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)
		ChainsInit(t, b, ex, randomChain)

		chainA := randomChain.chainIDs[0]
		srcChainA := randomChain.chainSources[chainA]

		t.Run("ChainProcessEvent Event Succeeds", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Log("Initial State")
			preState := AssertInvariants(t, b, randomChain)
			target := preState.chainHeads[chainA].localUnsafe.Number + 1

			newHash := testutils.RandomHash(randomChain.randomGenerator)

			// TODO: Add L1Origin and SequenceNumber fields?
			newLocalUnsafe := eth.L2BlockRef{
				Hash:       newHash,
				Number:     target,
				ParentHash: randomChain.chainBlocks[chainA][target-1].Hash,
				Time:       uint64(time.Now().Unix()),
			}

			t.Logf("Chain A block %d: %s\t Timestamp:%d", target, newLocalUnsafe.Hash.Hex(), newLocalUnsafe.Time)

			srcChainA.ExpectL2BlockRefByNumber(target, newLocalUnsafe, nil)
			srcChainA.ExpectFetchReceipts(newLocalUnsafe.Hash, nil, nil)

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.ChainProcessEvent{
					ChainID: chainA,
					Target:  target,
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.ChainProcessEvent{
						ChainID: chainA,
						Target:  target,
					}
				}, false))
			t.Log("ChainProcessEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			// Liveness property
			require.Equal(t, posState.chainHeads[chainA].localUnsafe.Number, target)
		})

		t.Run("ChainProcessEvent Event Fails", func(t *testing.T) {
			// Ensure the invariants hold in the initial state
			t.Logf("Initial State with seed %d", seed)
			preState := AssertInvariants(t, b, randomChain)
			chainABlocks := randomChain.chainBlocks[chainA]
			nextLocalUnsafe := preState.chainHeads[chainA].localUnsafe.Number + 1
			target := uint64(randomChain.randomGenerator.Int63n(int64(len(chainABlocks))))

			if target == nextLocalUnsafe {
				//This would be the right target to update the localUnsafe
				t.Skip()
			}

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.ChainProcessEvent{
					ChainID: chainA,
					Target:  target,
				},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.ChainProcessEvent{
						ChainID: chainA,
						Target:  target,
					}
				}, false))
			t.Log("ChainProcessEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})

}

// FuzzEventsPreserveState tests that various events preserve the state of the backend
func FuzzEventsPreserveState(f *testing.F) {

	f.Add(int64(30))

	f.Fuzz(func(t *testing.T, seed int64) {

		randomChain := chainParams.MakeRandomChain(seed)
		ex, b := ExecutorBackendInit(t, randomChain)
		ChainsInit(t, b, ex, randomChain)

		t.Log("Initial State")
		preState := AssertInvariants(t, b, randomChain)

		t.Run("LocalUnsafeUpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.LocalUnsafeUpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalUnsafeUpdateEvent{}
				}, false))
			t.Log("LocalUnsafeUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("LocalUnsafeReceivedEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.LocalUnsafeReceivedEvent{ChainID: randomChain.chainIDs[0]},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalUnsafeReceivedEvent{ChainID: randomChain.chainIDs[0]}
				}, false))
			t.Log("LocalUnsafeReceivedEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)

		})

		t.Run("CrossUnsafeUpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.CrossUnsafeUpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.CrossUnsafeUpdateEvent{}
				}, false))
			t.Log("CrossUnsafeUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("CrossSafeUpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.CrossSafeUpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.CrossSafeUpdateEvent{}
				}, false))
			t.Log("CrossSafeUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("FinalizedL1UpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.FinalizedL1UpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.FinalizedL1UpdateEvent{}
				}, false))
			t.Log("FinalizedL1UpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("FinalizedL2UpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.FinalizedL2UpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.FinalizedL2UpdateEvent{}
				}, false))
			t.Log("FinalizedL2UpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("InvalidateLocalSafeEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.InvalidateLocalSafeEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.InvalidateLocalSafeEvent{}
				}, false))
			t.Log("InvalidateLocalSafeEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("UpdateLocalSafeFailedEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.UpdateLocalSafeFailedEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.UpdateLocalSafeFailedEvent{}
				}, false))
			t.Log("UpdateLocalSafeFailedEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("LocalDerivedOriginUpdateEvent", func(t *testing.T) {
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.LocalDerivedOriginUpdateEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.LocalDerivedOriginUpdateEvent{}
				}, false))
			t.Log("LocalDerivedOriginUpdateEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		t.Run("FinalizedL1RequestEvent", func(t *testing.T) {
			// Handling this event changes the finalized database which is not part of the invariants
			ex.Enqueue(event.AnnotatedEvent{
				Ctx:          context.Background(),
				Event:        superevents.FinalizedL1RequestEvent{},
				EmitPriority: event.High,
			})

			require.NoError(t, ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.FinalizedL1RequestEvent{}
				}, false))
			t.Log("FinalizedL1RequestEvent processed")

			t.Log("Final State")
			// Safety properties
			posState := AssertInvariants(t, b, randomChain)
			AssertStateNotChange(t, randomChain, preState, posState)
		})

		err := b.Stop(context.Background())
		require.NoError(t, err)
		t.Log("stopped!")
	})

}

func fuzzConfigSet(t *testing.T, randomChain RandomChain) depset.FullConfigSetMerged {
	size := chainParams.chainCount
	staticDepSet := make(map[eth.ChainID]*depset.StaticConfigDependency, size)
	staticRollupCfgSet := make(map[eth.ChainID]*depset.StaticRollupConfig, size)
	zero := uint64(0)
	for _, chain := range randomChain.chainIDs {
		staticDepSet[chain] = &depset.StaticConfigDependency{}
		staticRollupCfgSet[chain] = &depset.StaticRollupConfig{
			InteropTime: &zero,
			BlockTime:   2,
		}
	}
	depSet, err := depset.NewStaticConfigDependencySet(staticDepSet)
	require.NoError(t, err)
	rollupCfgSet := depset.NewStaticRollupConfigSet(staticRollupCfgSet)
	fullCfgSet, err := depset.NewFullConfigSetMerged(rollupCfgSet, depSet)
	require.NoError(t, err)
	return fullCfgSet
}

func ExecutorBackendInit(t *testing.T, randomChain RandomChain) (ex *event.GlobalSyncExec, b *SupervisorBackend) {
	logger := testlog.Logger(t, log.LvlInfo)
	m := metrics.NoopMetrics
	dataDir := t.TempDir()
	fullCfgSet := fuzzConfigSet(t, randomChain)
	rollupCfgSet := fullCfgSet.RollupConfigSet.(depset.StaticRollupConfigSet)

	for _, chain := range randomChain.chainIDs {
		anchor := randomChain.chainBlocks[chain][0]
		rollupCfgSet[chain].Genesis = depset.Genesis{
			L1: types.BlockSealFromRef(randomChain.l1SourceMap[ChainBlock{chain: chain, block: anchor}]),
			L2: types.BlockSealFromRef(anchor.BlockRef()),
		}
	}
	cfg := &config.Config{
		Version:               "test",
		FullConfigSetSource:   fullCfgSet,
		SynchronousProcessors: true,
		MockRun:               false,
		SyncSources:           &syncnode.CLISyncNodes{},
		Datadir:               dataDir,
	}

	ex = event.NewGlobalSynchronous(context.Background())
	b, err := NewSupervisorBackend(context.Background(), logger, m, cfg, ex)
	require.NoError(t, err)
	t.Log("initialized!")

	l1Src := &testutils.MockL1Source{}
	b.AttachL1Source(l1Src)

	for _, chain := range randomChain.chainIDs {
		srcChain := randomChain.chainSources[chain]
		require.NoError(t, b.AttachProcessorSource(chain, srcChain))
	}

	err = b.Start(context.Background())
	require.NoError(t, err)
	t.Log("started!")

	return ex, b
}

func ChainsInit(t *testing.T, b *SupervisorBackend, ex *event.GlobalSyncExec, randomChain RandomChain) {
	GenerateReceiptsFromLogs(&randomChain)

	for _, chain := range randomChain.chainIDs {
		chainHeads := randomChain.chainHeads[chain]
		localUnsafe := randomChain.chainBlocks[chain][len(randomChain.chainBlocks[chain])-1].Number
		crossUnsafe := randomChain.chainBlocks[chain][chainHeads.crossUnsafe]
		localSafe := randomChain.chainBlocks[chain][chainHeads.localSafe].Number
		crossSafe := randomChain.chainBlocks[chain][chainHeads.crossSafe]

		t.Logf("Chain %d LocalUnsafe: %d CrossUnsafe: %d LocalSafe: %d CrossSafe: %d", chain, localUnsafe, crossUnsafe.Number, localSafe, crossSafe.Number)

		for i := 1; i <= int(crossSafe.Number); i++ {
			previous := randomChain.chainBlocks[chain][i-1]
			previousSource := randomChain.l1SourceMap[ChainBlock{chain: chain, block: previous}]
			block := randomChain.chainBlocks[chain][i]
			source := randomChain.l1SourceMap[ChainBlock{chain: chain, block: block}]

			for j := previousSource.Number + 1; j <= source.Number; j++ {
				crossSafe := superevents.LocalDerivedEvent{
					ChainID: chain,
					Derived: types.DerivedBlockRefPair{
						Derived: previous.BlockRef(),
						Source:  randomChain.l1Source[j],
					},
					NodeID: "test-node",
				}
				b.emitter.Emit(context.Background(), crossSafe)
			}
			crossSafe := superevents.LocalDerivedEvent{
				ChainID: chain,
				Derived: types.DerivedBlockRefPair{
					Derived: block.BlockRef(),
					Source:  source,
				},
				NodeID: "test-node",
			}
			b.emitter.Emit(context.Background(), crossSafe)
		}
	}
	ex.Drain()

	for _, chain := range randomChain.chainIDs {
		chainHeads := randomChain.chainHeads[chain]
		crossSafe := randomChain.chainBlocks[chain][chainHeads.crossSafe].Number
		localSafe := randomChain.chainBlocks[chain][chainHeads.localSafe].Number

		for i := int(crossSafe) + 1; i < len(randomChain.chainBlocks[chain]); i++ {
			block := randomChain.chainBlocks[chain][i]

			ex.Enqueue(event.AnnotatedEvent{
				Ctx: context.Background(),
				Event: superevents.ChainProcessEvent{
					ChainID: chain,
					Target:  block.Number,
				},
				EmitPriority: event.High,
			})

			ex.DrainUntil(
				func(ev event.Event) bool {
					return ev == superevents.ChainProcessEvent{
						ChainID: chain,
						Target:  block.Number}
				}, false)

			if block.Number <= localSafe {
				previous := randomChain.chainBlocks[chain][i-1]
				previousSource := randomChain.l1SourceMap[ChainBlock{chain: chain, block: previous}]
				source := randomChain.l1SourceMap[ChainBlock{chain: chain, block: block}]

				for j := previousSource.Number + 1; j <= source.Number; j++ {
					localSafe := superevents.LocalDerivedEvent{
						ChainID: chain,
						Derived: types.DerivedBlockRefPair{
							Derived: previous.BlockRef(),
							Source:  randomChain.l1Source[j],
						},
						NodeID: "test-node",
					}
					ex.Enqueue(event.AnnotatedEvent{
						Ctx:          context.Background(),
						Event:        localSafe,
						EmitPriority: event.High,
					})
					ex.DrainUntil(
						func(ev event.Event) bool {
							return ev == localSafe
						}, false)
				}
				localSafe := superevents.LocalDerivedEvent{
					ChainID: chain,
					Derived: types.DerivedBlockRefPair{
						Derived: block.BlockRef(),
						Source:  randomChain.l1SourceMap[ChainBlock{chain: chain, block: block}],
					},
					NodeID: "test-node",
				}
				ex.Enqueue(event.AnnotatedEvent{
					Ctx:          context.Background(),
					Event:        localSafe,
					EmitPriority: event.High,
				})
				ex.DrainUntil(
					func(ev event.Event) bool {
						return ev == localSafe
					}, false)
			}
		}
		localUnsafe, _ := b.chainDBs.LocalUnsafe(chain)
		crossUnsafe := types.BlockSealFromRef(randomChain.chainBlocks[chain][chainHeads.crossUnsafe].BlockRef())
		if crossUnsafe.Number > localUnsafe.Number {
			crossUnsafe = localUnsafe
		}
		err := b.chainDBs.UpdateCrossUnsafe(chain, crossUnsafe)
		require.NoError(t, err)
	}
}

func CrossUnsafe_LE_LocalUnsafe(t *testing.T, b *SupervisorBackend, chain eth.ChainID, state State) {

	localUnsafe, err := b.LocalUnsafe(context.Background(), chain)
	require.NoError(t, err)
	crossUnsafe, err := b.CrossUnsafe(context.Background(), chain)
	require.NoError(t, err)

	t.Logf("\t- Cross Unsafe head %d <= Local Unsafe head %d", crossUnsafe.Number, localUnsafe.Number)
	state.chainHeads[chain].crossUnsafe = crossUnsafe
	state.chainHeads[chain].localUnsafe = localUnsafe
	require.LessOrEqual(t, crossUnsafe.Number, localUnsafe.Number, "Chain %d: Cross Unsafe head %d is not less or equal than Local Unsafe head %d", chain, crossUnsafe.Number, localUnsafe.Number)
}

func CrossSafe_LE_LocalSafe(t *testing.T, b *SupervisorBackend, chain eth.ChainID, state State) {

	localSafe, err := b.LocalSafe(context.Background(), chain)
	crossSafe, _ := b.CrossSafe(context.Background(), chain)

	state.chainHeads[chain].crossSafe = crossSafe
	state.chainHeads[chain].localSafe = localSafe

	if err == types.ErrAwaitReplacementBlock {
		t.Logf("\t- Cross Safe head %d <= Local Safe awaiting replacement block", crossSafe.Derived.Number)
		return
	}
	t.Logf("\t- Cross Safe head %d <= Local Safe head %d", crossSafe.Derived.Number, localSafe.Derived.Number)
	require.LessOrEqual(t, crossSafe.Derived.Number, localSafe.Derived.Number, "Chain %d: Cross Safe head %d is not less or equal than Local Safe head %d", chain, crossSafe.Derived.Number, localSafe.Derived.Number)
}

func AssertInvariants(t *testing.T, b *SupervisorBackend, rc RandomChain) (state State) {
	state.chainHeads = make(map[eth.ChainID]*SafetyHeads)
	for _, chain := range rc.chainIDs {
		t.Logf("Chain %d:", chain)
		state.chainHeads[chain] = &SafetyHeads{}

		CrossUnsafe_LE_LocalUnsafe(t, b, chain, state)
		CrossSafe_LE_LocalSafe(t, b, chain, state)
	}
	return state
}

func AssertCrossUnsafeHeadUpdate(t *testing.T, rc RandomChain, preState State, posState State, expectNoUpdate eth.ChainID) {
	crossUnsafeUpdates := 0
	chainsToUpdate := len(rc.chainIDs)
	for _, chain := range rc.chainIDs {
		preCrossUnsafe := preState.chainHeads[chain].crossUnsafe.Number
		preLocalUnsafe := preState.chainHeads[chain].localUnsafe.Number
		posCrossUnsafe := posState.chainHeads[chain].crossUnsafe.Number
		if chain != expectNoUpdate {
			if preCrossUnsafe < preLocalUnsafe {
				// Ensure the cross unsafe head has been updated
				if posCrossUnsafe == preCrossUnsafe+1 {
					crossUnsafeUpdates++
					t.Logf("Cross Unsafe head for chain %d has been updated from %d to %d", chain, preCrossUnsafe, posCrossUnsafe)
				}
			} else {
				chainsToUpdate--
				require.Equal(t, posCrossUnsafe, preCrossUnsafe)
				t.Logf("Cross Unsafe head for chain %d was already equal to Local Unsafe", chain)
			}
		} else {
			chainsToUpdate--
			require.Equal(t, posCrossUnsafe, preCrossUnsafe, "Cross Unsafe head unexpectedly updated for chain %d", chain)
			t.Logf("Cross Unsafe head for chain %d was not updated because the candidate was invalid", chain)
		}
	}

	if chainsToUpdate > 0 && expectNoUpdate == eth.ChainIDFromUInt64(0) {
		// At least one chain must be updated
		require.Greater(t, crossUnsafeUpdates, 0)
	}
}

func AssertCrossSafeHeadUpdate(t *testing.T, rc RandomChain, preState State, posState State, expectNoUpdate eth.ChainID) {
	crossSafeUpdates := 0
	chainsToUpdate := len(rc.chainIDs)
	for _, chain := range rc.chainIDs {
		preCrossSafeDerived := preState.chainHeads[chain].crossSafe.Derived.Number
		preLocalSafeDerived := preState.chainHeads[chain].localSafe.Derived.Number
		posCrossSafeDerived := posState.chainHeads[chain].crossSafe.Derived.Number
		if chain != expectNoUpdate {
			if preCrossSafeDerived < preLocalSafeDerived {
				preCrossSafeSource := preState.chainHeads[chain].crossSafe.Source.Number
				posCrossSafeSource := posState.chainHeads[chain].crossSafe.Source.Number
				if preCrossSafeDerived < posCrossSafeDerived {
					require.Equal(t, posCrossSafeDerived, preCrossSafeDerived+1,
						"Cross Safe update must be incremental, instead got %d -> %d on chain %d", preCrossSafeDerived, posCrossSafeDerived, chain)
					require.Equal(t, preCrossSafeSource, posCrossSafeSource,
						"Cross Safe head unexpectedly updated for chain %d", chain)
					crossSafeUpdates++
					t.Logf("Cross Safe head for chain %d has been updated from %d to %d", chain, preCrossSafeDerived, posCrossSafeDerived)
				} else if posCrossSafeDerived == preCrossSafeDerived {
					if preCrossSafeSource < posCrossSafeSource {
						require.Equal(t, posCrossSafeSource, preCrossSafeSource+1,
							"Cross Safe scope bump should be incremental, instead got %d -> %d on chain %d", preCrossSafeSource, posCrossSafeSource, chain)
						crossSafeUpdates++
						t.Logf("Cross Safe head for chain %d has increased source from %d to %d", chain, preCrossSafeSource, posCrossSafeSource)
					}
				}
			} else {
				chainsToUpdate--
				require.Equal(t, posCrossSafeDerived, preCrossSafeDerived)
				t.Logf("Cross Safe head for chain %d was already equal to Local Safe", chain)
			}
		} else {
			chainsToUpdate--
			require.Equal(t, posCrossSafeDerived, preCrossSafeDerived, "Cross Safe head unexpectedly updated for chain %d", chain)
			t.Logf("Cross Safe head for chain %d was not updated because the candidate was invalid", chain)
		}
	}
	if chainsToUpdate > 0 && expectNoUpdate == eth.ChainIDFromUInt64(0) {
		// At least one chain must be updated
		require.Greater(t, crossSafeUpdates, 0)
	}
}

func AssertStateNotChange(t *testing.T, rc RandomChain, preState State, posState State) {
	for _, chain := range rc.chainIDs {
		require.Equal(t, preState.chainHeads[chain].localUnsafe, posState.chainHeads[chain].localUnsafe, "Local Unsafe head for chain %d was expected not to change", chain)
		require.Equal(t, preState.chainHeads[chain].crossUnsafe, posState.chainHeads[chain].crossUnsafe, "Cross Unsafe head for chain %d was expected not to change", chain)
		require.Equal(t, preState.chainHeads[chain].localSafe, posState.chainHeads[chain].localSafe, "Local Safe head for chain %d was expected not to change", chain)
		require.Equal(t, preState.chainHeads[chain].crossSafe, posState.chainHeads[chain].crossSafe, "Cross Safe head for chain %d was expected not to change", chain)
	}
}
