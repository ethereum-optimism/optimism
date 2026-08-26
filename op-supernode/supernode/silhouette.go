package supernode

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethlog "github.com/ethereum/go-ethereum/log"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
)

// assembleSilhouette turns one configured chain into a silhouette chain, in place.
//
// It is called from the chain loop BEFORE NewChainContainer, and everything it does is to the
// arguments that call is about to receive: the virtual-node config gets an in-process execution
// client that is the shim, and the initialization overrides get the proof-batch data source. The
// container built afterwards is a stock container — it is not told that its chain is unusual,
// because from where it stands it is not.
//
// The two shared L1 resources come from the supernode, which is the whole reason this lives here
// rather than in the silhouette package: a silhouette chain reads L1 exactly like every other chain
// does, through the same client and the same beacon, and giving it its own would be a second view
// of L1 that could disagree with the one the cross-safety round checks canonicality against.
func (s *Supernode) assembleSilhouette(
	log gethlog.Logger,
	decl silhouette.ManifestChain,
	vnCfg *opnodecfg.Config,
	overrides *rollupNode.InitializationOverrides,
) (*silhouette.Assembly, silhouette.LabelSource, error) {
	chainID := eth.ChainIDFromUInt64(decl.ChainID)
	cfg := decl.Config()

	labels, err := decl.LabelSource()
	if err != nil {
		return nil, 0, err
	}
	if s.beaconClient == nil {
		// A silhouette chain's history travels in blobs, so a beacon endpoint is not optional the
		// way it is for a calldata rollup. Saying so at startup beats deriving nothing and looking
		// like a stalled prover.
		return nil, 0, fmt.Errorf("chain %s is a silhouette chain and needs an L1 beacon endpoint", chainID)
	}
	l1Chain, err := silhouette.L1ChainConfig(cfg)
	if err != nil {
		return nil, 0, fmt.Errorf("chain %s: %w", chainID, err)
	}
	if got := vnCfg.Rollup.L2ChainID.Uint64(); got != decl.ChainID {
		return nil, 0, fmt.Errorf("silhouette manifest declares chain %d but its rollup config is for chain %d",
			decl.ChainID, got)
	}

	l1 := resources.NewNonCloseableL1Client(s.l1Client)
	assembly, err := silhouette.Assemble(log, cfg, silhouette.AssemblyConfig{
		Rollup: vnCfg,
		// The FROZEN genesis SystemConfig (DR-2) is read off the rollup config rather than
		// configured separately. There is exactly one right answer and the rollup config already
		// holds it; a second copy could only ever be a way to disagree with it.
		SysCfg:    vnCfg.Rollup.Genesis.SystemConfig,
		L1Chain:   l1Chain,
		L1:        l1,
		L1Headers: l1,
		Blobs:     resources.NewNonCloseableL1BeaconClient(s.beaconClient),
		Labels:    labels,
	}, vnCfg, overrides)
	if err != nil {
		return nil, 0, fmt.Errorf("assemble silhouette chain %s: %w", chainID, err)
	}
	// THE HALT, MADE OBSERVABLE. G3 D4 deliberately declines to call log.Crit for the fail-stop —
	// killing the process is this layer's decision, not the shim's — and left the callback unwired,
	// which meant a halted shim looked SLOW rather than dead: the process stays up, the RPC route
	// stays up, and every request is refused.
	//
	// The decision this layer makes is to say so loudly and keep running, not to exit. A halt is a
	// disagreement about proven history, and the state of a node that has stopped serving a wrong
	// answer is worth an operator's inspection; a process that killed itself would take its own
	// evidence with it, and on a multi-chain supernode it would take the OTHER chains down with it
	// too. So: a gauge to alert on, and a log line at the layer that owns the deployment.
	if assembly.Shim != nil {
		metrics := s.supernodeMetrics
		id := chainID.String()
		metrics.SilhouetteShimHalted.WithLabelValues(id).Set(0)
		assembly.Shim.OnHalt(func(reason error) {
			metrics.SilhouetteShimHalted.WithLabelValues(id).Set(1)
			log.Error("silhouette chain has fail-stopped and will refuse every further request; "+
				"this supernode's other chains are unaffected and this process is NOT exiting",
				"chain", chainID, "reason", reason)
		})
	}
	s.supernodeMetrics.SilhouetteProvenHead.WithLabelValues(chainID.String()).Set(0)
	s.supernodeMetrics.SilhouetteTrackerL1.WithLabelValues(chainID.String()).Set(0)

	// proofType and dependenciesVerified are in the startup line because they are the two properties
	// of a silhouette verifier that are invisible from the outside: every proving system and both wire
	// versions derive the chain, serve the same roots and report the same heads. Only these two say
	// what an accepted batch MEANT — whether the chain's state was proven or attested (V1G), and
	// whether its IMPORTS were ever checked (G7). An operator reading a runbook needs both, once, at
	// boot.
	log.Info("assembled silhouette chain",
		"chain", chainID,
		"labels", decl.Labels,
		"wireVersion", cfg.EffectiveWireVersion(),
		"dependenciesVerified", cfg.DependenciesVerified(),
		"proofType", cfg.ProofType,
		"anchor", cfg.Anchor.BlockNumber,
		"anchorOutputRoot", cfg.Anchor.OutputRoot)
	return assembly, labels, nil
}

// attachSilhouetteLogStores closes the loop between each silhouette chain's data source and the
// interop log database its exported messages are sealed into.
//
// This runs after interop is constructed because that is when the databases exist, and it is a hard
// error rather than a warning when interop is configured and a store is missing: a silhouette chain
// whose sink never attached would derive P perfectly and export nothing, so the failure would look
// like a chain that simply had no cross-chain traffic. Silence is the wrong shape for that bug.
func (s *Supernode) attachSilhouetteLogStores(log gethlog.Logger, in *interop.Interop) error {
	for chainID, assembly := range s.silhouettes {
		store, ok := in.LogsDBForChain(chainID)
		if !ok {
			return fmt.Errorf("silhouette chain %s has no interop log database", chainID)
		}
		if err := assembly.AttachLogStore(log, store); err != nil {
			return fmt.Errorf("attach log store for silhouette chain %s: %w", chainID, err)
		}
		log.Info("attached silhouette log sink", "chain", chainID)
	}
	return nil
}

// silhouetteObserveInterval is how often the gauges are refreshed. It is slower than anything it
// measures on purpose: these are for alerting on a stall over minutes, not for tracking a head.
const silhouetteObserveInterval = 5 * time.Second

// observeSilhouettes keeps the silhouette gauges current.
//
// A polling loop rather than a callback on every accepted batch, and one loop for every chain rather
// than one each: the values are a proven head and an L1 cursor, both of which are already state that
// something else owns, and reading them is cheaper than notifying about them. The alternative would
// put a metrics dependency inside the silhouette package, on the acceptance path, for a number an
// alert reads once a minute.
func (s *Supernode) observeSilhouettes(ctx context.Context) {
	if len(s.silhouettes) == 0 {
		return
	}
	ticker := time.NewTicker(silhouetteObserveInterval)
	defer ticker.Stop()
	for {
		for chainID, assembly := range s.silhouettes {
			id := chainID.String()
			if head, ok := assembly.Facts.Head(); ok {
				s.supernodeMetrics.SilhouetteProvenHead.WithLabelValues(id).Set(float64(head.Number))
			}
			if assembly.Tracker != nil {
				s.supernodeMetrics.SilhouetteTrackerL1.WithLabelValues(id).Set(float64(assembly.Tracker.Cursor()))
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// wrapSilhouetteChain gives a stock container the silhouette behaviours: proven ingestion, the
// import list its proofs declared, refusal to fetch receipts or to be invalidated, and (in the
// sequencer posture) labels from the proven head.
func wrapSilhouetteChain(log gethlog.Logger, inner cc.InteropChain, a *silhouette.Assembly, labels silhouette.LabelSource, metrics *resources.SupernodeMetrics) cc.InteropChain {
	logger := log.New("chain", a.ChainID)
	container := silhouette.NewContainer(logger, inner, a.Facts, labels)
	// A refused invalidation must be able to PAGE, not only to appear in a log.
	//
	// Under G7 the judge can find a block of a proof-carried chain invalid — its dependencies are on
	// the wire and really checked — and this node then refuses to replace it (G7G D3). The state that
	// leaves behind is operationally identical to the shim's halt: the chain cannot advance, the
	// dependency set's cross-safe frontier is pinned behind it, and no local action fixes it. The
	// difference is that a halt is a gauge on a component and this is a verdict about history, so it
	// gets its own counter rather than borrowing that one. The series is created at zero for every
	// declared chain, because an alert on an absent series never fires.
	if metrics != nil {
		counter := metrics.SilhouetteInvalidationsRefused.WithLabelValues(a.ChainID.String())
		counter.Add(0)
		container.OnInvalidationRefused(func(height uint64, hash common.Hash) {
			counter.Inc()
			logger.Error("the cross-safety judge found a block of this proof-carried chain INVALID and "+
				"this node refused to replace it; the dependency set's cross-safe frontier is now pinned "+
				"at this block and only a re-proof from the last valid point can clear it",
				"chain", a.ChainID, "height", height, "hash", hash)
		})
	}
	return container
}
