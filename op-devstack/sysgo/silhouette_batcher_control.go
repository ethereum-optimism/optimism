package sysgo

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// ProofBatchControl drives and observes the ordinary op-batcher used by silhouette acceptance
// tests. It does not load blocks or submit transactions; those remain entirely inside op-batcher.
type ProofBatchControl struct {
	t            devtest.T
	batcher      *L2Batcher
	hooks        *batcher.ProofBatchTestHooks
	waitForProof bool
}

func newProofBatchControl(t devtest.T, b *L2Batcher) *ProofBatchControl {
	t.Require().NotNil(b, "silhouette chain needs its ordinary op-batcher")
	t.Require().NotNil(b.proofBatchHooks, "silhouette op-batcher has no proof-batch hooks")
	return &ProofBatchControl{t: t, batcher: b, hooks: b.proofBatchHooks}
}

func (c *ProofBatchControl) Start(_ time.Duration) {
	err := c.batcher.service.TestDriver().StartBatchSubmitting()
	if err != nil && err.Error() != "batcher is already running" {
		c.t.Require().NoError(err, "start silhouette op-batcher")
	}
}

func (c *ProofBatchControl) Stop() {
	ctx, cancel := context.WithTimeout(c.t.Ctx(), 30*time.Second)
	defer cancel()
	err := c.batcher.service.TestDriver().StopBatchSubmittingIfRunning(ctx)
	c.t.Require().NoError(err, "stop silhouette op-batcher")
}

func (c *ProofBatchControl) MutateUntilApplied(fn func(*proofbatch.ProofBatch) bool) {
	c.hooks.MutateUntilApplied(fn)
}

func (c *ProofBatchControl) ProofBytesOnNext(proof []byte) {
	c.hooks.ProofBytesAfter(proof, c.BatchedHead())
	c.waitForProof = true
}

func (c *ProofBatchControl) SubmitNext() *proofbatch.ProofBatch {
	before := len(c.hooks.Envelopes())
	c.Start(0)
	env := c.waitEnvelope(2*time.Minute, func(i int, env batcher.ProofBatchEnvelope) bool {
		if i < before {
			return false
		}
		return !c.waitForProof || len(env.Proof) > 0
	})
	c.Stop()
	c.waitForProof = false
	if env == nil {
		return nil
	}
	out := env.Batch
	return &out
}

func (c *ProofBatchControl) BatchedHead() uint64 {
	var head uint64
	for _, env := range c.hooks.Envelopes() {
		for _, block := range env.Batch.Blocks {
			head = max(head, block.Number)
		}
	}
	return head
}

func (c *ProofBatchControl) WaitBatched(block eth.BlockID, timeout time.Duration) {
	if c.waitEnvelope(timeout, func(_ int, env batcher.ProofBatchEnvelope) bool {
		for _, candidate := range env.Batch.Blocks {
			if candidate.Number == block.Number && candidate.Hash == block.Hash {
				return true
			}
		}
		return false
	}) == nil {
		c.t.Require().FailNow("timed out waiting for op-batcher proof batch", "block %s", block)
	}
}

// PostedExport returns the first terminal encoding that included block. Keeping the first is
// important for replacement tests: a later corrected proof may encode the same height again.
func (c *ProofBatchControl) PostedExport(block eth.BlockID) (proofbatch.BlockExport, bool) {
	for _, env := range c.hooks.Envelopes() {
		for _, candidate := range env.Batch.Blocks {
			if candidate.Number == block.Number && candidate.Hash == block.Hash {
				return candidate, true
			}
		}
	}
	return proofbatch.BlockExport{}, false
}

func (c *ProofBatchControl) waitEnvelope(timeout time.Duration, matches func(int, batcher.ProofBatchEnvelope) bool) *batcher.ProofBatchEnvelope {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		for i, env := range c.hooks.Envelopes() {
			if matches(i, env) {
				return &env
			}
		}
		select {
		case <-c.t.Ctx().Done():
			c.t.Require().NoError(fmt.Errorf("wait for proof batch: %w", c.t.Ctx().Err()))
			return nil
		case <-deadline.C:
			return nil
		case <-c.hooks.Changed():
		}
	}
}
