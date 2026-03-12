package sysgo

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	challengerconfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
)

type TestSequencerRuntime struct {
	Name       string
	AdminRPC   string
	JWTSecret  [32]byte
	ControlRPC map[eth.ChainID]string
	Service    *sequencer.Service
}

func newTestSequencerRuntime(ts *testSequencer, name string) *TestSequencerRuntime {
	if ts == nil {
		return nil
	}
	if name == "" {
		name = ts.name
	}
	return &TestSequencerRuntime{
		Name:       name,
		AdminRPC:   ts.adminRPC,
		JWTSecret:  ts.jwtSecret,
		ControlRPC: copyControlRPCMap(ts.controlRPC),
		Service:    ts.service,
	}
}

type SingleChainNodeRuntime struct {
	Name        string
	IsSequencer bool
	EL          L2ELNode
	CL          L2CLNode
}

type SyncTesterRuntime struct {
	Service *SyncTesterService
	Node    *SingleChainNodeRuntime
}

type FlashblocksRuntimeSupport struct {
	Builder     *OPRBuilderNode
	RollupBoost *RollupBoostNode
}

type SingleChainInteropSupport struct {
	Migration     *interopMigrationState
	FullConfigSet depset.FullConfigSetMerged
	DependencySet depset.DependencySet
	Supervisor    Supervisor
}

type SingleChainRuntime struct {
	Keys devkeys.Keys

	L1Network *L1Network
	L2Network *L2Network

	L1EL *L1Geth
	L1CL *L1CLNode

	L2EL L2ELNode
	L2CL L2CLNode

	L2Batcher    *L2Batcher
	L2Proposer   *L2Proposer
	L2Challenger *L2Challenger

	FaucetService *faucet.Service
	TimeTravel    *clock.AdvancingClock
	TestSequencer *TestSequencerRuntime

	Nodes       map[string]*SingleChainNodeRuntime
	SyncTester  *SyncTesterRuntime
	Conductors  map[string]*Conductor
	Flashblocks *FlashblocksRuntimeSupport
	Interop     *SingleChainInteropSupport
	P2PEnabled  bool
}

type MultiChainNodeRuntime struct {
	Name      string
	Network   *L2Network
	EL        L2ELNode
	CL        L2CLNode
	Batcher   *L2Batcher
	Proposer  *L2Proposer
	Followers map[string]*SingleChainNodeRuntime
}

type MultiChainRuntime struct {
	Keys          devkeys.Keys
	Migration     *interopMigrationState
	FullConfigSet depset.FullConfigSetMerged
	DependencySet depset.DependencySet

	L1Network *L1Network
	L1EL      *L1Geth
	L1CL      *L1CLNode

	Chains map[string]*MultiChainNodeRuntime

	PrimarySupervisor   Supervisor
	SecondarySupervisor Supervisor
	Supernode           *SuperNode

	FaucetService      *faucet.Service
	TimeTravel         *clock.AdvancingClock
	TestSequencer      *TestSequencerRuntime
	L2ChallengerConfig *challengerconfig.Config
	DelaySeconds       uint64
}
