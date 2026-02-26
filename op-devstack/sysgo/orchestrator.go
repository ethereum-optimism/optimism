package sysgo

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Orchestrator struct {
	p devtest.P

	keys devkeys.Keys

	wb *worldBuilder

	// nil if no time travel is supported
	timeTravelClock *clock.AdvancingClock

	// options
	batcherOptions          []BatcherOption
	proposerOptions         []ProposerOption
	l2CLOptions             L2CLOptionBundle
	l2ELOptions             L2ELOptionBundle
	l2ChallengerOpts        l2ChallengerOpts
	SyncTesterELOptions     SyncTesterELOptionBundle
	deployerPipelineOptions []DeployerPipelineOption

	// Unified component registry - replaces the 15 separate locks.RWMap fields
	registry *stack.Registry

	// supernodes is stored separately because SupernodeID cannot be converted to ComponentID
	supernodes locks.RWMap[stack.SupernodeID, *SuperNode]

	// service name => prometheus endpoints to scrape
	l2MetricsEndpoints locks.RWMap[string, []PrometheusMetricsTarget]

	syncTester *SyncTesterService
	faucet     *FaucetService

	controlPlane *ControlPlane

	// sysHook is called after hydration of a new test-scope system frontend,
	// essentially a test-case preamble.
	sysHook stack.SystemHook

	jwtPath     string
	jwtSecret   [32]byte
	jwtPathOnce sync.Once
}

func (o *Orchestrator) Type() compat.Type {
	return compat.SysGo
}

func (o *Orchestrator) ClusterForL2(chainID eth.ChainID) (*Cluster, bool) {
	clusters := stack.RegistryGetByKind[*Cluster](o.registry, stack.KindCluster)
	for _, cluster := range clusters {
		if cluster.DepSet() != nil && cluster.DepSet().HasChain(chainID) {
			return cluster, true
		}
	}
	return nil, false
}

func (o *Orchestrator) ControlPlane() stack.ControlPlane {
	return o.controlPlane
}

func (o *Orchestrator) EnableTimeTravel() {
	if o.timeTravelClock == nil {
		o.timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
	}
}

// GetL2EL retrieves an L2 EL node by its ID from the registry.
// Supports polymorphic lookup: if the ID was converted from another L2EL-capable type
// (e.g., OPRBuilderNodeID), searches across all L2EL-capable kinds using same key/chainID.
func (o *Orchestrator) GetL2EL(id stack.L2ELNodeID) (L2ELNode, bool) {
	for _, kind := range stack.L2ELCapableKinds() {
		cid := stack.NewComponentID(kind, id.Key(), id.ChainID())
		if component, ok := o.registry.Get(cid); ok {
			if el, ok := component.(L2ELNode); ok {
				return el, true
			}
		}
	}
	return nil, false
}

var _ stack.Orchestrator = (*Orchestrator)(nil)

func NewOrchestrator(p devtest.P, hook stack.SystemHook) *Orchestrator {
	o := &Orchestrator{
		p:        p,
		sysHook:  hook,
		registry: stack.NewRegistry(),
	}
	o.controlPlane = &ControlPlane{o: o}
	return o
}

func (o *Orchestrator) P() devtest.P {
	return o.p
}

func (o *Orchestrator) writeDefaultJWT() (jwtPath string, secret [32]byte) {
	o.jwtPathOnce.Do(func() {
		// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
		o.jwtPath = filepath.Join(o.p.TempDir(), "jwt_secret")
		o.jwtSecret = [32]byte{123}
		err := os.WriteFile(o.jwtPath, []byte(hexutil.Encode(o.jwtSecret[:])), 0o600)
		require.NoError(o.p, err, "failed to prepare jwt file")
	})
	return o.jwtPath, o.jwtSecret
}

func (o *Orchestrator) Hydrate(sys stack.ExtensibleSystem) {
	o.sysHook.PreHydrate(sys)
	if o.timeTravelClock != nil {
		ttSys, ok := sys.(stack.TimeTravelSystem)
		if ok {
			ttSys.SetTimeTravelClock(o.timeTravelClock)
		}
	}

	// Hydrate all components in the unified registry.
	for _, kind := range stack.HydrationComponentKindOrder() {
		o.registry.RangeByKind(kind, func(id stack.ComponentID, component any) bool {
			if h, ok := component.(hydrator); ok {
				h.hydrate(sys)
			}
			return true
		})
	}

	o.supernodes.Range(rangeHydrateFn[stack.SupernodeID, *SuperNode](sys))

	if o.syncTester != nil {
		o.syncTester.hydrate(sys)
	}
	o.faucet.hydrate(sys)
	o.sysHook.PostHydrate(sys)
}

func (o *Orchestrator) RegisterL2MetricsTargets(id stack.IDWithChain, endpoints ...PrometheusMetricsTarget) {
	wasSet := o.l2MetricsEndpoints.SetIfMissing(id.Key(), endpoints)
	if !wasSet {
		existing, _ := o.l2MetricsEndpoints.Get(id.Key())
		o.p.Logger().Warn("multiple endpoints registered with the same key", "key", id.Key(), "existing", existing, "new", endpoints)
	}
}

// InteropTestControl returns the InteropTestControl for a given SupernodeID.
// Returns nil if the supernode doesn't exist or doesn't implement the interface.
// This function is for integration test control only.
func (o *Orchestrator) InteropTestControl(id stack.SupernodeID) stack.InteropTestControl {
	sn, ok := o.supernodes.Get(id)
	if !ok {
		return nil
	}
	return sn
}

type hydrator interface {
	hydrate(system stack.ExtensibleSystem)
}

func rangeHydrateFn[I any, H hydrator](sys stack.ExtensibleSystem) func(id I, v H) bool {
	return func(id I, v H) bool {
		v.hydrate(sys)
		return true
	}
}
