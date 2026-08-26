package silhouette

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// Assembly is one silhouette chain's in-process machinery: the fact store both halves read, the
// shim execution client, and the injected data source.
//
// It exists so that "P is a chain in the supernode's chains map" is a wiring statement rather than
// a new kind of container. Everything below is stock op-node — a virtual node, a derivation
// pipeline, an engine controller — pointed at two replaced endpoints.
type Assembly struct {
	ChainID eth.ChainID
	Facts   *FactStore
	Source  *DataSource

	// Labels is the posture this assembly was built for. It is kept because the two postures are
	// not two configurations of one object — they are two different sets of wired seams — and a
	// caller that has to ask "which one is this" is better served by a field than by testing
	// whether Shim is nil.
	Labels LabelSource

	// Shim is the execution client that replaces P's real one. It exists in the VERIFIER posture
	// only: the sequencer posture keeps P's real execution client, so there is nothing to replace
	// and a shim there would be a second, contradictory answer to `eth_getBlockByNumber`.
	Shim *Shim

	// Tracker drives the data source over L1 when no derivation pipeline does. It exists in the
	// SEQUENCER posture only. Run() is what makes it run.
	Tracker *ProvenHeadTracker

	// server is the in-process RPC server carrying the shim's namespaces. The caller owns stopping
	// it; it is returned rather than hidden because a shim that outlives its supernode is a leak
	// that only shows up under test restarts.
	server *gethrpc.Server
	client client.RPC
}

// restartableRPC prevents a virtual-node restart from closing the assembly-owned in-process
// connection. Assembly.Close closes the underlying client after every container has stopped.
type restartableRPC struct{ client.RPC }

func (restartableRPC) Close() {}

// Run drives whatever this assembly has to drive, until ctx is cancelled.
//
// In the verifier posture there is nothing to drive: the derivation pipeline pulls the data source
// as it traverses L1, so acceptance is a side effect of deriving the chain and this returns
// immediately. In the sequencer posture there is no pipeline, and this is the walk.
func (a *Assembly) Run(ctx context.Context) {
	if a.Tracker == nil {
		return
	}
	a.Tracker.Run(ctx)
}

// AssemblyConfig is what assembling a silhouette chain needs beyond its silhouette config.
type AssemblyConfig struct {
	// Rollup is P's rollup config — G2's generated artifact. Its Genesis must sit AT this node's
	// anchor when the anchor is above P's real block 0 (G3 F2): op-node always derives from its
	// rollup config's genesis, so a config whose genesis sat at P's real block 0 would ask the shim
	// for every block below the anchor and fail stop on every one of them.
	Rollup *opnodecfg.Config
	// L1Chain is the settlement chain's config, read only for the L1-info transaction's blob-fee
	// field.
	L1Chain *params.ChainConfig
	// SysCfg is P's FROZEN genesis SystemConfig (DR-2).
	SysCfg eth.SystemConfig
	// L1 and Blobs are the shared L1 access the supernode already holds.
	L1    L1Source
	Blobs BlobSource
	// L1Headers is the headers-only L1 access the forced-extension convention needs. It is
	// separate from L1 because the convention deliberately reads no receipts, and the narrower
	// type is what makes that visible at the call site rather than only in a comment.
	L1Headers L1Headers
	// Labels selects the verifier or sequencer posture. See LabelSource.
	Labels LabelSource
	// TrackerInterval is how long the sequencer posture's proven-head tracker waits after catching
	// up with L1 before looking again. Zero takes the tracker's own default. It is here for tests
	// and for a settlement chain with an unusual block time, not as an operator knob.
	TrackerInterval time.Duration
}

// SequencerPostureSeqWindow is the sequencing window the SEQUENCER posture gives the virtual node's
// own derivation pipeline, and it is the disarm of a one-hour bomb (G4 D5).
//
// The bomb: a sequencing window is how long stock derivation waits for batch data before declaring
// the epoch empty and force-generating empty batches (`base_batch_stage.go`, `deriveNextEmptyBatch`).
// On a node that SEQUENCES P, consolidation then compares those empty blocks against the real ones
// its execution client produced, they do not match, and it reorgs the real chain out
// (`attributes.go`, `reorgOutUnsafeChain`). P has no batcher by construction, so with the committed
// window of 300 L1 blocks that fires about an hour after cutover and keeps firing — a delayed,
// total loss of P's private history, on a timer, looking green the whole first hour.
//
// Why the answer is "wait forever" rather than a bigger finite number: the window answers "how long
// should I wait for batch data before concluding there is none". For this pipeline the honest answer
// is that there is no batch data and there never will be — P's public history arrives as proofs, on
// a different L1 address, read by a different code path — so any epoch this pipeline declares empty
// is an epoch it invented. The committed finite window (DR-2) is a statement about the PROVEN
// rendering of P, which the verifiers and the prover implement and which this value does not touch;
// see the `committed` copy in Assemble.
//
// Why it is set here and not in the operator's rollup.json: a rollup.json is a file that gets
// copied. This value is only sound for a pipeline whose engine is the real sequencer's and whose
// chain has no batch inbox, and a standalone op-node started from a rollup.json carrying it would
// size its L1 receipt cache at three halves of it (`op-node/node/node.go`, initL1Source — reached
// only when no L1Source override is supplied, which is never the case inside a supernode). Setting
// it in the posture branch means it cannot travel to a node the reasoning does not cover.
//
// 2^32 L1 blocks is about sixteen centuries of Ethereum. It is deliberately not MaxUint64: every
// stock consumer adds it to an L1 block number (`epoch.Number + SeqWindowSize`,
// `Genesis.L1.Number + RecoverMinSeqWindows*SeqWindowSize`), and a value that overflowed those sums
// would re-arm the bomb rather than disarm it.
const SequencerPostureSeqWindow uint64 = 1 << 32

// Assemble builds a silhouette chain's machinery and wires the op-node seams IN PLACE on the
// caller's virtual-node config and initialization overrides.
//
// IT BRANCHES ON THE POSTURE, and the branch is the difference between a node that verifies P and
// the node that sequences it. Everything below describes the VERIFIER posture, which is what a
// public node runs and what the default `labels` value selects; the sequencer posture replaces none
// of these seams and is described at assembleSequencerPosture.
//
// The two seams are the whole mechanism, and both are stock extension points rather than forks:
//
//   - vncfg.L2 becomes an in-process client to the shim, so the embedded op-node's engine client
//     and the chain container's own engine controller both talk to a service that serves
//     proof-committed facts and executes nothing (DR-1).
//   - overrides.DataSourceOverride becomes the proof-batch source, so the derivation pipeline reads
//     transcoded proof batches where it would read batcher transactions. Everything downstream of
//     retrieval is unmodified stock derivation.
//
// Note what is NOT replaced: the pipeline, the attributes builder, the batch queue, the engine
// controller, the safety ladder. A silhouette chain is a DRIVEN chain in the supernode's eyes, and
// that is the claim the assembly exists to make good on.
//
// The overrides must be the ones handed to NewChainContainer: the container's restart loop
// overwrites RPCHandler, MetricsRegistry and SuperAuthority on every virtual-node restart but
// leaves DataSourceOverride alone, so setting it once here survives restarts.
func Assemble(logger log.Logger, cfg *Config, ac AssemblyConfig,
	vncfg *opnodecfg.Config, overrides *rollupNode.InitializationOverrides,
) (*Assembly, error) {
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("silhouette config: %w", err)
	}
	if vncfg == nil || overrides == nil {
		return nil, fmt.Errorf("silhouette assembly needs a virtual-node config and initialization overrides")
	}
	if ac.Rollup == nil {
		return nil, fmt.Errorf("silhouette assembly needs P's rollup config")
	}
	chainID := eth.ChainIDFromUInt64(bigs.Uint64Strict(ac.Rollup.Rollup.L2ChainID))

	verifier, err := cfg.NewVerifier()
	if err != nil {
		return nil, fmt.Errorf("build proof verifier for chain %s: %w", chainID, err)
	}

	facts := &FactStore{}

	// The COMMITTED rollup config, and it is a COPY of the virtual node's rather than a pointer into
	// it. That is not tidiness, it is the load-bearing line of the sequencer posture.
	//
	// Everything on the proof-facing side is computed from this config: acceptance's block-time
	// spacing, and — the one that bites — the forced-extension convention, which is
	// `epoch.Number + SeqWindowSize` against the pipeline origin (forced.go, forced_at.go). The
	// sequencer posture edits the VIRTUAL NODE's window to disarm the empty-batch bomb
	// (SequencerPostureSeqWindow), and if the two were the same object that edit would silently
	// change which forced blocks the sequencer computes. Every verifier would compute a forced
	// extension where the sequencer computed none, so the sequencer's proven head and every
	// verifier's proven head would be different renderings of the same chain — the sequencer
	// disagreeing with its own verifiers about P's public history, which is the failure G4 D1
	// exists to prevent, arriving through a field nobody was looking at.
	committed := ac.Rollup.Rollup

	source := NewDataSource(logger.New("component", "proof-source"), cfg, &committed,
		ac.L1Chain, ac.SysCfg, ac.L1, ac.Blobs, verifier, facts)

	a := &Assembly{ChainID: chainID, Facts: facts, Source: source, Labels: ac.Labels}

	if ac.Labels == LabelsFromProvenHead {
		if err := assembleSequencerPosture(logger, cfg, ac, &committed, a, vncfg, overrides); err != nil {
			return nil, err
		}
		return a, nil
	}

	// The shim: the source's chaining and the shim's fail-stop read the same fact table, and there
	// is exactly one of those per assembly, so they cannot be handed different ones.
	shim := NewShim(logger.New("component", "shim"), &committed, ac.L1Chain, ac.SysCfg,
		ac.L1Headers, facts)
	shimRPC, server, err := shim.InProc()
	if err != nil {
		return nil, fmt.Errorf("start in-process shim for chain %s: %w", chainID, err)
	}

	// Seam 1: the execution client. PreparedL2Endpoints is op-node's own in-process endpoint type,
	// so both the virtual node and the chain container's engine controller share this one client.
	vncfg.L2 = &opnodecfg.PreparedL2Endpoints{Client: restartableRPC{shimRPC}}

	// Seam 2: the derivation input.
	overrides.DataSourceOverride = source

	// Seam 3, which is not a replacement but a restoration. Replacing the L2 endpoint with an
	// in-process service took P's query surface off the network; this puts it back at the chain's
	// own route, so `eth_getBlockByNumber` answers for P and `silhouette_blockProvenance` can say
	// what proved each block. Without it, G2 D8's ruling — that a silhouette chain declares itself
	// at the SERVICE layer rather than in a header field — would be true of a service nothing can
	// reach.
	overrides.ExtraAPIs = append(overrides.ExtraAPIs, shim.PublicAPIs()...)

	// A silhouette chain never sequences. Its blocks are produced by a private sequencer and
	// established by proofs; a public node that tried to build one would be inventing history, and
	// the shim would fail stop on it. Pinning this here rather than trusting configuration means a
	// supernode started in sequencer mode cannot accidentally sequence P.
	vncfg.Driver.SequencerEnabled = false
	vncfg.Driver.SequencerStopped = true

	// And it takes no gossip. This is a SAFETY pin, not a tidiness one.
	//
	// A silhouette chain's blocks arrive as proofs, so gossip can only ever offer this node a block
	// it will not use. Most of the time that is harmless — the shim refuses a payload at a height it
	// holds no fact for, without halting. But a payload at a height it DOES hold, with the same
	// parent and a different hash, is exactly the shape the shim treats as a fatal disagreement
	// about proven history, and it halts permanently (G3 D4). That is the correct response when the
	// claim comes from derivation; it is an absurd one when the claim came from a stranger on a
	// gossip topic.
	//
	// So the door is closed rather than guarded: a chain whose history is established by proof has
	// nothing to learn from a peer, and leaving the topic subscribed would let any peer halt a
	// verifier. Found by an agent standing this up on a local cluster, where P's virtual node
	// inherited the harness's P2P config.
	vncfg.P2P = nil
	vncfg.P2PSigner = nil

	a.Shim = shim
	a.server = server
	a.client = shimRPC
	return a, nil
}

// assembleSequencerPosture wires the sequencer side, and the whole of its design is in what it does
// NOT do.
//
// It does not replace the execution client: on this node P's container fronts the REAL one, which is
// producing blocks from the private sequencer. It does not override the derivation input: the shim's
// rendered block hashes are deliberately not `keccak(RLP(header))` (F-P1), so proof-derived
// attributes consolidated against real blocks would mismatch and reorg every one of them. It does
// not pin `SequencerEnabled=false`: this node is the block producer. It does not pin P2P away: the
// gossip hazard the verifier posture closes is a payload that could halt the SHIM, and there is no
// shim here.
//
// What it adds is the walk. The verifier posture gets its facts as a side effect of deriving P; with
// no pipeline the same DataSource has to be driven explicitly, and it is driven by the SAME code
// every verifier runs, over the same L1 blobs, under the same acceptance rules — which is the point
// (tracker.go). The sequencer's view of what P has proven is arrived at by the same means as
// everyone else's, so it cannot drift from theirs, and its log-sink ingestion is therefore the wire
// (G4 D1) rather than the receipts it happens to be sitting on.
//
// And it disarms the empty-batch bomb. See SequencerPostureSeqWindow.
func assembleSequencerPosture(logger log.Logger, cfg *Config, ac AssemblyConfig, committed *rollup.Config,
	a *Assembly, vncfg *opnodecfg.Config, overrides *rollupNode.InitializationOverrides,
) error {
	start, err := trackerStartBlock(cfg, committed)
	if err != nil {
		return fmt.Errorf("chain %s: %w", a.ChainID, err)
	}
	a.Tracker = NewProvenHeadTracker(logger.New("component", "proven-head"), a.Source, ac.L1Headers,
		start, ac.TrackerInterval)

	// THE FROZEN SAFE HEAD, and the three stock mechanisms that read it as a fault.
	//
	// On this node P's public safe head never moves. That is not a degradation to be worked around,
	// it is the deployment: there is no batcher and no DA, so the virtual node's own derivation has
	// nothing to consolidate and its safe label sits at genesis forever, while the label the cluster
	// actually consumes is taken from the proven head one layer out (Container.OptimisticAt). Three
	// stock behaviours are written on the assumption that a safe head which stops moving means
	// something is wrong, and each of them acts on it. All three are pinned here, for this posture
	// only, and all three are the same bug seen from three angles.
	//
	// (1) THE EMPTY-BATCH BOMB, which is the one that destroys P. See SequencerPostureSeqWindow.
	vncfg.Rollup.SeqWindowSize = SequencerPostureSeqWindow

	// (2) The safe-lag stall. `evalMaxSafeLag` parks the sequencer permanently once
	// `safe + maxSafeLag <= unsafe` (op-node/rollup/sequencing/sequencer.go), and on this chain that
	// becomes true `maxSafeLag` blocks after cutover and stays true forever. The stock default is 0,
	// which disables it, so this pin changes nothing on a correctly configured node — which is
	// exactly why it is a pin rather than a note: the failure mode of an operator copying a lag from
	// chain A's config is that P's block production stops dead, minutes after a cutover that looked
	// green, and the log line blames a safe head that is never going to catch up.
	vncfg.Driver.SequencerMaxSafeLag = 0

	// (3) The sync-start walk-back. FindL2Heads walks the L2 chain backwards looking for a safe head
	// it can trust, and stops early once it has seen `SyncLookback()` — the sequencing window — worth
	// of L1 blocks with a canonical origin (op-node/rollup/sync/start.go). Pin (1) makes that early
	// stop unreachable, so the walk runs to the finalized head, which on this chain is genesis: one
	// `L2BlockRefByHash` per L2 block of the whole chain, on every restart, growing forever. At a
	// two-second block time that is over a million calls per restart after a month.
	//
	// This pin is the cost of pin (1) and is named as such rather than presented as free. What it
	// skips is a consistency check on the L1 origins of the UNSAFE chain — and on this node the
	// unsafe chain is the one this process sequenced itself, while the public verdict on P comes from
	// proofs. The check can only ever conclude "safe = genesis", which is what it started from.
	vncfg.Sync.SkipSyncStartCheck = true

	// The one public surface. See sequencer_api.go: without it the proven head — the state this whole
	// posture turns on — is not readable anywhere, because `optimism_syncStatus` correctly reports the
	// virtual node's own frozen labels rather than the ones the cluster consumes.
	overrides.ExtraAPIs = append(overrides.ExtraAPIs,
		provenHeadAPIs(a.Facts, a.Tracker, committed.SeqWindowSize, vncfg.Rollup.SeqWindowSize)...)

	logger.Info("silhouette sequencer posture: labels follow the proven head",
		"chain", a.ChainID, "tracker_start_l1", start,
		"committed_seq_window", committed.SeqWindowSize,
		"pipeline_seq_window", vncfg.Rollup.SeqWindowSize)
	return nil
}

// trackerStartBlock resolves the first L1 block the proven-head tracker reads.
//
// This is where `l1StartBlock` becomes a live field. It was declared for exactly this consumer and
// read by nothing while the tracker had no production caller, and the two honest options were to
// delete it or to give it its meaning; giving it its meaning is smaller, because the runbook already
// writes it into every verifier config and the alternative is editing those files to remove a field
// whose intent was right.
//
// Its meaning is bounded rather than free. Acceptance is a pure function of L1 from the anchor
// upwards (G2 D5), so a tracker that started ABOVE the L1 block carrying the first batch would skip
// the batch that extends the anchor and then reject every later batch for not chaining — a
// configuration mistake that presents as a total proof outage. The anchor's own L1 origin is the
// floor: nothing below it can carry a batch this node would accept. So zero means "the anchor's L1
// origin", a value at or below it is honoured as the operator's choice to start earlier, and a value
// above it is refused rather than clamped, because clamping would silently correct a number the
// operator is going to read back and believe.
func trackerStartBlock(cfg *Config, committed *rollup.Config) (uint64, error) {
	floor := cfg.Anchor.L1Origin.Number
	if committed.Genesis.L1.Number < floor {
		// An anchor above P's genesis (G3 F2) makes the anchor's origin the right floor; an anchor AT
		// genesis makes them equal. Taking the lower of the two costs a few L1 blocks of catch-up and
		// cannot skip a carrier.
		floor = committed.Genesis.L1.Number
	}
	if cfg.L1StartBlock == 0 {
		return floor, nil
	}
	if cfg.L1StartBlock > floor {
		return 0, fmt.Errorf("l1StartBlock %d is above the anchor's L1 origin %d: the proven-head "+
			"tracker would skip the L1 blocks that can carry the batch extending the anchor, and "+
			"then reject every later batch for not chaining", cfg.L1StartBlock, floor)
	}
	return cfg.L1StartBlock, nil
}

// AttachLogStore gives the assembly the interop log database its exported messages are sealed
// into, building the sink over it.
//
// It is separate from Assemble because of construction order and not by preference: the supernode
// opens a chain's LogsDB inside the interop activity, which is built AFTER the chains map it takes
// as an argument. So the chain exists first and its message database exists second, and this is the
// call that closes the loop.
func (a *Assembly) AttachLogStore(logger log.Logger, store LogStore) error {
	if store == nil {
		return fmt.Errorf("chain %s: no log store", a.ChainID)
	}
	return a.Source.AttachLogSink(NewLogSink(logger.New("component", "log-sink"), store))
}

// Close stops the in-process shim server.
func (a *Assembly) Close() {
	if a.client != nil {
		a.client.Close()
	}
	if a.server != nil {
		a.server.Stop()
	}
}
