package interop

import (
	"context"
	"math/rand"
	"testing"

	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Fuzz Test: progressInterop with valid multi-chain states (P28, P29)
// =============================================================================

// FuzzProgressInteropValid tests that valid multi-chain states always result
// in successful commits to the VerifiedDB.
//
// Document coverage:
//   INV-6: L2 heads advance monotonically (sequential timestamp commits)
//   Step 5: Valid blocks → extend VerifiedDB
//
// Properties:
// P28: Timestamps are processed strictly sequentially (no gaps, no repeats)
// P29: Valid results are committed
func FuzzProgressInteropValid(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(12345))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		activationTS := uint64(1000 + rng.Intn(9000))
		numChains := 2 + rng.Intn(3)    // 2-4 chains
		numTimestamps := 2 + rng.Intn(5) // 2-6 timestamps to process

		chainIDs := generateChainIDs(numChains, 10, 10)

		verifiedDB, err := OpenVerifiedDB(t.TempDir())
		require.NoError(t, err)
		defer verifiedDB.Close()

		logsDBs := make(map[eth.ChainID]LogsDB)
		chains := make(map[eth.ChainID]cc.ChainContainer)
		for i, chainID := range chainIDs {
			mockDB := newFuzzMockLogsDB()
			mockDB.defaultContainsSeal = suptypes.BlockSeal{Number: 1, Timestamp: 1}
			logsDBs[chainID] = mockDB

			chains[chainID] = newMockChainContainer(uint64(10 + i*10))
		}

		interop := &Interop{
			log:                 gethlog.New(),
			logsDBs:             logsDBs,
			chains:              chains,
			verifiedDB:          verifiedDB,
			activationTimestamp: activationTS,
			ctx:                 context.Background(),
		}
		// Override verifyFn to always return valid results
		interop.verifyFn = func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
			result := Result{
				Timestamp:    ts,
				L2Heads:      make(map[eth.ChainID]eth.BlockID),
				InvalidHeads: make(map[eth.ChainID]eth.BlockID),
			}
			for chainID, block := range blocksAtTimestamp {
				result.L2Heads[chainID] = block
			}
			return result, nil
		}
		// Override cycleVerifyFn to always return valid (no cycles)
		interop.cycleVerifyFn = func(ts uint64, blocksAtTimestamp map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{
				Timestamp: ts,
				L2Heads:   blocksAtTimestamp,
			}, nil
		}

		// Process timestamps using progressInterop + handleResult
		for i := 0; i < numTimestamps; i++ {
			ts := activationTS + uint64(i)

			result, err := interop.progressInterop()
			require.NoError(t, err)

			// P29: Valid results should be committable
			require.True(t, result.IsValid(), "result at ts=%d should be valid", ts)

			err = interop.handleResult(result)
			require.NoError(t, err)

			// P28: Verify the commit succeeded and timestamp is sequential
			has, err := verifiedDB.Has(ts)
			require.NoError(t, err)
			require.True(t, has, "P28: timestamp %d should be committed", ts)

			lastTS, initialized := verifiedDB.LastTimestamp()
			require.True(t, initialized)
			require.Equal(t, ts, lastTS, "P28: lastTimestamp should match committed ts")
		}

		// Final verification: all timestamps committed sequentially
		for i := 0; i < numTimestamps; i++ {
			ts := activationTS + uint64(i)
			has, err := verifiedDB.Has(ts)
			require.NoError(t, err)
			require.True(t, has, "P28: all timestamps should be committed sequentially")
		}
	})
}

// =============================================================================
// Fuzz Test: progressInterop with invalid messages triggers invalidation (P29, P31)
// =============================================================================

// FuzzProgressInteropInvalid tests that invalid messages correctly trigger
// block invalidation through handleResult.
//
// Document coverage:
//   Step 4: Invalid B_j → invalidateBlock called per invalid chain (DenyList addition),
//           valid chains untouched. After invalidation, can resume at same timestamp.
//
// Properties:
// P29: Invalid results trigger block invalidation via invalidateBlock
// P31: After invalidation, the interop loop can resume from the same timestamp
func FuzzProgressInteropInvalid(f *testing.F) {
	f.Add(int64(1), 1)
	f.Add(int64(42), 2)
	f.Add(int64(100), 3)

	f.Fuzz(func(t *testing.T, seed int64, numInvalidChains int) {
		rng := rand.New(rand.NewSource(seed))

		if numInvalidChains < 1 {
			numInvalidChains = 1
		}
		if numInvalidChains > 5 {
			numInvalidChains = 5
		}

		activationTS := uint64(1000)
		numChains := numInvalidChains + 1 + rng.Intn(3)
		if numChains > 8 {
			numChains = 8
		}

		chainIDs := generateChainIDs(numChains, 10, 10)

		verifiedDB, err := OpenVerifiedDB(t.TempDir())
		require.NoError(t, err)
		defer verifiedDB.Close()

		// Create interop with chains using existing mockChainContainer
		chains := make(map[eth.ChainID]cc.ChainContainer)
		mocks := make(map[eth.ChainID]*mockChainContainer)
		for i, chainID := range chainIDs {
			mock := newMockChainContainer(uint64(10 + i*10))
			mock.currentL1 = eth.BlockRef{Number: 100, Hash: randomHash(rng)}
			mocks[chainID] = mock
			chains[chainID] = mock
		}

		interop := &Interop{
			log:                 gethlog.New(),
			logsDBs:             make(map[eth.ChainID]LogsDB),
			verifiedDB:          verifiedDB,
			activationTimestamp: activationTS,
			chains:              chains,
			ctx:                 context.Background(),
		}

		// Build an invalid result
		invalidResult := Result{
			Timestamp:    activationTS,
			L2Heads:      make(map[eth.ChainID]eth.BlockID),
			InvalidHeads: make(map[eth.ChainID]eth.BlockID),
		}

		for i, chainID := range chainIDs {
			blockHash := randomHash(rng)
			blockID := eth.BlockID{Number: uint64(100 + i), Hash: blockHash}
			invalidResult.L2Heads[chainID] = blockID
			if i < numInvalidChains {
				invalidResult.InvalidHeads[chainID] = blockID
			}
		}

		// P29: result with invalid heads should not be valid
		require.False(t, invalidResult.IsValid(), "P29: result with invalid heads should not be valid")
		require.Equal(t, numInvalidChains, len(invalidResult.InvalidHeads))

		// P29: handleResult with invalid result should call invalidateBlock on chains
		err = interop.handleResult(invalidResult)
		require.NoError(t, err)

		// Verify invalidateBlock was called for each invalid chain
		for _, chainID := range chainIDs[:numInvalidChains] {
			mock := mocks[chainID]
			mock.mu.Lock()
			calls := len(mock.invalidateBlockCalls)
			mock.mu.Unlock()
			require.Equal(t, 1, calls,
				"P29: invalidateBlock should be called once for invalid chain %s", chainID)
		}

		// Verify invalidateBlock was NOT called for valid chains
		for _, chainID := range chainIDs[numInvalidChains:] {
			mock := mocks[chainID]
			mock.mu.Lock()
			calls := len(mock.invalidateBlockCalls)
			mock.mu.Unlock()
			require.Equal(t, 0, calls,
				"valid chain %s should not have invalidateBlock called", chainID)
		}

		// P31: After invalidation, should be able to commit at the same timestamp
		validResult := generateVerifiedResult(rng, activationTS, chainIDs)

		err = verifiedDB.Commit(validResult)
		require.NoError(t, err, "P31: should be able to commit at same timestamp after invalid result")


		lastTS, initialized := verifiedDB.LastTimestamp()
		require.True(t, initialized)
		require.Equal(t, activationTS, lastTS)
	})
}

// =============================================================================
// Fuzz Test: Reset correctly rewinds state (P32)
// =============================================================================

// FuzzProgressInteropReset tests that Reset correctly rewinds both
// the logsDB and verifiedDB.
//
// Document coverage:
//   Step 3: C_t reorged out → rollback VerifiedDB (entries after rewindTS deleted),
//           rollback LogsDB (rewound to block before rewind timestamp),
//           currentL1 cleared to force re-evaluation.
//           Can resume committing at rewindTS+1.
//
// Properties:
// P32: Reset correctly rewinds both logsDB and verifiedDB
func FuzzProgressInteropReset(f *testing.F) {
	f.Add(int64(1), uint64(5))
	f.Add(int64(42), uint64(3))

	f.Fuzz(func(t *testing.T, seed int64, numCommits uint64) {
		rng := rand.New(rand.NewSource(seed))

		if numCommits < 2 {
			numCommits = 2
		}
		if numCommits > 20 {
			numCommits = 20
		}

		activationTS := uint64(1000)
		chainID := eth.ChainIDFromUInt64(10)

		verifiedDB, err := OpenVerifiedDB(t.TempDir())
		require.NoError(t, err)
		defer verifiedDB.Close()

		// Set up mock chain and logsDB
		mockDB := newFuzzMockLogsDB()
		mockDB.firstBlock = suptypes.BlockSeal{Number: 100, Timestamp: activationTS}

		mock := newMockChainContainer(10)
		mock.currentL1 = eth.BlockRef{Number: 100, Hash: randomHash(rng)}

		interop := &Interop{
			log:                 gethlog.New(),
			activationTimestamp: activationTS,
			verifiedDB:          verifiedDB,
			logsDBs:             map[eth.ChainID]LogsDB{chainID: mockDB},
			chains:              map[eth.ChainID]cc.ChainContainer{chainID: mock},
			ctx:                 context.Background(),
		}

		// Commit several timestamps
		singleChain := []eth.ChainID{chainID}
		for i := uint64(0); i < numCommits; i++ {
			ts := activationTS + i
			err := verifiedDB.Commit(generateVerifiedResult(rng, ts, singleChain))
			require.NoError(t, err)
		}

		// Pick a random rewind point
		rewindOffset := uint64(rng.Int63n(int64(numCommits)))
		rewindTS := activationTS + rewindOffset

		// Call Reset (exercises both resetLogsDB and resetVerifiedDB)
		// invalidatedBlock.Number must equal rewindTS so that targetBlock.Number = rewindTS - 1
		invalidatedBlock := eth.BlockRef{
			Number:     rewindTS,
			Hash:       randomHash(rng),
			ParentHash: randomHash(rng),
		}
		interop.Reset(chainID, rewindTS, invalidatedBlock)

		// P32: Verify logsDB was rewound
		require.Equal(t, 1, len(mockDB.rewindCalls), "P32: logsDB should have been rewound once")
		require.Equal(t, rewindTS-1, mockDB.rewindCalls[0].Number, "P32: logsDB should be rewound to block before rewind timestamp")

		// P32: Verify currentL1 was reset to force re-evaluation
		require.Equal(t, eth.BlockID{}, interop.CurrentL1(), "P32: currentL1 should be reset to empty after Reset")

		// P32: Verify verifiedDB state after rewind
		// RewindAfter(rewindTS) keeps entries at rewindTS and below, deletes those strictly after
		for i := uint64(0); i < numCommits; i++ {
			ts := activationTS + i
			has, err := verifiedDB.Has(ts)
			require.NoError(t, err)

			if ts <= rewindTS {
				require.True(t, has, "P32: timestamp %d at/before rewind point %d should still exist", ts, rewindTS)
			} else {
				require.False(t, has, "P32: timestamp %d after rewind point %d should be deleted", ts, rewindTS)
			}
		}

		// P32: Verify we can resume committing after the rewind point
		lastTS, initialized := verifiedDB.LastTimestamp()
		require.True(t, initialized)
		require.Equal(t, rewindTS, lastTS, "P32: lastTimestamp should be rewindTS")

		// Should be able to commit at rewindTS+1 (next sequential)
		err = verifiedDB.Commit(generateVerifiedResult(rng, rewindTS+1, singleChain))
		require.NoError(t, err, "P32: should be able to commit at rewindTS+1")
	})
}

// =============================================================================
// Fuzz Test: handleResult with empty results (P30)
// =============================================================================

// FuzzHandleResultEmpty tests that empty results are no-ops.
//
// Document coverage:
//   Step 1: When chains not ready, empty result returned → no state change.
//
// Properties:
// P30: Empty results do not modify state
func FuzzHandleResultEmpty(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		verifiedDB, err := OpenVerifiedDB(t.TempDir())
		require.NoError(t, err)
		defer verifiedDB.Close()

		interop := &Interop{
			log:                 gethlog.New(),
			verifiedDB:          verifiedDB,
			activationTimestamp: uint64(1000),
			ctx:                 context.Background(),
		}

		// Pre-commit some state
		activationTS := uint64(1000)
		err = verifiedDB.Commit(generateVerifiedResult(rng, activationTS, []eth.ChainID{eth.ChainIDFromUInt64(10)}))
		require.NoError(t, err)

		lastTSBefore, _ := verifiedDB.LastTimestamp()

		// Build random empty results
		emptyResult := Result{
			Timestamp:    activationTS + 1 + uint64(rng.Intn(100)),
			L2Heads:      make(map[eth.ChainID]eth.BlockID),
			InvalidHeads: make(map[eth.ChainID]eth.BlockID),
		}

		require.True(t, emptyResult.IsEmpty(), "result with no L2Heads should be empty")

		// P30: handleResult with empty result should be a no-op
		err = interop.handleResult(emptyResult)
		require.NoError(t, err)

		lastTSAfter, _ := verifiedDB.LastTimestamp()
		require.Equal(t, lastTSBefore, lastTSAfter, "P30: empty result should not change state")
	})
}
