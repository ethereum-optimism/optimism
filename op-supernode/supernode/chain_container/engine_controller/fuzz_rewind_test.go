package engine_controller

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// FuzzRewindToTimestamp tests the RewindToTimestamp function with random
// engine states and rewind targets.
//
// Properties:
// P25: Rewind never succeeds when target is before finalized head
// P26: After successful rewind, unsafe head == target block
// P27: After successful rewind, finalized head is unchanged
func FuzzRewindToTimestamp(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(12345))
	f.Add(int64(0))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		genesisTime := uint64(1000)
		blockTime := uint64(2)

		// Generate chain state: finalized <= safe <= unsafe
		finalizedNum := uint64(rng.Intn(20))
		safeNum := finalizedNum + uint64(rng.Intn(10))
		unsafeNum := safeNum + uint64(rng.Intn(10))

		// Target block: random position relative to chain state
		// Allow targets both before finalized (should fail) and at/after finalized (may succeed)
		targetNum := uint64(rng.Intn(int(unsafeNum) + 5))

		// Build block refs
		makeRef := func(num uint64) eth.L2BlockRef {
			ts := genesisTime + num*blockTime
			var parentHash common.Hash
			if num > 0 {
				parentHash = common.BigToHash(big.NewInt(int64(num - 1)))
			}
			return eth.L2BlockRef{
				Number:     num,
				Hash:       common.BigToHash(big.NewInt(int64(num))),
				ParentHash: parentHash,
				Time:       ts,
			}
		}

		targetRef := makeRef(targetNum)
		finalizedRef := makeRef(finalizedNum)
		safeRef := makeRef(safeNum)

		// Compute expected rewind targets
		expectedSafe := safeRef
		if targetNum < safeNum {
			expectedSafe = targetRef
		}
		expectedFinalized := finalizedRef

		// Build mock L2 with proper state
		l2 := &mockL2{
			refsByNumber: map[uint64]eth.L2BlockRef{
				targetNum: targetRef,
			},
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      safeRef,
				eth.Finalized: finalizedRef,
			},
			refsByLabelAfterFCU: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Unsafe:    targetRef,
				eth.Safe:      expectedSafe,
				eth.Finalized: expectedFinalized,
			},
			payloadsByNumber: map[uint64]*eth.ExecutionPayloadEnvelope{
				targetNum: {
					ExecutionPayload: &eth.ExecutionPayload{
						ParentHash:  targetRef.ParentHash,
						BlockNumber: eth.Uint64Quantity(targetRef.Number),
						Timestamp:   eth.Uint64Quantity(targetRef.Time),
						BlockHash:   targetRef.Hash,
						FeeRecipient: common.Address{0x01},
					},
				},
			},
		}

		rcfg := &rollup.Config{
			Genesis:   rollup.Genesis{L2: eth.BlockID{Number: 0}, L2Time: genesisTime},
			BlockTime: blockTime,
			L2ChainID: big.NewInt(420),
		}

		ec := &simpleEngineController{
			l2:     l2,
			rollup: rcfg,
			log:    gethlog.New(),
		}

		targetTimestamp := genesisTime + targetNum*blockTime
		err := ec.RewindToTimestamp(context.Background(), targetTimestamp)

		if targetNum < finalizedNum {
			// P25: Rewind never succeeds when target is before finalized head
			require.Error(t, err, "P25: rewind to block %d should fail (finalized=%d)", targetNum, finalizedNum)
			require.ErrorIs(t, err, ErrRewindOverFinalizedHead,
				"P25: should get ErrRewindOverFinalizedHead when target=%d < finalized=%d", targetNum, finalizedNum)
		} else {
			// Successful rewind
			require.NoError(t, err, "rewind to block %d should succeed (finalized=%d)", targetNum, finalizedNum)

			// P26: After successful rewind, unsafe head == target block
			// Verified by verifyRewindState inside RewindToTimestamp
			// We also verify the FCU was called with correct state
			require.NotNil(t, l2.lastFCUState)
			require.Equal(t, targetRef.Hash, l2.lastFCUState.HeadBlockHash,
				"P26: FCU head should be target block hash")

			// P27: After successful rewind, finalized head is unchanged
			require.Equal(t, expectedFinalized.Hash, l2.lastFCUState.FinalizedBlockHash,
				"P27: finalized head should be unchanged (or clamped to target if target < finalized)")

			// Verify NewPayload was called once (synthetic block)
			require.Equal(t, 1, l2.newPayloadCalls, "NewPayload should be called once")
			require.NotNil(t, l2.lastNewPayload)
			require.Equal(t, common.MaxAddress, l2.lastNewPayload.FeeRecipient,
				"synthetic payload should have modified fee recipient")

			// Verify ForkchoiceUpdate was called twice
			require.Equal(t, 2, l2.fcuCalls, "FCU should be called twice")
		}
	})
}

// FuzzComputeRewindTargets tests that computeRewindTargets correctly clamps
// safe and finalized heads.
//
// Properties:
// P25: Returns error when target < finalized
// P27: Finalized head is always <= target after clamping
func FuzzComputeRewindTargets(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		genesisTime := uint64(1000)
		blockTime := uint64(2)

		finalizedNum := uint64(rng.Intn(20))
		safeNum := finalizedNum + uint64(rng.Intn(10))
		targetNum := uint64(rng.Intn(int(safeNum) + 10))

		makeRef := func(num uint64) eth.L2BlockRef {
			return eth.L2BlockRef{
				Number: num,
				Hash:   common.BigToHash(big.NewInt(int64(num))),
				Time:   genesisTime + num*blockTime,
			}
		}

		targetRef := makeRef(targetNum)
		safeRef := makeRef(safeNum)
		finalizedRef := makeRef(finalizedNum)

		l2 := &mockL2{
			refsByLabel: map[eth.BlockLabel]eth.L2BlockRef{
				eth.Safe:      safeRef,
				eth.Finalized: finalizedRef,
			},
		}

		ec := &simpleEngineController{
			l2:     l2,
			rollup: &rollup.Config{Genesis: rollup.Genesis{L2Time: genesisTime}, BlockTime: blockTime, L2ChainID: big.NewInt(420)},
			log:    gethlog.New(),
		}

		safe, finalized, err := ec.computeRewindTargets(context.Background(), targetRef)

		if targetNum < finalizedNum {
			// P25: Must fail
			require.ErrorIs(t, err, ErrRewindOverFinalizedHead,
				"P25: target=%d < finalized=%d should fail", targetNum, finalizedNum)
		} else {
			require.NoError(t, err)

			// Safe should be min(currentSafe, target)
			if safeNum < targetNum {
				require.Equal(t, safeRef, safe, "safe should stay at currentSafe when < target")
			} else {
				require.Equal(t, targetRef, safe, "safe should clamp to target when >= target")
			}

			// P27: Finalized never moves forward
			if finalizedNum < targetNum {
				require.Equal(t, finalizedRef, finalized, "P27: finalized should stay at currentFinalized")
			} else {
				require.Equal(t, targetRef, finalized, "finalized should clamp to target when == target")
			}

			// Finalized is always <= safe
			require.LessOrEqual(t, finalized.Number, safe.Number,
				"finalized should always be <= safe")
		}
	})
}
