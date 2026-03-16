package interop

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// =============================================================================
// Mock ChainContainer for progressAndRecord fuzz testing
// =============================================================================

type fuzzMockChainContainer struct {
	chainID   eth.ChainID
	blockTime uint64

	syncStatus    *eth.SyncStatus
	syncStatusErr error

	// Per-timestamp block lookup (used when localSafeBlocks is populated)
	localSafeBlocks   map[uint64]eth.L2BlockRef
	localSafeBlock    eth.L2BlockRef
	localSafeBlockErr error

	invalidateCalls  []fuzzInvalidateCall
	invalidateRetErr error // if set, InvalidateBlock returns this error
}

type fuzzInvalidateCall struct {
	height      uint64
	payloadHash common.Hash
}

func (m *fuzzMockChainContainer) ID() eth.ChainID               { return m.chainID }
func (m *fuzzMockChainContainer) BlockTime() uint64              { return m.blockTime }
func (m *fuzzMockChainContainer) Start(_ context.Context) error  { return nil }
func (m *fuzzMockChainContainer) Stop(_ context.Context) error   { return nil }
func (m *fuzzMockChainContainer) Pause(_ context.Context) error  { return nil }
func (m *fuzzMockChainContainer) Resume(_ context.Context) error { return nil }
func (m *fuzzMockChainContainer) RegisterVerifier(_ activity.VerificationActivity) {
}
func (m *fuzzMockChainContainer) SetResetCallback(_ cc.ResetCallback) {}
func (m *fuzzMockChainContainer) RewindEngine(_ context.Context, _ uint64, _ eth.BlockRef) error {
	return nil
}
func (m *fuzzMockChainContainer) OutputRootAtL2BlockNumber(_ context.Context, _ uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *fuzzMockChainContainer) OptimisticOutputAtTimestamp(_ context.Context, _ uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *fuzzMockChainContainer) VerifiedAt(_ context.Context, _ uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *fuzzMockChainContainer) OptimisticAt(_ context.Context, _ uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *fuzzMockChainContainer) IsDenied(_ uint64, _ common.Hash) (bool, error) {
	return false, nil
}

func (m *fuzzMockChainContainer) SyncStatus(_ context.Context) (*eth.SyncStatus, error) {
	return m.syncStatus, m.syncStatusErr
}

func (m *fuzzMockChainContainer) LocalSafeBlockAtTimestamp(_ context.Context, ts uint64) (eth.L2BlockRef, error) {
	if m.localSafeBlockErr != nil {
		return eth.L2BlockRef{}, m.localSafeBlockErr
	}
	if m.localSafeBlocks != nil {
		if block, ok := m.localSafeBlocks[ts]; ok {
			return block, nil
		}
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return m.localSafeBlock, nil
}

func (m *fuzzMockChainContainer) FetchReceipts(_ context.Context, _ eth.BlockID) (eth.BlockInfo, gethTypes.Receipts, error) {
	// Always nil — loadLogs skips because logsDB is pre-sealed beyond the test timestamp
	return nil, nil, nil
}

func (m *fuzzMockChainContainer) InvalidateBlock(_ context.Context, height uint64, payloadHash common.Hash) (bool, error) {
	m.invalidateCalls = append(m.invalidateCalls, fuzzInvalidateCall{height: height, payloadHash: payloadHash})
	if m.invalidateRetErr != nil {
		return false, m.invalidateRetErr
	}
	return true, nil
}

var _ cc.ChainContainer = (*fuzzMockChainContainer)(nil)

// =============================================================================
// Setup helper
// =============================================================================

type fuzzProgressSetup struct {
	interop    *Interop
	verifiedDB *VerifiedDB
	mockChains map[eth.ChainID]*fuzzMockChainContainer
	chainIDs   []eth.ChainID
}

// newFuzzProgressSetup creates a fully wired Interop for progressAndRecord testing.
// LogsDBs are pre-sealed so loadLogs always skips (no chain I/O needed).
// Uses bbolt-backed VerifiedDB with t.TempDir() for correctness.
func newFuzzProgressSetup(t *testing.T, rng *rand.Rand, numChains int, activationTS uint64) *fuzzProgressSetup {
	t.Helper()

	chainIDs := generateChainIDs(numChains, 10, 10)

	verifiedDB, err := OpenVerifiedDB(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { verifiedDB.Close() })

	chains := make(map[eth.ChainID]cc.ChainContainer)
	logsDBs := make(map[eth.ChainID]LogsDB)
	mockChains := make(map[eth.ChainID]*fuzzMockChainContainer)

	for i, chainID := range chainIDs {
		blockHash := randomHash(rng)
		blockNum := uint64(100 + rng.Intn(1000))

		// Pre-seal the logsDB so loadLogs sees latestBlock >= block and skips
		mockDB := newFuzzMockLogsDB()
		mockDB.hasSealed = true
		mockDB.latestSealed = eth.BlockID{Number: blockNum + 10, Hash: blockHash}
		mockDB.sealedBlocks[blockNum+10] = suptypes.BlockSeal{
			Number:    blockNum + 10,
			Hash:      blockHash,
			Timestamp: activationTS + 100,
		}
		logsDBs[chainID] = mockDB

		// Use distinct L1 numbers per chain to make min-L1 deterministic
		// (avoids flaky assertions when two chains share the same L1 number but different hashes)
		l1Number := uint64(100 + i*100 + rng.Intn(50))

		mockChain := &fuzzMockChainContainer{
			chainID:   chainID,
			blockTime: 2,
			syncStatus: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{
					Number: l1Number,
					Hash:   randomHash(rng),
				},
			},
			localSafeBlock: eth.L2BlockRef{
				Number: blockNum,
				Hash:   blockHash,
				Time:   activationTS,
			},
		}
		chains[chainID] = mockChain
		mockChains[chainID] = mockChain
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	interop := &Interop{
		log:                 gethlog.New(),
		chains:              chains,
		logsDBs:             logsDBs,
		verifiedDB:          verifiedDB,
		activationTimestamp: activationTS,
		ctx:                 ctx,
	}

	return &fuzzProgressSetup{
		interop:    interop,
		verifiedDB: verifiedDB,
		mockChains: mockChains,
		chainIDs:   chainIDs,
	}
}

// =============================================================================
// Fuzz Test: progressAndRecord orchestration (P8-P12)
// =============================================================================

// FuzzProgressAndRecord tests the orchestration logic of progressAndRecord
// under eight scenario classes driven by the fuzzer.
//
// Document coverage:
//   Step 1: case 0 (chains ready → progress), case 3 (chains not ready → wait)
//   Step 2: case 3 (P11: currentL1 = min of collected L1s) [INV-8 partial]
//   Step 4: case 1 (P9: verifyFn invalids), case 2 (P10: cycleVerifyFn invalids),
//           case 6 (P16: invalidateBlock error)
//   Step 5: case 0 (P8: valid → extend VerifiedDB, update currentL1) [INV-9]
//   INV-7:  case 0 uses LocalSafeBlockAtTimestamp (highest block with ts ≤ t)
//   A3:     case 3 models "not yet synced" via NotFound
//   A5:     cases 4,5 model error propagation from chain queries
//
// Properties:
// P8:  Valid result → madeProgress=true, verifiedDB.Has(ts), currentL1=L1Inclusion
// P9:  Invalid from verifyFn → madeProgress=false, invalidateBlock called, currentL1 unchanged
// P10: Invalid from cycleVerifyFn → merge produces invalid result, invalidateBlock called
// P11: Chains not ready → madeProgress=false, currentL1=collected L1
// P12: collectCurrentL1 error → propagates, madeProgress=false
// P15: verifyFn error → propagates, madeProgress=false
// P16: invalidateBlock error → propagates through handleResult
// P17: pauseAtTimestamp → madeProgress=false, verifyFn not called
func FuzzProgressAndRecord(f *testing.F) {
	f.Add(int64(1), uint8(0), uint8(2))
	f.Add(int64(42), uint8(1), uint8(3))
	f.Add(int64(100), uint8(2), uint8(2))
	f.Add(int64(200), uint8(3), uint8(4))
	f.Add(int64(300), uint8(4), uint8(2))
	f.Add(int64(400), uint8(5), uint8(3))
	f.Add(int64(500), uint8(6), uint8(2))
	f.Add(int64(600), uint8(7), uint8(2))

	f.Fuzz(func(t *testing.T, seed int64, scenarioRaw uint8, numChainsRaw uint8) {
		rng := rand.New(rand.NewSource(seed))
		numChains := 2 + int(numChainsRaw%4) // 2-5
		scenario := scenarioRaw % 8
		activationTS := uint64(100000)

		setup := newFuzzProgressSetup(t, rng, numChains, activationTS)

		switch scenario {
		case 0: // Valid result
			l1Inclusion := eth.BlockID{Number: uint64(rng.Intn(1000)), Hash: randomHash(rng)}

			setup.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{
					Timestamp:    ts,
					L1Inclusion:  l1Inclusion,
					L2Heads:      blocks,
					InvalidHeads: make(map[eth.ChainID]eth.BlockID),
				}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{InvalidHeads: make(map[eth.ChainID]eth.BlockID)}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.NoError(t, err)
			require.True(t, madeProgress, "P8: valid result should advance")

			has, err := setup.verifiedDB.Has(activationTS)
			require.NoError(t, err)
			require.True(t, has, "P8: verifiedDB should have committed timestamp")

			require.Equal(t, l1Inclusion, setup.interop.CurrentL1(),
				"P8: currentL1 should equal result.L1Inclusion after valid advance")

		case 1: // Invalid from verifyFn
			numInvalid := 1 + rng.Intn(numChains)
			if numInvalid > numChains {
				numInvalid = numChains
			}

			setup.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				result := Result{
					Timestamp:    ts,
					L1Inclusion:  eth.BlockID{Number: uint64(rng.Intn(1000)), Hash: randomHash(rng)},
					L2Heads:      blocks,
					InvalidHeads: make(map[eth.ChainID]eth.BlockID),
				}
				count := 0
				for chainID, block := range blocks {
					if count < numInvalid {
						result.InvalidHeads[chainID] = block
						count++
					}
				}
				return result, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{InvalidHeads: make(map[eth.ChainID]eth.BlockID)}, nil
			}

			prevL1 := setup.interop.CurrentL1()
			madeProgress, err := setup.interop.progressAndRecord()
			require.NoError(t, err)
			require.False(t, madeProgress, "P9: invalid result should not advance")

			totalCalls := 0
			for _, mc := range setup.mockChains {
				totalCalls += len(mc.invalidateCalls)
			}
			require.Equal(t, numInvalid, totalCalls, "P9: invalidateBlock count must match invalid heads")

			require.Equal(t, prevL1, setup.interop.CurrentL1(),
				"P9: currentL1 should not change on invalid result")

		case 2: // Invalid from cycleVerifyFn (merge)
			targetChain := setup.chainIDs[rng.Intn(len(setup.chainIDs))]

			setup.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{
					Timestamp:    ts,
					L1Inclusion:  eth.BlockID{Number: uint64(rng.Intn(1000)), Hash: randomHash(rng)},
					L2Heads:      blocks,
					InvalidHeads: make(map[eth.ChainID]eth.BlockID),
				}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{
					InvalidHeads: map[eth.ChainID]eth.BlockID{
						targetChain: blocks[targetChain],
					},
				}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.NoError(t, err)
			require.False(t, madeProgress, "P10: cycle invalids should prevent progress")

			mc := setup.mockChains[targetChain]
			require.Len(t, mc.invalidateCalls, 1,
				"P10: cycle-invalidated chain should receive exactly one invalidateBlock call")

		case 3: // Chains not ready (NotFound)
			for _, mc := range setup.mockChains {
				mc.localSafeBlockErr = ethereum.NotFound
			}
			setup.interop.verifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("verifyFn must not be called when chains are not ready")
				return Result{}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("cycleVerifyFn must not be called when chains are not ready")
				return Result{}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.NoError(t, err)
			require.False(t, madeProgress, "P11: empty result should not advance")

			// currentL1 should be the minimum of the collected L1s
			var minL1 eth.BlockID
			first := true
			for _, mc := range setup.mockChains {
				l1 := mc.syncStatus.CurrentL1
				if first || l1.Number < minL1.Number {
					minL1 = l1.ID()
					first = false
				}
			}
			require.Equal(t, minL1, setup.interop.CurrentL1(),
				"P11: currentL1 should be min of collected L1s when chains not ready")

		case 4: // collectCurrentL1 error
			for _, mc := range setup.mockChains {
				mc.syncStatusErr = fmt.Errorf("sync error")
			}
			setup.interop.verifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("verifyFn must not be called when L1 collection fails")
				return Result{}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("cycleVerifyFn must not be called when L1 collection fails")
				return Result{}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.Error(t, err, "P12: L1 collection error should propagate")
			require.False(t, madeProgress, "P12: should not advance on error")

		case 5: // verifyFn returns error
			setup.interop.verifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{}, fmt.Errorf("verify error")
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("cycleVerifyFn must not be called when verifyFn errors")
				return Result{}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.Error(t, err, "P15: verifyFn error should propagate")
			require.False(t, madeProgress, "P15: should not advance on verifyFn error")

		case 6: // invalidateBlock returns error
			targetChain := setup.chainIDs[0]
			setup.mockChains[targetChain].invalidateRetErr = fmt.Errorf("invalidate error")

			setup.interop.verifyFn = func(ts uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{
					Timestamp:   ts,
					L1Inclusion: eth.BlockID{Number: 1, Hash: randomHash(rng)},
					L2Heads:     blocks,
					InvalidHeads: map[eth.ChainID]eth.BlockID{
						targetChain: blocks[targetChain],
					},
				}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				return Result{InvalidHeads: make(map[eth.ChainID]eth.BlockID)}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.Error(t, err, "P16: invalidateBlock error should propagate through handleResult")
			require.False(t, madeProgress, "P16: should not advance on invalidateBlock error")

		case 7: // pauseAtTimestamp prevents progress
			setup.interop.pauseAtTimestamp.Store(activationTS)
			setup.interop.verifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("verifyFn must not be called when paused")
				return Result{}, nil
			}
			setup.interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
				t.Fatal("cycleVerifyFn must not be called when paused")
				return Result{}, nil
			}

			madeProgress, err := setup.interop.progressAndRecord()
			require.NoError(t, err, "P17: pause should not produce error")
			require.False(t, madeProgress, "P17: should not advance when paused")

			// verifiedDB should not have been written to
			_, initialized := setup.verifiedDB.LastTimestamp()
			require.False(t, initialized, "P17: verifiedDB should remain uninitialized when paused")
		}
	})
}

// =============================================================================
// Fuzz Test: Sequential progressAndRecord with persistent chains (P13-P14)
// =============================================================================

// FuzzProgressAndRecordSequential tests that multiple sequential calls
// correctly advance the verifiedDB and track currentL1, with chains that
// persist across iterations and accumulate new blocks and messages.
//
// Unlike the single-shot FuzzProgressAndRecord, this test:
//   - Creates chains once and reuses them across all steps
//   - Adds a new block (with cross-chain messages) to each chain per step
//   - Uses real verifyInteropMessages (reads from logsDB) instead of mock verifyFn
//   - Messages in step N reference blocks sealed in steps 0..N-1
//
// Document coverage:
//
//	INV-1: Logs match blocks (real verifyInteropMessages reads logsDB blocks)
//	INV-3: Every block is cross-valid (real message verification per step)
//	INV-6: L2 heads advance monotonically (sequential timestamp commits)
//	INV-9: C_i tracks L1Inclusion at each step
//	Step 5: Valid blocks → extend VerifiedDB sequentially
//
// Properties:
// P13: Sequential timestamps commit correctly (verifiedDB.Has for each)
// P14: currentL1 tracks L1Inclusion across multiple advances
func FuzzProgressAndRecordSequential(f *testing.F) {
	f.Add(int64(1), uint8(3), uint8(2), uint8(1))
	f.Add(int64(42), uint8(5), uint8(3), uint8(2))
	f.Add(int64(100), uint8(2), uint8(4), uint8(0))

	f.Fuzz(func(t *testing.T, seed int64, numStepsRaw uint8, numChainsRaw uint8, numMsgsRaw uint8) {
		rng := rand.New(rand.NewSource(seed))
		numChains := 2 + int(numChainsRaw%4)    // 2-5
		numSteps := 2 + int(numStepsRaw%5)       // 2-6
		maxMsgsPerBlock := int(numMsgsRaw % 4)   // 0-3
		activationTS := uint64(100000)

		chainIDs := generateChainIDs(numChains, 10, 10)

		verifiedDB, err := OpenVerifiedDB(t.TempDir())
		require.NoError(t, err)
		defer verifiedDB.Close()

		chains := make(map[eth.ChainID]cc.ChainContainer)
		logsDBs := make(map[eth.ChainID]LogsDB)
		mockChains := make(map[eth.ChainID]*fuzzMockChainContainer)
		mockDBs := make(map[eth.ChainID]*fuzzMockLogsDB)

		// Base block number per chain (each chain starts at a different height)
		baseBlockNums := make(map[eth.ChainID]uint64)

		for i, chainID := range chainIDs {
			baseBlockNum := uint64(100 + i*1000)
			baseBlockNums[chainID] = baseBlockNum

			mockDB := newFuzzMockLogsDB()
			// Default Contains returns ErrConflict (unknown message) — only explicitly
			// registered queries succeed, just like in the algo fuzz tests.
			mockDB.defaultContainsErr = suptypes.ErrConflict
			logsDBs[chainID] = mockDB
			mockDBs[chainID] = mockDB

			l1Number := uint64(100 + i*100 + rng.Intn(50))
			mockChain := &fuzzMockChainContainer{
				chainID:         chainID,
				blockTime:       2,
				localSafeBlocks: make(map[uint64]eth.L2BlockRef),
				syncStatus: &eth.SyncStatus{
					CurrentL1: eth.L1BlockRef{
						Number: l1Number,
						Hash:   randomHash(rng),
					},
				},
			}
			chains[chainID] = mockChain
			mockChains[chainID] = mockChain
		}

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		interop := &Interop{
			log:                 gethlog.New(),
			chains:              chains,
			logsDBs:             logsDBs,
			verifiedDB:          verifiedDB,
			activationTimestamp: activationTS,
			ctx:                 ctx,
		}
		// Use real message verification — reads blocks/messages from logsDB
		interop.verifyFn = interop.verifyInteropMessages
		// Skip cycle verification (tested separately in fuzz_algo_test.go)
		interop.cycleVerifyFn = func(_ uint64, _ map[eth.ChainID]eth.BlockID) (Result, error) {
			return Result{InvalidHeads: make(map[eth.ChainID]eth.BlockID)}, nil
		}

		// Track L1Inclusions for P14 verification
		var lastL1Inclusion eth.BlockID

		for step := 0; step < numSteps; step++ {
			ts := activationTS + uint64(step)
			l1Inclusion := eth.BlockID{Number: uint64(step + 1), Hash: randomHash(rng)}

			// --- Add a new block for each chain at this timestamp ---
			for _, chainID := range chainIDs {
				blockNum := baseBlockNums[chainID] + uint64(step)
				blockHash := randomHash(rng)

				// Register the block in the chain container
				mockChains[chainID].localSafeBlocks[ts] = eth.L2BlockRef{
					Number: blockNum,
					Hash:   blockHash,
					Time:   ts,
				}

				mockDB := mockDBs[chainID]

				// Pre-seal the logsDB so loadLogs sees latestBlock >= block and skips.
				// We advance the seal each step to stay ahead.
				sealNum := blockNum + 10
				mockDB.hasSealed = true
				mockDB.latestSealed = eth.BlockID{Number: sealNum, Hash: blockHash}
				mockDB.sealedBlocks[sealNum] = suptypes.BlockSeal{
					Number:    sealNum,
					Hash:      blockHash,
					Timestamp: ts + 100,
				}

				// Build executing messages that reference other chains' prior blocks.
				// Step 0 has no prior blocks to reference, so messages start from step 1.
				execMsgs := make(map[uint32]*suptypes.ExecutingMessage)
				if step > 0 && maxMsgsPerBlock > 0 {
					numMsgs := 1 + rng.Intn(maxMsgsPerBlock)
					for m := 0; m < numMsgs; m++ {
						// Pick a random source chain (can be same or different)
						sourceIdx := rng.Intn(len(chainIDs))
						sourceChain := chainIDs[sourceIdx]

						// Reference a block from a previous step on the source chain
						prevStep := rng.Intn(step) // 0..step-1
						prevTS := activationTS + uint64(prevStep)
						prevBlockNum := baseBlockNums[sourceChain] + uint64(prevStep)

						logIdx := uint32(m)
						execMsg := &suptypes.ExecutingMessage{
							ChainID:   sourceChain,
							BlockNum:  prevBlockNum,
							LogIdx:    logIdx,
							Timestamp: prevTS,
							Checksum:  suptypes.MessageChecksum(randomHash(rng)),
						}
						execMsgs[logIdx] = execMsg

						// Register the Contains result on the source chain's logsDB
						query := containsQueryForMessage(execMsg)
						mockDBs[sourceChain].containsResults[query] = fuzzContainsResult{
							seal: suptypes.BlockSeal{
								Number:    prevBlockNum,
								Timestamp: prevTS,
							},
						}
					}
				}

				// Register the block data in the logsDB
				mockDB.blocks[blockNum] = fuzzBlockData{
					ref:      eth.BlockRef{Hash: blockHash, Number: blockNum, Time: ts},
					logCount: uint32(len(execMsgs)),
					execMsgs: execMsgs,
				}
			}

			// --- Override verifyInteropMessages' L1Inclusion ---
			// The real verifyInteropMessages doesn't set L1Inclusion (it's set by
			// the caller in production). We wrap the real fn to inject it.
			realVerifyFn := interop.verifyInteropMessages
			capturedL1 := l1Inclusion
			interop.verifyFn = func(verifyTS uint64, blocks map[eth.ChainID]eth.BlockID) (Result, error) {
				result, err := realVerifyFn(verifyTS, blocks)
				if err != nil {
					return result, err
				}
				result.L1Inclusion = capturedL1
				return result, nil
			}

			// --- Run progressAndRecord ---
			madeProgress, err := interop.progressAndRecord()
			require.NoError(t, err)
			require.True(t, madeProgress, "P13: step %d should advance", step)

			// P13: Verify timestamp was committed
			has, err := verifiedDB.Has(ts)
			require.NoError(t, err)
			require.True(t, has, "P13: verifiedDB should have timestamp %d after step %d", ts, step)

			// P14: currentL1 tracks L1Inclusion
			require.Equal(t, l1Inclusion, interop.CurrentL1(),
				"P14: currentL1 should track L1Inclusion at step %d", step)

			// Verify the committed result has L2Heads for all chains
			committed, err := verifiedDB.Get(ts)
			require.NoError(t, err)
			require.Equal(t, len(chainIDs), len(committed.L2Heads),
				"all chains should have L2Heads in committed result at step %d", step)

			lastL1Inclusion = l1Inclusion
		}

		// Final verification: all timestamps committed sequentially
		lastTS, initialized := verifiedDB.LastTimestamp()
		require.True(t, initialized, "P13: verifiedDB should be initialized after sequential commits")
		require.Equal(t, activationTS+uint64(numSteps-1), lastTS,
			"P13: lastTimestamp should equal activation + numSteps - 1")

		_ = lastL1Inclusion
	})
}