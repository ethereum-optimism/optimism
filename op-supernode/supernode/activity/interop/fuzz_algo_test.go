package interop

import (
	"math"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Source/dest chain ID pair used across multiple fuzz tests.
var (
	fuzzSourceChainID = eth.ChainIDFromUInt64(10)
	fuzzDestChainID   = eth.ChainIDFromUInt64(8453)
)

// =============================================================================
// Fuzz Mock: configurable LogsDB for fuzz testing
// =============================================================================

// fuzzMockLogsDB is a more configurable mock that supports per-block behavior
type fuzzMockLogsDB struct {
	// Per block-number: block ref, exec msgs
	blocks map[uint64]fuzzBlockData
	// Default contains behavior
	containsResults map[suptypes.ContainsQuery]fuzzContainsResult
	// Fallback contains behavior
	defaultContainsSeal suptypes.BlockSeal
	defaultContainsErr  error
	// First sealed block info
	firstBlock    suptypes.BlockSeal
	firstBlockErr error
	// Sealed block tracking (for progressInterop integration)
	latestSealed eth.BlockID
	hasSealed    bool
	sealedBlocks map[uint64]suptypes.BlockSeal
	// Reset tracking
	rewindCalls []eth.BlockID
	clearCalls  int
}

type fuzzBlockData struct {
	ref      eth.BlockRef
	logCount uint32
	execMsgs map[uint32]*suptypes.ExecutingMessage
	err      error
}

type fuzzContainsResult struct {
	seal suptypes.BlockSeal
	err  error
}

func newFuzzMockLogsDB() *fuzzMockLogsDB {
	return &fuzzMockLogsDB{
		blocks:          make(map[uint64]fuzzBlockData),
		containsResults: make(map[suptypes.ContainsQuery]fuzzContainsResult),
		sealedBlocks:    make(map[uint64]suptypes.BlockSeal),
	}
}

func (m *fuzzMockLogsDB) LatestSealedBlock() (eth.BlockID, bool) {
	return m.latestSealed, m.hasSealed
}
func (m *fuzzMockLogsDB) FindSealedBlock(number uint64) (suptypes.BlockSeal, error) {
	if seal, ok := m.sealedBlocks[number]; ok {
		return seal, nil
	}
	return suptypes.BlockSeal{}, nil
}

func (m *fuzzMockLogsDB) FirstSealedBlock() (suptypes.BlockSeal, error) {
	if m.firstBlockErr != nil {
		return suptypes.BlockSeal{}, m.firstBlockErr
	}
	return m.firstBlock, nil
}

func (m *fuzzMockLogsDB) OpenBlock(blockNum uint64) (eth.BlockRef, uint32, map[uint32]*suptypes.ExecutingMessage, error) {
	if bd, ok := m.blocks[blockNum]; ok {
		return bd.ref, bd.logCount, bd.execMsgs, bd.err
	}
	return eth.BlockRef{}, 0, nil, suptypes.ErrSkipped
}

func (m *fuzzMockLogsDB) Contains(query suptypes.ContainsQuery) (suptypes.BlockSeal, error) {
	if r, ok := m.containsResults[query]; ok {
		return r.seal, r.err
	}
	if m.defaultContainsErr != nil {
		return suptypes.BlockSeal{}, m.defaultContainsErr
	}
	return m.defaultContainsSeal, nil
}

func (m *fuzzMockLogsDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *suptypes.ExecutingMessage) error {
	return nil
}
func (m *fuzzMockLogsDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.latestSealed = block
	m.hasSealed = true
	m.sealedBlocks[block.Number] = suptypes.BlockSeal{
		Hash:      block.Hash,
		Number:    block.Number,
		Timestamp: timestamp,
	}
	return nil
}
func (m *fuzzMockLogsDB) Rewind(inv reads.Invalidator, newHead eth.BlockID) error {
	m.rewindCalls = append(m.rewindCalls, newHead)
	return nil
}
func (m *fuzzMockLogsDB) Clear(inv reads.Invalidator) error {
	m.clearCalls++
	return nil
}
func (m *fuzzMockLogsDB) Close() error { return nil }

var _ LogsDB = (*fuzzMockLogsDB)(nil)

// =============================================================================
// Fuzz Test: Valid messages never produce InvalidHeads (P1, P3)
// =============================================================================

// FuzzVerifyInteropMessagesValid generates random valid multi-chain states and
// verifies that valid cross-chain messages always result in a valid Result.
//
// Document coverage:
//   INV-3: Every block in LogsDB is cross-valid (valid executing messages, no cycles)
//
// Properties:
// P1: Valid cross-chain messages never produce InvalidHeads
// P3: Result.IsValid() ↔ len(InvalidHeads) == 0
func FuzzVerifyInteropMessagesValid(f *testing.F) {
	f.Add(int64(1), uint8(3), uint8(2), uint64(500000))
	f.Add(int64(42), uint8(2), uint8(0), uint64(ExpiryTime+1))
	f.Add(int64(12345), uint8(5), uint8(3), uint64(ExpiryTime))
	f.Add(int64(0), uint8(4), uint8(1), uint64(2*ExpiryTime))

	f.Fuzz(func(t *testing.T, seed int64, numChainsRaw uint8, numMsgsRaw uint8, execTSRaw uint64) {
		rng := rand.New(rand.NewSource(seed))

		numChains := 2 + int(numChainsRaw%4) // 2-5 chains
		maxMsgsPerBlock := int(numMsgsRaw % 4) // 0-3 messages per block
		execTimestamp := 100000 + (execTSRaw % 900000)

		chainIDs := generateChainIDs(numChains, 10, 10)
		setup := generateChainSetup(rng, chainIDs, execTimestamp)
		populateValidMessages(rng, &setup, maxMsgsPerBlock)

		interop := newFuzzInterop(setup.LogsDBs)

		result, err := interop.verifyInteropMessages(execTimestamp, setup.BlocksAtTimestamp)
		require.NoError(t, err)

		// P1: Valid messages never produce InvalidHeads
		require.True(t, result.IsValid(), "P1: valid messages should produce valid result, got InvalidHeads: %v", result.InvalidHeads)

		// P3: IsValid() ↔ len(InvalidHeads) == 0
		require.Empty(t, result.InvalidHeads, "P3: InvalidHeads should be empty for valid result")

		// Verify all chains are in L2Heads
		for _, chainID := range chainIDs {
			require.Contains(t, result.L2Heads, chainID, "all chains should be in L2Heads")
			require.Equal(t, setup.BlocksAtTimestamp[chainID], result.L2Heads[chainID])
		}
	})
}

// =============================================================================
// Fuzz Test: Each invalidation type is correctly detected (P2)
// =============================================================================

// FuzzVerifyInteropMessagesFails generates states with various invalidation types
// and verifies they are correctly detected.
//
// Document coverage:
//   INV-3: Cross-validity violations are detected (5 invalidation types)
//   Step 4: Invalid blocks are identified for DenyList addition
//
// Properties:
// P2: Every invalidation type is correctly detected
func FuzzVerifyInteropMessagesFails(f *testing.F) {
	f.Add(int64(1), uint8(0))
	f.Add(int64(42), uint8(1))
	f.Add(int64(100), uint8(2))
	f.Add(int64(200), uint8(3))
	f.Add(int64(300), uint8(4))

	f.Fuzz(func(t *testing.T, seed int64, invalidationType uint8) {
		rng := rand.New(rand.NewSource(seed))

		execTimestamp := uint64(1000000)
		destBlockHash := randomHash(rng)
		destBlockNum := uint64(100 + rng.Intn(1000))

		sourceDB := newFuzzMockLogsDB()
		destDB := newFuzzMockLogsDB()

		var execMsg *suptypes.ExecutingMessage

		invType := invalidationType % 5
		switch invType {
		case 0: // Unknown source chain - source not in logsDBs
			unknownChain := eth.ChainIDFromUInt64(9999)
			execMsg = generateExecutingMessage(rng, unknownChain, execTimestamp-100, 0)

		case 1: // Timestamp violation - initTimestamp > execTimestamp
			initTS := execTimestamp + 1 + uint64(rng.Intn(1000))
			execMsg = generateExecutingMessage(rng, fuzzSourceChainID, initTS, 0)

		case 2: // Expired message
			initTS := execTimestamp - ExpiryTime - 1 - uint64(rng.Intn(10000))
			execMsg = generateExecutingMessage(rng, fuzzSourceChainID, initTS, 0)

		case 3: // Message not found (ErrConflict from Contains)
			initTS := execTimestamp - 1 - uint64(rng.Intn(int(ExpiryTime-1)))
			execMsg = generateExecutingMessage(rng, fuzzSourceChainID, initTS, 0)
			sourceDB.defaultContainsErr = suptypes.ErrConflict

		case 4: // Block hash mismatch
			setBlockData(destDB, randomHash(rng), destBlockNum, execTimestamp)

			logsDBs := map[eth.ChainID]LogsDB{
				fuzzSourceChainID: sourceDB,
				fuzzDestChainID:   destDB,
			}

			interop := newFuzzInterop(logsDBs)
			blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
				fuzzDestChainID: {Number: destBlockNum, Hash: destBlockHash},
			}

			result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
			require.NoError(t, err)
			require.False(t, result.IsValid(), "P2: block hash mismatch should be detected")
			require.Contains(t, result.InvalidHeads, fuzzDestChainID)
			return
		}

		if invType != 4 {
			setBlockDataWithMsgs(destDB, destBlockHash, destBlockNum, execTimestamp,
				map[uint32]*suptypes.ExecutingMessage{0: execMsg})

			logsDBs := map[eth.ChainID]LogsDB{
				fuzzSourceChainID: sourceDB,
				fuzzDestChainID:   destDB,
			}

			interop := newFuzzInterop(logsDBs)
			blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
				fuzzDestChainID: {Number: destBlockNum, Hash: destBlockHash},
			}

			result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
			require.NoError(t, err)
			require.False(t, result.IsValid(), "P2: invalidation type %d should be detected", invType)
			require.Contains(t, result.InvalidHeads, fuzzDestChainID, "P2: dest chain should be in InvalidHeads")
		}
	})
}

// =============================================================================
// Fuzz Test: Expiry boundary exact values (P4)
// =============================================================================

// FuzzVerifyExpiryBoundary tests timestamps at the exact expiry boundary.
//
// Document coverage:
//   INV-3: Cross-validity expiry window boundary correctness
//
// Properties:
// P4: execMsg.Timestamp + ExpiryTime overflow doesn't cause false positive/negative
func FuzzVerifyExpiryBoundary(f *testing.F) {
	f.Add(int64(1), uint64(1000000))
	f.Add(int64(42), uint64(ExpiryTime+1))
	f.Add(int64(100), uint64(ExpiryTime))
	f.Add(int64(200), uint64(math.MaxUint64-ExpiryTime))
	f.Add(int64(300), uint64(math.MaxUint64))

	f.Fuzz(func(t *testing.T, seed int64, execTimestamp uint64) {
		rng := rand.New(rand.NewSource(seed))

		// Skip trivially invalid exec timestamps (must be > 0 for any valid init timestamp)
		if execTimestamp == 0 {
			return
		}

		destBlockHash := randomHash(rng)
		destBlockNum := uint64(100)

		// Test boundary conditions around the expiry time.
		// Skip cases where initTS + ExpiryTime would overflow uint64
		// (unrealistic in practice — timestamps are Unix seconds).
		type boundaryTest struct {
			name        string
			initTS      uint64
			expectValid bool
		}

		var tests []boundaryTest

		// Exactly at boundary: initTS + ExpiryTime == execTimestamp → valid (not <)
		if execTimestamp >= ExpiryTime {
			exactBoundaryTS := execTimestamp - ExpiryTime
			tests = append(tests, boundaryTest{
				name:        "exact_boundary",
				initTS:      exactBoundaryTS,
				expectValid: true,
			})

			// One past expiry: initTS + ExpiryTime < execTimestamp → expired
			if exactBoundaryTS > 0 {
				tests = append(tests, boundaryTest{
					name:        "one_past_expiry",
					initTS:      exactBoundaryTS - 1,
					expectValid: false,
				})
			}
		}

		// One before expiry: initTS + ExpiryTime > execTimestamp → valid
		if execTimestamp > ExpiryTime {
			initTS := execTimestamp - ExpiryTime + 1
			if initTS <= math.MaxUint64-ExpiryTime {
				tests = append(tests, boundaryTest{
					name:        "one_before_expiry",
					initTS:      initTS,
					expectValid: true,
				})
			}
		}

		// Equal timestamp: valid unless initTS + ExpiryTime overflows uint64
		if execTimestamp <= math.MaxUint64-ExpiryTime {
			tests = append(tests, boundaryTest{
				name:        "equal_timestamp",
				initTS:      execTimestamp,
				expectValid: true,
			})
		}

		// One less than exec timestamp: valid if within expiry window
		if execTimestamp > 0 {
			ts := execTimestamp - 1
			if ts <= math.MaxUint64-ExpiryTime {
				tests = append(tests, boundaryTest{
					name:        "one_less",
					initTS:      ts,
					expectValid: ts+ExpiryTime >= execTimestamp,
				})
			}
		}

		for _, tc := range tests {
			sourceDB := newFuzzMockLogsDB()
			sourceDB.defaultContainsSeal = suptypes.BlockSeal{Number: 1, Timestamp: tc.initTS}

			destDB := newFuzzMockLogsDB()

			execMsg := generateExecutingMessage(rng, fuzzSourceChainID, tc.initTS, 0)
			setBlockDataWithMsgs(destDB, destBlockHash, destBlockNum, execTimestamp,
				map[uint32]*suptypes.ExecutingMessage{0: execMsg})

			interop := newFuzzInterop(map[eth.ChainID]LogsDB{
				fuzzSourceChainID: sourceDB,
				fuzzDestChainID:   destDB,
			})

			blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
				fuzzDestChainID: {Number: destBlockNum, Hash: destBlockHash},
			}

			result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
			require.NoError(t, err)

			if tc.expectValid {
				require.True(t, result.IsValid(), "P4: %s at execTS=%d, initTS=%d should be valid", tc.name, execTimestamp, tc.initTS)
			} else {
				require.False(t, result.IsValid(), "P4: %s at execTS=%d, initTS=%d should be invalid", tc.name, execTimestamp, tc.initTS)
			}
		}
	})
}

// =============================================================================
// Fuzz Test: Expiry overflow causes false expiration (P4 overflow)
// =============================================================================

// FuzzVerifyExpiryOverflow tests that when execMsg.Timestamp is large enough
// for Timestamp + ExpiryTime to overflow uint64, the production code's
// unchecked addition wraps around and falsely expires a valid message.
//
// Document coverage:
//   INV-3: BUG — uint64 overflow in expiry check violates cross-validity correctness.
//
// The production check (algo.go:167) is:
//
//	if execMsg.Timestamp+ExpiryTime < executingTimestamp { → ErrMessageExpired }
//
// When Timestamp > MaxUint64-ExpiryTime, the LHS overflows to a small value,
// making the condition true even though the message is not actually expired
// (initTS <= execTS, and the "real" age is execTS - initTS which is < ExpiryTime).
//
// This test demonstrates the bug: a message whose true age is well within the
// expiry window gets incorrectly rejected due to uint64 overflow.
func FuzzVerifyExpiryOverflow(f *testing.F) {
	// Seeds that place initTS in the overflow region (initTS + ExpiryTime > MaxUint64)
	f.Add(int64(1), uint64(0))   // offset 0: initTS = MaxUint64 - ExpiryTime + 1
	f.Add(int64(2), uint64(100)) // offset 100: initTS = MaxUint64 - ExpiryTime + 101
	f.Add(int64(3), uint64(ExpiryTime-1))

	f.Fuzz(func(t *testing.T, seed int64, offset uint64) {
		rng := rand.New(rand.NewSource(seed))

		// Clamp offset so initTS doesn't wrap around itself
		maxOffset := uint64(ExpiryTime - 1)
		if offset > maxOffset {
			offset = offset % maxOffset
		}

		// Place initTS in the overflow zone: initTS + ExpiryTime will wrap uint64
		initTS := (math.MaxUint64 - ExpiryTime + 1) + offset

		// Sanity: confirm this is actually in the overflow zone
		if initTS <= math.MaxUint64-ExpiryTime {
			return // not in overflow zone, skip
		}

		// execTimestamp must be >= initTS (timestamp ordering) and the "real"
		// age (execTS - initTS) must be <= ExpiryTime so the message is
		// logically valid.
		//
		// Pick execTS in [initTS, initTS + ExpiryTime/2] but clamp to MaxUint64.
		age := uint64(rng.Int63n(int64(ExpiryTime/2))) + 1
		execTimestamp := initTS + age
		if execTimestamp < initTS {
			// execTimestamp itself overflowed — skip
			return
		}

		// The message is logically valid:
		//   initTS <= execTimestamp              (timestamp ordering satisfied)
		//   execTimestamp - initTS <= ExpiryTime (within expiry window)
		//
		// But the production code computes initTS + ExpiryTime which overflows:
		//   (initTS + ExpiryTime) wraps to a small number < execTimestamp
		//   → falsely returns ErrMessageExpired

		destBlockHash := randomHash(rng)
		destBlockNum := uint64(100)

		sourceDB := newFuzzMockLogsDB()
		sourceDB.defaultContainsSeal = suptypes.BlockSeal{Number: 1, Timestamp: initTS}

		destDB := newFuzzMockLogsDB()

		execMsg := generateExecutingMessage(rng, fuzzSourceChainID, initTS, 0)
		setBlockDataWithMsgs(destDB, destBlockHash, destBlockNum, execTimestamp,
			map[uint32]*suptypes.ExecutingMessage{0: execMsg})

		interop := newFuzzInterop(map[eth.ChainID]LogsDB{
			fuzzSourceChainID: sourceDB,
			fuzzDestChainID:   destDB,
		})

		blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
			fuzzDestChainID: {Number: destBlockNum, Hash: destBlockHash},
		}

		result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
		require.NoError(t, err)

		// The message is logically valid (within expiry window), so a correct
		// implementation would return IsValid() == true.
		//
		// However, due to uint64 overflow in the production code, the message
		// is falsely expired. We assert the ACTUAL (buggy) behavior here so
		// that:
		// 1. The test documents the overflow bug.
		// 2. If the production code is fixed (e.g. by rewriting the check as
		//    `execTimestamp - initTS > ExpiryTime`), this assertion will flip
		//    and the test must be updated to expect IsValid() == true.
		overflowedSum := initTS + ExpiryTime // wraps around
		require.Less(t, overflowedSum, initTS,
			"sanity: addition must have overflowed")
		require.Less(t, overflowedSum, execTimestamp,
			"sanity: overflowed sum should be < execTimestamp, triggering false expiry")

		require.False(t, result.IsValid(),
			"BUG(P4-overflow): message with initTS=%d, execTS=%d (age=%d, ExpiryTime=%d) "+
				"is logically valid but falsely expired due to uint64 overflow in "+
				"initTS+ExpiryTime (overflows to %d)",
			initTS, execTimestamp, execTimestamp-initTS, ExpiryTime, overflowedSum)
		require.Contains(t, result.InvalidHeads, fuzzDestChainID,
			"BUG(P4-overflow): dest chain should be in InvalidHeads due to false expiry")
	})
}

// =============================================================================
// Fuzz Test: ErrSkipped path (P5)
// =============================================================================

// FuzzVerifyFirstBlockSkipped tests the ErrSkipped fallback path when
// OpenBlock fails for the first block in the logsDB.
//
// Properties:
// P5: First block (ErrSkipped path) correctly handles hash mismatch
func FuzzVerifyFirstBlockSkipped(f *testing.F) {
	f.Add(int64(1), true)
	f.Add(int64(42), false)
	f.Add(int64(100), true)

	f.Fuzz(func(t *testing.T, seed int64, hashMatch bool) {
		rng := rand.New(rand.NewSource(seed))

		chainID := eth.ChainIDFromUInt64(10)
		blockNum := uint64(rng.Intn(10000))
		timestamp := uint64(100000 + rng.Intn(900000))
		expectedHash := randomHash(rng)

		var firstBlockHash common.Hash
		if hashMatch {
			firstBlockHash = expectedHash
		} else {
			firstBlockHash = randomHash(rng)
			// Ensure it's actually different
			for firstBlockHash == expectedHash {
				firstBlockHash = randomHash(rng)
			}
		}

		mockDB := newFuzzMockLogsDB()
		// OpenBlock returns ErrSkipped (first block in DB)
		mockDB.blocks[blockNum] = fuzzBlockData{
			err: suptypes.ErrSkipped,
		}
		// FirstSealedBlock returns the first block info
		mockDB.firstBlock = suptypes.BlockSeal{
			Hash:      firstBlockHash,
			Number:    blockNum,
			Timestamp: timestamp,
		}

		interop := newFuzzInterop(map[eth.ChainID]LogsDB{chainID: mockDB})

		blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
			chainID: {Number: blockNum, Hash: expectedHash},
		}

		result, err := interop.verifyInteropMessages(timestamp, blocksAtTimestamp)
		require.NoError(t, err)

		// P5: The chain should always be in L2Heads
		require.Contains(t, result.L2Heads, chainID, "P5: chain should be in L2Heads")

		if hashMatch {
			// Hash matches: should be valid
			require.True(t, result.IsValid(), "P5: matching first block hash should be valid")
			require.NotContains(t, result.InvalidHeads, chainID)
		} else {
			// Hash mismatch: should mark as invalid
			require.False(t, result.IsValid(), "P5: mismatching first block hash should be invalid")
			require.Contains(t, result.InvalidHeads, chainID, "P5: chain should be in InvalidHeads on hash mismatch")
		}
	})
}

// =============================================================================
// Fuzz Test: Multiple invalid messages (P6)
// =============================================================================

// FuzzVerifyMultipleInvalidMessages tests that blocks with multiple invalid
// executing messages are still correctly detected as invalid.
//
// Document coverage:
//   INV-3: Cross-validity detection works regardless of number of invalid messages
//
// Properties:
// P6: Block with multiple invalid messages still gets marked invalid
func FuzzVerifyMultipleInvalidMessages(f *testing.F) {
	f.Add(int64(1), 2)
	f.Add(int64(42), 5)
	f.Add(int64(100), 10)

	f.Fuzz(func(t *testing.T, seed int64, numInvalidMsgs int) {
		rng := rand.New(rand.NewSource(seed))

		// Bound the number of invalid messages
		if numInvalidMsgs < 1 {
			numInvalidMsgs = 1
		}
		if numInvalidMsgs > 20 {
			numInvalidMsgs = 20
		}

		execTimestamp := uint64(1000000)
		destBlockHash := randomHash(rng)
		destBlockNum := uint64(100)

		sourceDB := newFuzzMockLogsDB()
		sourceDB.defaultContainsErr = suptypes.ErrConflict

		destDB := newFuzzMockLogsDB()

		execMsgs := generateInvalidExecMsgs(rng, fuzzSourceChainID, numInvalidMsgs, execTimestamp)
		setBlockDataWithMsgs(destDB, destBlockHash, destBlockNum, execTimestamp, execMsgs)

		interop := newFuzzInterop(map[eth.ChainID]LogsDB{
			fuzzSourceChainID: sourceDB,
			fuzzDestChainID:   destDB,
		})

		blocksAtTimestamp := map[eth.ChainID]eth.BlockID{
			fuzzDestChainID: {Number: destBlockNum, Hash: destBlockHash},
		}

		result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
		require.NoError(t, err)

		// P6: Block should be marked invalid regardless of which message was checked first
		require.False(t, result.IsValid(), "P6: block with %d invalid messages should be invalid", numInvalidMsgs)
		require.Contains(t, result.InvalidHeads, fuzzDestChainID, "P6: dest chain should be in InvalidHeads")
	})
}

// =============================================================================
// Fuzz Test: Missing chains silently excluded (P7)
// =============================================================================

// FuzzVerifyMissingChains tests that chains not in logsDBs are silently
// excluded from the Result.
//
// Properties:
// P7: Missing chains in logsDBs are consistently excluded from Result
func FuzzVerifyMissingChains(f *testing.F) {
	f.Add(int64(1), 3, 1)
	f.Add(int64(42), 5, 2)

	f.Fuzz(func(t *testing.T, seed int64, totalChains int, registeredChains int) {
		rng := rand.New(rand.NewSource(seed))

		if totalChains < 1 {
			totalChains = 1
		}
		if totalChains > 10 {
			totalChains = 10
		}
		if registeredChains < 0 {
			registeredChains = 0
		}
		if registeredChains > totalChains {
			registeredChains = totalChains
		}

		execTimestamp := uint64(100000 + rng.Intn(900000))

		chainIDs := generateChainIDs(totalChains, 10, 10)

		// Only register first `registeredChains` chains
		logsDBs := make(map[eth.ChainID]LogsDB)
		blocksAtTimestamp := make(map[eth.ChainID]eth.BlockID)

		for i, chainID := range chainIDs {
			blockHash := randomHash(rng)
			blockNum := uint64(rng.Intn(10000))
			blocksAtTimestamp[chainID] = eth.BlockID{Number: blockNum, Hash: blockHash}

			if i < registeredChains {
				mockDB := newFuzzMockLogsDB()
				setBlockData(mockDB, blockHash, blockNum, execTimestamp)
				logsDBs[chainID] = mockDB
			}
		}

		interop := newFuzzInterop(logsDBs)

		result, err := interop.verifyInteropMessages(execTimestamp, blocksAtTimestamp)
		require.NoError(t, err)

		// P7: Only registered chains should be in L2Heads
		for i, chainID := range chainIDs {
			if i < registeredChains {
				require.Contains(t, result.L2Heads, chainID, "P7: registered chain should be in L2Heads")
			} else {
				require.NotContains(t, result.L2Heads, chainID, "P7: unregistered chain should NOT be in L2Heads")
			}
		}
	})
}

// =============================================================================
// Fuzz Test: Result type properties (P34-P36)
// =============================================================================

// FuzzResultProperties tests the Result type's IsValid, IsEmpty, and
// ToVerifiedResult methods with random data.
//
// Properties:
// P34: Result.IsValid() == (len(InvalidHeads) == 0)
// P35: ToVerifiedResult() strips invalid heads, preserves other fields
// P36: Empty results correctly detected
func FuzzResultProperties(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(0))

	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))

		numL2Heads := rng.Intn(5)
		numInvalidHeads := rng.Intn(3)

		result := Result{
			Timestamp: uint64(rng.Intn(1000000)),
			L1Inclusion: eth.BlockID{
				Hash:   randomHash(rng),
				Number: uint64(rng.Intn(1000)),
			},
			L2Heads:      make(map[eth.ChainID]eth.BlockID),
			InvalidHeads: make(map[eth.ChainID]eth.BlockID),
		}

		// Optionally make it empty
		makeEmpty := rng.Intn(10) == 0
		if makeEmpty {
			result.L1Inclusion = eth.BlockID{}
			numL2Heads = 0
			numInvalidHeads = 0
		}

		for i := 0; i < numL2Heads; i++ {
			chainID := eth.ChainIDFromUInt64(uint64(10 + i*10))
			result.L2Heads[chainID] = eth.BlockID{Hash: randomHash(rng), Number: uint64(rng.Intn(1000))}
		}

		for i := 0; i < numInvalidHeads; i++ {
			chainID := eth.ChainIDFromUInt64(uint64(100 + i*10))
			result.InvalidHeads[chainID] = eth.BlockID{Hash: randomHash(rng), Number: uint64(rng.Intn(1000))}
		}

		// P34: IsValid ↔ no invalid heads
		require.Equal(t, len(result.InvalidHeads) == 0, result.IsValid(), "P34: IsValid should match InvalidHeads emptiness")

		// P36: IsEmpty detection
		isActuallyEmpty := result.L1Inclusion == (eth.BlockID{}) && len(result.L2Heads) == 0 && len(result.InvalidHeads) == 0
		require.Equal(t, isActuallyEmpty, result.IsEmpty(), "P36: IsEmpty should match actual emptiness")

		// P35: ToVerifiedResult strips InvalidHeads
		verified := result.ToVerifiedResult()
		require.Equal(t, result.Timestamp, verified.Timestamp, "P35: timestamp preserved")
		require.Equal(t, result.L1Inclusion, verified.L1Inclusion, "P35: L1Inclusion preserved")
		require.Equal(t, len(result.L2Heads), len(verified.L2Heads), "P35: L2Heads preserved")
		for chainID, blockID := range result.L2Heads {
			require.Equal(t, blockID, verified.L2Heads[chainID], "P35: L2Head for chain %s preserved", chainID)
		}
	})
}
