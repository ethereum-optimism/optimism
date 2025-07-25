package msg

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/sync/errgroup"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestInitExecMsg tests basic interop messaging
func TestInitExecMsg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	// Trigger random init message at chain A
	initIntent, _ := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(30)))
	// Make sure supervisor indexes block which includes init message
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)
	// Single event in tx so index is 0
	bob.SendExecMessage(initIntent, 0)
}

// TestInitExecMsgWithDSL tests basic interop messaging with contract DSL
func TestInitExecMsgWithDSL(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	rng := rand.New(rand.NewSource(1234))
	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)
	require := t.Require()

	eventLoggerAddress := alice.DeployEventLogger()

	clientA := sys.L2ELA.Escape().EthClient()
	clientB := sys.L2ELB.Escape().EthClient()

	// Initialize eventLogger binding
	eventLogger := bindings.NewBindings[bindings.EventLogger](bindings.WithClient(clientA), bindings.WithTest(t), bindings.WithTo(eventLoggerAddress))
	// Initialize crossL2Inbox binding
	crossL2Inbox := bindings.NewBindings[bindings.CrossL2Inbox](bindings.WithClient(clientB), bindings.WithTest(t), bindings.WithTo(common.HexToAddress(predeploys.CrossL2Inbox)))

	// manually build topics and data for EventLogger
	topics := []eth.Bytes32{}
	for range rng.Intn(5) {
		var topic [32]byte
		copy(topic[:], testutils.RandomData(rng, 32))
		topics = append(topics, topic)
	}
	data := testutils.RandomData(rng, rng.Intn(30))

	// Write: Alice triggers initiating message
	receipt := contract.Write(alice, eventLogger.EmitLog(topics, data))
	block, err := clientA.BlockRefByNumber(t.Ctx(), receipt.BlockNumber.Uint64())
	require.NoError(err)

	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Manually build identifier, message, accesslist for executing message
	// Single event in tx so index is 0
	logIdx := uint32(0)
	payload := suptypes.LogToMessagePayload(receipt.Logs[logIdx])
	identifier := suptypes.Identifier{
		Origin:      eventLoggerAddress,
		BlockNumber: receipt.BlockNumber.Uint64(),
		LogIndex:    logIdx,
		Timestamp:   block.Time,
		ChainID:     sys.L2ELA.ChainID(),
	}
	payloadHash := crypto.Keccak256Hash(payload)
	msgHash := eth.Bytes32(payloadHash)
	msg := suptypes.Message{
		Identifier: identifier, PayloadHash: payloadHash,
	}
	accessList := types.AccessList{{
		Address:     predeploys.CrossL2InboxAddr,
		StorageKeys: suptypes.EncodeAccessList([]suptypes.Access{msg.Access()}),
	}}

	call := crossL2Inbox.ValidateMessage(identifier, msgHash)

	// Read not using the DSL. Therefore you need to manually error handle and also set context
	_, err = contractio.Read(call, t.Ctx())
	// Will revert because access list not provided
	require.Error(err)
	// Provide access list using txplan
	_, err = contractio.Read(call, t.Ctx(), txplan.WithAccessList(accessList))
	// Success because access list made storage slot warm
	require.NoError(err)

	// Read: Trigger executing message
	contract.Read(call, txplan.WithAccessList(accessList))

	// Write: Bob triggers executing message
	contract.Write(bob, call, txplan.WithAccessList(accessList))
}

// TestRandomDirectedGraph tests below scenario:
// Construct random directed graph of messages.
func TestRandomDirectedGraph(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := presets.NewSimpleInterop(t)
	logger := sys.Log.With("Test", "TestRandomDirectedGraph")
	rng := rand.New(rand.NewSource(1234))
	require := sys.T.Require()

	// interop network has at least two chains
	l2ChainNum := 2

	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	// Deploy eventLoggers per every L2 chains because initiating messages can happen on any L2 chains
	eventLoggerAddresses := []common.Address{alice.DeployEventLogger(), bob.DeployEventLogger()}

	// pubSubPairCnt is the count of (publisher, subscriber) pairs which
	// - publisher initiates messages
	// - subscriber validates messages
	pubSubPairCnt := 5
	// txCnt is the count of transactions that each publisher emits
	txCnt := 3
	// fundAmount is the ETH amount to fund publishers and subscribers
	fundAmount := eth.OneTenthEther

	// jitter randomizes tx
	jitter := func(rng *rand.Rand) {
		time.Sleep(time.Duration(rng.Intn(250)) * time.Millisecond)
	}

	// fund EOAs per chain
	eoasPerChain := make([][]*dsl.EOA, l2ChainNum)
	for chainIdx, funder := range []*dsl.Funder{sys.FunderA, sys.FunderB} {
		eoas := funder.NewFundedEOAs(pubSubPairCnt, fundAmount)
		eoasPerChain[chainIdx] = eoas
	}

	// runPubSubPair spawns publisher goroutine, paired with subscriber goroutine
	runPubSubPair := func(pubEOA, subEOA *dsl.EOA, eventLoggerAddress common.Address, localRng *rand.Rand) error {
		ctx, cancel := context.WithCancel(t.Ctx())
		defer cancel()

		g, ctx := errgroup.WithContext(ctx)

		ch := make(chan *txintent.IntentTx[*txintent.MultiTrigger, *txintent.InteropOutput])

		publisherRng := rand.New(rand.NewSource(localRng.Int63()))
		subscriberRng := rand.New(rand.NewSource(localRng.Int63()))

		// publisher initiates txCnt transactions that includes multiple random messages
		g.Go(func() error {
			defer close(ch)
			for range txCnt {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					tx, receipt, err := pubEOA.SendPackedRandomInitMessages(publisherRng, eventLoggerAddress)
					if err != nil {
						return fmt.Errorf("publisher error: %w", err)
					}
					logger.Info("Initiate messages included", "chainID", tx.PlannedTx.ChainID.Value(), "blockNumber", receipt.BlockNumber, "block", receipt.BlockHash)
					select {
					case ch <- tx:
					case <-ctx.Done():
						return ctx.Err()
					}
					jitter(publisherRng)
				}
			}
			return nil
		})

		// subscriber validates every messages that was initiated by the publisher
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case dependsOn, ok := <-ch:
					if !ok {
						return nil
					}
					tx, receipt, err := subEOA.SendPackedExecMessages(dependsOn)
					if err != nil {
						return fmt.Errorf("subscriber error: %w", err)
					}
					logger.Info("Validate messages included", "blockNumber", receipt.BlockNumber, "block", receipt.BlockHash)
					logger.Info("Message dependency",
						"sourceChainID", dependsOn.PlannedTx.ChainID.Value(),
						"destChainID", tx.PlannedTx.ChainID.Value(),
						"sourceBlockNum", dependsOn.PlannedTx.IncludedBlock.Value().Number,
						"destBlockNum", receipt.BlockNumber)
					jitter(subscriberRng)
				}
			}
		})
		return g.Wait()
	}

	var g errgroup.Group

	runPubSubPairWrapper := func(sourceIdx, destIdx, pairIdx int, localRng *rand.Rand) error {
		return runPubSubPair(eoasPerChain[sourceIdx][pairIdx], eoasPerChain[destIdx][pairIdx], eventLoggerAddresses[sourceIdx], localRng)
	}

	for pairIdx := range pubSubPairCnt {
		// randomize source and destination L2 chain
		sourceIdx := rng.Intn(2)
		destIdx := 1 - sourceIdx
		// localRng is needed per pubsub pair because rng cannot be shared without mutex
		localRng := rand.New(rand.NewSource(rng.Int63()))
		g.Go(func() error {
			return runPubSubPairWrapper(sourceIdx, destIdx, pairIdx, localRng)
		})
	}
	require.NoError(g.Wait())
}

// TestInitExecMultipleMsg tests below scenario:
// Transaction initiates and executes multiple messages of self
func TestInitExecMultipleMsg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()
	// Intent to initiate two message(or emit event) on chain A
	initCalls := []txintent.Call{
		interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 15),
		interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 13),
	}
	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger two events
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate messages included", "block", receiptA.BlockHash)
	require.Equal(2, len(receiptA.Logs))

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to validate messages on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// Two events in tx so use every index
	indexes := []int{0, 1}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate messages included", "block", receiptB.BlockHash)

	// Check two ExecutingMessage triggered
	require.Equal(2, len(receiptB.Logs))
}

// TestExecSameMsgTwice tests below scenario:
// Transaction that executes the same message twice.
func TestExecSameMsgTwice(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// Intent to initiate message(or emit event) on chain A
	txA := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](alice.Plan())
	randomInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10)
	txA.Content.Set(randomInitTrigger)

	// Trigger single event
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate message included", "block", receiptA.BlockHash)

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to validate same message two times on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// Single event in tx so indexes are 0, 0
	indexes := []int{0, 0}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate messages included", "block", receiptB.BlockHash)

	// Check two ExecutingMessage triggered
	require.Equal(2, len(receiptB.Logs))
	// Check two messages are identical
	require.Equal(receiptB.Logs[0].Topics, receiptB.Logs[1].Topics)
}

// TestExecDifferentTopicCount tests below scenario:
// Execute message that links with initiating message with: 0, 1, 2, 3, or 4 topics in it
func TestExecDifferentTopicCount(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// Intent to initiate message with different topic counts on chain A
	initCalls := make([]txintent.Call, 5)
	for topicCnt := range 5 {
		initCalls[topicCnt] = interop.RandomInitTrigger(rng, eventLoggerAddress, topicCnt, 10)
	}
	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger five events, each have {0, 1, 2, 3, 4} topics in it
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate messages included", "block", receiptA.BlockHash)
	require.Equal(5, len(receiptA.Logs))

	for topicCnt := range 5 {
		require.Equal(topicCnt, len(receiptA.Logs[topicCnt].Topics))
	}

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to validate message on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// Five events in tx so use every index
	indexes := []int{0, 1, 2, 3, 4}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate message included", "block", receiptB.BlockHash)

	// Check five ExecutingMessage triggered
	require.Equal(5, len(receiptB.Logs))
}

// TestExecMsgOpaqueData tests below scenario:
// Execute message that links with initiating message with: 0, 10KB of opaque event data in it
func TestExecMsgOpaqueData(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// Intent to initiate message with two messages: 0, 10KB of opaque event data
	initCalls := make([]txintent.Call, 2)
	emptyInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 0)      // 0B
	largeInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10_000) // 10KB
	initCalls[0] = emptyInitTrigger
	initCalls[1] = largeInitTrigger

	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger two events
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate messages included", "block", receiptA.BlockHash)
	require.Equal(2, len(receiptA.Logs))
	require.Equal(emptyInitTrigger.OpaqueData, receiptA.Logs[0].Data)
	require.Equal(largeInitTrigger.OpaqueData, receiptA.Logs[1].Data)

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to validate messages on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// Two events in tx so use every index
	indexes := []int{0, 1}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate messages included", "block", receiptB.BlockHash)

	// Check two ExecutingMessage triggered
	require.Equal(2, len(receiptB.Logs))
}

// TestExecMsgDifferEventIndexInSingleTx tests below scenario:
// Execute message that links with initiating message with: first, random or last event of a tx.
func TestExecMsgDifferEventIndexInSingleTx(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	alice, bob := sys.FunderA.NewFundedEOA(eth.OneTenthEther), sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// Intent to initiate message with multiple messages, all included in single tx
	eventCnt := 10
	initCalls := make([]txintent.Call, eventCnt)
	for index := range eventCnt {
		initCalls[index] = interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(100))
	}

	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger multiple events
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate messages included", "block", receiptA.BlockHash)
	require.Equal(eventCnt, len(receiptA.Logs))

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Intent to validate messages on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// first, random or last event of a tx.
	indexes := []int{0, 1 + rng.Intn(eventCnt-1), eventCnt - 1}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate messages included", "block", receiptB.BlockHash)

	// Check three ExecutingMessage triggered
	require.Equal(len(indexes), len(receiptB.Logs))
}

type invalidAttributeType string

const (
	randomOrigin                     invalidAttributeType = "randomOrigin"
	randomBlockNumber                invalidAttributeType = "randomBlockNumber"
	randomLogIndex                   invalidAttributeType = "randomLogIndex"
	randomTimestamp                  invalidAttributeType = "randomTimestamp"
	randomChainID                    invalidAttributeType = "randomChainID"
	mismatchedLogIndex               invalidAttributeType = "mismatchedLogIndex"
	mismatchedTimestamp              invalidAttributeType = "mismatchedTimestamp"
	msgNotPresent                    invalidAttributeType = "msgNotPresent"
	logIndexGreaterOrEqualToEventCnt invalidAttributeType = "logIndexGreaterOrEqualToEventCnt"
)

// executeIndexedFault builds on top of txintent.ExecuteIndexed to inject a fault for the identifier of message
func executeIndexedFault(
	executor common.Address,
	events *plan.Lazy[*txintent.InteropOutput],
	index int,
	rng *rand.Rand,
	faults []invalidAttributeType,
	destChainID eth.ChainID,
) func(ctx context.Context) (*txintent.ExecTrigger, error) {
	return func(ctx context.Context) (*txintent.ExecTrigger, error) {
		execTrigger, err := txintent.ExecuteIndexed(executor, events, index)(ctx)
		if err != nil {
			return nil, err
		}
		newMsg := execTrigger.Msg
		for _, fault := range faults {
			switch fault {
			case randomOrigin:
				newMsg.Identifier.Origin = testutils.RandomAddress(rng)
			case randomBlockNumber:
				// make sure that the faulty blockNumber does not exceed type(uint64).max for CrossL2Inbox check
				newMsg.Identifier.BlockNumber = rng.Uint64() / 2
			case randomLogIndex:
				// make sure that the faulty logIndex does not exceed type(uint32).max for CrossL2Inbox check
				newMsg.Identifier.LogIndex = rng.Uint32() / 2
			case randomTimestamp:
				// make sure that the faulty Timestamp does not exceed type(uint64).max for CrossL2Inbox check
				newMsg.Identifier.Timestamp = rng.Uint64() / 2
			case randomChainID:
				newMsg.Identifier.ChainID = eth.ChainIDFromBytes32([32]byte(testutils.RandomData(rng, 32)))
			case mismatchedLogIndex:
				// valid msg within block, but mismatching event index
				newMsg.Identifier.LogIndex += 1
			case mismatchedTimestamp:
				// within time window, but mismatching block
				newMsg.Identifier.Timestamp += 2
			case msgNotPresent:
				// valid chain but msg not there
				// use destination chain ID because initiating message is not present in dest chain
				newMsg.Identifier.ChainID = destChainID
			case logIndexGreaterOrEqualToEventCnt:
				// execute implied-conflict message: point to event-index >= number of logs
				// number of logs == number of entries
				// so set the invalid logindex to number of entries
				newMsg.Identifier.LogIndex = uint32(len(events.Value().Entries))
			default:
				panic("invalid type")
			}
		}
		return &txintent.ExecTrigger{
			Executor: executor,
			Msg:      newMsg,
		}, nil
	}
}

// TestExecMessageInvalidAttributes tests below scenario:
// Execute message, but with one or more invalid attributes inside identifiers
func TestExecMessageInvalidAttributes(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSimpleInterop(t)
	require := sys.T.Require()
	logger := t.Logger()

	rng := rand.New(rand.NewSource(1234))
	// honest EOA which initiates messages
	alice := sys.FunderA.NewFundedEOA(eth.OneTenthEther)
	// honest EOA which executes messages
	bob := sys.FunderB.NewFundedEOA(eth.OneTenthEther)
	// malicious EOA which creates executing messages with invalid attributes
	chuck := sys.FunderB.NewFundedEOA(eth.OneTenthEther)

	eventLoggerAddress := alice.DeployEventLogger()

	// Intent to initiate messages(or emit events) on chain A
	initCalls := []txintent.Call{
		interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10),
		interop.RandomInitTrigger(rng, eventLoggerAddress, 2, 95),
		interop.RandomInitTrigger(rng, eventLoggerAddress, 1, 50),
	}
	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](alice.Plan())
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: initCalls})

	// Trigger multiple events
	receiptA, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Initiate messages included", "block", receiptA.BlockHash)

	// Make sure supervisor syncs the chain A events
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	faultsLists := [][]invalidAttributeType{
		// test each identifier attributes to be faulty for upper bound tests
		{randomOrigin}, {randomBlockNumber}, {randomLogIndex}, {randomTimestamp}, {randomChainID},
		// test for every attributes to be faulty for upper bound tests
		{randomOrigin, randomBlockNumber, randomLogIndex, randomTimestamp, randomChainID},
		// test for non-random invalid attributes
		{mismatchedLogIndex}, {mismatchedTimestamp}, {msgNotPresent}, {logIndexGreaterOrEqualToEventCnt},
	}

	for _, faults := range faultsLists {
		logger.Info("Attempt to validate message with invalid attribute", "faults", faults)
		// Intent to validate message on chain B
		txC := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](chuck.Plan())
		txC.Content.DependOn(&txA.Result)

		// Random select event index in tx for injecting faults
		eventIdx := rng.Intn(len(initCalls))
		txC.Content.Fn(executeIndexedFault(constants.CrossL2Inbox, &txA.Result, eventIdx, rng, faults, chuck.ChainID()))

		// make sure that the transaction is not reverted by CrossL2Inbox...
		gas, err := txC.PlannedTx.Gas.Eval(t.Ctx())
		require.NoError(err)
		require.Greater(gas, uint64(0))

		// but rather not included at chain B because of supervisor check
		// chain B L2 EL will query supervisor to check whether given message is valid
		// supervisor will throw ErrConflict(conflicting data), and L2 EL will drop tx
		_, err = txC.PlannedTx.Included.Eval(t.Ctx())
		require.Error(err)
		logger.Info("Validate message not included")
	}

	// we now attempt to execute msg correctly
	// Intent to validate message on chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](bob.Plan())
	txB.Content.DependOn(&txA.Result)

	// Three events in tx so use every index
	indexes := []int{0, 1, 2}
	txB.Content.Fn(txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, indexes))

	receiptB, err := txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(err)
	logger.Info("Validate message included", "block", receiptB.BlockHash)

	// Check three ExecutingMessage triggered
	require.Equal(3, len(receiptB.Logs))
}
