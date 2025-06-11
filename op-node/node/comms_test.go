package node

import (
	"context"
	"math/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type testTracer struct {
	got string
}

func (t *testTracer) OnNewL1Head(ctx context.Context, sig eth.L1BlockRef) {
	t.got += "L1Head: " + sig.ID().String() + "\n"
}

func (t *testTracer) OnUnsafeL2Payload(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
	t.got += "P2P in: from: " + string(from) + " id: " + payload.ID().String() + "\n"
}

func (t *testTracer) OnPublishL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
	t.got += "P2P out: " + payload.ID().String() + "\n"
}

var _ Tracer = (*testTracer)(nil)

// TestTracer tests that the tracer traces each thing as expected
func TestTracer(t *testing.T) {
	tr := &testTracer{}
	d := NewTracerDeriver(tr)
	rng := rand.New(rand.NewSource(123))

	l1Head := testutils.RandomBlockRef(rng)
	d.OnEvent(status.L1UnsafeEvent{L1Unsafe: l1Head})
	require.Equal(t, "L1Head: "+l1Head.ID().String()+"\n", tr.got)
	tr.got = ""

	id := testutils.RandomBlockID(rng)
	block := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockHash: id.Hash, BlockNumber: eth.Uint64Quantity(id.Number)}}

	d.OnEvent(p2p.ReceivedBlockEvent{From: "foo", Envelope: block})
	require.Equal(t, "P2P in: from: foo id: "+id.String()+"\n", tr.got)
	tr.got = ""

	d.OnEvent(TracePublishBlockEvent{Envelope: block})
	require.Equal(t, "P2P out: "+id.String()+"\n", tr.got)
	tr.got = ""

	require.NoError(t, d.ctx.Err())
	d.Unattach()
	require.Error(t, d.ctx.Err())
}
