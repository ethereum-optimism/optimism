package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const pcdGenesisTimeOffset = 600

type pcdChainArtifacts struct {
	chainID     common.Hash
	genesisPath string
	rollupPath  string
}

type pcdJourneyFixture struct {
	t                     *testing.T
	runner                *CLITestRunner
	workdir               string
	cacheDir              string
	l1RPC                 string
	l1Client              *ethclient.Client
	privateKey            string
	deployer              common.Address
	devKeys               *devkeys.MnemonicDevKeys
	chainIDs              []common.Hash
	superchainOutput      opcm.DeploySuperchainOutput
	implementationsOutput opcm.DeployImplementationsOutput
	postBootstrapL1State  pcdL1State
	chainArtifacts        []pcdChainArtifacts
	dependencySetPath     string
	prestate              common.Hash
}

func newPCDJourneyFixture(t *testing.T, chainIDs []common.Hash) *pcdJourneyFixture {
	t.Helper()
	lgr := testlog.Logger(t, log.LevelError)
	l1RPC, l1Client := devnet.DefaultAnvilRPC(t, lgr)
	t.Cleanup(l1Client.Close)
	privateKey, key, devKeys := shared.DefaultPrivkey(t)
	runner := NewCLITestRunner(t, WithL1RPC(l1RPC), WithPrivateKey(privateKey))

	return &pcdJourneyFixture{
		t:          t,
		runner:     runner,
		workdir:    runner.GetWorkDir(),
		cacheDir:   filepath.Join(runner.GetWorkDir(), ".cache"),
		l1RPC:      l1RPC,
		l1Client:   l1Client,
		privateKey: privateKey,
		deployer:   crypto.PubkeyToAddress(key.PublicKey),
		devKeys:    devKeys,
		chainIDs:   append([]common.Hash(nil), chainIDs...),
	}
}

// This setup uses the CLI sequence from TestCLIBootstrap in bootstrap_test.go.
func (f *pcdJourneyFixture) bootstrapOPCM() {
	f.t.Helper()
	f.t.Log("PCD stage: bootstrap OPCM")
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	superchainProxyAdminOwner := shared.AddrFor(f.t, f.devKeys, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID))
	guardian := shared.AddrFor(f.t, f.devKeys, devkeys.SuperchainConfigGuardianKey.Key(l1ChainID))
	challenger := shared.AddrFor(f.t, f.devKeys, devkeys.ChallengerRole.Key(l1ChainID))
	l1ProxyAdminOwner := shared.AddrFor(f.t, f.devKeys, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID))

	superchainOutputFile := filepath.Join(f.workdir, "bootstrap-superchain.json")
	f.runner.ExpectSuccessWithNetwork(f.t, []string{
		"bootstrap", "superchain",
		"--outfile", superchainOutputFile,
		"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
		"--guardian", guardian.Hex(),
	}, nil)
	f.readJSON(superchainOutputFile, &f.superchainOutput)
	require.NoError(f.t, addresses.CheckNoZeroAddresses(f.superchainOutput))

	implementationsOutputFile := filepath.Join(f.workdir, "bootstrap-implementations.json")
	f.runner.ExpectSuccessWithNetwork(f.t, []string{
		"bootstrap", "implementations",
		"--outfile", implementationsOutputFile,
		"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
		"--superchain-config-proxy", f.superchainOutput.SuperchainConfigProxy.Hex(),
		"--l1-proxy-admin-owner", l1ProxyAdminOwner.Hex(),
		"--superchain-proxy-admin", f.superchainOutput.SuperchainProxyAdmin.Hex(),
		"--challenger", challenger.Hex(),
	}, nil)
	f.readJSON(implementationsOutputFile, &f.implementationsOutput)
	require.NotEqual(f.t, common.Address{}, f.implementationsOutput.OpcmV2)

	probe := pcdL1Probe{client: f.l1Client, deployer: f.deployer}
	f.postBootstrapL1State = probe.read(f.t, nil)
}

func (f *pcdJourneyFixture) runInit(gameType embedded.GameType) (*state.Intent, *state.State) {
	f.t.Helper()
	f.t.Log("PCD stage: init")
	intent, st := cliInitIntent(f.t, f.runner, devnet.DefaultChainID, f.chainIDs)
	intent.OPCMAddress = &f.implementationsOutput.OpcmV2
	intent.SuperchainConfigProxy = &f.superchainOutput.SuperchainConfigProxy
	intent.SuperchainRoles = nil
	l1ChainID := new(big.Int).SetUint64(devnet.DefaultChainID)
	for i, chain := range intent.Chains {
		chainID := new(uint256.Int).SetBytes32(chain.ID[:])
		intent.Chains[i] = shared.NewChainIntent(f.t, f.devKeys, l1ChainID, chainID, standard.GasLimit)
		intent.Chains[i].DeployOverrides = map[string]any{"respectedGameType": gameType}
	}
	require.NoError(f.t, intent.WriteToFile(filepath.Join(f.workdir, "intent.toml")))
	return intent, st
}

func (f *pcdJourneyFixture) runPrepare() *state.State {
	f.t.Helper()
	f.t.Log("PCD stage: prepare")
	f.runner.ExpectSuccessWithNetwork(f.t, []string{
		"prepare",
		"--workdir", f.workdir,
		"--genesis-time-offset", strconv.FormatUint(pcdGenesisTimeOffset, 10),
	}, nil)
	st, err := pipeline.ReadState(f.workdir)
	require.NoError(f.t, err)
	return st
}

func (f *pcdJourneyFixture) runInspect() []pcdChainArtifacts {
	f.t.Helper()
	f.t.Log("PCD stage: inspect")
	statePath := filepath.Join(f.workdir, "state.json")
	stateBefore, err := os.ReadFile(statePath)
	require.NoError(f.t, err)
	stateHashBefore := sha256.Sum256(stateBefore)

	f.chainArtifacts = make([]pcdChainArtifacts, 0, len(f.chainIDs))
	for _, chainID := range f.chainIDs {
		artifactDir := filepath.Join(f.workdir, "artifacts", chainID.Hex())
		require.NoError(f.t, os.MkdirAll(artifactDir, 0o755))
		artifacts := pcdChainArtifacts{
			chainID:     chainID,
			genesisPath: filepath.Join(artifactDir, "genesis.json"),
			rollupPath:  filepath.Join(artifactDir, "rollup.json"),
		}

		f.runner.ExpectSuccess(f.t, []string{
			"inspect", "genesis",
			"--workdir", f.workdir,
			"--outfile", artifacts.genesisPath,
			chainID.Hex(),
		}, nil)
		require.FileExists(f.t, artifacts.genesisPath)
		var genesis core.Genesis
		f.readJSON(artifacts.genesisPath, &genesis)

		f.runner.ExpectSuccess(f.t, []string{
			"inspect", "rollup",
			"--workdir", f.workdir,
			"--outfile", artifacts.rollupPath,
			chainID.Hex(),
		}, nil)
		require.FileExists(f.t, artifacts.rollupPath)
		var rollupConfig rollup.Config
		f.readJSON(artifacts.rollupPath, &rollupConfig)

		f.chainArtifacts = append(f.chainArtifacts, artifacts)
	}

	stateAfter, err := os.ReadFile(statePath)
	require.NoError(f.t, err)
	require.Equal(f.t, stateHashBefore, sha256.Sum256(stateAfter), "inspect changed state.json")
	return append([]pcdChainArtifacts(nil), f.chainArtifacts...)
}

func (f *pcdJourneyFixture) writeDependencySet() string {
	f.t.Helper()
	f.t.Log("PCD stage: write dependency set")
	st, err := pipeline.ReadState(f.workdir)
	require.NoError(f.t, err)
	require.NotNil(f.t, st.InteropDepSet)

	data, err := json.Marshal(st.InteropDepSet)
	require.NoError(f.t, err)
	f.dependencySetPath = filepath.Join(f.workdir, "interop-depset.json")
	require.NoError(f.t, os.WriteFile(f.dependencySetPath, data, 0o644))
	require.FileExists(f.t, f.dependencySetPath)

	written, err := os.ReadFile(f.dependencySetPath)
	require.NoError(f.t, err)
	parsed, err := depset.ParseJSONDependencySet(bytes.NewReader(written))
	require.NoError(f.t, err)
	require.Len(f.t, parsed.Chains(), len(f.chainIDs))
	for _, chainID := range f.chainIDs {
		require.Truef(f.t, parsed.HasChain(eth.ChainIDFromBytes32(chainID)), "dependency set is missing chain %s", chainID.Hex())
	}
	return f.dependencySetPath
}

func (f *pcdJourneyFixture) runPrestate(prestate common.Hash) *state.State {
	f.t.Helper()
	f.t.Log("PCD stage: prestate")
	f.runner.ExpectSuccess(f.t, []string{
		"prestate",
		"--workdir", f.workdir,
		"--dispute-absolute-prestate", prestate.Hex(),
	}, nil)

	st, err := pipeline.ReadState(f.workdir)
	require.NoError(f.t, err)
	for _, chainID := range f.chainIDs {
		chain, err := st.Chain(chainID)
		require.NoError(f.t, err)
		require.Equalf(
			f.t,
			prestate,
			chain.Prestate,
			"committed prestate differs for chain %s: expected %s, observed %s",
			chainID.Hex(),
			prestate,
			chain.Prestate,
		)
	}
	f.prestate = prestate
	return st
}

func (f *pcdJourneyFixture) runContinue() *state.State {
	f.t.Helper()
	f.t.Log("PCD stage: continue")
	prepared, err := pipeline.ReadState(f.workdir)
	require.NoError(f.t, err)
	require.NotNil(f.t, prepared.PreparedDeployment)
	preparedSnapshot, err := prepared.PreparedDeployment.Clone()
	require.NoError(f.t, err)

	f.runner.ExpectSuccessWithNetwork(f.t, []string{
		"continue",
		"--workdir", f.workdir,
	}, nil)

	continued, err := pipeline.ReadState(f.workdir)
	require.NoError(f.t, err)
	require.Nil(f.t, continued.AppliedIntent)
	require.Equal(f.t, preparedSnapshot, continued.PreparedDeployment)
	for _, chainID := range f.chainIDs {
		require.Truef(f.t, continued.IsChainDeployed(chainID), "chain %s was not deployed", chainID.Hex())
		chain, err := continued.Chain(chainID)
		require.NoError(f.t, err)
		require.NotNilf(f.t, chain.Continuation, "chain %s has no continuation checkpoint", chainID.Hex())
	}
	return continued
}

func (f *pcdJourneyFixture) readJSON(path string, out any) {
	f.t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(f.t, err)
	require.NoError(f.t, json.Unmarshal(data, out))
}
