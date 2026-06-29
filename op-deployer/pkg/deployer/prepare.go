package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

// placeholderRole is a non-zero sentinel used for the role addresses in the
// prediction dry-run.
var placeholderRole = common.Address{0x01}

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
	// Prestate is the absolute prestate hash to write to state. When set it takes
	// precedence over any faultGameAbsolutePrestate intent override. It may be empty,
	// in which case the prestate is resolved from the intent (if present at all).
	Prestate string

	privateKeyECDSA *ecdsa.PrivateKey
	prestate        common.Hash
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

	if c.Prestate != "" {
		prestate := common.HexToHash(c.Prestate)
		if prestate == (common.Hash{}) {
			return fmt.Errorf("prestate must be a non-zero hash, got %q", c.Prestate)
		}
		c.prestate = prestate
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
		Workdir:    cliCtx.String(WorkdirFlagName),
		Logger:     l,
		PrivateKey: cliCtx.String(PrivateKeyFlagName),
		L1RPCUrl:   cliCtx.String(L1RPCURLFlagName),
		CacheDir:   cliCtx.String(CacheDirFlagName),
		Prestate:   cliCtx.String(PrestateFlagName),
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

	if len(intent.Chains) == 0 {
		return fmt.Errorf("intent has no chains to prepare")
	}

	// Download the L1 artifacts referenced by the intent so the dry-run uses the
	// same DeployOPChain script as the eventual broadcast.
	l1ArtifactsFS, err := artifacts.Download(ctx, intent.L1ContractsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to download L1 artifacts: %w", err)
	}

	// The sender of the deploy transaction.
	deployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)

	l1RPC, err := rpc.Dial(cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer l1RPC.Close()

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		broadcaster.NoopBroadcaster(),
		cfg.Logger,
		deployer,
		l1ArtifactsFS,
		l1RPC,
	)
	if err != nil {
		return fmt.Errorf("failed to create forked L1 script host: %w", err)
	}

	deployScript, err := opcm.NewDeployOPChainScript(l1Host)
	if err != nil {
		return fmt.Errorf("failed to load DeployOPChain script: %w", err)
	}

	interopDepSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	if err != nil {
		return fmt.Errorf("failed to build interop dependency set: %w", err)
	}
	st.InteropDepSet = interopDepSet

	for _, chain := range intent.Chains {
		// Resolve the absolute prestate and enforce the permissionless gate before
		// predicting, so a misconfigured chain fails fast. A permissionless game type
		// requires a resolved prestate; a permissioned-only chain may leave it unset.
		prestate, err := resolvePrestate(cfg.prestate, intent, chain)
		if err != nil {
			return fmt.Errorf("failed to resolve prestate for chain %s: %w", chain.ID.Hex(), err)
		}

		permissionless, err := isPermissionlessDeployment(intent, chain)
		if err != nil {
			return fmt.Errorf("failed to resolve game type for chain %s: %w", chain.ID.Hex(), err)
		}
		if permissionless && prestate == (common.Hash{}) {
			return fmt.Errorf(
				"chain %s enables a permissionless game type but no prestate was resolved; pass --%s or set the %s intent override",
				chain.ID.Hex(), PrestateFlagName, faultGameAbsolutePrestateKey,
			)
		}

		dci, err := makePredictionInput(intent, st, chain)
		if err != nil {
			return fmt.Errorf("failed to build prediction input for chain %s: %w", chain.ID.Hex(), err)
		}

		out, err := deployScript.Run(dci)
		if err != nil {
			return fmt.Errorf("failed to predict L1 addresses for chain %s: %w", chain.ID.Hex(), err)
		}

		// Record the predicted addresses into the chain state marked as not deployed yet.
		st.SetChainContracts(chain.ID, pipeline.OpChainContractsFromDeployOutput(out), false)

		if prestate != (common.Hash{}) {
			st.SetChainPrestate(chain.ID, prestate)
			cfg.Logger.Info("resolved prestate", "chain", chain.ID.Hex(), "prestate", prestate)
		}

		cfg.Logger.Info(
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

	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// faultGameAbsolutePrestateKey is the deploy-override key that carries the absolute
// prestate in the intent. It matches the json/toml tag of ChainProofParams.DisputeAbsolutePrestate.
const faultGameAbsolutePrestateKey = "faultGameAbsolutePrestate"

// isPermissionlessDeployment reports whether the chain's resolved respected game type
// runs a permissionless fault dispute game (which requires a real absolute prestate).
// The game type is resolved from the standard default merged with the intent overrides,
// matching how the deploy resolves it.
func isPermissionlessDeployment(intent *state.Intent, chain *state.ChainIntent) (bool, error) {
	proofParams, err := jsonutil.MergeJSON(
		state.ChainProofParams{DisputeGameType: standard.DisputeGameType},
		intent.GlobalDeployOverrides,
		chain.DeployOverrides,
	)
	if err != nil {
		return false, fmt.Errorf("error merging proof params from overrides: %w", err)
	}
	return isPermissionlessGameType(proofParams.DisputeGameType), nil
}

func isPermissionlessGameType(gameType uint32) bool {
	switch gameTypes.GameType(gameType) {
	case gameTypes.PermissionedGameType, gameTypes.SuperPermissionedGameType:
		return false
	default:
		return true
	}
}

// resolvePrestate resolves the absolute prestate for a chain. A non-zero flag value
// takes precedence; otherwise it looks for a faultGameAbsolutePrestate override on the
// chain first and then the global overrides. It returns the zero hash when no prestate
// is declared anywhere, which is valid for permissioned-only deployments.
func resolvePrestate(flagPrestate common.Hash, intent *state.Intent, chain *state.ChainIntent) (common.Hash, error) {
	if flagPrestate != (common.Hash{}) {
		return flagPrestate, nil
	}

	for _, overrides := range []map[string]any{chain.DeployOverrides, intent.GlobalDeployOverrides} {
		raw, ok := overrides[faultGameAbsolutePrestateKey]
		if !ok {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			return common.Hash{}, fmt.Errorf("%s override must be a hex string, got %T", faultGameAbsolutePrestateKey, raw)
		}
		prestate := common.HexToHash(s)
		if prestate == (common.Hash{}) {
			return common.Hash{}, fmt.Errorf("%s override must be a non-zero hash, got %q", faultGameAbsolutePrestateKey, s)
		}
		return prestate, nil
	}

	return common.Hash{}, nil
}

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

	gasLimit := chain.GasLimit
	if gasLimit == 0 {
		gasLimit = standard.GasLimit
	}

	return opcm.DeployOPChainInput{
		OpChainProxyAdminOwner: placeholderRole,
		SystemConfigOwner:      placeholderRole,
		Batcher:                placeholderRole,
		UnsafeBlockSigner:      placeholderRole,
		Proposer:               placeholderRole,
		Challenger:             placeholderRole,

		BasefeeScalar:     standard.BasefeeScalar,
		BlobBaseFeeScalar: standard.BlobBaseFeeScalar,
		L2ChainId:         chain.ID.Big(),
		Opcm:              *intent.OPCMAddress,
		SaltMixer:         st.Create2Salt.String(),
		GasLimit:          gasLimit,

		// Default dispute params
		DisputeGameType:         standard.DisputeGameType,
		DisputeAbsolutePrestate: standard.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:     new(big.Int).SetUint64(standard.DisputeMaxGameDepth),
		DisputeSplitDepth:       new(big.Int).SetUint64(standard.DisputeSplitDepth),
		DisputeClockExtension:   standard.DisputeClockExtension,
		DisputeMaxClockDuration: standard.DisputeMaxClockDuration,

		SuperchainConfig: *intent.SuperchainConfigProxy,
	}, nil
}
