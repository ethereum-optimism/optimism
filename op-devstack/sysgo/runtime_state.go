package sysgo

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	challengerconfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	coredepset "github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	// Node is set on the op-node-verifier path; nil on the supernode path.
	Node *SingleChainNodeRuntime
	// EL is the sync-tester-backed L2ELNode.
	EL L2ELNode
	// CL drives the sync-tester EL (op-node or SuperNodeProxy).
	CL L2CLNode
}

type FlashblocksRuntimeSupport struct {
	Builder     *OPRBuilderNode
	RollupBoost *RollupBoostNode
}

type SingleChainInteropSupport struct {
	Migration     *interopMigrationState
	FullConfigSet config.FullConfigSetMerged
	DependencySet coredepset.DependencySet
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

func (r *SingleChainRuntime) VMConfig(t devtest.T, dir string) *vm.Config {
	konaHostPath, err := rustbin.Spec{
		SrcDir:  "rust/kona",
		Package: "kona-host",
		Binary:  "kona-host",
	}.EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "locate/build kona-host")

	rollupCfgPath := filepath.Join(dir, "rollup.json")
	rollupBytes, err := json.Marshal(r.L2Network.RollupConfig())
	t.Require().NoError(err, "marshal rollup config")
	t.Require().NoError(os.WriteFile(rollupCfgPath, rollupBytes, 0o644), "write rollup config")

	l1GenesisPath := filepath.Join(dir, "l1-genesis.json")
	l1GenesisBytes, err := json.Marshal(r.L1Network.Genesis())
	t.Require().NoError(err, "marshal l1 genesis")
	t.Require().NoError(os.WriteFile(l1GenesisPath, l1GenesisBytes, 0o644), "write l1 genesis")

	return &vm.Config{
		L1:                r.L1EL.UserRPC(),
		L1Beacon:          r.L1CL.BeaconHTTPAddr(),
		L2s:               []string{r.L2EL.UserRPC()},
		RollupConfigPaths: []string{rollupCfgPath},
		L1GenesisPath:     l1GenesisPath,
		Server:            konaHostPath,
	}
}

type MultiChainNodeRuntime struct {
	Name        string
	Network     *L2Network
	EL          L2ELNode
	CL          L2CLNode
	SupernodeCL L2CLNode
	Batcher     *L2Batcher
	Proposer    *L2Proposer
	Followers   map[string]*SingleChainNodeRuntime
}

type MultiChainRuntime struct {
	Keys          devkeys.Keys
	Migration     *interopMigrationState
	FullConfigSet config.FullConfigSetMerged
	DependencySet coredepset.DependencySet

	L1Network *L1Network
	L1EL      *L1Geth
	L1CL      *L1CLNode

	Chains map[string]*MultiChainNodeRuntime

	Supernode *SuperNode

	FaucetService      *faucet.Service
	TimeTravel         *clock.AdvancingClock
	TestSequencer      *TestSequencerRuntime
	L2ChallengerConfig *challengerconfig.Config
	DelaySeconds       uint64
	InteropFilter      *InteropFilter // nil if not using interop filter
	SyncTester         *SyncTesterRuntime
}
