//go:build !ci

package upgrade

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestPostInbox(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 60)
	devtest.RunParallel(t, []*dsl.L2Network{sys.L2A, sys.L2B}, func(t devtest.T, net *dsl.L2Network) {
		require := t.Require()
		activationBlock := net.AwaitActivation(t, forks.Lagoon)

		el := net.PrimaryEL()
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
	sys := presets.NewTwoL2SupernodeInterop(t, 60)
	require := t.Require()
	logger := t.Logger()

	// Wait for networks to be online by waiting for blocks
	sys.L1Network.WaitForBlock()
	sys.L2A.WaitForBlock()
	sys.L2B.WaitForBlock()

	lagoonTime := sys.L2A.Escape().ChainConfig().LagoonTime
	require.NotNil(lagoonTime, "Lagoon activation time must be set")

	logger.Info("Starting comprehensive post-interop upgrade tests", "lagoonTime", *lagoonTime)

	// 1. Check that chains reach cross-safe past the activation block
	logger.Info("Checking cross-safe progression past activation block")
	testActivationCrossSafe(t, sys)

	// 2. Check that chains have safety progression for each level
	logger.Info("Checking safety progression")
	testSafetyProgression(t, sys)

	// 3. Confirms that interop message can be included
	logger.Info("Testing interop message inclusion")
	testInteropMessageInclusion(t, sys)

	logger.Info("All comprehensive post-interop upgrade tests completed successfully")
}

// testActivationCrossSafe checks that cross-safe advances past the interop activation block on both chains
func testActivationCrossSafe(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
	logger := t.Logger()

	logger.Info("Waiting for L2 chains to reach interop activation time")

	devtest.RunParallel(t, []*dsl.L2Network{sys.L2A, sys.L2B}, func(t devtest.T, net *dsl.L2Network) {
		// Gate test to not time out before upgrade happens
		forkTimestamp := net.Escape().ChainConfig().LagoonTime
		t.Gate().NotNil(forkTimestamp, "Must have fork configured")
		t.Gate().Greater(*forkTimestamp, uint64(0), "Must not start fork at genesis")
		upgradeTime := time.Unix(int64(*forkTimestamp), 0)
		if deadline, hasDeadline := t.Deadline(); hasDeadline {
			t.Gate().True(upgradeTime.Before(deadline), "test must not time out before upgrade happens")
		}

		activationBlock := net.AwaitActivation(t, forks.Lagoon)

		// Wait for the corresponding CL to reach cross-safe past activation
		if net.ChainID() == sys.L2A.ChainID() {
			sys.L2ACL.Reached(safety.CrossSafe, activationBlock.Number, 60)
		} else {
			sys.L2BCL.Reached(safety.CrossSafe, activationBlock.Number, 60)
		}

		logger.Info("Validating activation block timing",
			"chainID", net.ChainID(),
			"derivedBlockNumber", activationBlock.Number,
			"lagoonTime", *forkTimestamp)
	})

	logger.Info("Activation cross-safe validation completed successfully")
}

// testSafetyProgression checks that chains have safety progression for each level
func testSafetyProgression(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
	logger := t.Logger()
	logger.Info("Testing safety progression")

	delta := uint64(3) // Minimum blocks of progression expected
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, delta, 30),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, delta, 30),

		sys.L2ACL.AdvancedFn(safety.LocalSafe, delta, 30),
		sys.L2BCL.AdvancedFn(safety.LocalSafe, delta, 30),

		sys.L2ACL.AdvancedFn(safety.CrossSafe, delta, 60),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, delta, 60),
	)

	logger.Info("Safety progression validation completed successfully")
}

// testInteropMessageInclusion confirms that interop messages can be included post-upgrade
func testInteropMessageInclusion(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
	logger := t.Logger()
	logger.Info("Starting interop message inclusion test")

	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	eventLoggerAddress := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)

	rng := rand.New(rand.NewSource(1234))
	initMsg := alice.SendRandomInitMessage(rng, eventLoggerAddress, rng.Intn(5), rng.Intn(30))

	// Wait for chain A to advance so supernode indexes the init message
	sys.L2A.WaitForBlock()

	testValidateMessageNeedsAccessList(t, sys, initMsg)

	execMsg := bob.SendExecMessage(initMsg)

	// Verify cross-safe progression for both messages
	dsl.CheckAll(t,
		sys.L2ACL.ReachedRefFn(safety.CrossSafe, initMsg.BlockID(), 60),
		sys.L2BCL.ReachedRefFn(safety.CrossSafe, execMsg.BlockID(), 60),
	)

	logger.Info("Interop message inclusion test completed successfully")
}

// testValidateMessageNeedsAccessList checks that the CrossL2Inbox code from the upgrade bundle
// accepts a message only when the access list declares it.
func testValidateMessageNeedsAccessList(t devtest.T, sys *presets.TwoL2SupernodeInterop, initMsg *dsl.InitMessage) {
	require := t.Require()
	result, err := initMsg.Tx.Result.Eval(t.Ctx())
	require.NoError(err)
	require.NotEmpty(result.Entries)
	msg := result.Entries[0]

	crossL2Inbox := bindings.NewBindings[bindings.CrossL2Inbox](
		bindings.WithClient(sys.L2ELB.Escape().EthClient()),
		bindings.WithTest(t),
		bindings.WithTo(predeploys.CrossL2InboxAddr))
	call := crossL2Inbox.ValidateMessage(msg.Identifier, eth.Bytes32(msg.PayloadHash))

	_, err = contractio.Read(call, t.Ctx())
	require.ErrorContains(errutil.TryAddRevertReason(err), hexutil.Encode(dsl.ErrorSelector("NotInAccessList()")),
		"validateMessage must revert when the access list does not declare the message")

	accessList, err := (&txintent.ExecTrigger{Executor: predeploys.CrossL2InboxAddr, Msg: msg}).AccessList()
	require.NoError(err)
	_, err = contractio.Read(call, t.Ctx(), txplan.WithAccessList(accessList))
	require.NoError(err, "validateMessage must succeed when the access list declares the message")
}
