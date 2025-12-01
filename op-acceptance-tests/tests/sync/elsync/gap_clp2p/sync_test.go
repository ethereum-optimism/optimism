package gap_clp2p

import (
	"bytes"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestReachUnsafeTipByAppendingUnsafePayload(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()

	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// First make verifier reach unsafe tip
	logger.Info("Initial trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)

	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// Try once more to check that filling in the gap works again
	logger.Info("Second trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)
}

// TestCLUnsafeNotRewoundOnInvalidDuringELSync verifies that the CL's unsafe head
// is not rewound when the EL returns INVALID for a payload during EL sync.
//
// When the EL is still syncing and cannot append new blocks, ForkchoiceUpdate
// returns SYNCING. In this state, the CL may continue to advance its unsafe head
// as it processes new targets, creating temporary divergence from the EL.
//
// The test then crafts a payload that the EL can still validate—even though it is
// not appendable to the EL's current head—by introducing a detectable fault in the
// payload itself (e.g., malformed ExtraData). The CL relays this payload through
// engine_newPayload, and the EL immediately responds INVALID based on intrinsic
// payload validation. The EL does not advance or trigger sync for this payload,
// and the CL's unsafe head remains unchanged, without rewinding.
//
// This confirms that an INVALID response during EL sync halts advancement but does
// not cause the CL's unsafe head to regress, preserving the last known valid head
// while maintaining correct Engine API semantics.
func TestCLUnsafeNotRewoundOnInvalidDuringELSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	logger := t.Logger()
	require := t.Require()

	// Advance few blocks to make sure reference node advanced
	sys.L2CL.Advanced(types.LocalUnsafe, 7, 30)

	// Restart L2CLB to always trigger an EL Sync
	sys.L2CLB.Stop()
	// Wipe out L2ELB state to start from genesis
	sys.L2ELB.Stop()
	sys.L2ELB.Start()
	sys.L2CLB.Start()

	// At this point, L2ELB has no ELP2P and no safe advancement because batcher is stopped
	startNum := sys.L2ELB.BlockRefByLabel(eth.Unsafe).Number
	sys.L2CLB.UnsafeHead().NumEqualTo(startNum)

	attempts := 3
	// Check CL and EL divergence when there is a unsafe gap
	for _, gap := range []uint64{3, 5} {
		targetNum := startNum + gap
		sys.L2CLB.SignalTarget(sys.L2EL, targetNum)
		sys.L2ELB.NotAdvanced(eth.Unsafe, 5)
		sys.L2ELB.UnsafeHead().NumEqualTo(startNum)
		// Check FCU returns SYNCING
		sys.L2ELB.ForkchoiceUpdate(sys.L2EL, targetNum, startNum, startNum, nil).Retry(attempts).ResultAllSyncing()
		// Even though EL did not advance, CL advanced
		sys.L2CLB.UnsafeHead().NumEqualTo(targetNum)
		logger.Info("CL and EL diverged", "CL", targetNum, "EL", startNum)
	}

	// Inject invalid payload that can be only checked by the EL
	// Must choose payload number after than CL unsafe to make the payload sent to EL
	targetNum := sys.L2CLB.UnsafeHead().BlockRef.Number + 1
	payload := sys.L2EL.PayloadByNumber(targetNum)
	// inject fault to the payload
	// Altering extradata makes EL return INVALID even if the EL does not have state to validate
	// EL will not trigger EL Sync because EL already knows that the payload is INVALID
	payload.ExecutionPayload.ExtraData = bytes.Repeat([]byte{0xFF}, 32)
	newHash, ok := payload.CheckBlockHash()
	require.False(ok)
	logger.Info("Injected fault to payload", "newHash", newHash, "prevHash", payload.ExecutionPayload.BlockHash)
	payload.ExecutionPayload.BlockHash = newHash
	_, ok = payload.CheckBlockHash()
	require.True(ok)
	sys.L2CLB.PostUnsafePayload(payload)
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, attempts)
	sys.L2ELB.NotAdvanced(eth.Unsafe, attempts)
	// EL did not advance
	sys.L2ELB.UnsafeHead().NumEqualTo(startNum)
	// CL did not advance
	sys.L2CLB.UnsafeHead().NumEqualTo(startNum + 5)
	// Check newPayload returns INVALID
	// ex) op-geth error msg: "ignoring bad block: holocene extraData should be 9 bytes, got 32"
	sys.L2ELB.NewPayloadRaw(payload).IsInvalid()

	t.Cleanup(func() {
		sys.L2ELB.Start()
		sys.L2CLB.Start()
	})
}
