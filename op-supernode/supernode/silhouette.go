package supernode

import (
	"context"
	"fmt"
	"time"

	gethlog "github.com/ethereum/go-ethereum/log"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
)

// assembleSilhouette connects a stock virtual node to a standalone op-silhouette-el. The supernode
// owns no proof verifier or fact table; it only consumes the EL's receipt-free interop RPC.
func (s *Supernode) assembleSilhouette(
	ctx context.Context,
	log gethlog.Logger,
	decl silhouette.ManifestChain,
	vnCfg *opnodecfg.Config,
) (*silhouette.RemoteAssembly, error) {
	chainID := eth.ChainIDFromUInt64(decl.ChainID)

	if err := decl.CheckRole(); err != nil {
		return nil, err
	}
	if got := bigs.Uint64Strict(vnCfg.Rollup.L2ChainID); got != decl.ChainID {
		return nil, fmt.Errorf("silhouette manifest declares chain %d but its rollup config is for chain %d",
			decl.ChainID, got)
	}
	rpc, _, err := vnCfg.L2.Setup(ctx, log.New("component", "silhouette-el-client"),
		&vnCfg.Rollup, &opmetrics.NoopRPCMetrics{})
	if err != nil {
		return nil, fmt.Errorf("connect to standalone silhouette EL for chain %s: %w", chainID, err)
	}
	var self *silhouette.SelfDeclaration
	if err := rpc.CallContext(ctx, &self, "silhouette_selfDeclaration"); err != nil {
		rpc.Close()
		return nil, fmt.Errorf("chain %s EL does not expose the silhouette component API: %w", chainID, err)
	}
	if self == nil || !self.ProofRendered || self.ExecutesTransactions || uint64(self.L2ChainID) != decl.ChainID {
		rpc.Close()
		return nil, fmt.Errorf("chain %s EL returned an incompatible silhouette self-declaration", chainID)
	}
	assembly := silhouette.NewRemoteAssembly(log.New("chain", chainID), chainID, rpc)

	// Proof-rendered verifier nodes never sequence or accept gossip. These are virtual-node posture
	// settings; all proof interpretation remains inside the standalone EL.
	vnCfg.Driver.SequencerEnabled = false
	vnCfg.Driver.SequencerStopped = true
	vnCfg.P2P = nil
	vnCfg.P2PSigner = nil
	s.supernodeMetrics.SilhouetteProvenHead.WithLabelValues(chainID.String()).Set(0)
	log.Info("connected standalone silhouette EL", "chain", chainID, "client", self.Client)
	return assembly, nil
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
		if err := assembly.AttachLogStore(store); err != nil {
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
			status, err := assembly.Source.Status(ctx)
			if err == nil && status.HeadFact != nil {
				s.supernodeMetrics.SilhouetteProvenHead.WithLabelValues(id).Set(float64(*status.HeadFact))
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
// import list its proofs declared, refusal to fetch receipts, replacement-aware invalidation, and
// proof-aware invalidation.
func wrapSilhouetteChain(log gethlog.Logger, inner cc.InteropChain, a *silhouette.RemoteAssembly) cc.InteropChain {
	logger := log.New("chain", a.ChainID)
	return silhouette.NewContainer(logger, inner, a.Source, a.LogSink)
}
