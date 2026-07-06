package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
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
	"github.com/ethereum/go-ethereum/crypto"
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
		Workdir:    cliCtx.String(WorkdirFlagName),
		Logger:     l,
		PrivateKey: cliCtx.String(PrivateKeyFlagName),
		L1RPCUrl:   cliCtx.String(L1RPCURLFlagName),
		CacheDir:   cliCtx.String(CacheDirFlagName),
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

	// The sender of the deploy dry-run. Pin it into the state so a re-run cannot
	// silently switch deployer keys.
	deployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)
	if err := st.CheckL1PredictSender(deployer); err != nil {
		return err
	}
	st.L1PredictSenderAddress = &deployer
	st.Prepared = true

	if err := st.EnsureCreate2Salt(); err != nil {
		return err
	}

	// Download the L1 artifacts referenced by the intent so the dry-run uses the
	// same DeployOPChain script as the eventual broadcast.
	l1ArtifactsFS, err := artifacts.Download(ctx, intent.L1ContractsLocator, ioutil.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to download L1 artifacts: %w", err)
	}

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

	if err := predictChains(cfg.Logger, intent, st, deployScript.Run); err != nil {
		return err
	}

	if err := pipeline.WriteState(cfg.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// predictChains predicts the L1 addresses for each undeployed chain in the intent
// and records them as not deployed. Chains that have been deployed are skipped so a
// re-runs don't overwrite their recorded addresses.
func predictChains(lgr log.Logger, intent *state.Intent, st *state.State, run func(opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error)) error {
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			lgr.Info("skipping already deployed chain", "chain", chain.ID.Hex())
			continue
		}

		dci, err := makePredictionInput(intent, st, chain)
		if err != nil {
			return fmt.Errorf("failed to build prediction input for chain %s: %w", chain.ID.Hex(), err)
		}

		out, err := run(dci)
		if err != nil {
			return fmt.Errorf("failed to predict L1 addresses for chain %s: %w", chain.ID.Hex(), err)
		}

		st.SetChainContracts(chain.ID, pipeline.OpChainContractsFromDeployOutput(out), false)

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

	proofParams, err := pipeline.ResolveChainProofParams(intent, chain)
	if err != nil {
		return opcm.DeployOPChainInput{}, fmt.Errorf("failed to resolve dispute params: %w", err)
	}

	return opcm.DeployOPChainInput{
		OpChainProxyAdminOwner: standard.PlaceholderAddress,
		SystemConfigOwner:      standard.PlaceholderAddress,
		Batcher:                standard.PlaceholderAddress,
		UnsafeBlockSigner:      standard.PlaceholderAddress,
		Proposer:               standard.PlaceholderAddress,
		Challenger:             standard.PlaceholderAddress,

		BasefeeScalar:     standard.BasefeeScalar,
		BlobBaseFeeScalar: standard.BlobBaseFeeScalar,
		L2ChainId:         chain.ID.Big(),
		Opcm:              *intent.OPCMAddress,
		SaltMixer:         st.Create2Salt.String(),
		GasLimit:          gasLimit,

		DisputeGameType:              proofParams.DisputeGameType,
		DisputeAbsolutePrestate:      proofParams.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:          new(big.Int).SetUint64(proofParams.DisputeMaxGameDepth),
		DisputeSplitDepth:            new(big.Int).SetUint64(proofParams.DisputeSplitDepth),
		DisputeClockExtension:        proofParams.DisputeClockExtension,
		DisputeMaxClockDuration:      proofParams.DisputeMaxClockDuration,
		AllowCustomDisputeParameters: proofParams.DangerouslyAllowCustomDisputeParameters,

		SuperchainConfig:    *intent.SuperchainConfigProxy,
		OperatorFeeScalar:   chain.OperatorFeeScalar,
		OperatorFeeConstant: chain.OperatorFeeConstant,
		UseCustomGasToken:   chain.IsCustomGasTokenEnabled(),
	}, nil
}
