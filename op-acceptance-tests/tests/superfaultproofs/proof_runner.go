package superfaultproofs

import (
	"strconv"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/proofs"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ProofRunner executes the proof checks attached to one shared interop scenario.
// Implementations are constructed here so scenario internals remain private.
type ProofRunner interface {
	run(t devtest.T, sys *presets.SingleChainInterop, chains []*chain, data *scenarioProofData)
}

type scenarioProofData struct {
	fpvmTransitions       []*transitionTest
	fpvmStartTimestamp    uint64
	cannonTargetTimestamp uint64
	zkCheckpoint          *zkCheckpoint
}

type zkCheckpoint struct {
	endTimestamp      uint64
	trustedL1Head     eth.BlockID
	expectReplacement bool
}

type konaProofRunner struct{}

type cannonKonaSuperProofRunner struct{}

type sp1ProofRunner struct {
	executorPath string
	nativeCore   bool
}

// NewKonaProofRunner constructs the existing Kona FPVM and challenger runner.
func NewKonaProofRunner() ProofRunner {
	return konaProofRunner{}
}

// NewCannonKonaSuperProofRunner constructs a full super-cannon-kona dispute-game runner.
// Unlike NewKonaProofRunner, this executes the kona-client ELF through Cannon MIPS.
func NewCannonKonaSuperProofRunner() ProofRunner {
	return cannonKonaSuperProofRunner{}
}

// NewSP1NativeProofRunner constructs the fast runner that replays witnesses through the shared
// native range and consolidation cores.
func NewSP1NativeProofRunner() ProofRunner {
	return sp1ProofRunner{nativeCore: true}
}

// NewSP1FullProofRunner constructs the opt-in runner used by the full-ELF smoke tests.
func NewSP1FullProofRunner(executorPath string) ProofRunner {
	return sp1ProofRunner{executorPath: executorPath}
}

func runScenarioProofs(t devtest.T, sys *presets.SingleChainInterop, chains []*chain, data *scenarioProofData, runners ...ProofRunner) {
	if len(runners) == 0 {
		runners = []ProofRunner{NewKonaProofRunner()}
	}
	for _, runner := range runners {
		runner.run(t, sys, chains, data)
	}
}

func hasSP1Runner(runners []ProofRunner) bool {
	for _, runner := range runners {
		if _, ok := runner.(sp1ProofRunner); ok {
			return true
		}
	}
	return false
}

func (konaProofRunner) run(t devtest.T, sys *presets.SingleChainInterop, _ []*chain, data *scenarioProofData) {
	t.Require().NotEmpty(data.fpvmTransitions, "Kona runner requires FPVM transition cases")
	challengerCfg := sys.L2ChainA.Escape().L2Challengers()[0].Config()
	gameDepth := sys.DisputeGameFactory().GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()
	for _, test := range data.fpvmTransitions {
		t.Run(test.Name+"-fpp", func(t devtest.T) {
			runKonaInteropProgram(t, challengerCfg.CannonKona, test.L1Head.Hash,
				test.AgreedClaim, crypto.Keccak256Hash(test.DisputedClaim),
				test.ClaimTimestamp, test.ExpectValid)
		})
		t.Run(test.Name+"-challenger", func(t devtest.T) {
			runChallengerProviderTest(t, sys.SuperRoots.QueryAPI(), gameDepth,
				data.fpvmStartTimestamp, test.ClaimTimestamp, test)
		})
	}
}

func (cannonKonaSuperProofRunner) run(t devtest.T, sys *presets.SingleChainInterop, _ []*chain, data *scenarioProofData) {
	t.Require().NotZero(data.cannonTargetTimestamp, "Cannon runner requires a target timestamp")
	t.Run("CannonKonaSuperGame", func(t devtest.T) {
		attacker := sys.FunderL1.NewFundedEOA(eth.ThousandEther)
		game := sys.DisputeGameFactory().StartSuperCannonKonaGame(
			attacker,
			proofs.WithL2SequenceNumber(data.cannonTargetTimestamp),
		)

		// The root is the canonical super root. Dispute the target timestamp with an invalid
		// claim, wait for the honest challenger to counter it, and then descend into the
		// Cannon trace. Reaching the bottom step requires the kona-client MIPS ELF to execute
		// the target transition.
		claim := game.DisputeL2SequenceNumber(
			attacker,
			game.RootClaim(),
			data.cannonTargetTimestamp,
		)
		game.LogGameData()
		claim = claim.WaitForCounterClaim()
		claim = game.DisputeToStep(attacker, claim, 1000)
		game.LogGameData()
		claim.WaitForCountered()
	})
}

func (r sp1ProofRunner) run(t devtest.T, sys *presets.SingleChainInterop, chains []*chain, data *scenarioProofData) {
	executorPath := r.executorPath
	if r.nativeCore {
		var err error
		executorPath, err = rustbin.Spec{
			SrcDir:  "rust/kona",
			Package: "kona-sp1-super-range-executor",
			Binary:  "kona-sp1-super-range-executor",
		}.EnsureExists(t.Ctx(), t.Logger())
		t.Require().NoError(err, "locate kona-sp1 super-range executor")
		if err != nil {
			return
		}
	}
	t.Require().NotEmpty(executorPath, "SP1 super-range executor path is required")
	t.Require().NotNil(data.zkCheckpoint, "SP1 runner requires a ZK checkpoint")
	name := "SP1FullELF"
	if r.nativeCore {
		name = "SP1NativeCore"
	}
	t.Run(name, func(t devtest.T) {
		checkpoint := data.zkCheckpoint
		validateZKCheckpoint(t, sys, chains, checkpoint)
		cfg := sys.L2ChainA.Escape().L2Challengers()[0].Config().CannonKona
		args := superRangeExecutorArgs(t, sys, chains, cfg, checkpoint.trustedL1Head, checkpoint.endTimestamp)
		if r.nativeCore {
			args = append(args, "--native-core")
		}
		t.Require().True(
			rustbin.RunKonaSP1SuperRange(t, t.Logger(), executorPath, t.TempDir(), args...),
			"expected kona-sp1 super-range executor to accept timestamp %d",
			checkpoint.endTimestamp,
		)
	})
}

func newZKCheckpoint(t devtest.T, sys *presets.SingleChainInterop, endTimestamp uint64, expectReplacement bool) *zkCheckpoint {
	resp := sys.SuperRoots.SuperRootAtTimestamp(endTimestamp)
	t.Require().NotNil(resp.Data, "expected verified super-root data at timestamp %d", endTimestamp)
	return &zkCheckpoint{
		endTimestamp:      endTimestamp,
		trustedL1Head:     resp.Data.VerifiedRequiredL1,
		expectReplacement: expectReplacement,
	}
}

func newZKCheckpointForRunners(
	t devtest.T,
	sys *presets.SingleChainInterop,
	endTimestamp uint64,
	expectReplacement bool,
	runners []ProofRunner,
) *zkCheckpoint {
	if !hasSP1Runner(runners) {
		return nil
	}
	return newZKCheckpoint(t, sys, endTimestamp, expectReplacement)
}

func validateZKCheckpoint(t devtest.T, sys *presets.SingleChainInterop, chains []*chain, checkpoint *zkCheckpoint) {
	resp := sys.SuperRoots.SuperRootAtTimestamp(checkpoint.endTimestamp)
	t.Require().NotNil(resp.Data, "expected verified super-root data at timestamp %d", checkpoint.endTimestamp)
	t.Require().Equal(checkpoint.trustedL1Head, resp.Data.VerifiedRequiredL1,
		"verified required L1 changed for timestamp %d", checkpoint.endTimestamp)
	expectedChainIDs := make([]eth.ChainID, len(chains))
	for i, chain := range chains {
		expectedChainIDs[i] = chain.ID
	}
	t.Require().NotEmpty(expectedChainIDs, "dependency set must contain at least one chain")
	t.Require().Equal(expectedChainIDs, resp.ChainIDs,
		"checkpoint dependency-set chains must match the proof system")
	t.Require().Len(resp.OptimisticAtTimestamp, len(expectedChainIDs),
		"every dependency-set chain must have an optimistic output")

	verified, ok := resp.Data.Super.(*eth.SuperV1)
	t.Require().True(ok, "verified super-root data must be SuperV1")
	if !ok {
		return
	}
	t.Require().Equal(checkpoint.endTimestamp, verified.Timestamp)
	t.Require().Len(verified.Chains, len(expectedChainIDs),
		"verified super root must contain every dependency-set chain")
	for i, output := range verified.Chains {
		t.Require().Equal(expectedChainIDs[i], output.ChainID,
			"verified super-root chains must match the proof system")
	}
	trustedL1 := sys.L1EL.BlockRefByNumber(checkpoint.trustedL1Head.Number)
	t.Require().Equal(checkpoint.trustedL1Head.Hash, trustedL1.Hash,
		"trusted L1 head must be canonical at block %d", checkpoint.trustedL1Head.Number)
	canonicalL1Hashes := map[uint64]common.Hash{
		trustedL1.Number: trustedL1.Hash,
	}

	replacements := 0
	for i, chainID := range expectedChainIDs {
		optimistic, exists := resp.OptimisticAtTimestamp[chainID]
		t.Require().Truef(exists, "missing optimistic output for chain %s", chainID)
		if !exists {
			continue
		}
		t.Require().NotNilf(optimistic.Output, "missing optimistic output preimage for chain %s", chainID)
		t.Require().LessOrEqualf(optimistic.RequiredL1.Number, checkpoint.trustedL1Head.Number,
			"optimistic output for chain %s requires L1 block %d after trusted head %d",
			chainID, optimistic.RequiredL1.Number, checkpoint.trustedL1Head.Number)
		canonicalHash, exists := canonicalL1Hashes[optimistic.RequiredL1.Number]
		if !exists {
			canonical := sys.L1EL.BlockRefByNumber(optimistic.RequiredL1.Number)
			canonicalHash = canonical.Hash
			canonicalL1Hashes[canonical.Number] = canonical.Hash
		}
		t.Require().Equalf(optimistic.RequiredL1.Hash, canonicalHash,
			"optimistic output for chain %s is not supported by the trusted L1 chain", chainID)
		if optimistic.OutputRoot != verified.Chains[i].Output {
			replacements++
		}
	}
	if checkpoint.expectReplacement {
		t.Require().Positive(replacements, "expected at least one optimistic root replacement")
	} else {
		t.Require().Zero(replacements, "canonical checkpoint must have matching optimistic and verified roots")
	}
}

func superRangeExecutorArgs(
	t devtest.T,
	sys *presets.SingleChainInterop,
	chains []*chain,
	cfg vm.Config,
	l1Head eth.BlockID,
	endTimestamp uint64,
) []string {
	t.Require().NotEmpty(chains, "SP1 super-range requires at least one chain")
	t.Require().Len(cfg.RollupConfigPaths, len(chains), "SP1 super-range requires one rollup config per chain")
	t.Require().NotEmpty(cfg.DepsetConfigPath, "SP1 super-range requires a dependency-set config")
	l2NodeAddresses := make([]string, len(chains))
	for i, chain := range chains {
		l2NodeAddresses[i] = chain.EL.Escape().UserRPC()
	}
	args := []string{
		"--supernode-address", sys.SuperRoots.UserRPC(),
		"--l1-node-address", sys.L1EL.Escape().UserRPC(),
		"--l1-beacon-address", sys.L1CL.BeaconHTTPAddr(),
		"--l2-node-addresses", strings.Join(l2NodeAddresses, ","),
		"--l1-head", l1Head.Hash.Hex(),
		"--end-timestamp", strconv.FormatUint(endTimestamp, 10),
		"--rollup-config-paths", strings.Join(cfg.RollupConfigPaths, ","),
		"--depset-cfg", cfg.DepsetConfigPath,
	}
	if cfg.L1GenesisPath != "" {
		args = append(args, "--l1-config-path", cfg.L1GenesisPath)
	}
	return args
}
