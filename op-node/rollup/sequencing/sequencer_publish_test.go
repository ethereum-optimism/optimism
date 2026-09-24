package sequencing

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	rollupsync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// gatedPublishNetwork holds the first publish until released or canceled. All
// accesses from the test follow synctest.Wait, including reads of published.
// Unlike FakeAsyncGossip, this exercises the actual queue and in-flight context.
type gatedPublishNetwork struct {
	gate      chan struct{}
	firstCtx  context.Context
	attempts  int
	published []*eth.ExecutionPayloadEnvelope
}

func (*gatedPublishNetwork) GossipTimestampThreshold() time.Duration { return time.Minute }

func (n *gatedPublishNetwork) SignAndPublishL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) error {
	n.attempts++
	if n.attempts == 1 {
		n.firstCtx = ctx
		select {
		case <-n.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	n.published = append(n.published, envelope)
	return nil
}

func queueInsertedBlock(s *seqTestSetup, hash byte) *eth.ExecutionPayloadEnvelope {
	ref := eth.L2BlockRef{
		Hash:       common.Hash{hash},
		ParentHash: s.seq.unsafeHead.Hash,
		Number:     s.seq.unsafeHead.Number + 1,
		Time:       uint64(time.Now().Unix()),
	}
	envelope := &eth.ExecutionPayloadEnvelope{ExecutionPayload: &eth.ExecutionPayload{
		BlockHash:   ref.Hash,
		ParentHash:  ref.ParentHash,
		BlockNumber: eth.Uint64Quantity(ref.Number),
		Timestamp:   eth.Uint64Quantity(ref.Time),
	}}
	s.seq.insertAndPublish(envelope, ref, time.Time{})
	return envelope
}

// newPublishQueueSetup must be used inside a synctest bubble. It leaves two
// successfully inserted blocks in the real publisher: one in flight, one queued.
func newPublishQueueSetup(t *testing.T) (*seqTestSetup, *gatedPublishNetwork, []*eth.ExecutionPayloadEnvelope, *engine.EngineController) {
	t.Helper()
	s := newSeqSetup(t)
	net := &gatedPublishNetwork{gate: make(chan struct{})}
	gossiper := async.NewAsyncGossiper(t.Context(), net, log.New(), metrics.NoopMetrics)
	s.seq.asyncGossip = gossiper
	gossiper.Start()
	t.Cleanup(gossiper.Stop)
	// Exercise the same authoritative head setter and invalidation wiring as
	// the driver, while keeping the execution RPC scripted for this unit test.
	ec := engine.NewEngineController(t.Context(), nil, log.New(), metrics.NoopMetrics,
		s.deps.cfg, &rollupsync.Config{}, nil, nil, nil)
	ec.SetUnsafeHead(s.head)
	ec.SetUnsafeHeadInvalidator(gossiper)
	s.deps.eng.processPayloadFn = func(_ context.Context, _ *eth.ExecutionPayloadEnvelope, ref eth.L2BlockRef, _ time.Time) error {
		ec.SetUnsafeHead(ref)
		return nil
	}
	first := queueInsertedBlock(s, 0xa1)
	synctest.Wait()
	require.Equal(t, 1, net.attempts)
	second := queueInsertedBlock(s, 0xa2)
	synctest.Wait()
	require.Empty(t, net.published)
	return s, net, []*eth.ExecutionPayloadEnvelope{first, second}, ec
}

func TestSequencerInvalidatesPublishBacklog(t *testing.T) {
	for _, change := range []string{"reset", "forced reset", "rewind", "sibling", "unproven advance", "stopped reorg", "restart"} {
		t.Run(change, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s, net, _, ec := newPublishQueueSetup(t)
				head := s.seq.unsafeHead
				switch change {
				case "reset":
					deliver(s.seq, rollup.ResetEvent{Err: errors.New("reset chain")})
				case "forced reset":
					// EngineController.forceReset need not send ResetEvent first.
					deliver(s.seq, engine.EngineResetConfirmedEvent{LocalUnsafe: s.head})
				case "rewind":
					ec.SetUnsafeHead(s.head)
					deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: s.head})
				case "sibling", "stopped reorg":
					if change == "stopped reorg" {
						_, err := s.seq.Stop(t.Context())
						require.NoError(t, err)
					}
					head.Hash = common.Hash{0xb1}
					ec.SetUnsafeHead(head)
					deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})
				case "unproven advance":
					head.Hash = common.Hash{0xb2}
					head.ParentHash = common.Hash{0xb1}
					head.Number += 2
					ec.SetUnsafeHead(head)
					deliver(s.seq, engine.ForkchoiceUpdateEvent{UnsafeL2Head: head})
				case "restart":
					_, err := s.seq.Stop(t.Context())
					require.NoError(t, err)
					require.NoError(t, s.seq.Start(t.Context(), head.Hash))
				}
				require.ErrorIs(t, net.firstCtx.Err(), context.Canceled,
					"chain invalidation cancels in-flight publication without waiting for the signer")
				synctest.Wait()
				require.Equal(t, 1, net.attempts, "no queued descendant was attempted")
				require.Empty(t, net.published, "abandoned blocks were not published")

				// The publisher itself remains alive: a newly accepted block can
				// publish without restarting its loop or releasing the old signer.
				fresh := queueInsertedBlock(s, 0xc1)
				synctest.Wait()
				require.Equal(t, []*eth.ExecutionPayloadEnvelope{fresh}, net.published)
			})
		})
	}
}

func TestSequencerPreservesCanonicalPublishBacklog(t *testing.T) {
	for _, change := range []string{"insertion echo", "safe head only", "direct extension", "stopped extension"} {
		t.Run(change, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s, net, queued, ec := newPublishQueueSetup(t)
				head := s.seq.unsafeHead
				ev := engine.ForkchoiceUpdateEvent{UnsafeL2Head: head}
				switch change {
				case "safe head only":
					ev.SafeL2Head = s.head
				case "direct extension", "stopped extension":
					if change == "stopped extension" {
						_, err := s.seq.Stop(t.Context())
						require.NoError(t, err)
					}
					ev.UnsafeL2Head = eth.L2BlockRef{
						Hash:       common.Hash{0xa3},
						ParentHash: head.Hash,
						Number:     head.Number + 1,
						Time:       head.Time + s.deps.cfg.BlockTime,
					}
				}
				ec.SetUnsafeHead(ev.UnsafeL2Head)
				deliver(s.seq, ev)
				synctest.Wait()
				require.NoError(t, net.firstCtx.Err(), "canonical backlog must survive ordinary head updates")
				close(net.gate)
				synctest.Wait()
				require.Equal(t, queued, net.published, "both ancestors publish in order")
			})
		})
	}
}
