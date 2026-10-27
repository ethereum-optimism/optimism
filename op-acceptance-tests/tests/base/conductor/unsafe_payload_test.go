package conductor

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

func newUnsafePayloadSystem(t devtest.T) *presets.SingleChainMultiNode {
	sys := presets.NewSingleChainWithIsolatedVerifier(t,
		presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
			cfg.Stopped = true
		}),
	)
	sys.L2CLB.UnsafeHead().NumEqualTo(0)
	return sys
}

// TestPostedUnsafePayloadAdvancesVerifier verifies an operator can inject an
// unsafe payload through the admin API and advance an otherwise disconnected
// verifier.
func TestPostedUnsafePayloadAdvancesVerifier(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newUnsafePayloadSystem(t)
	sys.L2CL.AdvancedUnsafe(2, 30)

	payload := sys.L2EL.PayloadByNumber(1)
	sys.L2CLB.PostUnsafePayload(payload)
	sys.L2CLB.Reached(safety.LocalUnsafe, 1, 30)
}

// TestPostedUnsafePayloadRejectsBadBlockHash verifies malformed payloads are
// rejected synchronously instead of being acknowledged and dropped later.
func TestPostedUnsafePayloadRejectsBadBlockHash(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newUnsafePayloadSystem(t)
	sys.L2CL.AdvancedUnsafe(2, 30)

	payload := sys.L2EL.PayloadByNumber(2)
	payload.ExecutionPayload.BlockHash = common.Hash{0xaa}
	err := sys.L2CLB.Escape().RollupAPI().PostUnsafePayload(t.Ctx(), payload)
	t.Require().ErrorContains(err, "payload has bad block hash")
}
