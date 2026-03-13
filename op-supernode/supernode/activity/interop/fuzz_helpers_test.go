package interop

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

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