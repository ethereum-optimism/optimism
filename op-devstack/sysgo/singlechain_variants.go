package sysgo

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opconductor "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	synctesterconfig "github.com/ethereum-optimism/optimism/op-sync-tester/config"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
	stconf "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/config"
	sttypes "github.com/ethereum-optimism/optimism/op-sync-tester/synctester/backend/types"
	"github.com/ethereum/go-ethereum/log"
)

func NewSingleChainMultiNodeRuntime(t devtest.T, withP2P bool) *SingleChainRuntime {
	return NewSingleChainMultiNodeRuntimeWithConfig(t, withP2P, PresetConfig{})
}

func NewSingleChainMultiNodeRuntimeWithConfig(t devtest.T, withP2P bool, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalRuntimeWithConfig(t, cfg)
	nodeB := addSingleChainOpNode(t, runtime, "b", false, "", cfg.GlobalL2CLOptions...)
	if withP2P {
		connectSingleChainNodes(t, runtime.L2EL, runtime.L2CL, nodeB)
	}
	runtime.P2PEnabled = withP2P
	return runtime
}

// NewSingleChainMultiNodeNoFaultProofsRuntimeWithConfig is like
// NewSingleChainMultiNodeRuntimeWithConfig but skips the proposer and
// challenger (no cannon prestate artifacts required). Intended for tests that
// only exercise sequencer + batcher + verifier derivation.
func NewSingleChainMultiNodeNoFaultProofsRuntimeWithConfig(t devtest.T, withP2P bool, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalNoFaultProofsRuntimeWithConfig(t, cfg)
	// With P2P and an op-con-node verifier, the verifier has no built-in P2P: it
	// gets the unsafe chain over gossip via the op-conp2p sidecar (peered to the
	// sequencer) and derives its safe chain from L1. No follow source, no CL P2P
	// peering (which op-con-node cannot do).
	if withP2P && devstackL2CLKind() == MixedL2CLOpCon {
		addSingleChainOpConVerifierWithSidecar(t, runtime, "b", cfg.GlobalL2CLOptions...)
		runtime.P2PEnabled = true
		return runtime
	}
	// Otherwise wire the verifier's L2 follow source to the sequencer's L2
	// execution RPC. A consensus-only verifier (op-con-node) without P2P pulls the
	// unsafe chain from this RPC and consolidates its L1-derived safe chain against
	// it. op-node verifiers accept the same L2 follow source and additionally peer
	// over CL P2P when withP2P is set.
	followSource := runtime.L2EL.UserRPC()
	nodeB := addSingleChainOpNode(t, runtime, "b", false, followSource, cfg.GlobalL2CLOptions...)
	if withP2P {
		connectSingleChainNodes(t, runtime.L2EL, runtime.L2CL, nodeB)
	}
	runtime.P2PEnabled = withP2P
	return runtime
}

// NewSingleChainMultiNodeNoFaultProofsBareVerifierRuntime builds the
// no-fault-proofs runtime with a BARE verifier: no follow source, no CL P2P, no
// sidecar. The test drives the verifier's unsafe chain entirely via
// admin_postUnsafePayload (the dsl SignalTarget/PostUnsafePayload path). Works for
// both an op-node and an op-con-node verifier — both accept posted unsafe
// payloads and apply them in order, filling gaps before advancing the head.
func NewSingleChainMultiNodeNoFaultProofsBareVerifierRuntime(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalNoFaultProofsRuntimeWithConfig(t, cfg)
	addSingleChainOpNode(t, runtime, "b", false, "", cfg.GlobalL2CLOptions...)
	runtime.P2PEnabled = false
	return runtime
}

// addSingleChainOpConVerifierWithSidecar launches an op-con-node verifier whose
// unsafe chain is delivered over OP gossip by the op-conp2p sidecar (peered to
// the sequencer), rather than follow-mode. It seeds the expected unsafe-block
// signer (the genesis signer is the sequencer P2P role address) so the verifier's
// admin_verifyUnsafePayload can attribute gossiped blocks until an on-chain
// rotation is observed.
func addSingleChainOpConVerifierWithSidecar(t devtest.T, runtime *SingleChainRuntime, name string, l2Opts ...L2CLOption) *SingleChainNodeRuntime {
	addrFor := intentbuilder.RoleToAddrProvider(t, runtime.Keys, runtime.L2Network.ChainID())
	signer := addrFor(devkeys.SequencerP2PRole)
	t.Require().NoError(os.Setenv("DEVSTACK_OPCON_UNSAFE_SIGNER", signer.Hex()),
		"set op-con-node unsafe-block signer env")

	// op-con-node verifier with no follow source: unsafe head arrives via gossip.
	nodeB := addSingleChainOpNode(t, runtime, name, false, "", l2Opts...)

	// The sidecar consumes the same standard rollup config (chain id + fork times
	// drive the gossip topics) and peers with the sequencer's gossip address.
	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	StartOpConP2PSidecar(t, "op-conp2p-"+name, nodeB.CL.UserRPC(), rollupPath, SequencerGossipMultiaddr(t, runtime.L2CL))
	return nodeB
}

// NewSingleChainOpConSequencerP2PRuntime builds the mirror image of the
// with-P2P verifier preset: op-con-node runs AS the sequencer — it builds L2
// blocks on its op-reth, SIGNS each one (SequencerP2PRole key, the deployed
// SystemConfig unsafe-block signer), and serves the signed envelopes on its
// payload websocket. The op-conp2p sidecar subscribes to that feed and
// PUBLISHES each block to OP gossip, where a stock op-node verifier (its own
// gossip stack, static-peered by the sidecar) must accept and execute them.
//
// The sequencer launches stopped and is only started (admin_startSequencer)
// once the verifier reports the sidecar as a connected peer, so the first
// built block already has a live gossip route — there is no batcher in this
// topology to fill unsafe gaps via L1 derivation.
func NewSingleChainOpConSequencerP2PRuntime(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node sequencer P2P preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		// No batcher/proposer/challenger: the assertion surface is unsafe-head
		// propagation over gossip; nothing here consumes L1 batches.
		StartBatcher: false,
	})

	// The verifier peer is deliberately a STOCK op-node (bypassing the
	// DEVSTACK_L2CL_KIND dispatch that owns the verifier slots): the strongest
	// wire-format proof is op-node's own gossip decode + signature check
	// accepting op-con-node's blocks.
	nodeB := addSingleChainForcedOpNodeVerifier(t, runtime, "b", cfg.GlobalL2CLOptions...)

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	// Publish sidecar: bridge the sequencer's signed-payload feed onto gossip,
	// dialing the verifier as its static peer. Flood publish removes the
	// mesh-formation delay so the first block is not dropped. The verdict RPC
	// (node.rpc) points back at the sequencer itself: gossipsub validates
	// locally published messages too, and the sequencer attributes its own
	// signature and accepts.
	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	StartOpConP2PSidecar(t, "op-conp2p-pub", opcon.UserRPC(), rollupPath,
		SequencerGossipMultiaddr(t, nodeB.CL),
		WithSignedPayloadWS(opcon.SignedPayloadWS()),
		WithFloodPublish(),
	)

	// Only sequence once the gossip route exists end to end.
	awaitCLPeerCount(t, nodeB.CL, 1)
	startOpConSequencer(t, opcon)

	// Peering is fully managed by the sidecar's static dial; nothing for the
	// preset frontends to manage.
	runtime.P2PEnabled = false
	return runtime
}

// NewSingleChainOpConSequencerFanOutP2PRuntime extends the op-con-node
// sequencer P2P topology to fan a single signed unsafe-block feed out to TWO
// verifiers over gossip: a stock op-node ("hub", real gossip stack) and an
// op-con-node fronted by a receive sidecar ("b", the assertion target). It
// exercises the FULL op-con↔op-con loop — op-con-node signs and publishes, the
// publish sidecar gossips, a receive sidecar delegates the verdict back to an
// op-con-node verifier — in one topology, alongside op-con→op-node.
//
// The publish sidecar floodpublishes DIRECTLY to both the op-node hub and the
// receive sidecar (not via mesh relay), so the very first block reaches both —
// essential here because there is no batcher to backfill an unsafe gap. The
// op-node hub doubles as the readiness oracle: once it reports both sidecars as
// connected peers, the gossip routes exist and the sequencer is started.
func NewSingleChainOpConSequencerFanOutP2PRuntime(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node sequencer fan-out P2P preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		StartBatcher: false, // unsafe-head propagation over gossip is the surface
	})

	// The expected unsafe-block signer for the op-con-node verifier's verdict
	// (the op-node hub uses the on-chain SystemConfig value). Set by the primary
	// starter already, but assert it is present for clarity.
	addrFor := intentbuilder.RoleToAddrProvider(t, runtime.Keys, runtime.L2Network.ChainID())
	t.Require().Equal(addrFor(devkeys.SequencerP2PRole).Hex(), os.Getenv("DEVSTACK_OPCON_UNSAFE_SIGNER"),
		"sequencer signer env must match the SystemConfig unsafe-block signer")

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	// Gossip hub: a stock op-node verifier with its own gossip stack. It is the
	// publish sidecar's readiness oracle (queryable peer count) and an
	// independent op-node receiver of the same feed.
	hub := addSingleChainForcedOpNodeVerifier(t, runtime, "hub", cfg.GlobalL2CLOptions...)
	hubAddr := SequencerGossipMultiaddr(t, hub.CL)

	// op-con-node verifier "b" (the assertion target), fronted by a receive
	// sidecar. The sidecar dials the hub so the hub counts it as a peer
	// (readiness), and is itself dialed directly by the publish sidecar (block
	// delivery). Node "b" has no follow source: its unsafe head can only come
	// from gossip via this sidecar.
	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	nodeB := addSingleChainOpNode(t, runtime, "b", false, "", cfg.GlobalL2CLOptions...)
	recvSidecar := StartOpConP2PSidecar(t, "op-conp2p-recv-b", nodeB.CL.UserRPC(), rollupPath, hubAddr)

	// Publish sidecar: bridge the sequencer's signed feed onto gossip, dialing
	// BOTH receivers directly so floodpublish delivers block 1 to each.
	StartOpConP2PSidecar(t, "op-conp2p-pub", opcon.UserRPC(), rollupPath,
		strings.Join([]string{hubAddr, recvSidecar.GossipMultiaddr()}, ","),
		WithSignedPayloadWS(opcon.SignedPayloadWS()),
		WithFloodPublish(),
	)

	// Both sidecars connected to the hub ⇒ the gossip routes are live. Start
	// sequencing only then, so no early block is published into a dead network.
	//
	// NOTE: this gates on the hub seeing both sidecars, which does not directly
	// observe the publish→receive-sidecar edge that floodpublishes block 1 to the
	// assertion target (node "b" has no batcher/backfill, so a lost block 1 wedges
	// it). In practice the publish sidecar has been dialing the receive sidecar for
	// the whole awaitCLPeerCount window plus the ~block-time build delay before
	// block 1, and localhost dial+subscribe is sub-second, so the edge is up first.
	// A fully race-free gate would need op-conp2p to report its own connected
	// topic-peer count (it exposes no such RPC today); tracked as a follow-up.
	awaitCLPeerCount(t, hub.CL, 2)
	startOpConSequencer(t, opcon)

	runtime.P2PEnabled = false
	return runtime
}

// NewSingleChainOpConSequencerP2PWrongSignerRuntime is the negative security
// variant of NewSingleChainOpConSequencerP2PRuntime: the op-con-node sequencer
// signs each unsafe block with a key that is NOT the deployed SystemConfig
// unsafe-block signer. The publish sidecar relays the (validly-hash-guarded but
// wrongly-signed) envelope onto gossip verbatim, and the stock op-node verifier
// must REJECT it in gossip validation (signature attribution failure) — so the
// verifier's unsafe head must NOT advance even though the sequencer's own head
// does.
func NewSingleChainOpConSequencerP2PWrongSignerRuntime(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node sequencer P2P preset requires DEVSTACK_L2CL_KIND=op-con-node")

	// A fixed key that is deliberately NOT any deployed role key, so its address
	// is not the SystemConfig unsafe-block signer. Blocks signed with it must be
	// rejected by the verifier.
	const wrongSignerKeyHex = "1111111111111111111111111111111111111111111111111111111111111111"

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld: newDefaultSingleChainWorld,
		StartPrimary: func(t devtest.T, keys devkeys.Keys, world singleChainRuntimeWorld,
			l1EL *L1Geth, l1CL *L1CLNode, jwtPath string, jwtSecret [32]byte, cfg PresetConfig) singleChainPrimaryRuntime {
			// The sequencer resolves the canonical unsafe-block signer (the deployed
			// SystemConfig / SequencerP2PRole address) from its L1 derivation, so
			// DEVSTACK_OPCON_UNSAFE_SIGNER is only a fallback and is set to that same
			// correct address here to match the other single-chain presets (they all
			// write the same value under devtest.ParallelT, keeping the shared env
			// write race-free). Because the sequencer signs with wrongSignerKeyHex,
			// its OWN admin_verifyUnsafePayload — which the op-conp2p publish sidecar
			// invokes as its gossipsub validator, and which go-libp2p-pubsub runs
			// even on locally-published messages — rejects the block, so the sidecar
			// never propagates it. That rejection-at-the-source is the property under
			// test: a non-canonically-signed block is refused entry to gossip.
			secret, err := keys.Secret(devkeys.SequencerP2PRole.Key(world.L2Network.ChainID().ToBig()))
			t.Require().NoError(err, "derive SequencerP2PRole secret")
			t.Require().NoError(os.Setenv("DEVSTACK_OPCON_UNSAFE_SIGNER",
				crypto.PubkeyToAddress(secret.PublicKey).Hex()), "seed expected signer env")
			return startOpConSequencerPrimaryWithSignerKey(t, world, l1EL, l1CL, jwtPath, jwtSecret, wrongSignerKeyHex)
		},
		StartBatcher: false,
	})

	nodeB := addSingleChainForcedOpNodeVerifier(t, runtime, "b", cfg.GlobalL2CLOptions...)

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	rollupPath := writeStandardRollupConfig(t, runtime.L2Network)
	StartOpConP2PSidecar(t, "op-conp2p-pub", opcon.UserRPC(), rollupPath,
		SequencerGossipMultiaddr(t, nodeB.CL),
		WithSignedPayloadWS(opcon.SignedPayloadWS()),
		WithFloodPublish(),
	)

	awaitCLPeerCount(t, nodeB.CL, 1)
	startOpConSequencer(t, opcon)

	runtime.P2PEnabled = false
	return runtime
}

// NewSingleChainOpConSequencerWSFollowRuntime wires the sidecar-less opql
// distribution path: the op-con-node sequencer signs each unsafe block and
// serves the signed envelopes on its payload websocket; an op-con-node verifier
// consumes that feed DIRECTLY via --unsafe-payload-ws (the push analog of
// --l2-follow-rpc) — no gossip, no sidecar — verifying each block's signature
// against the expected unsafe-block signer before ingesting it. The batcher
// runs, so the verifier's safe chain derives from L1 and consolidates against
// the WS-fed unsafe chain.
//
// The sequencer launches stopped and is only started once the verifier is up:
// the verifier's startup gates on its initial WS subscribe, and the feed is
// best-effort live-forward (a block published before the verifier subscribes
// would be an unsafe gap healable only via L1 derivation, muddying the
// unsafe-path assertion).
func NewSingleChainOpConSequencerWSFollowRuntime(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	t.Require().Equal(MixedL2CLOpCon, devstackL2CLKind(),
		"the op-con-node sequencer WS-follow preset requires DEVSTACK_L2CL_KIND=op-con-node")

	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:   newDefaultSingleChainWorld,
		StartPrimary: startOpConSequencerPrimary,
		// Batcher on: safe-head derivation runs alongside the WS unsafe feed.
		StartBatcher: true,
	})

	opcon, ok := runtime.L2CL.(*OpConNode)
	t.Require().True(ok, "primary sequencer must be op-con-node")
	t.Require().NotEmpty(opcon.SignedPayloadWS(), "op-con-node sequencer must serve the signed-payload ws")

	// Verifier "b": no follow source, no sidecar — its unsafe head can only come
	// from the sequencer's signed-payload websocket. The expected unsafe-block
	// signer env was seeded by startOpConSequencerPrimary. Startup blocks until
	// the initial WS subscribe succeeds (the feed is already served while the
	// sequencer is stopped), so once this returns the route is live.
	wsOpts := append(append([]L2CLOption{}, cfg.GlobalL2CLOptions...),
		L2CLOpConUnsafePayloadWS(opcon.SignedPayloadWS()))
	addSingleChainOpNode(t, runtime, "b", false, "", wsOpts...)

	startOpConSequencer(t, opcon)

	runtime.P2PEnabled = false
	return runtime
}

// startOpConSequencerPrimary starts the primary as an op-con-node SEQUENCER
// (signing + signed-payload websocket) paired with its own execution engine.
// The signing key is the SequencerP2PRole secret so op-node verifiers accept
// the blocks against the deployed SystemConfig's unsafe-block signer; the same
// address seeds the sequencer's own admin_verifyUnsafePayload (the sidecar's
// validator delegates self-published blocks back to it).
func startOpConSequencerPrimary(
	t devtest.T,
	keys devkeys.Keys,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	cfg PresetConfig,
) singleChainPrimaryRuntime {
	secret, err := keys.Secret(devkeys.SequencerP2PRole.Key(world.L2Network.ChainID().ToBig()))
	t.Require().NoError(err, "derive SequencerP2PRole secret")
	// Seed the deployed SystemConfig unsafe-block signer as the expected signer
	// so op-con-node verifiers (and the sequencer's own self-verdict) attribute
	// the blocks; op-node verifiers use the on-chain value directly.
	signerAddr := crypto.PubkeyToAddress(secret.PublicKey)
	t.Require().NoError(os.Setenv("DEVSTACK_OPCON_UNSAFE_SIGNER", signerAddr.Hex()),
		"seed op-con-node unsafe-block signer env")
	return startOpConSequencerPrimaryWithSignerKey(t, world, l1EL, l1CL, jwtPath, jwtSecret,
		hex.EncodeToString(crypto.FromECDSA(secret)))
}

// startOpConSequencerPrimaryWithSignerKey is startOpConSequencerPrimary with an
// explicit unsafe-block signing key, so a test can sign with a key OTHER than
// the deployed SystemConfig signer (the wrong-signer rejection test). It does
// NOT touch DEVSTACK_OPCON_UNSAFE_SIGNER — the caller owns what the verifier
// expects.
func startOpConSequencerPrimaryWithSignerKey(
	t devtest.T,
	world singleChainRuntimeWorld,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	jwtPath string,
	jwtSecret [32]byte,
	signerKeyHex string,
) singleChainPrimaryRuntime {
	l2EL := startSequencerEL(t, world.L2Network, jwtPath, jwtSecret, NewELNodeIdentity(0))
	seqSigning := &opConSequencerSigning{
		signerKeyHex: signerKeyHex,
		startStopped: true,
	}
	l2CL := startMixedOpConNode(t, world.L1Network, world.L2Network, l1EL, l1CL, l2EL,
		"sequencer", "sequencer", true, "", "", "", seqSigning, nil)
	return singleChainPrimaryRuntime{EL: l2EL, CL: l2CL}
}

// addSingleChainForcedOpNodeVerifier launches an op-node verifier (with its own
// op-reth EL and gossip stack) REGARDLESS of DEVSTACK_L2CL_KIND. Used when the
// op-con-node under test occupies the sequencer slot and the verifier must be a
// stock op-node.
func addSingleChainForcedOpNodeVerifier(t devtest.T, runtime *SingleChainRuntime, name string, l2Opts ...L2CLOption) *SingleChainNodeRuntime {
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELForKey(t, runtime.L2Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0))
	l2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:           name,
		IsSequencer:   false,
		NoDiscovery:   true,
		EnableReqResp: true,
		UseReqResp:    true,
		L2CLOptions:   l2Opts,
	})
	node := newSingleChainNodeRuntime(name, false, l2EL, l2CL)
	runtime.Nodes[name] = node
	return node
}

// awaitCLPeerCount blocks until the CL reports at least min connected gossip
// peers (opp2p_peers), or fails the test after 60s.
func awaitCLPeerCount(t devtest.T, cl L2CLNode, min uint) {
	p2pClient, err := GetP2PClient(t.Ctx(), t.Logger(), cl)
	t.Require().NoError(err, "p2p client for peer-count wait")
	ctx, cancel := context.WithTimeout(t.Ctx(), 60*time.Second)
	defer cancel()
	err = retry.Do0(ctx, 120, retry.Fixed(500*time.Millisecond), func() error {
		dump, err := p2pClient.Peers(ctx, true)
		if err != nil {
			return err
		}
		if dump.TotalConnected < min {
			return fmt.Errorf("connected peers %d < %d", dump.TotalConnected, min)
		}
		return nil
	})
	t.Require().NoError(err, "CL never reached %d connected gossip peers", min)
}

// startOpConSequencer enables block production on a stopped op-con-node
// sequencer via admin_startSequencer.
func startOpConSequencer(t devtest.T, node *OpConNode) {
	client, err := gethrpc.DialContext(t.Ctx(), node.UserRPC())
	t.Require().NoError(err, "dial op-con-node admin rpc")
	defer client.Close()
	ctx, cancel := context.WithTimeout(t.Ctx(), 10*time.Second)
	defer cancel()
	var result any
	t.Require().NoError(client.CallContext(ctx, &result, "admin_startSequencer"),
		"admin_startSequencer on the op-con-node sequencer")
	t.Logger().Info("op-con-node sequencing started", "rpc", node.UserRPC())
}

// writeStandardRollupConfig marshals the network's rollup config (the standard
// op-node schema) to a temp file for the op-conp2p sidecar.
func writeStandardRollupConfig(t devtest.T, l2Net *L2Network) string {
	data, err := json.Marshal(l2Net.rollupCfg)
	t.Require().NoError(err, "marshal rollup config for op-conp2p sidecar")
	path := filepath.Join(t.TempDir(), "rollup.json")
	t.Require().NoError(os.WriteFile(path, data, 0o644), "write rollup config for op-conp2p sidecar")
	return path
}

func NewSingleChainTwoVerifiersRuntime(t devtest.T) *SingleChainRuntime {
	return NewSingleChainTwoVerifiersRuntimeWithConfig(t, PresetConfig{})
}

func NewSingleChainTwoVerifiersRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewSingleChainMultiNodeRuntimeWithConfig(t, true, cfg)
	nodeB := runtime.Nodes["b"]
	t.Require().NotNil(nodeB, "missing single-chain node b")
	nodeC := addSingleChainOpNode(t, runtime, "c", false, nodeB.CL.UserRPC(), cfg.GlobalL2CLOptions...)

	connectSingleChainNodes(t, runtime.L2EL, runtime.L2CL, nodeC)
	connectSingleChainNodes(t, nodeB.EL, nodeB.CL, nodeC)

	// Follow legacy behavior: test-sequencer is wired against node "b".
	replaceSingleChainTestSequencer(t, runtime, "dev", nodeB)
	return runtime
}

func NewSimpleWithSyncTesterRuntime(t devtest.T) *SingleChainRuntime {
	return NewSimpleWithSyncTesterRuntimeWithConfig(t, PresetConfig{})
}

func NewSimpleWithSyncTesterRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	runtime := NewMinimalRuntimeWithConfig(t, cfg)
	syncTester := startSyncTesterService(t, map[eth.ChainID]string{
		runtime.L2Network.ChainID(): runtime.L2EL.UserRPC(),
	})
	syncTesterELCfg := DefaultSyncTesterELConfig()
	if len(cfg.GlobalSyncTesterELOptions) > 0 {
		syncTesterELTarget := NewComponentTarget("sync-tester-el", runtime.L2Network.ChainID())
		for _, opt := range cfg.GlobalSyncTesterELOptions {
			if opt == nil {
				continue
			}
			opt.Apply(t, syncTesterELTarget, syncTesterELCfg)
		}
	}
	syncTesterEL := startSyncTesterELNode(
		t,
		runtime.L2EL.JWTPath(),
		syncTester,
		NewComponentTarget("sync-tester-el", runtime.L2Network.ChainID()),
		syncTesterELCfg,
	)
	jwtSecret := readJWTSecretFromPath(t, runtime.L2EL.JWTPath())
	l2CL2 := startL2CLNode(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, syncTesterEL, jwtSecret, l2CLNodeStartConfig{
		Key:           "verifier",
		IsSequencer:   false,
		NoDiscovery:   true,
		EnableReqResp: true,
		UseReqResp:    true,
		L2CLOptions:   cfg.GlobalL2CLOptions,
	})
	node := newSingleChainNodeRuntime("verifier", false, syncTesterEL, l2CL2)
	runtime.Nodes[node.Name] = node
	connectSingleChainCLPeer(t, runtime.L2CL, node.CL)
	runtime.SyncTester = &SyncTesterRuntime{
		Service: syncTester,
		Node:    node,
	}
	return runtime
}

func NewMinimalWithConductorsRuntime(t devtest.T) *SingleChainRuntime {
	return NewMinimalWithConductorsRuntimeWithConfig(t, PresetConfig{})
}

func NewMinimalWithConductorsRuntimeWithConfig(t devtest.T, cfg PresetConfig) *SingleChainRuntime {
	// Conductor tests only exercise sequencing leadership. They do not need a
	// challenger, and rust e2e jobs do not build cannon artifacts.
	runtime := newSingleChainRuntimeWithConfig(t, cfg, singleChainRuntimeSpec{
		BuildWorld:      newDefaultSingleChainWorld,
		StartPrimary:    startDefaultSingleChainPrimary,
		StartBatcher:    true,
		StartProposer:   true,
		StartChallenger: false,
	})
	nodeB := addSingleChainOpNode(t, runtime, "b", true, "", cfg.GlobalL2CLOptions...)
	nodeC := addSingleChainOpNode(t, runtime, "c", true, "", cfg.GlobalL2CLOptions...)

	conductorA := startConductorNode(t, "sequencer", runtime.L2Network, runtime.L2CL.(*OpNode), runtime.L2EL, true, false)
	conductorB := startConductorNode(t, "b", runtime.L2Network, nodeB.CL.(*OpNode), nodeB.EL, false, true)
	conductorC := startConductorNode(t, "c", runtime.L2Network, nodeC.CL.(*OpNode), nodeC.EL, false, true)
	startConductorCluster(t, conductorA, []*Conductor{conductorB, conductorC})

	runtime.Conductors = map[string]*Conductor{
		"sequencer": conductorA,
		"b":         conductorB,
		"c":         conductorC,
	}
	return runtime
}

func connectSingleChainNodes(t devtest.T, sourceEL L2ELNode, sourceCL L2CLNode, target *SingleChainNodeRuntime) {
	connectL2ELPeers(t, t.Logger(), sourceEL.UserRPC(), target.EL.UserRPC(), false)
	connectSingleChainCLPeer(t, sourceCL, target.CL)
}

func connectSingleChainCLPeer(t devtest.T, sourceCL, targetCL L2CLNode) {
	connectL2CLPeers(t, t.Logger(), sourceCL, targetCL)
}

func replaceSingleChainTestSequencer(t devtest.T, runtime *SingleChainRuntime, name string, node *SingleChainNodeRuntime) {
	l2CL, ok := node.CL.(*OpNode)
	t.Require().True(ok, "single-chain test sequencer requires an op-node CL node")
	testSequencer := startTestSequencer(
		t,
		runtime.Keys,
		runtime.L2EL.JWTPath(),
		readJWTSecretFromPath(t, runtime.L2EL.JWTPath()),
		runtime.L1Network,
		runtime.L1EL,
		runtime.L1CL,
		node.EL,
		runtime.L2Network,
		l2CL,
	)
	runtime.TestSequencer = newTestSequencerRuntime(testSequencer, name)
}

func addSingleChainOpNode(
	t devtest.T,
	runtime *SingleChainRuntime,
	name string,
	isSequencer bool,
	followSource string,
	l2Opts ...L2CLOption,
) *SingleChainNodeRuntime {
	jwtPath := runtime.L2EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELForKey(t, runtime.L2Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0))
	l2CL := startL2CLForKey(t, runtime.Keys, runtime.L1Network, runtime.L2Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, name, name, isSequencer, followSource, l2Opts)
	node := newSingleChainNodeRuntime(name, isSequencer, l2EL, l2CL)
	runtime.Nodes[name] = node
	return node
}

func startSyncTesterService(t devtest.T, chainRPCs map[eth.ChainID]string) *SyncTesterService {
	require := t.Require()
	syncTesters := make(map[sttypes.SyncTesterID]*stconf.SyncTesterEntry)
	for chainID, elRPC := range chainRPCs {
		id := sttypes.SyncTesterID(fmt.Sprintf("dev-sync-tester-%s", chainID))
		syncTesters[id] = &stconf.SyncTesterEntry{
			ELRPC:   endpoint.MustRPC{Value: endpoint.URL(elRPC)},
			ChainID: chainID,
		}
	}
	cfg := &synctesterconfig.Config{
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		SyncTesters: &stconf.Config{
			SyncTesters: syncTesters,
		},
	}
	logger := t.Logger().New("component", "sync-tester")
	srv, err := synctester.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "must setup sync tester service")
	require.NoError(srv.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		logger.Info("Closing sync tester")
		_ = srv.Stop(ctx)
		logger.Info("Closed sync tester")
	})
	return &SyncTesterService{
		service: srv,
	}
}

func startSyncTesterELNode(
	t devtest.T,
	jwtPath string,
	syncTester *SyncTesterService,
	target ComponentTarget,
	cfg *SyncTesterELConfig,
) *SyncTesterEL {
	node := &SyncTesterEL{
		target:     target,
		jwtPath:    jwtPath,
		config:     cfg,
		p:          t,
		syncTester: syncTester,
	}
	node.Start()
	t.Cleanup(node.Stop)
	return node
}

func startConductorNode(
	t devtest.T,
	conductorName string,
	l2Net *L2Network,
	opNode *OpNode,
	l2EL L2ELNode,
	bootstrap bool,
	paused bool,
) *Conductor {
	require := t.Require()
	serverID := conductorName
	require.NotEmpty(serverID, "conductor ID key cannot be empty")

	var conductorRPCEndpoint atomic.Value
	conductorRPCEndpoint.Store("")
	opNode.cfg.ConductorEnabled = true
	opNode.cfg.ConductorRpcTimeout = 5 * time.Second
	opNode.cfg.ConductorRpc = func(ctx context.Context) (string, error) {
		for {
			if endpoint, _ := conductorRPCEndpoint.Load().(string); endpoint != "" {
				return endpoint, nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	opNode.cfg.Driver.SequencerStopped = true
	opNode.Stop()
	opNode.Start()

	cfg := opconductor.Config{
		ConsensusAddr:           "127.0.0.1",
		ConsensusPort:           0,
		ConsensusAdvertisedAddr: "",
		RaftServerID:            serverID,
		RaftStorageDir:          filepath.Join(t.TempDir(), "raft"),
		RaftBootstrap:           bootstrap,
		RaftSnapshotInterval:    120 * time.Second,
		RaftSnapshotThreshold:   8192,
		RaftTrailingLogs:        10240,
		RaftHeartbeatTimeout:    1000 * time.Millisecond,
		RaftLeaderLeaseTimeout:  500 * time.Millisecond,
		NodeRPC:                 opNode.UserRPC(),
		ExecutionRPC:            l2EL.UserRPC(),
		Paused:                  paused,
		HealthCheck: opconductor.HealthCheckConfig{
			Interval:       3600,
			UnsafeInterval: 3600,
			SafeInterval:   3600,
			MinPeerCount:   1,
		},
		RollupCfg:      *l2Net.rollupCfg,
		RPCEnableProxy: false,
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelInfo,
			Format: oplog.FormatText,
			Color:  false,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
			ListenPort: 0,
		},
	}

	logger := t.Logger().New("component", "conductor", "name", conductorName, "chain", l2Net.ChainID())
	svc, err := opconductor.New(t.Ctx(), &cfg, logger, "0.0.1")
	require.NoError(err)
	require.NoError(svc.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		logger.Info("Closing conductor")
		if err := svc.Stop(ctx); err != nil {
			logger.Warn("Failed to close conductor cleanly", "err", err)
		}
	})

	out := &Conductor{
		name:              conductorName,
		chainID:           l2Net.ChainID(),
		serverID:          serverID,
		consensusEndpoint: svc.ConsensusEndpoint(),
		rpcEndpoint:       svc.HTTPEndpoint(),
		service:           svc,
	}
	conductorRPCEndpoint.Store(svc.HTTPEndpoint())
	return out
}

func startConductorCluster(t devtest.T, bootstrap *Conductor, members []*Conductor) {
	require := t.Require()
	ctx, cancel := context.WithTimeout(t.Ctx(), 90*time.Second)
	defer cancel()

	err := retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		if !bootstrap.service.Leader(ctx) {
			return errors.New("bootstrap conductor is not leader yet")
		}
		return nil
	})
	require.NoError(err, "bootstrap conductor never became leader")

	for _, member := range members {
		err := retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return bootstrap.service.AddServerAsNonvoter(ctx, member.ServerID(), member.ConsensusEndpoint(), 0)
		})
		require.NoErrorf(err, "failed to add conductor %s as non-voter", member.ServerID())

		err = retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return bootstrap.service.AddServerAsVoter(ctx, member.ServerID(), member.ConsensusEndpoint(), 0)
		})
		require.NoErrorf(err, "failed to add conductor %s as voter", member.ServerID())
	}

	expectedServers := 1 + len(members)
	err = retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
		membership, err := bootstrap.service.ClusterMembership(ctx)
		if err != nil {
			return err
		}
		if len(membership.Servers) != expectedServers {
			return fmt.Errorf("expected %d conductors in cluster membership, got %d", expectedServers, len(membership.Servers))
		}
		return nil
	})
	require.NoError(err, "conductor cluster did not converge to expected membership")

	cluster := append([]*Conductor{bootstrap}, members...)
	for _, conductor := range cluster {
		err := retry.Do0(ctx, 40, retry.Fixed(250*time.Millisecond), func() error {
			return conductor.service.Resume(ctx)
		})
		require.NoErrorf(err, "failed to resume conductor %s", conductor.ServerID())
	}

	for _, conductor := range cluster {
		err := retry.Do0(ctx, 90, retry.Fixed(500*time.Millisecond), func() error {
			if !conductor.service.SequencerHealthy(ctx) {
				return fmt.Errorf("conductor %s sequencer is not healthy yet", conductor.ServerID())
			}
			return nil
		})
		require.NoErrorf(err, "conductor %s never became healthy", conductor.ServerID())
	}
}
