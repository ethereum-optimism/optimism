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
	deployerPipelineOptions []DeployerPipelineOption

	superchains    locks.RWMap[stack.SuperchainID, *Superchain]
	clusters       locks.RWMap[stack.ClusterID, *Cluster]
	l1Nets         locks.RWMap[eth.ChainID, *L1Network]
	l2Nets         locks.RWMap[eth.ChainID, *L2Network]
	l1ELs          locks.RWMap[stack.L1ELNodeID, *L1ELNode]
	l1CLs          locks.RWMap[stack.L1CLNodeID, *L1CLNode]
	l2ELs          locks.RWMap[stack.L2ELNodeID, L2ELNode]
	l2CLs          locks.RWMap[stack.L2CLNodeID, L2CLNode]
	supervisors    locks.RWMap[stack.SupervisorID, *Supervisor]
	testSequencers locks.RWMap[stack.TestSequencerID, *TestSequencer]
	batchers       locks.RWMap[stack.L2BatcherID, *L2Batcher]
	challengers    locks.RWMap[stack.L2ChallengerID, *L2Challenger]
	proposers      locks.RWMap[stack.L2ProposerID, *L2Proposer]

	syncTester *SyncTester
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
	for _, cluster := range o.clusters.Values() {
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

var _ stack.Orchestrator = (*Orchestrator)(nil)

func NewOrchestrator(p devtest.P, hook stack.SystemHook) *Orchestrator {
	o := &Orchestrator{p: p, sysHook: hook}
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
	o.superchains.Range(rangeHydrateFn[stack.SuperchainID, *Superchain](sys))
	o.clusters.Range(rangeHydrateFn[stack.ClusterID, *Cluster](sys))
	o.l1Nets.Range(rangeHydrateFn[eth.ChainID, *L1Network](sys))
	o.l2Nets.Range(rangeHydrateFn[eth.ChainID, *L2Network](sys))
	o.l1ELs.Range(rangeHydrateFn[stack.L1ELNodeID, *L1ELNode](sys))
	o.l1CLs.Range(rangeHydrateFn[stack.L1CLNodeID, *L1CLNode](sys))
	o.l2ELs.Range(rangeHydrateFn[stack.L2ELNodeID, L2ELNode](sys))
	o.l2CLs.Range(rangeHydrateFn[stack.L2CLNodeID, L2CLNode](sys))
	o.supervisors.Range(rangeHydrateFn[stack.SupervisorID, *Supervisor](sys))
	o.testSequencers.Range(rangeHydrateFn[stack.TestSequencerID, *TestSequencer](sys))
	o.batchers.Range(rangeHydrateFn[stack.L2BatcherID, *L2Batcher](sys))
	o.challengers.Range(rangeHydrateFn[stack.L2ChallengerID, *L2Challenger](sys))
	o.proposers.Range(rangeHydrateFn[stack.L2ProposerID, *L2Proposer](sys))
	o.faucet.hydrate(sys)
	o.sysHook.PostHydrate(sys)
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
