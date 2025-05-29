package interop

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-program/chainconfig"
	"github.com/ethereum-optimism/optimism/op-program/client/boot"
	"github.com/ethereum-optimism/optimism/op-program/client/interop/types"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	test2 "github.com/ethereum-optimism/optimism/op-program/client/l1/test"
	"github.com/ethereum-optimism/optimism/op-program/client/l2"
	"github.com/ethereum-optimism/optimism/op-program/client/l2/test"
	"github.com/ethereum-optimism/optimism/op-program/client/tasks"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/cross"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type chainSetupOpts struct {
	expiryWindow uint64
}

func WithExpiryWindow(window uint64) func(*chainSetupOpts) {
	return func(opts *chainSetupOpts) {
		opts.expiryWindow = window
	}
}

func setupTwoChains(opts ...func(*chainSetupOpts)) (*staticConfigSource, *eth.SuperV1, *stubTasks) {
	chainSetupOpts := &chainSetupOpts{}
	for _, opt := range opts {
		opt(chainSetupOpts)
	}

	rollupCfg1 := *chaincfg.OPSepolia()
	chainCfg1 := *chainconfig.OPSepoliaChainConfig()

	rollupCfg2 := *chaincfg.OPSepolia()
	rollupCfg2.L2ChainID = new(big.Int).SetUint64(42)
	chainCfg2 := *chainconfig.OPSepoliaChainConfig()
	chainCfg2.ChainID = rollupCfg2.L2ChainID

	// activate interop at genesis for both
	rollupCfg1.InteropTime = new(uint64)
	rollupCfg2.InteropTime = new(uint64)

	agreedSuperRoot := &eth.SuperV1{
		Timestamp: rollupCfg1.Genesis.L2Time + 1234,
		Chains: []eth.ChainIDAndOutput{
			{ChainID: eth.ChainIDFromBig(rollupCfg1.L2ChainID), Output: eth.OutputRoot(&eth.OutputV0{BlockHash: common.Hash{0x11}})},
			{ChainID: eth.ChainIDFromBig(rollupCfg2.L2ChainID), Output: eth.OutputRoot(&eth.OutputV0{BlockHash: common.Hash{0x22}})},
		},
	}

	var ds *depset.StaticConfigDependencySet
	if chainSetupOpts.expiryWindow > 0 {
		ds, _ = depset.NewStaticConfigDependencySetWithMessageExpiryOverride(map[eth.ChainID]*depset.StaticConfigDependency{
			eth.ChainIDFromBig(rollupCfg1.L2ChainID): {},
			eth.ChainIDFromBig(rollupCfg2.L2ChainID): {},
		}, chainSetupOpts.expiryWindow)
	} else {
		ds, _ = depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
			eth.ChainIDFromBig(rollupCfg1.L2ChainID): {},
			eth.ChainIDFromBig(rollupCfg2.L2ChainID): {},
		})
	}
	configSource := &staticConfigSource{
		rollupCfgs:   []*rollup.Config{&rollupCfg1, &rollupCfg2},
		chainConfigs: []*params.ChainConfig{&chainCfg1, &chainCfg2},
		depset:       ds,
		chainIDs:     []eth.ChainID{eth.ChainIDFromBig(rollupCfg1.L2ChainID), eth.ChainIDFromBig(rollupCfg2.L2ChainID)},
	}
	tasksStub := &stubTasks{
		l2SafeHead: eth.L2BlockRef{Number: 918429823450218}, // Past the claimed block
		blockHash:  common.Hash{0x22},
		outputRoot: eth.Bytes32{0x66},
	}
	return configSource, agreedSuperRoot, tasksStub
}

func TestDeriveBlockForFirstChainFromSuperchainRoot(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()

	outputRootHash := common.Hash(eth.SuperRoot(agreedSuperRoot))
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[outputRootHash] = &types.TransitionState{SuperRoot: agreedSuperRoot.Marshal()}

	expectedIntermediateRoot := &types.TransitionState{
		SuperRoot: agreedSuperRoot.Marshal(),
		PendingProgress: []types.OptimisticBlock{
			{BlockHash: tasksStub.blockHash, OutputRoot: tasksStub.outputRoot},
		},
		Step: 1,
	}

	expectedClaim := expectedIntermediateRoot.Hash()
	verifyResult(t, logger, tasksStub, configSource, l2PreimageOracle, outputRootHash, agreedSuperRoot.Timestamp+100000, expectedClaim)
}

func TestDeriveBlockForSecondChainFromTransitionState(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()
	agreedTransitionState := &types.TransitionState{
		SuperRoot: agreedSuperRoot.Marshal(),
		PendingProgress: []types.OptimisticBlock{
			{BlockHash: common.Hash{0xaa}, OutputRoot: eth.Bytes32{6: 22}},
		},
		Step: 1,
	}
	outputRootHash := agreedTransitionState.Hash()
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[outputRootHash] = agreedTransitionState
	expectedIntermediateRoot := &types.TransitionState{
		SuperRoot: agreedSuperRoot.Marshal(),
		PendingProgress: []types.OptimisticBlock{
			{BlockHash: common.Hash{0xaa}, OutputRoot: eth.Bytes32{6: 22}},
			{BlockHash: tasksStub.blockHash, OutputRoot: tasksStub.outputRoot},
		},
		Step: 2,
	}

	expectedClaim := expectedIntermediateRoot.Hash()
	verifyResult(t, logger, tasksStub, configSource, l2PreimageOracle, outputRootHash, agreedSuperRoot.Timestamp+100000, expectedClaim)
}

func TestNoOpStep(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()
	agreedTransitionState := &types.TransitionState{
		SuperRoot: agreedSuperRoot.Marshal(),
		PendingProgress: []types.OptimisticBlock{
			{BlockHash: common.Hash{0xaa}, OutputRoot: eth.Bytes32{6: 22}},
			{BlockHash: tasksStub.blockHash, OutputRoot: tasksStub.outputRoot},
		},
		Step: 2,
	}
	outputRootHash := agreedTransitionState.Hash()
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[outputRootHash] = agreedTransitionState
	expectedIntermediateRoot := *agreedTransitionState // Copy agreed state
	expectedIntermediateRoot.Step = 3

	expectedClaim := expectedIntermediateRoot.Hash()
	verifyResult(t, logger, tasksStub, configSource, l2PreimageOracle, outputRootHash, agreedSuperRoot.Timestamp+100000, expectedClaim)
}

var (
	initiatingMessageTopic   = crypto.Keccak256Hash([]byte("Test()"))
	initPayloadHash          = crypto.Keccak256Hash(initiatingMessageTopic[:])
	initiatingMessageOrigin  = common.Address{0xaa}
	initiatingMessageOrigin2 = common.Address{0xbb}
)

func TestDeriveBlockForConsolidateStep(t *testing.T) {
	createExecMessage := func(initIncludedIn uint64, config *staticConfigSource, initChainID eth.ChainID) supervisortypes.Message {
		rollupCfg, err := config.RollupConfig(initChainID)
		if err != nil {
			panic(err)
		}
		exec := supervisortypes.Message{
			Identifier: supervisortypes.Identifier{
				Origin:      initiatingMessageOrigin,
				BlockNumber: initIncludedIn,
				LogIndex:    0,
				Timestamp:   initIncludedIn * rollupCfg.BlockTime,
				ChainID:     eth.ChainIDFromBig(rollupCfg.L2ChainID),
			},
			PayloadHash: initPayloadHash,
		}
		return exec
	}

	createInitLog := func() *gethTypes.Log {
		return &gethTypes.Log{
			Address: initiatingMessageOrigin,
			Topics:  []common.Hash{initiatingMessageTopic},
		}
	}

	cases := []struct {
		name     string
		testCase consolidationTestCase
	}{
		{
			name:     "HappyPath",
			testCase: consolidationTestCase{},
		},
		{
			name: "HappyPathWithValidMessages-ExecOnChainB",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init := createInitLog()
					exec := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					return map[eth.ChainID][]*gethTypes.Log{chainA: {init}, chainB: {convertExecutingMessageToLog(t, exec)}}
				},
			},
		},
		{
			name: "HappyPathWithValidMessages-ExecOnChainA",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init := createInitLog()
					execMsg := createExecMessage(includeBlockNumbers[chainB], config, chainB)
					exec := convertExecutingMessageToLog(t, execMsg)
					return map[eth.ChainID][]*gethTypes.Log{chainA: {exec}, chainB: {init}}
				},
			},
		},
		{
			name: "HappyPathWithValidMessages-ExecOnChainB-NonZeroLogIndex",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init1 := &gethTypes.Log{
						Address: initiatingMessageOrigin,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					init2 := &gethTypes.Log{
						Address: initiatingMessageOrigin2,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					exec := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					exec.Identifier.Origin = init2.Address
					exec.Identifier.LogIndex = 1
					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {init1, init2},
						chainB: {convertExecutingMessageToLog(t, exec)},
					}
				},
			},
		},
		{
			name: "HappyPathWithValidMessages-IntraBlockCycle",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					initA := createInitLog()
					initB := createInitLog()

					execMsgA := createExecMessage(includeBlockNumbers[chainB], config, chainB)
					execA := convertExecutingMessageToLog(t, execMsgA)
					execMsgB := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					execB := convertExecutingMessageToLog(t, execMsgB)
					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {initA, execA},
						chainB: {initB, execB},
					}
				},
			},
		},
		{
			name: "ReplaceChainB-UnknownChainID",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init := createInitLog()
					exec := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					exec.Identifier.ChainID = eth.ChainIDFromUInt64(0xdeadbeef)
					return map[eth.ChainID][]*gethTypes.Log{chainA: {init}, chainB: {convertExecutingMessageToLog(t, exec)}}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
		{
			name: "ReplaceChainB-InvalidLogIndex",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init1 := &gethTypes.Log{
						Address: initiatingMessageOrigin,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					init2 := &gethTypes.Log{
						Address: initiatingMessageOrigin2,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					exec := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					exec.Identifier.Origin = init2.Address
					exec.Identifier.LogIndex = 0
					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {init1, init2},
						chainB: {convertExecutingMessageToLog(t, exec)},
					}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
		{
			name: "ReplaceChainB-InvalidPayloadHash",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init := createInitLog()
					execMsg := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					execMsg.PayloadHash = crypto.Keccak256Hash([]byte("invalid hash"))
					return map[eth.ChainID][]*gethTypes.Log{chainA: {init}, chainB: {convertExecutingMessageToLog(t, execMsg)}}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
		{
			name: "ReplaceChainB-InvalidTimestamp",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init := createInitLog()
					execMsg := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					execMsg.Identifier.Timestamp = execMsg.Identifier.Timestamp - 1
					return map[eth.ChainID][]*gethTypes.Log{chainA: {init}, chainB: {convertExecutingMessageToLog(t, execMsg)}}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
		{
			name: "ReplaceBothChains",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					invalidExecMsg := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					invalidExecMsg.PayloadHash = crypto.Keccak256Hash([]byte("invalid hash"))
					log := convertExecutingMessageToLog(t, invalidExecMsg)
					return map[eth.ChainID][]*gethTypes.Log{chainA: {log}, chainB: {log}}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainA, chainB}
				},
			},
		},
		{
			name: "ReplaceBothChains-CascadingReorg",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					initA := createInitLog()
					initB := createInitLog()

					execMsgA := createExecMessage(includeBlockNumbers[chainB], config, chainB)
					execA := convertExecutingMessageToLog(t, execMsgA)
					execMsgB := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					execMsgB.PayloadHash = crypto.Keccak256Hash([]byte("invalid hash"))
					execB := convertExecutingMessageToLog(t, execMsgB)

					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {initA, execA},
						chainB: {initB, execB},
					}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainA, chainB}
				},
			},
		},
		{
			name: "ReplaceChainB-BlockNumberTooBig",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init1 := &gethTypes.Log{
						Address: initiatingMessageOrigin,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					init2 := &gethTypes.Log{
						Address: initiatingMessageOrigin2,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					exec := createExecMessage(1_000_000, config, chainA)
					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {init1, init2},
						chainB: {convertExecutingMessageToLog(t, exec)},
					}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
		{
			name: "ReplaceChainB-LogIndexTooBig",
			testCase: consolidationTestCase{
				logBuilderFn: func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log {
					chainA := config.chainIDs[0]
					chainB := config.chainIDs[1]
					init1 := &gethTypes.Log{
						Address: initiatingMessageOrigin,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					init2 := &gethTypes.Log{
						Address: initiatingMessageOrigin2,
						Topics:  []common.Hash{initiatingMessageTopic},
					}
					exec := createExecMessage(includeBlockNumbers[chainA], config, chainA)
					exec.Identifier.Origin = init2.Address
					exec.Identifier.LogIndex = 1_000_000
					return map[eth.ChainID][]*gethTypes.Log{
						chainA: {init1, init2},
						chainB: {convertExecutingMessageToLog(t, exec)},
					}
				},
				expectBlockReplacements: func(config *staticConfigSource) []eth.ChainID {
					chainB := config.chainIDs[1]
					return []eth.ChainID{chainB}
				},
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			runConsolidationTestCase(t, tt.testCase)
		})
	}
}

// expectBlockReplacementsFn returns the chain indexes containing an optimistic block that must be replaced
type expectBlockReplacementsFn func(config *staticConfigSource) (chaiIDsToReplace []eth.ChainID)

type logBuilderFn func(includeBlockNumbers map[eth.ChainID]uint64, config *staticConfigSource) map[eth.ChainID][]*gethTypes.Log

type consolidationTestCase struct {
	expectBlockReplacements expectBlockReplacementsFn
	logBuilderFn            logBuilderFn
}

func runConsolidationTestCase(t *testing.T, testCase consolidationTestCase) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()
	defer tasksStub.AssertExpectations(t)
	rng := rand.New(rand.NewSource(123))

	configA := configSource.rollupCfgs[0]
	configB := configSource.rollupCfgs[1]
	chainIDA := eth.ChainIDFromBig(configA.L2ChainID)
	chainIDB := eth.ChainIDFromBig(configB.L2ChainID)

	block1A, _ := createBlock(rng, configA, 1, nil)
	block1B, _ := createBlock(rng, configB, 1, nil)

	var logA, logB []*gethTypes.Log
	if testCase.logBuilderFn != nil {
		logs := testCase.logBuilderFn(
			map[eth.ChainID]uint64{chainIDA: block1A.NumberU64() + 1, chainIDB: block1B.NumberU64() + 1},
			configSource,
		)
		logA = logs[chainIDA]
		logB = logs[chainIDB]
	}
	block2A, block2AReceipts := createBlock(rng, configA, 2, gethTypes.Receipts{{Logs: logA}})
	block2B, block2BReceipts := createBlock(rng, configB, 2, gethTypes.Receipts{{Logs: logB}})

	pendingOutputs := map[eth.ChainID]*eth.OutputV0{
		chainIDA: createOutput(block2A.Hash()),
		chainIDB: createOutput(block2B.Hash()),
	}
	finalTransitionState := &types.TransitionState{
		SuperRoot: agreedSuperRoot.Marshal(),
		PendingProgress: []types.OptimisticBlock{
			{BlockHash: block2A.Hash(), OutputRoot: eth.OutputRoot(pendingOutputs[chainIDA])},
			{BlockHash: block2B.Hash(), OutputRoot: eth.OutputRoot(pendingOutputs[chainIDB])},
		},
		Step: ConsolidateStep,
	}
	outputRootHash := finalTransitionState.Hash()
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[outputRootHash] = finalTransitionState

	l2PreimageOracle.Outputs[common.Hash(agreedSuperRoot.Chains[0].Output)] = createOutput(block1A.Hash())
	l2PreimageOracle.Outputs[common.Hash(agreedSuperRoot.Chains[1].Output)] = createOutput(block1B.Hash())
	l2PreimageOracle.Blocks[block1A.Hash()] = block1A
	l2PreimageOracle.Blocks[block2A.Hash()] = block2A
	l2PreimageOracle.Blocks[block2B.Hash()] = block2B

	l2PreimageOracle.Receipts[block2A.Hash()] = block2AReceipts
	l2PreimageOracle.Receipts[block2B.Hash()] = block2BReceipts

	finalRoots := map[eth.ChainID]eth.Bytes32{
		chainIDA: finalTransitionState.PendingProgress[0].OutputRoot,
		chainIDB: finalTransitionState.PendingProgress[1].OutputRoot,
	}
	if testCase.expectBlockReplacements != nil {
		for _, chainIDToReplace := range testCase.expectBlockReplacements(configSource) {
			// stub output root preimage of the replaced block
			replacedBlockOutput := pendingOutputs[chainIDToReplace]
			replacedBlockOutputRoot := common.Hash(eth.OutputRoot(replacedBlockOutput))
			l2PreimageOracle.Outputs[replacedBlockOutputRoot] = replacedBlockOutput

			replacedChainRollupConfig, err := configSource.RollupConfig(chainIDToReplace)
			require.NoError(t, err)
			depositsOnlyBlock, depositsOnlyBlockReceipts := createBlock(rng, replacedChainRollupConfig, 2, nil)
			depositsOnlyOutput := createOutput(depositsOnlyBlock.Hash())
			depositsOnlyOutputRoot := eth.OutputRoot(depositsOnlyOutput)
			var out eth.Bytes32
			for _, ch := range agreedSuperRoot.Chains {
				if ch.ChainID == chainIDToReplace {
					out = ch.Output
				}
			}
			tasksStub.ExpectBuildDepositOnlyBlock(common.Hash{}, out, depositsOnlyBlock.Hash(), depositsOnlyOutputRoot)
			finalRoots[chainIDToReplace] = depositsOnlyOutputRoot
			// stub the preimages in the replacement block
			l2PreimageOracle.Blocks[depositsOnlyBlock.Hash()] = depositsOnlyBlock
			l2PreimageOracle.Outputs[common.Hash(depositsOnlyOutputRoot)] = depositsOnlyOutput
			l2PreimageOracle.Receipts[depositsOnlyBlock.Hash()] = depositsOnlyBlockReceipts
		}
	}
	expectedClaim := common.Hash(eth.SuperRoot(&eth.SuperV1{
		Timestamp: agreedSuperRoot.Timestamp + 1,
		Chains: []eth.ChainIDAndOutput{
			{
				ChainID: chainIDA,
				Output:  finalRoots[chainIDA],
			},
			{
				ChainID: chainIDB,
				Output:  finalRoots[chainIDB],
			},
		},
	}))

	verifyResult(
		t,
		logger,
		tasksStub,
		configSource,
		l2PreimageOracle,
		outputRootHash,
		agreedSuperRoot.Timestamp+100000,
		expectedClaim,
	)
}

func createOutput(blockHash common.Hash) *eth.OutputV0 {
	return &eth.OutputV0{BlockHash: blockHash}
}

func convertExecutingMessageToLog(t *testing.T, msg supervisortypes.Message) *gethTypes.Log {
	id := msg.Identifier
	data := make([]byte, 0, 32*5)
	data = append(data, make([]byte, 12)...)
	data = append(data, id.Origin.Bytes()...)
	data = append(data, make([]byte, 32-8)...)
	data = append(data, binary.BigEndian.AppendUint64(nil, id.BlockNumber)...)
	data = append(data, make([]byte, 32-4)...)
	data = append(data, binary.BigEndian.AppendUint32(nil, id.LogIndex)...)
	data = append(data, make([]byte, 32-8)...)
	data = append(data, binary.BigEndian.AppendUint64(nil, id.Timestamp)...)
	b := id.ChainID.Bytes32()
	data = append(data, b[:]...)
	require.Equal(t, len(data), 32*5)
	return &gethTypes.Log{
		Address: params.InteropCrossL2InboxAddress,
		Topics:  []common.Hash{supervisortypes.ExecutingMessageEventTopic, msg.PayloadHash},
		Data:    data,
	}
}

func createBlock(rng *rand.Rand,
	config *rollup.Config,
	blockNum int64, receipts gethTypes.Receipts) (*gethTypes.Block, gethTypes.Receipts) {
	block, randomReceipts := testutils.RandomBlock(rng, 1)
	receipts = append(receipts, randomReceipts...)
	header := block.Header()
	header.Time = uint64(blockNum) * config.BlockTime
	header.Number = big.NewInt(blockNum)
	return gethTypes.NewBlock(
		header,
		block.Body(),
		receipts,
		trie.NewStackTrie(nil),
		gethTypes.DefaultBlockConfig,
	), receipts
}

func TestTraceExtensionOnceClaimedTimestampIsReached(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()
	agreedPrestatehash := common.Hash(eth.SuperRoot(agreedSuperRoot))
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[agreedPrestatehash] = &types.TransitionState{SuperRoot: agreedSuperRoot.Marshal()}

	// We have reached the game's timestamp so should just trace extend the agreed claim
	expectedClaim := agreedPrestatehash
	verifyResult(t, logger, tasksStub, configSource, l2PreimageOracle, agreedPrestatehash, agreedSuperRoot.Timestamp, expectedClaim)
}

func TestPanicIfAgreedPrestateIsAfterGameTimestamp(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	configSource, agreedSuperRoot, tasksStub := setupTwoChains()
	agreedPrestatehash := common.Hash(eth.SuperRoot(agreedSuperRoot))
	l2PreimageOracle, _ := test.NewStubOracle(t)
	l2PreimageOracle.TransitionStates[agreedPrestatehash] = &types.TransitionState{SuperRoot: agreedSuperRoot.Marshal()}

	// We have reached the game's timestamp so should just trace extend the agreed claim
	expectedClaim := agreedPrestatehash
	require.PanicsWithValue(t, fmt.Sprintf("agreed prestate timestamp %v is after the game timestamp %v", agreedSuperRoot.Timestamp, agreedSuperRoot.Timestamp-1), func() {
		verifyResult(t, logger, tasksStub, configSource, l2PreimageOracle, agreedPrestatehash, agreedSuperRoot.Timestamp-1, expectedClaim)
	})
}

func TestHazardSet_ExpiredMessageShortCircuitsInclusionCheck(t *testing.T) {
	// This test is also covered by safe_update_test.go in op-supervisor.
	// However, since this short-circuit behavior is critical for fault proofs, we doubly assert the desired behavior here to prevent a regression.

	runTest := func(t *testing.T, expiryWindow uint64, expectInclusionCheck bool) {
		logger := testlog.Logger(t, log.LevelError)
		configSource, agreedSuperRoot, tasksStub := setupTwoChains(WithExpiryWindow(expiryWindow))
		defer tasksStub.AssertExpectations(t)
		rng := rand.New(rand.NewSource(123))

		configA := configSource.rollupCfgs[0]
		configB := configSource.rollupCfgs[1]

		initLog := &gethTypes.Log{Address: initiatingMessageOrigin, Topics: []common.Hash{initiatingMessageTopic}}
		block1A, _ := createBlock(rng, configA, 1, gethTypes.Receipts{{Logs: []*gethTypes.Log{initLog}}})

		exec := supervisortypes.Message{
			Identifier: supervisortypes.Identifier{
				Origin:      initiatingMessageOrigin,
				BlockNumber: 1,
				Timestamp:   block1A.Time(),
				ChainID:     eth.ChainIDFromBig(configB.L2ChainID),
			},
			PayloadHash: initPayloadHash,
		}
		logA := convertExecutingMessageToLog(t, exec)
		block2A, block2AReceipts := createBlock(rng, configA, 2, gethTypes.Receipts{{Logs: []*gethTypes.Log{logA}}})
		block2B, block2BReceipts := createBlock(rng, configB, 2, nil)

		pendingOutputs := [2]*eth.OutputV0{0: createOutput(block2A.Hash()), 1: createOutput(block2B.Hash())}
		transitionState := &types.TransitionState{
			SuperRoot: agreedSuperRoot.Marshal(),
			PendingProgress: []types.OptimisticBlock{
				{BlockHash: block2A.Hash(), OutputRoot: eth.OutputRoot(pendingOutputs[0])},
				{BlockHash: block2B.Hash(), OutputRoot: eth.OutputRoot(pendingOutputs[1])},
			},
			Step: ConsolidateStep,
		}
		l2PreimageOracle, _ := test.NewStubOracle(t)
		l2PreimageOracle.Blocks[block2A.Hash()] = block2A
		l2PreimageOracle.Blocks[block2B.Hash()] = block2B
		l2PreimageOracle.Receipts[block2A.Hash()] = block2AReceipts
		l2PreimageOracle.Receipts[block2B.Hash()] = block2BReceipts
		for _, chain := range agreedSuperRoot.Chains {
			l2PreimageOracle.Outputs[common.Hash(chain.Output)] = &eth.OutputV0{}
		}

		consolidateState := newConsolidateState(transitionState)
		consolidateDeps, err := newConsolidateCheckDeps(configSource.depset, configSource, transitionState, agreedSuperRoot.Chains, l2PreimageOracle, consolidateState)
		require.NoError(t, err)

		mockConsolidateDeps := &mockConsolidateDeps{consolidateCheckDeps: consolidateDeps}
		mockConsolidateDeps.
			On("Contains", mock.Anything, mock.Anything).Return(supervisortypes.BlockSeal{}, supervisortypes.ErrConflict).
			Maybe()

		linker := depset.LinkCheckFn(func(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool {
			window := configSource.depset.MessageExpiryWindow()
			return initTimestamp+window >= execInTimestamp
		})
		deps := &cross.UnsafeHazardDeps{UnsafeStartDeps: mockConsolidateDeps}
		candidate := supervisortypes.BlockSeal{
			Hash:      block2A.Hash(),
			Number:    block2A.NumberU64(),
			Timestamp: block2A.Time(),
		}
		_, err = cross.NewHazardSet(deps, linker, logger, eth.ChainIDFromBig(configA.L2ChainID), candidate)
		require.ErrorIs(t, err, supervisortypes.ErrConflict)

		if expectInclusionCheck {
			mockConsolidateDeps.AssertCalled(t, "Contains", mock.Anything, mock.Anything)
		} else {
			mockConsolidateDeps.AssertNotCalled(t, "Contains", mock.Anything, mock.Anything)
		}
		mockConsolidateDeps.AssertExpectations(t)
	}

	t.Run("expired message short-circuits inclusion check", func(t *testing.T) {
		runTest(t, 1, false)
	})
	t.Run("message not expired does not short-circuit inclusion check", func(t *testing.T) {
		runTest(t, 2, true)
	})
}

type mockConsolidateDeps struct {
	mock.Mock
	*consolidateCheckDeps
}

func (m *mockConsolidateDeps) Contains(chainID eth.ChainID, query supervisortypes.ContainsQuery) (supervisortypes.BlockSeal, error) {
	out := m.Mock.Called(chainID, query)
	return out.Get(0).(supervisortypes.BlockSeal), out.Error(1)
}

func verifyResult(t *testing.T, logger log.Logger, tasks *stubTasks, configSource *staticConfigSource, l2PreimageOracle *test.StubBlockOracle, agreedPrestate common.Hash, gameTimestamp uint64, expectedClaim common.Hash) {
	bootInfo := &boot.BootInfoInterop{
		AgreedPrestate: agreedPrestate,
		GameTimestamp:  gameTimestamp,
		Claim:          expectedClaim,
		Configs:        configSource,
	}
	l1Oracle := test2.NewStubOracle(t)
	for _, chainID := range configSource.chainIDs {
		rollupCfg, err := configSource.RollupConfig(chainID)
		require.NoError(t, err)
		// Assuming the anchor block of the L2 on the L1 is the same timestamp as the genesis block of L1.
		l1Oracle.Blocks[rollupCfg.Genesis.L1.Hash] = &testutils.MockBlockInfo{
			InfoHash: rollupCfg.Genesis.L1.Hash,
			InfoNum:  rollupCfg.Genesis.L1.Number,
			InfoTime: rollupCfg.Genesis.L2Time,
		}
	}
	err := runInteropProgram(logger, bootInfo, l1Oracle, l2PreimageOracle, tasks)
	require.NoError(t, err)
}

type stubTasks struct {
	mock.Mock
	l2SafeHead eth.L2BlockRef
	blockHash  common.Hash
	outputRoot eth.Bytes32
	err        error
}

var _ taskExecutor = (*stubTasks)(nil)

func (t *stubTasks) RunDerivation(
	_ log.Logger,
	_ *rollup.Config,
	_ *params.ChainConfig,
	_ common.Hash,
	_ eth.Bytes32,
	_ uint64,
	_ l1.Oracle,
	_ l2.Oracle,
) (tasks.DerivationResult, error) {
	return tasks.DerivationResult{
		Head:       t.l2SafeHead,
		BlockHash:  t.blockHash,
		OutputRoot: t.outputRoot,
	}, t.err
}

func (t *stubTasks) BuildDepositOnlyBlock(
	logger log.Logger,
	rollupCfg *rollup.Config,
	l2ChainConfig *params.ChainConfig,
	l1Head common.Hash,
	agreedL2OutputRoot eth.Bytes32,
	l1Oracle l1.Oracle,
	l2Oracle l2.Oracle,
	optimisticBlock *gethTypes.Block,
	db l2.KeyValueStore,
) (common.Hash, eth.Bytes32, error) {
	out := t.Mock.Called(
		logger,
		rollupCfg,
		l2ChainConfig,
		l1Head,
		agreedL2OutputRoot,
		l1Oracle,
		l2Oracle,
		optimisticBlock,
		db,
	)
	return out.Get(0).(common.Hash), out.Get(1).(eth.Bytes32), nil
}

func (t *stubTasks) ExpectBuildDepositOnlyBlock(
	expectL1Head common.Hash,
	expectAgreedL2OutputRoot eth.Bytes32,
	depositOnlyBlockHash common.Hash,
	depositOnlyOutputRoot eth.Bytes32,
) {
	t.Mock.On(
		"BuildDepositOnlyBlock",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		expectL1Head,
		expectAgreedL2OutputRoot,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Once().Return(depositOnlyBlockHash, depositOnlyOutputRoot, nil)
}

type staticConfigSource struct {
	rollupCfgs   []*rollup.Config
	chainConfigs []*params.ChainConfig
	depset       *depset.StaticConfigDependencySet
	chainIDs     []eth.ChainID
}

func (s *staticConfigSource) RollupConfig(chainID eth.ChainID) (*rollup.Config, error) {
	for _, cfg := range s.rollupCfgs {
		if eth.ChainIDFromBig(cfg.L2ChainID) == chainID {
			return cfg, nil
		}
	}
	panic(fmt.Sprintf("no rollup config found for chain %d", chainID))
}

func (s *staticConfigSource) ChainConfig(chainID eth.ChainID) (*params.ChainConfig, error) {
	for _, cfg := range s.chainConfigs {
		if eth.ChainIDFromBig(cfg.ChainID) == chainID {
			return cfg, nil
		}
	}
	panic(fmt.Sprintf("no chain config found for chain %d", chainID))
}

func (s *staticConfigSource) DependencySet(chainID eth.ChainID) (depset.DependencySet, error) {
	return s.depset, nil
}
