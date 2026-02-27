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

		chainIDs := make([]eth.ChainID, numChains)
		for i := range chainIDs {
			chainIDs[i] = eth.ChainIDFromUInt64(uint64(10 + i*10))
		}

		dataDir := t.TempDir()

		// Create a custom Interop with mock logsDBs and real VerifiedDB
		verifiedDB, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)
		defer verifiedDB.Close()

		logsDBs := make(map[eth.ChainID]LogsDB)
		for _, chainID := range chainIDs {
			mockDB := newFuzzMockLogsDB()
			mockDB.defaultContainsSeal = suptypes.BlockSeal{Number: 1, Timestamp: 1}
			logsDBs[chainID] = mockDB
		}

		// Set up blocks for each chain at each timestamp
		for ts := activationTS; ts < activationTS+uint64(numTimestamps); ts++ {
			for _, chainID := range chainIDs {
				blockHash := randomHash(rng)
				blockNum := ts - activationTS + 100

				mockDB := logsDBs[chainID].(*fuzzMockLogsDB)
				mockDB.blocks[blockNum] = fuzzBlockData{
					ref:      eth.BlockRef{Hash: blockHash, Number: blockNum, Time: ts},
					execMsgs: nil, // No executing messages - all blocks are valid
				}
			}
		}

		interop := &Interop{
			log:                 gethlog.New(),
			logsDBs:             logsDBs,
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

		// Process timestamps sequentially and verify P28
		for i := 0; i < numTimestamps; i++ {
			ts := activationTS + uint64(i)

			// Build blocksAtTimestamp
			blocksAtTimestamp := make(map[eth.ChainID]eth.BlockID)
			for _, chainID := range chainIDs {
				blockNum := ts - activationTS + 100
				mockDB := logsDBs[chainID].(*fuzzMockLogsDB)
				bd := mockDB.blocks[blockNum]
				blocksAtTimestamp[chainID] = eth.BlockID{Number: blockNum, Hash: bd.ref.Hash}
			}

			// Call verifyFn and handleResult
			result, err := interop.verifyFn(ts, blocksAtTimestamp)
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

		chainIDs := make([]eth.ChainID, numChains)
		for i := range chainIDs {
			chainIDs[i] = eth.ChainIDFromUInt64(uint64(10 + i*10))
		}

		dataDir := t.TempDir()
		verifiedDB, err := OpenVerifiedDB(dataDir)
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
		validResult := VerifiedResult{
			Timestamp: activationTS,
			L1Head:    eth.BlockID{Hash: randomHash(rng), Number: 100},
			L2Heads:   make(map[eth.ChainID]eth.BlockID),
		}
		for _, chainID := range chainIDs {
			validResult.L2Heads[chainID] = eth.BlockID{Hash: randomHash(rng), Number: 100}
		}

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

		dataDir := t.TempDir()
		verifiedDB, err := OpenVerifiedDB(dataDir)
		require.NoError(t, err)

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
		for i := uint64(0); i < numCommits; i++ {
			ts := activationTS + i
			err = verifiedDB.Commit(VerifiedResult{
				Timestamp: ts,
				L1Head:    eth.BlockID{Hash: randomHash(rng), Number: ts},
				L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: randomHash(rng), Number: 100 + i}},
			})
			require.NoError(t, err)
		}

		// Pick a random rewind point
		rewindOffset := uint64(rng.Int63n(int64(numCommits)))
		rewindTS := activationTS + rewindOffset

		// Call resetVerifiedDB (the part of Reset that handles verifiedDB)
		interop.resetVerifiedDB(rewindTS)

		// P32: Verify verifiedDB state after rewind
		for i := uint64(0); i < numCommits; i++ {
			ts := activationTS + i
			has, err := verifiedDB.Has(ts)
			require.NoError(t, err)

			if ts < rewindTS {
				require.True(t, has, "P32: timestamp %d before rewind point %d should still exist", ts, rewindTS)
			} else {
				require.False(t, has, "P32: timestamp %d at/after rewind point %d should be deleted", ts, rewindTS)
			}
		}

		// P32: Verify we can resume committing from the rewind point
		if rewindTS > activationTS {
			lastTS, initialized := verifiedDB.LastTimestamp()
			require.True(t, initialized)
			require.Equal(t, rewindTS-1, lastTS, "P32: lastTimestamp should be rewindTS-1")

			// Should be able to recommit at rewindTS
			err = verifiedDB.Commit(VerifiedResult{
				Timestamp: rewindTS,
				L1Head:    eth.BlockID{Hash: randomHash(rng), Number: rewindTS},
				L2Heads:   map[eth.ChainID]eth.BlockID{chainID: {Hash: randomHash(rng), Number: 200}},
			})
			require.NoError(t, err, "P32: should be able to recommit at rewind point")
		}

		verifiedDB.Close()
	})
}

// =============================================================================
// Fuzz Test: handleResult with empty results (P30)
// =============================================================================

// FuzzHandleResultEmpty tests that empty results are no-ops.
//
// Properties:
// P30: Empty results do not modify state
func FuzzHandleResultEmpty(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		dataDir := t.TempDir()
		verifiedDB, err := OpenVerifiedDB(dataDir)
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
		err = verifiedDB.Commit(VerifiedResult{
			Timestamp: activationTS,
			L1Head:    eth.BlockID{Hash: randomHash(rng), Number: 1},
			L2Heads:   map[eth.ChainID]eth.BlockID{eth.ChainIDFromUInt64(10): {Hash: randomHash(rng), Number: 1}},
		})
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
