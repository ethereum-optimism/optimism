package sysgo

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// This file holds the req-resp-recovery variants of the op-con-node sequencer
// P2P topology (singlechain_variants.go): the op-con-node sequencer signs +
// publishes its unsafe chain to OP gossip via the op-conp2p sidecar, and the
// sidecar ALSO serves standard P2P req-resp (payloads-by-number) backed by the
// sequencer's Direct Sync pull endpoint (opql_getUnsafePayload) — so a stock
// op-node verifier can BACKFILL unsafe blocks it never saw on gossip. There is
// no batcher in these topologies: req-resp against the sidecar is the
// verifier's only route to a missed span.

// NewSingleChainOpConLateJoinReqRespRuntime builds the late-joining-verifier
// req-resp topology, the opcon adaptation of the depreqres suite: the
// op-con-node sequencer starts producing signed blocks with NO verifier and NO
// gossip route, and only after it has produced preproducedBlocks does the
// stock op-node verifier (req-resp sync enabled) plus the publish sidecar come
// up. The sidecar joins the sequencer's Direct Sync feed LIVE (a fresh
// subscription with no cursor replays nothing — ws_server.rs treats an absent
// fromBlock as a live join), so the pre-produced span is structurally absent
// from gossip: the verifier can only obtain it by req-resp reverse sync
// through the sidecar, triggered by the first live gossiped block whose parent
// it lacks.
func NewSingleChainOpConLateJoinReqRespRuntime(t devtest.T, cfg PresetConfig, preproducedBlocks uint64) *SingleChainRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node late-join req-resp preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		// No batcher: the missed span must be unhealable via L1, so req-resp is
		// the verifier's only backfill route.
		StartBatcher: false,
	})

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	// Sequence into the void: no verifier and no sidecar exist yet, so these
	// blocks are never gossiped. They stay in the sequencer's signed replay ring
	// (default 1024 blocks — far deeper than this span), serveable via the Direct
	// Sync pull endpoint.
	startOpConSequencer(t, opcon)
	awaitL2ELReachedBlock(t, runtime.L2EL, preproducedBlocks)

	// The verifier joins LATE: a stock op-node with req-resp sync enabled both
	// ways (addSingleChainForcedOpNodeVerifier sets EnableReqResp + UseReqResp).
	nodeB := addSingleChainForcedOpNodeVerifier(t, runtime, "b", cfg.GlobalL2CLOptions...)

	// Publish sidecar: dials the verifier, subscribes to the sequencer's feed
	// LIVE (no cursor — the pre-produced span is deliberately not replayed), and
	// serves req-resp payloads-by-number from the sequencer's ring. Flood publish
	// removes the mesh-formation delay so the first live block reaches the
	// verifier promptly (that block's unknown parent is the reverse-sync trigger).
	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	StartOpConP2PSidecar(t, "op-conp2p-pub", opcon.UserRPC(), rollupPath,
		SequencerGossipMultiaddr(t, nodeB.CL),
		WithSignedPayloadWS(opcon.SignedPayloadWS()),
		WithFloodPublish(),
	)
	awaitCLPeerCount(t, nodeB.CL, 1)

	runtime.P2PEnabled = false
	return runtime
}

// OpConPublishSidecarControl lets a test stop and restart the publish sidecar
// mid-run, modelling a sidecar outage. Restart launches a FRESH op-conp2p
// process with the same flags (new ports; it re-dials the verifier as its
// static peer): a fresh process holds no in-memory publish cursor, so it joins
// the sequencer's Direct Sync feed live and the outage window is never
// re-gossiped — recovery of that span is the verifier's req-resp reverse sync
// through the restarted sidecar.
type OpConPublishSidecarControl struct {
	t          devtest.T
	verifierCL L2CLNode
	start      func() *OpConP2PSidecar
	sidecar    *OpConP2PSidecar
}

// Stop kills the sidecar process; the verifier loses its only gossip peer.
func (c *OpConPublishSidecarControl) Stop() {
	c.sidecar.Stop()
}

// Restart launches a fresh sidecar with the same flags and blocks until the
// verifier reports it as a connected gossip peer again.
func (c *OpConPublishSidecarControl) Restart() {
	c.sidecar = c.start()
	awaitCLPeerCount(c.t, c.verifierCL, 1)
}

// NewSingleChainOpConSequencerP2PRestartableSidecarRuntime is
// NewSingleChainOpConSequencerP2PRuntime (op-con-node sequencer signs +
// publishes via the op-conp2p sidecar to a stock op-node verifier; no batcher)
// with the sidecar handed back to the test through an
// OpConPublishSidecarControl, so the test can model a sidecar outage and
// assert gap recovery.
func NewSingleChainOpConSequencerP2PRestartableSidecarRuntime(t devtest.T, cfg PresetConfig) (*SingleChainRuntime, *OpConPublishSidecarControl) {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node sequencer P2P preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		// No batcher: unsafe propagation (gossip + req-resp) is the only route.
		StartBatcher: false,
	})

	nodeB := addSingleChainForcedOpNodeVerifier(t, runtime, "b", cfg.GlobalL2CLOptions...)

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	verifierAddr := SequencerGossipMultiaddr(t, nodeB.CL)
	startSidecar := func() *OpConP2PSidecar {
		return StartOpConP2PSidecar(t, "op-conp2p-pub", opcon.UserRPC(), rollupPath,
			verifierAddr,
			WithSignedPayloadWS(opcon.SignedPayloadWS()),
			WithFloodPublish(),
		)
	}

	control := &OpConPublishSidecarControl{t: t, verifierCL: nodeB.CL, start: startSidecar}
	control.sidecar = startSidecar()

	// Only sequence once the gossip route exists end to end.
	awaitCLPeerCount(t, nodeB.CL, 1)
	startOpConSequencer(t, opcon)

	runtime.P2PEnabled = false
	return runtime, control
}

// awaitL2ELReachedBlock polls the EL's eth_blockNumber until it reaches the
// target height (or fails the test after ~2 minutes). Used to let the
// sequencer pre-produce a span before later topology pieces come up, so dsl
// frontends (which only exist once the preset is built) are not needed yet.
func awaitL2ELReachedBlock(t devtest.T, el L2ELNode, target uint64) {
	client, err := gethrpc.DialContext(t.Ctx(), el.UserRPC())
	t.Require().NoError(err, "dial sequencer L2 EL rpc")
	defer client.Close()
	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	defer cancel()
	err = retry.Do0(ctx, 240, retry.Fixed(500*time.Millisecond), func() error {
		var head hexutil.Uint64
		if err := client.CallContext(ctx, &head, "eth_blockNumber"); err != nil {
			return err
		}
		if uint64(head) < target {
			return fmt.Errorf("L2 head %d < target %d", uint64(head), target)
		}
		return nil
	})
	t.Require().NoError(err, "sequencer never pre-produced %d blocks", target)
}
