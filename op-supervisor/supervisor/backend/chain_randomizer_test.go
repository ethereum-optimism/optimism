package backend

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	types2 "github.com/ethereum/go-ethereum/core/types"
	params2 "github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/params"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/processors"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func ExecMsgForLog(chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: processors.LogToPayloadHash(log),
	}
	topics, data := msg.EncodeEvent()
	return &types2.Log{
		Address: params2.InteropCrossL2InboxAddress,
		Data:    data,
		Topics:  topics,
		Index:   log.Index,
	}
}

type ChainBlock struct {
	chain eth.ChainID
	block *eth.L2BlockRef
}

type ChainHeads struct {
	// These are block numbers on the chain
	localSafe   uint64
	localUnsafe uint64
	crossSafe   uint64
	crossUnsafe uint64
}

type RandomChainParams struct {
	chainCount int

	minLength int
	maxLength int

	sameTimestampFrequency int // Percentage [0-100]
	dependencyChance       int // Percentage [0-100]
}

type L1Assignments struct {
	L1Block  eth.BlockRef
	L2Blocks []*ChainBlock
}

type RandomChain struct {
	randomGenerator *rand.Rand
	cutoffs         struct {
		crossUnsafe int
		crossSafe   int
		localUnsafe int
		localSafe   int
	}
	chainIDs      []eth.ChainID
	allBlocks     []*ChainBlock
	cbIndices     map[ChainBlock]int // Lookup for a ChainBlock's index in allBlocks
	generatedLogs map[ChainBlock][]*types2.Log
	dependencies  map[ChainBlock][]*ChainBlock
	chainSources  map[eth.ChainID]*MockProcessorSource
	chainBlocks   map[eth.ChainID][]*eth.L2BlockRef
	chainHeads    map[eth.ChainID]*ChainHeads
	l1SourceMap   map[ChainBlock]eth.BlockRef
	l1Source      map[uint64]eth.BlockRef
}

func (rc *RandomChain) ChainInfo(chainid eth.ChainID) (blocks []*eth.L2BlockRef, heads ChainHeads) {
	blocks = rc.chainBlocks[chainid]
	heads = *rc.chainHeads[chainid]
	return blocks, heads
}

func (p *RandomChainParams) MakeRandomChain(seed int64) (res RandomChain) {
	r := rand.New(rand.NewSource(seed))

	// Add two special blocks to be used when creating invalid dependencies
	totalLength := randomInRange(r, p.minLength, p.maxLength) + 2
	// First block has a timestamp far in the past, already expired (used in InsertDependencyToExpiredMessage)
	expiredBlockIndex := 0
	// Last block has a timestamp in the future (used in InsertFutureDependency)
	futureBlockIndex := totalLength - 1

	// Heads (and candidates) must be between the two special blocks
	localUnsafe := futureBlockIndex - 1
	localSafe := randomInRange(r, expiredBlockIndex+2, futureBlockIndex)
	crossSafe := randomInRange(r, expiredBlockIndex+1, localSafe)
	crossUnsafe := randomInRange(r, crossSafe, localUnsafe)
	res = RandomChain{
		randomGenerator: r,
		cutoffs: struct {
			crossUnsafe int
			crossSafe   int
			localUnsafe int
			localSafe   int
		}{
			crossUnsafe: crossUnsafe,
			crossSafe:   crossSafe,
			localUnsafe: localUnsafe,
			localSafe:   localSafe,
		},
		chainIDs:      make([]eth.ChainID, 0, p.chainCount),
		allBlocks:     make([]*ChainBlock, 0, totalLength),
		cbIndices:     make(map[ChainBlock]int),
		generatedLogs: make(map[ChainBlock][]*types2.Log),
		dependencies:  make(map[ChainBlock][]*ChainBlock),
		chainSources:  make(map[eth.ChainID]*MockProcessorSource),
		chainBlocks:   make(map[eth.ChainID][]*eth.L2BlockRef),
		chainHeads:    make(map[eth.ChainID]*ChainHeads),
		l1SourceMap:   make(map[ChainBlock]eth.BlockRef),
		l1Source:      make(map[uint64]eth.BlockRef),
	}

	for i := range p.chainCount {
		chain := eth.ChainIDFromUInt64(testChainIDOffset + uint64(i))
		res.chainBlocks[chain] = make([]*eth.L2BlockRef, 0)
		res.chainSources[chain] = &MockProcessorSource{}
		res.chainHeads[chain] = &ChainHeads{}
		res.chainIDs = append(res.chainIDs, chain)
	}

	//
	// Create array of all blocks
	//
	chainUninit := eth.ChainIDFromUInt64(0)
	timeStampCount := 1 // Can't be greater than p.chainCount
	var newBlock *ChainBlock
	for i := range totalLength {
		allBlocks := res.allBlocks
		if i == 0 {
			// First block has a timestamp far in the past, already expired (used in InsertDependencyToExpiredMessage)
			randomBlock := testutils.RandomL2BlockRef(r)
			randomBlock.Time = 0
			newBlock = &ChainBlock{chainUninit, &randomBlock}
		} else if i == 1 {
			// Set the initial timestamp so that the block at index 0 is already expired
			randomBlock := testutils.NextRandomL2Ref(r, 100, *allBlocks[0].block, eth.BlockID{})
			randomBlock.Time = params.MessageExpiryTimeSecondsInterop + 1
			newBlock = &ChainBlock{chainUninit, &randomBlock}
		} else {
			// Use NextRandomRef for timestamp coherence.
			randomBlock := testutils.NextRandomL2Ref(r, 100, *allBlocks[len(allBlocks)-1].block, eth.BlockID{})

			// Repeat timestamps with some probability, with two caveats:
			// - Can only have one block per chain with the same timestamp,
			// - Last block must have a unique future timestamp, so it can be used in InsertFutureDependency.
			if timeStampCount < p.chainCount && i < futureBlockIndex && r.Intn(100) < p.sameTimestampFrequency {
				randomBlock.Time = allBlocks[len(allBlocks)-1].block.Time
				timeStampCount++
			} else {
				randomBlock.Time += 1 // Increment because NextRandomRef could return a block with the same timestamp
				timeStampCount = 1
			}
			newBlock = &ChainBlock{chainUninit, &randomBlock}
		}
		res.allBlocks = append(res.allBlocks, newBlock)
	}

	//
	// Assign blocks to random L2 chains
	//
	chainSelections := make([]eth.ChainID, p.chainCount)
	copy(chainSelections, res.chainIDs)
	shuffleChains := func() {
		r.Shuffle(len(chainSelections), func(i, j int) {
			chainSelections[i], chainSelections[j] = chainSelections[j], chainSelections[i]
		})
	}

	nextChain := 0
	var prevBlock *eth.L2BlockRef
	for i, cb := range res.allBlocks {
		block := cb.block
		if i == 0 || prevBlock.Time != block.Time {
			shuffleChains()
			nextChain = 0
		}
		chainid := chainSelections[nextChain]
		cb.chain = chainid
		nextChain++

		if len(res.chainBlocks[chainid]) == 0 {
			block.Number = 0
			block.ParentHash = common.Hash{}
		} else {
			chainBlocks := res.chainBlocks[chainid]
			lastblock := chainBlocks[len(chainBlocks)-1]
			block.Number = lastblock.Number + 1
			block.ParentHash = lastblock.Hash
		}

		// Assign the cross/local heads based on where the cutoffs are
		if i <= res.cutoffs.localSafe {
			res.chainHeads[chainid].localSafe = block.Number
		}
		if i <= res.cutoffs.localUnsafe {
			res.chainHeads[chainid].localUnsafe = block.Number
		}
		if i <= res.cutoffs.crossSafe {
			res.chainHeads[chainid].crossSafe = block.Number
		}
		if i <= res.cutoffs.crossUnsafe {
			res.chainHeads[chainid].crossUnsafe = block.Number
		}

		res.cbIndices[*cb] = i
		res.chainSources[chainid].ExpectL2BlockRefByNumber(block.Number, *block, nil)
		res.chainBlocks[chainid] = append(res.chainBlocks[chainid], block)
		prevBlock = block
	}

	//
	// Create random dependencies between all blocks
	//
	for initIndex, initcb := range res.allBlocks {
		// Add an unimportant message at index 0 that can be modified later by the InsertCycle function
		addRandomInitiatingMessage(r, &res, initcb)

		block := initcb.block
		if block.Number == 0 {
			continue
		}

		for r.Intn(100) < p.dependencyChance {
			execIndex := randomInRange(r, initIndex, totalLength)
			execcb := res.allBlocks[execIndex]
			if block.Number == 0 {
				continue
			}
			res.dependencies[*execcb] = append(res.dependencies[*execcb], initcb)
		}
	}

	// Add dependencies for candidates
	candidateDependencyChance := p.dependencyChance
	crossUnsafeCandidate := GetCrossUnsafeCandidate(res)
	crossSafeCandidate := GetCrossSafeCandidate(res)

	addCandidateDeps := func(candidate *ChainBlock) {
		if candidate != nil {
			time := candidate.block.Time
			candidateIndex := res.cbIndices[*candidate]
			index := candidateIndex - 1
			// Find earliest block with the same timestamp as the candidate
			for res.allBlocks[index].block.Time == time {
				index--
			}
			// Iterate over this range of blocks and add dependencies between them
			for i := candidateIndex; index+1 < i; i-- {
				for r.Intn(100) < candidateDependencyChance {
					execcb := res.allBlocks[i]
					dependencyIndex := randomInRange(r, index+1, i)
					initcb := res.allBlocks[dependencyIndex]
					if initcb.block.Number == 0 {
						continue
					}
					res.dependencies[*execcb] = append(res.dependencies[*execcb], initcb)
				}
			}
		}
	}

	addCandidateDeps(crossUnsafeCandidate)
	addCandidateDeps(crossSafeCandidate)

	// Construct the dependencies by creating initiating/executing message pairs
	for _, execcb := range res.allBlocks {
		for _, initcb := range res.dependencies[*execcb] {
			initiatingLog := addRandomInitiatingMessage(r, &res, initcb)
			addExecutingMessage(&res, execcb, initcb, initiatingLog)
		}
	}

	//
	// Make L1 derivation info
	//
	taken := 0
	nextL1 := testutils.RandomBlockRef(r)
	for taken < totalLength {
		nextL1 = testutils.NextRandomRef(r, nextL1)
		take := randomInRange(r, 1, 5) // Take 1-4 L2 blocks
		take = min(totalLength-taken, take)
		for _, l2Block := range res.allBlocks[taken : taken+take] {
			res.l1SourceMap[*l2Block] = nextL1
		}
		res.l1Source[nextL1.Number] = nextL1
		taken += take
	}

	return res
}

func addRandomInitiatingMessage(r *rand.Rand, res *RandomChain, initcb *ChainBlock) *types2.Log {
	initiatingLog := testutils.RandomLog(r)
	initiatingLog.Index = uint(len(res.generatedLogs[*initcb]))
	res.generatedLogs[*initcb] = append(res.generatedLogs[*initcb], initiatingLog)
	return initiatingLog
}

func addExecutingMessage(res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := ExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(res.generatedLogs[*execcb]))
	res.generatedLogs[*execcb] = append(res.generatedLogs[*execcb], execLog)
}

func addExecutingMessageWithDependency(res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	addExecutingMessage(res, execcb, initcb, initiatingLog)
	res.dependencies[*execcb] = append(res.dependencies[*execcb], initcb)
}

func addInvalidExecutingMessage(r *rand.Rand, res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := InvalidExecMsgForLog(r, res, initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = uint(len(res.generatedLogs[*execcb]))
	res.generatedLogs[*execcb] = append(res.generatedLogs[*execcb], execLog)
}

func insertExecutingMessageAt(i uint, res *RandomChain, execcb *ChainBlock, initcb *ChainBlock, initiatingLog *types2.Log) {
	execLog := ExecMsgForLog(initcb.chain, *initcb.block, initiatingLog)
	execLog.Index = i
	res.generatedLogs[*execcb][i] = execLog
}

func GenerateReceiptsFromLogs(res *RandomChain) {
	for _, cb := range res.allBlocks {
		chain, block := cb.chain, cb.block
		logs := res.generatedLogs[*cb]
		rcpt := types2.Receipt{
			Logs: logs,
		}
		source := res.chainSources[chain]
		source.ExpectFetchReceipts(block.Hash, types2.Receipts{&rcpt}, nil)
	}
}

// Returns a random integer in the interval [lowerIncluding, upperExcluding)
func randomInRange(r *rand.Rand, lowerIncluding int, upperExcluding int) int {
	return r.Intn(upperExcluding-lowerIncluding) + lowerIncluding
}

func InvalidExecMsgForLog(r *rand.Rand, res *RandomChain, chain eth.ChainID, block eth.L2BlockRef, log *types2.Log) *types2.Log {
	msg := types.Message{
		Identifier: types.Identifier{
			Origin:      log.Address,
			BlockNumber: block.Number,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time,
			ChainID:     chain,
		},
		PayloadHash: processors.LogToPayloadHash(log),
	}

	switch r.Intn(5) {
	case 0:
		// Invalid origin
		msg.Identifier.Origin = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	case 1:
		// Invalid block number
		msg.Identifier.BlockNumber += uint64(randomInRange(r, 1, 10))
	case 2:
		// Invalid log index
		msg.Identifier.LogIndex += uint32(randomInRange(r, 1, 5))
	case 3:
		// Invalid timestamp
		msg.Identifier.Timestamp -= uint64(randomInRange(r, 1, 100))
	case 4:
		// Invalid chain ID
		impossibleChainID := testChainIDOffset + len(res.chainIDs)
		msg.Identifier.ChainID = eth.ChainIDFromUInt64(uint64(impossibleChainID))
	}

	topics, data := msg.EncodeEvent()
	return &types2.Log{
		Address: params2.InteropCrossL2InboxAddress,
		Data:    data,
		Topics:  topics,
		Index:   log.Index,
	}
}

func InsertMessageWithInvalidIdentifier(r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidateBlock := res.allBlocks[candidateIndex]
	randomIndex := r.Intn(candidateIndex + 1)
	randomBlock := res.allBlocks[randomIndex]
	randomLogIndex := r.Intn(len(res.generatedLogs[*randomBlock]))
	randomLog := res.generatedLogs[*randomBlock][randomLogIndex]

	addInvalidExecutingMessage(r, res, candidateBlock, randomBlock, randomLog)
}

func InvalidateBlock(t *testing.T, res *RandomChain, candidate *ChainBlock) {
	r := res.randomGenerator
	switch r.Intn(5) {
	case 0:
		InsertCycle(t, r, res, candidate)
	case 1:
		InsertSelfDependency(r, res, candidate)
	case 2:
		InsertFutureDependency(t, r, res, res.cbIndices[*candidate])
	case 3:
		InsertDependencyToExpiredMessage(t, r, res, res.cbIndices[*candidate])
	case 4:
		InsertMessageWithInvalidIdentifier(r, res, res.cbIndices[*candidate])
	default:
	}
}

func InsertFutureDependency(t *testing.T, r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidateBlock := res.allBlocks[candidateIndex]
	t.Logf("Inserting a future dependency in candidate (%s, %2d)'s hazard set", candidateBlock.chain, candidateBlock.block.Number)

	// Find the next block with a timestamp in the future (guaranteed to exist since we added a special block at the end)
	i := candidateIndex + 1
	for res.allBlocks[i].block.Time <= candidateBlock.block.Time {
		i++
	}

	// Randomly pick a future block and create an executing message to it
	futureIndex := randomInRange(r, i, len(res.allBlocks))
	futureBlock := res.allBlocks[futureIndex]
	initiatingLog := addRandomInitiatingMessage(r, res, futureBlock)
	addExecutingMessageWithDependency(res, candidateBlock, futureBlock, initiatingLog)
}

func InsertDependencyToExpiredMessage(t *testing.T, r *rand.Rand, res *RandomChain, candidateIndex int) {
	candidate := res.allBlocks[candidateIndex]

	// We set the timestamps so that this is true for every block that can be selected as candidate
	require.Less(t, uint64(params.MessageExpiryTimeSecondsInterop), candidate.block.Time)

	// Any timestamp below this is expired
	expiryTimestamp := candidate.block.Time - params.MessageExpiryTimeSecondsInterop

	// Iterate until we find the first unexpired block
	i := 0
	for res.allBlocks[i].block.Time < expiryTimestamp {
		i++
	}

	// i is at least 1 since the block at index 0 is guaranteed to be expired
	expiredIndex := r.Intn(i)
	expiredBlock := res.allBlocks[expiredIndex]
	initiatingLog := addRandomInitiatingMessage(r, res, expiredBlock)
	addExecutingMessageWithDependency(res, candidate, expiredBlock, initiatingLog)
}

func InsertSelfDependency(r *rand.Rand, res *RandomChain, candidate *ChainBlock) {
	// Create a random initiating message to be inserted at index N+1
	initiatingLog := testutils.RandomLog(r)
	initiatingLog.Index = uint(len(res.generatedLogs[*candidate]) + 1)

	// Insert executing message at index N
	addExecutingMessageWithDependency(res, candidate, candidate, initiatingLog)

	// Insert initiating message at index N+1
	res.generatedLogs[*candidate] = append(res.generatedLogs[*candidate], initiatingLog)
}

func listHazards(t *testing.T, res *RandomChain, candidate *ChainBlock) []*ChainBlock {
	hazards := make([]*ChainBlock, 0)
	includedHazards := make(map[eth.ChainID]*ChainBlock)

	// Add the candidate itself as a hazard
	stack := []*ChainBlock{candidate}

	for len(stack) > 0 {
		// Pop hazard from the stack
		hazard := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Check if we already found a hazard from this chain
		includedHazard, ok := includedHazards[hazard.chain]
		if ok {
			// Ensure that there are not two different hazards from the same chain
			require.Equal(t, includedHazard.block.ID(), hazard.block.ID())
		} else {
			// If not already included, add hazard to the list
			hazards = append(hazards, hazard)
			includedHazards[hazard.chain] = hazard

			// For each new hazard, add all dependencies with the same timestamp to the stack
			for _, dependency := range res.dependencies[*hazard] {
				if dependency.block.Time == candidate.block.Time {
					stack = append(stack, dependency)
				}
			}
		}
	}

	return hazards
}

func InsertCycle(t *testing.T, r *rand.Rand, res *RandomChain, candidate *ChainBlock) {
	t.Logf("Inserting a cycle in candidate (%s, %2d)'s hazard set", candidate.chain, candidate.block.Number)

	candidateHazards := listHazards(t, res, candidate)
	t.Logf("Size of (%s, %2d)'s hazard set: %d", candidate.chain, candidate.block.Number, len(candidateHazards))
	cycleStart := candidateHazards[r.Intn(len(candidateHazards))]
	t.Logf("Picked random hazard set element to start the cycle: (%s, %2d)", cycleStart.chain, cycleStart.block.Number)

	// If the random element is equal to the candidate, no need to compute the hazards again
	var subHazards []*ChainBlock
	if cycleStart.chain == candidate.chain {
		require.Equal(t, cycleStart.block.Number, candidate.block.Number)
		subHazards = candidateHazards
	} else {
		subHazards = listHazards(t, res, cycleStart)
		t.Logf("Size of (%s, %2d)'s hazard set: %d", cycleStart.chain, cycleStart.block.Number, len(subHazards))
	}

	cycleEnd := subHazards[r.Intn(len(subHazards))]
	t.Logf("Picked random hazard set element to end the cycle: (%s, %2d)", cycleEnd.chain, cycleEnd.block.Number)

	// Add executing message from first log of cycleEnd to last log of cycleStart
	lastIndex := len(res.generatedLogs[*cycleStart]) - 1
	initiatingLog := res.generatedLogs[*cycleStart][lastIndex]
	// Replace dummy message at index 0
	insertExecutingMessageAt(0, res, cycleEnd, cycleStart, initiatingLog)
	res.dependencies[*cycleEnd] = append(res.dependencies[*cycleEnd], cycleStart)
	t.Logf("Added cyclic dependency: (%s, %2d) -> (%s, %2d)", cycleEnd.chain, cycleEnd.block.Number, cycleStart.chain, cycleStart.block.Number)
}

func GetCrossUnsafeCandidate(rc RandomChain) (block *ChainBlock) {
	for _, chain := range rc.chainIDs {
		if rc.chainHeads[chain].crossUnsafe < rc.chainHeads[chain].localUnsafe {
			return &ChainBlock{
				chain: chain,
				block: rc.chainBlocks[chain][rc.chainHeads[chain].crossUnsafe+1],
			}
		}
	}
	return nil
}

func GetCrossSafeCandidate(rc RandomChain) (block *ChainBlock) {
	for _, chain := range rc.chainIDs {
		if rc.chainHeads[chain].crossSafe < rc.chainHeads[chain].localSafe {
			return &ChainBlock{
				chain: chain,
				block: rc.chainBlocks[chain][rc.chainHeads[chain].crossSafe+1],
			}
		}
	}
	return nil
}
