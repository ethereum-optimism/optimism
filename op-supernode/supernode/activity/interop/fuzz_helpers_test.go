package interop

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Document Invariant → Fuzz Test Coverage Mapping
// =============================================================================
//
// Reference: Architecture document "Supernode State and Invariants"
//
// INVARIANTS (after cross-validation for timestamp t, before t+1):
//
// INV-1: ℓ^j_i is the list of logs for block B^j_i (logs match blocks)
//   → FuzzProcessBlockLogs (P11, P12): Verifies AddLog called per log with correct
//     indices and parent block. SealBlock called with correct parent hash.
//   ⚠ Gap: Log content/hash correctness not verified.
//
// INV-2: B^j_i is parent of B^j_{i+1} (chain continuity in LogsDB)
//   → FuzzProcessBlockLogs (P11): Parent hash handling for first block (virtual
//     parent seal) and subsequent blocks.
//   ⚠ Gap: Multi-block chain continuity across timestamps not tested.
//
// INV-3: Every B^j_i is cross-valid (all executing messages valid, no cycles)
//   → FuzzVerifyInteropMessagesValid (P1, P3): Valid messages → valid result.
//   → FuzzVerifyInteropMessagesFails (P2): All 5 invalidation types detected
//     (unknown source, timestamp violation, expired, conflict, hash mismatch).
//   → FuzzVerifyExpiryBoundary (P4): Exact expiry boundary correctness.
//   → FuzzVerifyExpiryOverflow (P4-overflow): Documents uint64 overflow bug in
//     algo.go:167 (initTS + ExpiryTime wraps, falsely expires valid messages).
//   → FuzzVerifyMultipleInvalidMessages (P6): Multiple invalid msgs in one block.
//   → FuzzProgressInteropValid (P28, P29): Valid multi-chain → committed to VerifiedDB.
//
// INV-4: C^j_{t_0} = B^j_0 (first verified head = first logsDB block)
//   ⚠ Not verified: Would require correlating VerifiedDB entries with LogsDB
//     state at activation timestamp. Mocking strategy pre-seals logsDB.
//
// INV-5: C^j_t = B^j_{n_j} (last verified head = last logsDB block)
//   ⚠ Not verified: Same reason as INV-4 — LogsDB is pre-sealed in fuzz tests.
//
// INV-6: C^j_i ∈ {C^j_{i+1}, parent(C^j_{i+1})} (L2 heads monotonic)
//   → FuzzProgressAndRecordSequential (P13): Sequential timestamps commit correctly.
//   → FuzzProgressInteropValid (P28): Sequential processing verified via VerifiedDB.
//   ⚠ Partial: Checks VerifiedDB sequencing, not that L2 head advances by ≤1 block.
//
// INV-7: C^j_i is highest block on chain j with timestamp ≤ i
//   → FuzzProgressAndRecord case 0 (P8): Uses LocalSafeBlockAtTimestamp return value.
//   ⚠ Partial: Relies on mock returning correct block; doesn't verify the query logic.
//
// INV-8: C_{t_0}, ..., C_t on same linear L1 chain (no L1 forks in history)
//   → FuzzProgressAndRecord case 3 (P11): currentL1 = min of collected L1s.
//   → FuzzProgressAndRecordSequential (P14): Tracks L1Inclusion across steps.
//   ⚠ Not verified: Linear chain property (parent links). Only tracks L1 numbers.
//
// INV-9: C_i = max(B'_1, ..., B'_k) where B'_j is L1 derivation block of C^j_i
//   → FuzzProgressAndRecord case 0 (P8): currentL1 = result.L1Inclusion after valid.
//   → FuzzProgressAndRecordSequential (P14): currentL1 tracks L1Inclusion each step.
//   ⚠ Partial: Verified at orchestration level; derivation-level max not tested.
//
// INV-10: B^j_i ∉ D_j (logsDB/verifiedDB blocks never in DenyList)
//   ⚠ Not verified: DenyList is inside ChainContainer, fully mocked in fuzz tests.
//
// INV-11: ∀B ∈ D_j: timestamp(B) ≤ t+1 (DenyList only has near-future blocks)
//   ⚠ Not verified: DenyList is inside ChainContainer, fully mocked in fuzz tests.
//
// ASSUMPTIONS ON L1/L2 CHAINS:
//
// A1: At most one block per chain per timestamp (no duplicate timestamps)
//   → Implicitly assumed in all tests (one block per chain in BlocksAtTimestamp).
//
// A2: L2 reorgs only if L1 reorgs
//   ⚠ Not tested: Requires L1/L2 reorg simulation beyond interop scope.
//
// A3: L2 eventually syncs to L1 (given sufficient time)
//   → FuzzProgressAndRecord case 3 (P11): Models "not yet synced" via NotFound.
//
// A4: DenyList causes deposit-only block replacement
//   ⚠ Not tested: VirtualNode behavior is outside interop package scope.
//
// A5: Queries can return arbitrary results (due to concurrent reorgs)
//   → FuzzProgressAndRecord cases 4,5 (P12, P15): Error propagation from chain queries.
//   ⚠ Partial: Tests error paths, not arbitrary-but-plausible results.
//
// STATE CHANGES FROM CROSS-VALIDATION:
//
// Step 1: For each chain j, get highest block B_j with ts ≤ t+1; wait if not ready
//   → FuzzProgressAndRecord case 0 (P8): Chains ready → valid result committed.
//   → FuzzProgressAndRecord case 3 (P11): Chains not ready → no progress, no error.
//   → FuzzVerifyCanAddTimestamp (P9, P13): Gap detection and activation ts handling.
//
// Step 2: Verify B'_j on same linear L1 chain; if not, wait for L2 to sync
//   → FuzzProgressAndRecord case 3 (P11): currentL1 = min of collected L1s.
//   ⚠ Not verified: L1 linear chain consistency check across rounds.
//
// Step 3: If C_t reorged out → rollback VerifiedDB, prune DenyList, prune LogsDB
//   → FuzzProgressInteropReset (P32): Reset rewinds logsDB and verifiedDB,
//     clears currentL1, verifies entries before/after rewind point,
//     can resume committing at rewindTS+1.
//   → FuzzVerifiedDBCommitRewind (P16-P18): VerifiedDB rewind invariants.
//   ⚠ Not verified: DenyList pruning on reorg (DenyList is mocked).
//
// Step 4: If any B_j invalid → add to DenyList, reset ChainContainer
//   → FuzzProgressInteropInvalid (P29): invalidateBlock called per invalid chain,
//     NOT called for valid chains.
//   → FuzzProgressInteropInvalid (P31): Can commit at same timestamp after invalidation.
//   → FuzzProgressAndRecord case 1 (P9): Invalid from verifyFn → invalidateBlock called,
//     currentL1 unchanged, madeProgress=false.
//   → FuzzProgressAndRecord case 2 (P10): Invalid from cycleVerifyFn → merged invalids.
//   → FuzzProgressAndRecord case 6 (P16): invalidateBlock error propagation.
//   ⚠ Not verified: DenyList addition, VirtualNode destruction/recreation (mocked).
//
// Step 5: If all valid → extend VerifiedDB with (t+1, C_{t+1}, C^1_{t+1}, ..., C^k_{t+1})
//   → FuzzProgressAndRecord case 0 (P8): verifiedDB.Has(ts), currentL1 = L1Inclusion.
//   → FuzzProgressAndRecordSequential (P13, P14): Sequential multi-step commits.
//   → FuzzProgressInteropValid (P28, P29): All timestamps committed sequentially.
//   → FuzzVerifiedDBCommitRewind (P15, P19, P20): Sequential commit enforcement,
//     error discrimination, JSON round-trip.
//   → FuzzVerifiedDBFirstCommit: First commit at any timestamp succeeds.
//   → FuzzVerifiedDBPersistence: Data survives close/reopen (P20).
//
// ADDITIONAL PROPERTIES (beyond document invariants):
//
// P5:  ErrSkipped path (first block in logsDB)
//   → FuzzVerifyFirstBlockSkipped: Hash match/mismatch on first sealed block.
//
// P7:  Missing chains silently excluded from Result
//   → FuzzVerifyMissingChains: Chains not in logsDBs not in L2Heads.
//
// P17: pauseAtTimestamp prevents progress
//   → FuzzProgressAndRecord case 7: No progress, no error, verifyFn never called.
//
// P30: Empty results are no-ops
//   → FuzzHandleResultEmpty: Empty result doesn't modify verifiedDB state.
//
// P34-P36: Result type algebraic properties
//   → FuzzResultProperties: IsValid ↔ no InvalidHeads, IsEmpty, ToVerifiedResult.
//
// COVERAGE GAPS SUMMARY:
//   - INV-4, INV-5: VerifiedDB ↔ LogsDB head correspondence (pre-sealed mocks)
//   - INV-10, INV-11: DenyList invariants (DenyList fully mocked)
//   - INV-8 (linear L1): Only number tracking, no parent-link verification
//   - A2, A4: L1/L2 reorg propagation and deposit-only replacement (VirtualNode scope)
//   - Step 2: L1 linear chain consistency check between consecutive rounds
//   - Multi-block LogsDB chain continuity (INV-2 across timestamps)
//
// =============================================================================
// Shared fuzz test helpers — reusable generators and builders
// =============================================================================

// randomHash generates a random common.Hash from the given rng.
func randomHash(rng *rand.Rand) common.Hash {
	var h common.Hash
	rng.Read(h[:])
	return h
}

// generateChainIDs creates a slice of chain IDs with deterministic values.
// baseID is the starting chain ID; subsequent IDs increment by step.
func generateChainIDs(count int, baseID, step uint64) []eth.ChainID {
	chainIDs := make([]eth.ChainID, count)
	for i := range chainIDs {
		chainIDs[i] = eth.ChainIDFromUInt64(baseID + uint64(i)*step)
	}
	return chainIDs
}

// fuzzChainBlock holds the block parameters for a single chain in a fuzz scenario.
type fuzzChainBlock struct {
	Hash      common.Hash
	Number    uint64
	Timestamp uint64
}

// fuzzChainSetup holds all the state generated for a multi-chain fuzz scenario.
type fuzzChainSetup struct {
	ChainIDs          []eth.ChainID
	LogsDBs           map[eth.ChainID]LogsDB
	MockDBs           map[eth.ChainID]*fuzzMockLogsDB
	BlocksAtTimestamp map[eth.ChainID]eth.BlockID
	ChainBlocks       map[eth.ChainID]fuzzChainBlock
}

// generateChainSetup creates mock LogsDBs and random blocks for each chain.
// All blocks share the same timestamp (execTimestamp).
// Default Contains behavior is set to ErrConflict so only explicitly registered queries succeed.
func generateChainSetup(rng *rand.Rand, chainIDs []eth.ChainID, execTimestamp uint64) fuzzChainSetup {
	setup := fuzzChainSetup{
		ChainIDs:          chainIDs,
		LogsDBs:           make(map[eth.ChainID]LogsDB),
		MockDBs:           make(map[eth.ChainID]*fuzzMockLogsDB),
		BlocksAtTimestamp: make(map[eth.ChainID]eth.BlockID),
		ChainBlocks:       make(map[eth.ChainID]fuzzChainBlock),
	}

	for _, chainID := range chainIDs {
		blockHash := randomHash(rng)
		blockNum := uint64(rng.Intn(10000))

		setup.ChainBlocks[chainID] = fuzzChainBlock{
			Hash:      blockHash,
			Number:    blockNum,
			Timestamp: execTimestamp,
		}

		setup.BlocksAtTimestamp[chainID] = eth.BlockID{Number: blockNum, Hash: blockHash}

		mockDB := newFuzzMockLogsDB()
		mockDB.defaultContainsErr = suptypes.ErrConflict
		setup.MockDBs[chainID] = mockDB
		setup.LogsDBs[chainID] = mockDB
	}

	return setup
}

// generateValidInitTimestamp generates a timestamp that is strictly less than execTimestamp
// and within the ExpiryTime window, suitable for a valid executing message.
func generateValidInitTimestamp(rng *rand.Rand, execTimestamp uint64) uint64 {
	minTimestamp := uint64(0)
	if execTimestamp > ExpiryTime {
		minTimestamp = execTimestamp - ExpiryTime
	}
	initTimestamp := minTimestamp + uint64(rng.Int63n(int64(execTimestamp-minTimestamp)))
	if initTimestamp >= execTimestamp {
		initTimestamp = execTimestamp - 1
	}
	return initTimestamp
}

// generateExecutingMessage creates a random ExecutingMessage with the given parameters.
func generateExecutingMessage(rng *rand.Rand, sourceChain eth.ChainID, initTimestamp uint64, logIdx uint32) *suptypes.ExecutingMessage {
	return &suptypes.ExecutingMessage{
		ChainID:   sourceChain,
		BlockNum:  uint64(rng.Intn(10000)),
		LogIdx:    logIdx,
		Timestamp: initTimestamp,
		Checksum:  suptypes.MessageChecksum(randomHash(rng)),
	}
}

// containsQueryForMessage builds the ContainsQuery that production code constructs
// for the given executing message.
func containsQueryForMessage(execMsg *suptypes.ExecutingMessage) suptypes.ContainsQuery {
	return suptypes.ContainsQuery{
		BlockNum:  execMsg.BlockNum,
		LogIdx:    execMsg.LogIdx,
		Timestamp: execMsg.Timestamp,
		Checksum:  execMsg.Checksum,
	}
}

// registerContainsResult registers a successful Contains result on the source chain's
// mock DB for the given executing message.
func registerContainsResult(mockDBs map[eth.ChainID]*fuzzMockLogsDB, execMsg *suptypes.ExecutingMessage) {
	query := containsQueryForMessage(execMsg)
	mockDBs[execMsg.ChainID].containsResults[query] = fuzzContainsResult{
		seal: suptypes.BlockSeal{Number: execMsg.BlockNum, Timestamp: execMsg.Timestamp},
	}
}

// populateValidMessages generates valid executing messages for each chain's block
// and registers the corresponding Contains queries on source chain mocks.
// Messages reference random source chains from the provided chainIDs.
func populateValidMessages(rng *rand.Rand, setup *fuzzChainSetup, maxMsgsPerBlock int) {
	for _, chainID := range setup.ChainIDs {
		cb := setup.ChainBlocks[chainID]
		mockDB := setup.MockDBs[chainID]

		execMsgs := make(map[uint32]*suptypes.ExecutingMessage)

		for j := 0; j < maxMsgsPerBlock; j++ {
			sourceIdx := rng.Intn(len(setup.ChainIDs))
			sourceChain := setup.ChainIDs[sourceIdx]

			initTimestamp := generateValidInitTimestamp(rng, cb.Timestamp)
			logIdx := uint32(j)
			execMsg := generateExecutingMessage(rng, sourceChain, initTimestamp, logIdx)
			execMsgs[logIdx] = execMsg

			registerContainsResult(setup.MockDBs, execMsg)
		}

		mockDB.blocks[cb.Number] = fuzzBlockData{
			ref:      eth.BlockRef{Hash: cb.Hash, Number: cb.Number, Time: cb.Timestamp},
			logCount: uint32(len(execMsgs)),
			execMsgs: execMsgs,
		}
	}
}

// setBlockData sets a simple block (no executing messages) on a chain's mock DB.
func setBlockData(mockDB *fuzzMockLogsDB, blockHash common.Hash, blockNum, timestamp uint64) {
	mockDB.blocks[blockNum] = fuzzBlockData{
		ref: eth.BlockRef{Hash: blockHash, Number: blockNum, Time: timestamp},
	}
}

// setBlockDataWithMsgs sets a block with executing messages on a chain's mock DB.
func setBlockDataWithMsgs(mockDB *fuzzMockLogsDB, blockHash common.Hash, blockNum, timestamp uint64, execMsgs map[uint32]*suptypes.ExecutingMessage) {
	mockDB.blocks[blockNum] = fuzzBlockData{
		ref:      eth.BlockRef{Hash: blockHash, Number: blockNum, Time: timestamp},
		execMsgs: execMsgs,
	}
}

// newFuzzInterop creates an Interop instance with the given logsDBs for fuzz testing.
func newFuzzInterop(logsDBs map[eth.ChainID]LogsDB) *Interop {
	return &Interop{
		log:     gethlog.New(),
		logsDBs: logsDBs,
	}
}

// generateVerifiedResult creates a VerifiedResult with random L2Heads for each chain.
func generateVerifiedResult(rng *rand.Rand, timestamp uint64, chainIDs []eth.ChainID) VerifiedResult {
	result := VerifiedResult{
		Timestamp:   timestamp,
		L1Inclusion: eth.BlockID{Hash: randomHash(rng), Number: uint64(rng.Intn(1000))},
		L2Heads:     make(map[eth.ChainID]eth.BlockID, len(chainIDs)),
	}
	for _, chainID := range chainIDs {
		result.L2Heads[chainID] = eth.BlockID{Hash: randomHash(rng), Number: uint64(rng.Intn(1000))}
	}
	return result
}

// generateInvalidExecMsgs creates a map of executing messages that will fail
// Contains checks (because no matching query is registered on sourceDB).
func generateInvalidExecMsgs(rng *rand.Rand, sourceChainID eth.ChainID, count int, execTimestamp uint64) map[uint32]*suptypes.ExecutingMessage {
	execMsgs := make(map[uint32]*suptypes.ExecutingMessage, count)
	for i := 0; i < count; i++ {
		logIdx := uint32(i)
		execMsgs[logIdx] = &suptypes.ExecutingMessage{
			ChainID:   sourceChainID,
			BlockNum:  uint64(rng.Intn(10000)),
			LogIdx:    logIdx,
			Timestamp: execTimestamp - 1 - uint64(rng.Intn(int(ExpiryTime-1))),
			Checksum:  suptypes.MessageChecksum(randomHash(rng)),
		}
	}
	return execMsgs
}