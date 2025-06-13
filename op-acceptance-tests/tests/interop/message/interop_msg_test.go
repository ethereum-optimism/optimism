package msg

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

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
	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
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

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)

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
