package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type PrepareConfig struct {
	Workdir string
	Logger  log.Logger
	// PrivateKey of the deployer account. Its address is used as the dry-run
	// sender, operator MUST use the same key for the eventual broadcast.
	PrivateKey string
	// L1RPCUrl is the L1 endpoint that prepare forks to dry-run OPCM.deploy.
	L1RPCUrl string
	// CacheDir is where downloaded artifacts are cached.
	CacheDir string
	// GenesisTimeOffset is the number of seconds added to the L1 anchor block's timestamp
	// to produce the committed L2 genesis timestamp.
	GenesisTimeOffset uint64

	privateKeyECDSA *ecdsa.PrivateKey
}

func (c *PrepareConfig) Check() error {
	if c.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if c.PrivateKey == "" {
		return fmt.Errorf("private key must be specified")
	}
	privECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	c.privateKeyECDSA = privECDSA

	if c.L1RPCUrl == "" {
		return fmt.Errorf("l1 RPC URL must be specified")
	}

	return nil
}

func PrepareCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

		return Prepare(ctx, newPrepareConfig(cliCtx, l))
	}
}

// newPrepareConfig maps the CLI flags onto a PrepareConfig. The private key is
// parsed and validated later in Check, mirroring apply.
func newPrepareConfig(cliCtx *cli.Context, l log.Logger) PrepareConfig {
	return PrepareConfig{
		Workdir:           cliCtx.String(WorkdirFlagName),
		Logger:            l,
		PrivateKey:        cliCtx.String(PrivateKeyFlagName),
		L1RPCUrl:          cliCtx.String(L1RPCURLFlagName),
		CacheDir:          cliCtx.String(CacheDirFlagName),
		GenesisTimeOffset: cliCtx.Uint64(GenesisTimeOffsetFlagName),
	}
}

func Prepare(ctx context.Context, cfg PrepareConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for prepare: %w", err)
	}

	// prepare predicts the L1 addresses by dry-running OPCM.deploy against a fork
	// of the live L1.
	intent, err := pipeline.ReadIntent(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	st, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return err
	}

	// A state produced by the apply pipeline MUST NOT be prepared.
	if err := st.CheckNotApplied(); err != nil {
		return err
	}

	if err := checkReservedOverrides(intent, st); err != nil {
		return err
	}

	if len(intent.Chains) == 0 {
		return fmt.Errorf("intent has no chains to prepare")
	}

	// prepare predicts against an already deployed OPCM, so the intent must pin it.
	if intent.OPCMAddress == nil {
		return fmt.Errorf("intent.opcmAddress must be set to predict against an existing OPCM")
	}

	// A rerun must use the same deployer and OPCM or the predicted addresses may change.
	deployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)
	opcmAddr := *intent.OPCMAddress
	if err := st.CheckL1PredictInputs(deployer, opcmAddr); err != nil {
		return err
	}

	if err := st.EnsureCreate2Salt(); err != nil {
		return err
	}

	// Download both artifact bundles so prepare can commit their contents to the snapshot.
	// Only the L1 artifacts are used by the address-prediction script.
	bundle, err := artifacts.DownloadBundle(
		ctx,
		intent.L1ContractsLocator,
		intent.L2ContractsLocator,
		ioutil.BarProgressor(),
		cfg.CacheDir,
	)
	if err != nil {
		return err
	}

	l1RPC, err := rpc.Dial(cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1RPC.Close()

	if err := validateL1ChainID(ctx, l1RPC, intent); err != nil {
		return err
	}

	if err := resolveSuperchainConfigProxy(ctx, l1RPC, intent, opcmAddr); err != nil {
		return err
	}

	l1Client := ethclient.NewClient(l1RPC)
	if err := checkOPCMHasCode(ctx, l1Client, opcmAddr); err != nil {
		return err
	}

	// Reject early a game type the pinned OPCM cannot deploy.
	superRoot, err := opcm.ReadSuperRootEnabled(ctx, l1Client, opcmAddr)
	if err != nil {
		return fmt.Errorf("failed to read the OPCM game mode at %s: %w", opcmAddr, err)
	}
	if err := validateInitialGameTypes(intent, st, superRoot, opcmAddr); err != nil {
		return err
	}

	// Record the implementations the pinned OPCM installs.
	impls, err := opcm.ReadImplementations(ctx, l1Client, opcmAddr)
	if err != nil {
		return fmt.Errorf("failed to read implementations from OPCM at %s: %w", opcmAddr, err)
	}
	st.ImplementationsDeployment = impls

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		cfg.Logger,
		deployer,
		bundle.L1,
		l1RPC,
	)
	if err != nil {
		return fmt.Errorf("failed to create forked L1 script host: %w", err)
	}

	superDeployment, superRoles, err := pipeline.PopulateSuperchainState(
		&pipeline.Env{Logger: cfg.Logger, L1ScriptHost: l1Host},
		opcmAddr,
		*intent.SuperchainConfigProxy,
	)
	if err != nil {
		return fmt.Errorf("failed to read superchain deployment: %w", err)
	}
	st.SuperchainDeployment = superDeployment
	st.SuperchainRoles = superRoles

	deployScript, err := opcm.NewDeployOPChainScript(l1Host)
	if err != nil {
		return fmt.Errorf("failed to load DeployOPChain script: %w", err)
	}

	// Fetch safe block once, so all unpinned chains without an override share one anchor.
	// This is also the reference block for the overrides.
	safe, err := pipeline.FetchL1BlockRefByNumber(ctx, l1RPC, "safe")
	if err != nil {
		return fmt.Errorf("failed to fetch L1 safe block: %w", err)
	}

	selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
		return selectAnchorBlock(ctx, l1RPC, safe, overrideHash)
	}

	if err := prepareChains(cfg.Logger, intent, st, deployScript.Run, selectAnchor, safe, cfg.GenesisTimeOffset); err != nil {
		return err
	}
	st.PreparedDeployment, err = pipeline.NewPreparedDeployment(intent, st, deployer, opcmAddr, bundle)
	if err != nil {
		return fmt.Errorf("failed to freeze prepared deployment: %w", err)
	}

	// Build L2 genesis from the addresses and genesis time just committed.
	genesisEnv := &pipeline.Env{Logger: cfg.Logger, Deployer: deployer}
	if err := generateGenesisForChains(genesisEnv, intent, bundle, st); err != nil {
		return err
	}

	// Compute the real genesis output root from that same genesis.
	if err := computeGenesisOutputRootsForChains(genesisEnv, intent, st); err != nil {
		return err
	}

	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// Validate the dependency set before mutating chain state.
func prepareChains(
	lgr log.Logger,
	intent *state.Intent,
	st *state.State,
	run func(opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error),
	selectAnchor func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error),
	safe *state.L1BlockRefJSON,
	genesisTimeOffset uint64,
) error {
	if err := pipeline.ValidateNoDuplicateChainIDs(intent.Chains); err != nil {
		return err
	}

	initialGameTypes := make([]uint32, 0, len(intent.Chains))
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		proofParams, err := pipeline.ResolveChainProofParams(intent, chain)
		if err != nil {
			return fmt.Errorf("failed to resolve initial dispute game type for chain %s: %w", chain.ID.Hex(), err)
		}
		initialGameTypes = append(initialGameTypes, proofParams.DisputeGameType)
	}
	if err := pipeline.ValidateInitialGameTypeSet(initialGameTypes); err != nil {
		return err
	}

	interopDepSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	if err != nil {
		return fmt.Errorf("failed to create interop dependency set: %w", err)
	}
	st.InteropDepSet = interopDepSet

	return predictChains(lgr, intent, st, run, selectAnchor, safe, genesisTimeOffset)
}

// validateL1ChainID checks that the L1 RPC endpoint serves the chain the intent was
// written for, so a prediction cannot silently run against a fork of the wrong L1.
func validateL1ChainID(ctx context.Context, l1RPC *rpc.Client, intent *state.Intent) error {
	l1ChainID, err := ethclient.NewClient(l1RPC).ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get L1 chain ID: %w", err)
	}
	if l1ChainID.Cmp(intent.L1ChainIDBig()) != 0 {
		return fmt.Errorf("l1 chain ID mismatch: got %d, expected %d", l1ChainID, intent.L1ChainID)
	}
	return nil
}

// checkOPCMHasCode rejects an unusable OPCM address.
func checkOPCMHasCode(ctx context.Context, l1Client *ethclient.Client, opcmAddr common.Address) error {
	code, err := l1Client.CodeAt(ctx, opcmAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to read code at OPCM address %s: %w", opcmAddr, err)
	}
	if len(code) == 0 {
		return fmt.Errorf("no contract code at intent.opcmAddress %s", opcmAddr)
	}
	return nil
}

// validateInitialGameTypes rejects any chain whose resolved game type belongs to the family the
// pinned OPCM does not install. Deployed chains are skipped, their games being fixed on L1.
func validateInitialGameTypes(intent *state.Intent, st *state.State, superRoot bool, opcmAddr common.Address) error {
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		proofParams, err := pipeline.ResolveChainProofParams(intent, chain)
		if err != nil {
			return fmt.Errorf("failed to resolve initial dispute game type for chain %s: %w", chain.ID.Hex(), err)
		}
		if err := pipeline.ValidateInitialGameTypeForOPCM(proofParams.DisputeGameType, superRoot, opcmAddr); err != nil {
			return fmt.Errorf("chain %s: %w", chain.ID.Hex(), err)
		}
	}
	return nil
}

// resolveSuperchainConfigProxy fills the intent's superchainConfigProxy from the pinned
// OPCM when it is unset.
func resolveSuperchainConfigProxy(ctx context.Context, l1RPC *rpc.Client, intent *state.Intent, opcmAddr common.Address) error {
	if intent.SuperchainConfigProxy != nil {
		return nil
	}
	return fmt.Errorf("intent.superchainConfigProxy must be set to predict against an existing OPCM at %s", opcmAddr.Hex())
}

// predictChains predicts and records contract L1 addresses for undeployed chains.
// It pins each chain's anchor and derived genesis time before prediction.
// Reruns revalidate and reuse that pair instead of recomputing it.
func predictChains(
	lgr log.Logger,
	intent *state.Intent,
	st *state.State,
	run func(opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error),
	selectAnchor func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error),
	safe *state.L1BlockRefJSON,
	genesisTimeOffset uint64,
) error {
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			lgr.Info("skipping already deployed chain", "chain", chain.ID.Hex())
			continue
		}

		var genesisTime hexutil.Uint64
		if pinned := pinnedAnchorState(st, chain.ID); pinned != nil {
			// A prior run already committed this chain's anchor and genesis time. Check the intent's override
			// and the block commitment are the same.
			if chain.L1StartBlockHash != nil && *chain.L1StartBlockHash != pinned.StartBlock.Hash {
				return fmt.Errorf(
					"chain %s: the l1StartBlockHash override (%s) conflicts with the anchor block pinned by a previous run (%s), clear the chain's state to re-pin",
					chain.ID.Hex(), chain.L1StartBlockHash.Hex(), pinned.StartBlock.Hash.Hex(),
				)
			}

			// Validate the pinned anchor block is still valid.
			if _, err := selectAnchor(&pinned.StartBlock.Hash); err != nil {
				return fmt.Errorf("pinned anchor block for chain %s is no longer valid: %w", chain.ID.Hex(), err)
			}
			genesisTime = *pinned.GenesisTime
			lgr.Info(
				"reusing pinned anchor block and genesis time",
				"chain", chain.ID.Hex(),
				"number", uint64(pinned.StartBlock.Number),
				"hash", pinned.StartBlock.Hash,
				"genesisTime", uint64(genesisTime),
			)
		} else {
			// Resolve the reorg-safe anchor block before the dry-run.
			anchorBlock, err := selectAnchor(chain.L1StartBlockHash)
			if err != nil {
				return fmt.Errorf("failed to select anchor block for chain %s: %w", chain.ID.Hex(), err)
			}

			// TODO(#20916): A reasonable minimum will be enforced in the future, once the L2 deployment is benchmarked.
			// Commit the anchor and the genesis time.
			genesisTime = hexutil.Uint64(uint64(anchorBlock.Time) + genesisTimeOffset)
			st.PinChainAnchor(chain.ID, anchorBlock, genesisTime)
			lgr.Info(
				"pinned anchor block and genesis time",
				"chain", chain.ID.Hex(),
				"number", uint64(anchorBlock.Number),
				"hash", anchorBlock.Hash,
				"genesisTime", uint64(genesisTime),
			)
		}

		// The deployment must land after the current safe head, so a genesis time at or
		// below its timestamp can no longer be met.
		if uint64(genesisTime) <= uint64(safe.Time) {
			return fmt.Errorf(
				"chain %s: the committed genesis time (%d) is not after the current L1 safe head timestamp (%d), the deployment window has elapsed; "+
					"use a newer anchor block or a larger --%s (for a pin from a previous run, clear the chain's state to re-pin)",
				chain.ID.Hex(), uint64(genesisTime), uint64(safe.Time), GenesisTimeOffsetFlagName,
			)
		}

		dci, err := makePredictionInput(intent, st, chain)
		if err != nil {
			return fmt.Errorf("failed to build prediction input for chain %s: %w", chain.ID.Hex(), err)
		}
		requirements, err := pipeline.ResolveInitialDeployRequirements(dci.DisputeGameType)
		if err != nil {
			return fmt.Errorf("failed to resolve initial deploy requirements for chain %s: %w", chain.ID.Hex(), err)
		}

		out, err := run(dci)
		if err != nil {
			return fmt.Errorf("failed to predict L1 addresses for chain %s: %w", chain.ID.Hex(), err)
		}

		// Record the predicted addresses, marked not deployed yet.
		st.SetChainContracts(chain.ID, pipeline.OpChainContractsFromDeployOutput(out), false)
		chainState, err := st.Chain(chain.ID)
		if err != nil {
			return fmt.Errorf("failed to clear prepared inputs for chain %s: %w", chain.ID.Hex(), err)
		}

		chainState.ClearDerivedArtifacts()
		gameType := dci.DisputeGameType
		chainState.InitialGameType = &gameType

		if requirements.RequiresPrestate {
			lgr.Info(
				"selected prestate must be committed; run op-deployer prestate before continue",
				"chain", chain.ID.Hex(),
			)
		}

		lgr.Info(
			"predicted L1 addresses",
			"chain", chain.ID.Hex(),
			"systemConfigProxy", out.SystemConfigProxy,
			"optimismPortalProxy", out.OptimismPortalProxy,
			"l1StandardBridgeProxy", out.L1StandardBridgeProxy,
			"l1CrossDomainMessengerProxy", out.L1CrossDomainMessengerProxy,
			"disputeGameFactoryProxy", out.DisputeGameFactoryProxy,
			"anchorStateRegistryProxy", out.AnchorStateRegistryProxy,
		)
	}

	return nil
}

// generateGenesisForChains builds each chain's L2 genesis allocs from the addresses and genesis
// time predictChains just committed.
func generateGenesisForChains(pEnv *pipeline.Env, intent *state.Intent, bundle artifacts.Bundle, st *state.State) error {
	for _, chain := range intent.Chains {
		if err := pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID); err != nil {
			return fmt.Errorf("failed to generate L2 genesis for chain %s: %w", chain.ID.Hex(), err)
		}
	}
	return nil
}

// computeGenesisOutputRootsForChains computes and persists every chain's genesis block hash and
// starting anchor proposal from the L2 genesis generateGenesisForChains just built.
func computeGenesisOutputRootsForChains(pEnv *pipeline.Env, intent *state.Intent, st *state.State) error {
	if err := pipeline.ComputeGenesisOutputRoots(pEnv, intent, st); err != nil {
		return fmt.Errorf("failed to compute genesis output roots: %w", err)
	}
	return nil
}

// checkReservedOverrides rejects deploy overrides for values that prepare commits
// into state.
func checkReservedOverrides(intent *state.Intent, st *state.State) error {
	if key, ok := state.FindPinnedOverrideKey(intent.GlobalDeployOverrides); ok {
		return fmt.Errorf(
			"globalDeployOverrides key %q is reserved by the prepare flow: set the anchor block via the chain's l1StartBlockHash and the genesis time via --%s",
			key, GenesisTimeOffsetFlagName,
		)
	}
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		if key, ok := state.FindPinnedOverrideKey(chain.DeployOverrides); ok {
			return fmt.Errorf(
				"chain %s: deployOverrides key %q is reserved by the prepare flow: set the anchor block via l1StartBlockHash and the genesis time via --%s",
				chain.ID.Hex(), key, GenesisTimeOffsetFlagName,
			)
		}
	}
	return nil
}

// pinnedAnchorState returns the chain's state when a prior prepare run committed both
// its anchor block and genesis time, and nil otherwise.
func pinnedAnchorState(st *state.State, id common.Hash) *state.ChainState {
	chainState, err := st.Chain(id)
	if err != nil || chainState.StartBlock == nil || chainState.GenesisTime == nil {
		return nil
	}
	return chainState
}

// Sentinel input for the prediction dry-run of permissionless deploys.
var predictionStartingAnchorRoot = common.Hash{0x01}

// makePredictionInput builds the DeployOPChain input for the prediction dry-run.
// The OPCM, superchain config and salt mixer are taken from the committed intent
// and state so the prediction matches the eventual broadcast. Role addresses are set to
// placeholders since they are not relevant for the prediction.
func makePredictionInput(intent *state.Intent, st *state.State, chain *state.ChainIntent) (opcm.DeployOPChainInput, error) {
	if intent.OPCMAddress == nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("intent.opcmAddress must be set to predict against an existing OPCM")
	}
	if intent.SuperchainConfigProxy == nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("intent.superchainConfigProxy must be set")
	}

	proofParams, err := pipeline.ResolveChainProofParams(intent, chain)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("failed to resolve dispute params: %w", err)
	}
	requirements, err := pipeline.ResolveInitialDeployRequirements(proofParams.DisputeGameType)
	if err != nil {
		return opcm.DeployOPChainInput{}, err
	}

	// Prediction runs against an already existing OPCM
	placeholderRoles := state.ChainRoles{
		L1ProxyAdminOwner: standard.PlaceholderAddress,
		SystemConfigOwner: standard.PlaceholderAddress,
		Batcher:           standard.PlaceholderAddress,
		UnsafeBlockSigner: standard.PlaceholderAddress,
		Proposer:          standard.PlaceholderAddress,
		Challenger:        standard.PlaceholderAddress,
	}

	// Permissioned deploys use the placeholder anchor broadcast by apply. Permissionless
	// deploys use a sentinel because their real anchor depends on the addresses predicted
	// here, and the placeholder is rejected for them.
	startingAnchorRoot := opcm.DefaultStartingAnchorProposal()

	if requirements.Permissionless {
		startingAnchorRoot.Root = predictionStartingAnchorRoot
	}

	return pipeline.BuildDeployOPChainInput(
		proofParams,
		placeholderRoles,
		*intent.OPCMAddress,
		*intent.SuperchainConfigProxy,
		chain.ID,
		st.Create2Salt.String(),
		chain.GasLimit,
		startingAnchorRoot,
		chain,
	), nil
}
