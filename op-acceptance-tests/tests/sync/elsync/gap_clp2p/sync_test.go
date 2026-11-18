package gap_clp2p

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestReachUnsafeTipByAppendingUnsafePayload(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newGapCLP2PSystem(t)
	logger := t.Logger()

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// First make verifier reach unsafe tip
	logger.Info("Initial trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// Try once more to check that filling in the gap works again
	logger.Info("Second trial for appending payload until tip")
	sys.L2CLB.AppendUnsafePayloadUntilTip(sys.L2ELB, sys.L2EL, 400)
}

// TestCLUnsafeKeptOnInvalidPayloadDuringELSync checks how the verifier CL handles a payload
// that its EL rejects as INVALID while EL sync is in progress. The EL has only genesis and
// no EL P2P, and the CL's unsafe head is ahead of it, as earlier SYNCING responses left it.
//
// The CL must not adopt the invalid payload, and must not rewind its unsafe head to the EL
// head either: an invalid payload says nothing about the payloads before it. A valid payload
// at the same height must still be accepted afterwards.
//
// The payload is invalidated with a transaction that cannot be decoded: a single 0xff byte, an
// RLP list header with no body. The Engine API lists INVALID as the response when transactions
// contains zero-length or invalid entries, and both op-geth and op-reth decode the transactions
// before they look for the parent block, so they answer INVALID rather than SYNCING. A
// zero-length transaction is the case the spec names, but go-ethereum and alloy compute
// different transaction roots for an empty entry, so kona-node's admin API rejects it with a
// bad block hash.
func TestCLUnsafeKeptOnInvalidPayloadDuringELSync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newGapCLP2PSystem(t)
	require := t.Require()

	sys.L2CL.Advanced(safety.LocalUnsafe, 7, 30)

	// Restart the verifier with an empty EL so that the CL starts EL sync.
	sys.L2CLB.Stop()
	sys.L2ELB.Stop()
	sys.L2ELB.Wipe()
	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.UnsafeHead().IsGenesis()

	// The EL cannot connect this payload to genesis, so it answers SYNCING and the CL's unsafe
	// head moves ahead of the EL.
	const syncTarget = 3
	sys.L2CLB.SignalTarget(sys.L2EL, syncTarget)
	sys.L2CLB.Reached(safety.LocalUnsafe, syncTarget, 30)

	invalid := sys.L2EL.PayloadByNumber(syncTarget + 1)
	invalid.ExecutionPayload.Transactions = append(invalid.ExecutionPayload.Transactions, eth.Data{0xff})
	newHash, _ := invalid.CheckBlockHash()
	invalid.ExecutionPayload.BlockHash = newHash
	sys.L2CLB.PostUnsafePayload(invalid)

	sys.L2CLB.NotAdvanced(safety.LocalUnsafe, 5)
	sys.L2CLB.UnsafeHead().NumEqualTo(syncTarget)
	// The EL answers INVALID for this payload, not SYNCING, so the checks above exercise the
	// CL's handling of an INVALID newPayload response.
	sys.L2ELB.NewPayloadRaw(invalid).IsInvalid()

	// The CL still accepts the valid payload at that height.
	sys.L2CLB.SignalTarget(sys.L2EL, syncTarget+1)
	sys.L2CLB.Reached(safety.LocalUnsafe, syncTarget+1, 30)
	require.Equal(sys.L2EL.BlockRefByNumber(syncTarget+1).Hash, sys.L2CLB.UnsafeHead().BlockRef.Hash,
		"verifier CL unsafe head should be the sequencer's block %d after the invalid payload", syncTarget+1)
}
