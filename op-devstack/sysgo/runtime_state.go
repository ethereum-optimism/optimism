package sysgo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen/config"
	challengerconfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	coredepset "github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
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
	// ZKChallengerSuperRootRPCProxy is set when the runtime starts a ZK challenger.
	ZKChallengerSuperRootRPCProxy *StallableProxy

	TimeTravel    *clock.AdvancingClock
	TestSequencer *TestSequencerRuntime

	Nodes      map[string]*SingleChainNodeRuntime
	SyncTester *SyncTesterRuntime
	Conductors map[string]*Conductor
	Interop    *SingleChainInteropSupport
	P2PEnabled bool
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
	Name    string
	Network *L2Network
	// EL is the chain's primary EL. In light-sequencer presets it is the
	// follow-mode sequencer's own EL; in virtual-sequencer presets it is the
	// same EL the supernode VN drives.
	EL          L2ELNode
	CL          L2CLNode
	SupernodeCL L2CLNode
	// SupernodeEL is the EL the supernode VN reads/writes. In light-sequencer
	// presets it is distinct from EL (production topology: separate ELs joined
	// only by L1 + P2P); in virtual-sequencer presets it equals EL.
	SupernodeEL L2ELNode
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

	TimeTravel         *clock.AdvancingClock
	TestSequencer      *TestSequencerRuntime
	L2ChallengerConfig *challengerconfig.Config
	startZKProposerFn  func() *ZKProposerRuntime
	zkProposer         *ZKProposerRuntime
	DelaySeconds       uint64
	InteropFilter      *InteropFilter // nil if not using interop filter
	SyncTester         *SyncTesterRuntime
}

// StartZKProposer starts the configured kona-sp1-proposer. It is intended for
// tests that seed dispute games with WithoutHonestProposer before allowing the
// proposer to observe them.
func (r *MultiChainRuntime) StartZKProposer(t devtest.T) *ZKProposerRuntime {
	proposer, err := r.startZKProposerRuntime()
	t.Require().NoError(err)
	return proposer
}

// ZKProposer returns the running kona-sp1-proposer.
func (r *MultiChainRuntime) ZKProposer(t devtest.T) *ZKProposerRuntime {
	proposer, err := r.zkProposerRuntime()
	t.Require().NoError(err)
	return proposer
}

func (r *MultiChainRuntime) startZKProposerRuntime() (*ZKProposerRuntime, error) {
	if r.startZKProposerFn == nil {
		if r.zkProposer != nil {
			return nil, errors.New("ZK proposer is already started")
		}
		return nil, errors.New("ZK proposer is not configured")
	}
	start := r.startZKProposerFn
	r.startZKProposerFn = nil
	r.zkProposer = start()
	return r.zkProposer, nil
}

func (r *MultiChainRuntime) zkProposerRuntime() (*ZKProposerRuntime, error) {
	if r.zkProposer != nil {
		return r.zkProposer, nil
	}
	if r.startZKProposerFn != nil {
		return nil, errors.New("ZK proposer is configured but not started; call StartZKProposer")
	}
	return nil, errors.New("ZK proposer is not configured")
}
