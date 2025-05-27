package upgrade

import (
	"math/rand"
	"testing"
	"time"

	stypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
)

func TestPostInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {
		require := t.Require()
		activationBlock := net.AwaitActivation(t, rollup.Interop)

		el := net.Escape().L2ELNode(match.FirstL2EL)
		implAddrBytes, err := el.EthClient().GetStorageAt(t.Ctx(), predeploys.CrossL2InboxAddr,
			genesis.ImplementationSlot, activationBlock.Hash.String())
		require.NoError(err)
		implAddr := common.BytesToAddress(implAddrBytes[:])
		require.NotEqual(common.Address{}, implAddr)
		code, err := el.EthClient().CodeAtHash(t.Ctx(), implAddr, activationBlock.Hash)
		require.NoError(err)
		require.NotEmpty(code)
	})
}

func TestPostInteropUpgradeComprehensive(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)
	require := t.Require()
	logger := t.Logger()

	// Wait for networks to be online by waiting for blocks
	sys.L1Network.WaitForBlock()
	sys.L2ChainA.WaitForBlock()
	sys.L2ChainB.WaitForBlock()

	// Get interop activation time
	interopTime := sys.L2ChainA.Escape().ChainConfig().InteropTime
	require.NotNil(interopTime, "InteropTime must be set")

	logger.Info("Starting comprehensive post-interop upgrade tests", "interopTime", *interopTime)

	// 1. Check that anchor block of supervisor matches the activation block
	logger.Info("Checking supervisor anchor block matches activation block")
	testSupervisorAnchorBlock(t, sys)

	// 2. Check that the supervisor has safety progression for each level
	logger.Info("Checking supervisor safety progression")
	testSupervisorSafetyProgression(t, sys)

	// 3. Confirms that interop message can be included
	logger.Info("Testing interop message inclusion")
	testInteropMessageInclusion(t, sys)

	logger.Info("All comprehensive post-interop upgrade tests completed successfully")
}

// testSupervisorAnchorBlock checks that the supervisor's anchor block has been set and matches the upgrade timestamp
func testSupervisorAnchorBlock(t devtest.T, sys *presets.SimpleInterop) {
	logger := t.Logger()

	// Use the DSL helper for anchor block validation
	logger.Info("Testing supervisor anchor block functionality")

	// Phase 1: Wait for L2 chains to reach interop activation time
	logger.Info("Phase 1: Waiting for L2 chains to reach interop activation time")

	devtest.RunParallel(t, sys.L2Networks(), func(t devtest.T, net *dsl.L2Network) {

		// Gate test to not time out before upgrade happens
		forkTimestamp := net.Escape().ChainConfig().InteropTime
		t.Gate().NotNil(forkTimestamp, "Must have fork configured")
		t.Gate().Greater(*forkTimestamp, uint64(0), "Must not start fork at genesis")
		upgradeTime := time.Unix(int64(*forkTimestamp), 0)
		if deadline, hasDeadline := t.Deadline(); hasDeadline {
			t.Gate().True(upgradeTime.Before(deadline), "test must not time out before upgrade happens")
		}

		activationBlock := net.AwaitActivation(t, rollup.Interop)
		sys.Supervisor.WaitForL2HeadToAdvanceTo(net.ChainID(), stypes.CrossSafe, activationBlock)

		logger.Info("Validating anchor block timing",
			"chainID", net.ChainID(),
			"derivedBlockNumber", activationBlock.Number,
			"interopTime", *forkTimestamp)
	})

	logger.Info("Supervisor anchor block validation completed successfully")
}

// testSupervisorSafetyProgression checks that supervisor has safety progression for each level
func testSupervisorSafetyProgression(t devtest.T, sys *presets.SimpleInterop) {
	logger := t.Logger()
	logger.Info("Testing supervisor safety progression")

	delta := uint64(3) // Minimum blocks of progression expected
	dsl.CheckAll(t,
		sys.L2CLA.AdvancedFn(stypes.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(stypes.LocalUnsafe, delta, 30),

		sys.L2CLA.AdvancedFn(stypes.LocalSafe, delta, 30),
		sys.L2CLB.AdvancedFn(stypes.LocalSafe, delta, 30),

		sys.L2CLA.AdvancedFn(stypes.CrossUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(stypes.CrossUnsafe, delta, 30),

		sys.L2CLA.AdvancedFn(stypes.CrossSafe, delta, 30),
		sys.L2CLB.AdvancedFn(stypes.CrossSafe, delta, 30),
	)

	logger.Info("Supervisor safety progression validation completed successfully")
}

// testInteropMessageInclusion confirms that interop messages can be included post-upgrade
func testInteropMessageInclusion(t devtest.T, sys *presets.SimpleInterop) {
	logger := t.Logger()
	logger.Info("Starting interop message inclusion test")

	// Phase 1: Setup test accounts and contracts
	alice, bob, eventLoggerAddress := setupInteropTestEnvironment(sys)

	// Phase 2: Send init message on chain A
	rng := rand.New(rand.NewSource(1234))
	initIntent, initReceipt := alice.SendInitMessage(interop.RandomInitTrigger(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(30)))

	// Make sure supervisor indexes block which includes init message
	sys.Supervisor.WaitForUnsafeHeadToAdvance(alice.ChainID(), 2)

	// Single event in tx so index is 0
	_, execReceipt := bob.SendExecMessage(initIntent, 0)

	// Phase 5: Verify cross-safe progression
	verifyInteropMessagesProgression(t, sys, initReceipt, execReceipt)

	logger.Info("Interop message inclusion test completed successfully")
}

// setupInteropTestEnvironment creates test accounts and deploys necessary contracts
func setupInteropTestEnvironment(sys *presets.SimpleInterop) (alice, bob *dsl.EOA, eventLoggerAddress common.Address) {

	// Create EOAs for interop messaging
	alice = sys.FunderA.NewFundedEOA(eth.OneEther)
	bob = sys.FunderB.NewFundedEOA(eth.OneEther)

	// Deploy event logger contract on chain A
	eventLoggerAddress = alice.DeployEventLogger()

	// Wait for chains to catch up
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	return alice, bob, eventLoggerAddress
}

// verifyInteropMessagesProgression verifies cross-safe progression for both init and exec messages
func verifyInteropMessagesProgression(t devtest.T, sys *presets.SimpleInterop, initReceipt, execReceipt *types.Receipt) {
	logger := t.Logger()

	// Verify cross-safe progression for both messages
	dsl.CheckAll(t,
		sys.L2CLA.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: initReceipt.BlockNumber.Uint64(),
			Hash:   initReceipt.BlockHash,
		}, 60),
		sys.L2CLB.ReachedRefFn(stypes.CrossSafe, eth.BlockID{
			Number: execReceipt.BlockNumber.Uint64(),
			Hash:   execReceipt.BlockHash,
		}, 60),
	)

	logger.Info("Cross-safe progression verified for both init and exec messages")
}
