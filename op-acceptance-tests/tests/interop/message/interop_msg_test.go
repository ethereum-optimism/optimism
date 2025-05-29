package msg

import (
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

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
	sys.Supervisor.AdvancedUnsafeHead(alice.ChainID(), 2)
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
	eventLoggerCallFactory := bindings.NewEventLoggerCallFactory(
		bindings.WithClient(clientA), bindings.WithTest(t), bindings.WithTo(eventLoggerAddress),
	)
	eventLogger := bindings.NewEventLogger(eventLoggerCallFactory)

	// Initialize crossL2Inbox binding
	crossL2InboxCallFactory := bindings.NewCrossL2InboxCallFactory(
		bindings.WithClient(clientB), bindings.WithTest(t),
	)
	crossL2InboxCallFactory.WithDefaultAddr()
	crossL2Inbox := bindings.NewCrossL2Inbox(crossL2InboxCallFactory)

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

	sys.Supervisor.AdvancedUnsafeHead(alice.ChainID(), 2)

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
