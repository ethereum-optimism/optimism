package silhouette

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
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
	tracker *ProvenHeadTracker
	sink    *LogSink
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
}

// Assemble builds a silhouette chain's proof observer beside an otherwise stock op-node virtual
// node. The virtual node reads the ordinary empty-batch carrier posted by the producer; this observer
// reads the proof envelope in the same L1 transaction and supplies imports to the interop judge.
//
// A silhouette supernode is verifier-only. Private sequencing is performed by a LightCL-based
// sequencing node and is deliberately not a mode of this assembly.
//
// vncfg.L2 names the standalone op-silhouette-el service, just as a normal virtual node names its
// execution client. No op-node derivation hook is installed: its data source, frame queue, channel
// decoder, attributes builder, engine controller and safety ladder are all stock.
//
// Note what is NOT replaced: the pipeline, the attributes builder, the batch queue, the engine
// controller, the safety ladder. A silhouette chain is a DRIVEN chain in the supernode's eyes, and
// that is the claim the assembly exists to make good on.
func Assemble(logger log.Logger, cfg *Config, ac AssemblyConfig,
	vncfg *opnodecfg.Config,
) (*Assembly, error) {
	if err := cfg.Check(); err != nil {
		return nil, fmt.Errorf("silhouette config: %w", err)
	}
	if vncfg == nil {
		return nil, fmt.Errorf("silhouette assembly needs a virtual-node config")
	}
	if ac.Rollup == nil {
		return nil, fmt.Errorf("silhouette assembly needs P's rollup config")
	}
	if vncfg.L2 == nil {
		return nil, fmt.Errorf("silhouette assembly needs an op-silhouette-el endpoint")
	}
	chainID := eth.ChainIDFromUInt64(bigs.Uint64Strict(ac.Rollup.Rollup.L2ChainID))

	verifier, err := cfg.NewVerifier()
	if err != nil {
		return nil, fmt.Errorf("build proof verifier for chain %s: %w", chainID, err)
	}

	facts := &FactStore{}

	// Keep a private copy because proof acceptance and forced-extension calculations treat the
	// rollup config as immutable.
	committed := ac.Rollup.Rollup

	source := NewDataSource(logger.New("component", "proof-source"), cfg, &committed,
		ac.L1Chain, ac.SysCfg, ac.L1, ac.Blobs, verifier, facts)

	start := cfg.L1StartBlock
	if start == 0 {
		start = committed.Genesis.L1.Number
	}
	a := &Assembly{
		ChainID: chainID,
		Facts:   facts,
		Source:  source,
		tracker: NewProvenHeadTracker(logger.New("component", "proof-walker"), source, ac.L1Headers, start, 2*time.Second),
	}

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

	return a, nil
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
	sink := NewLogSink(logger.New("component", "log-sink"), store)
	if err := a.Source.AttachLogSink(sink); err != nil {
		return err
	}
	a.sink = sink
	return nil
}

// LogSink returns the sink shared by proof acceptance and replacement invalidation.
func (a *Assembly) LogSink() *LogSink { return a.sink }

// Run observes the authenticated proof stream independently of the stock virtual node. The latter
// derives only the ordinary empty-batch carrier and talks to the standalone magic EL.
func (a *Assembly) Run(ctx context.Context) { a.tracker.Run(ctx) }

// Close is retained for assembly ownership symmetry; Run is stopped by the supernode lifecycle.
func (a *Assembly) Close() {}
