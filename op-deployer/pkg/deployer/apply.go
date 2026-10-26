package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

type ApplyConfig struct {
	L1RPCUrl         string
	Workdir          string
	PrivateKey       string
	DeploymentTarget DeploymentTarget
	Logger           log.Logger
	CacheDir         string
	privateKeyECDSA  *ecdsa.PrivateKey
}

func (a *ApplyConfig) Check() error {
	if a.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}

	if a.PrivateKey != "" {
		privECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(a.PrivateKey, "0x"))
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		a.privateKeyECDSA = privECDSA
	}

	if a.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if a.DeploymentTarget == DeploymentTargetGenesis {
		if a.L1RPCUrl != "" {
			return fmt.Errorf("l1-rpc-url should not be specified when deployment-target is genesis")
		}
	}

	if a.DeploymentTarget == DeploymentTargetLive {
		if a.L1RPCUrl == "" {
			return fmt.Errorf("l1 RPC URL must be specified for live deployment")
		}

		if a.privateKeyECDSA == nil {
			return fmt.Errorf("private key must be specified for live deployment")
		}
	}

	return nil
}

func ApplyCLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := logcli.ReadCLIConfig(cliCtx)
		l := logcli.NewLogger(logcli.AppOut(cliCtx), logCfg)
		logcli.SetGlobalLogHandler(l.Handler())

		l1RPCUrl := cliCtx.String(L1RPCURLFlagName)
		workdir := cliCtx.String(WorkdirFlagName)
		privateKey := cliCtx.String(PrivateKeyFlagName)
		cacheDir := cliCtx.String(CacheDirFlagName)
		depTarget, err := NewDeploymentTarget(cliCtx.String(DeploymentTargetFlag.Name))
		if err != nil {
			return fmt.Errorf("failed to parse deployment target: %w", err)
		}

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

		if err := Apply(ctx, ApplyConfig{
			L1RPCUrl:         l1RPCUrl,
			Workdir:          workdir,
			PrivateKey:       privateKey,
			DeploymentTarget: depTarget,
			Logger:           l,
			CacheDir:         cacheDir,
		}); err != nil {
			return err
		}

		if depTarget != DeploymentTargetLive {
			return nil
		}
		if cliCtx.Bool(NoVerifyFlag.Name) {
			l.Warn("Contract verification skipped", "reason", "--no-verify was set")
			return nil
		}

		stateFile := fmt.Sprintf("%s/state.json", workdir)
		if err := func() error {
			chainID, err := ChainIDFromRPC(ctx, l1RPCUrl)
			if err != nil {
				return fmt.Errorf("failed to get chain ID: %w", err)
			}

			intent, err := pipeline.ReadIntent(workdir)
			if err != nil {
				return fmt.Errorf("failed to read intent: %w", err)
			}

			return verify.AutoVerify(
				ctx,
				l,
				l1RPCUrl,
				bigs.Uint64Strict(chainID),
				stateFile,
				stateFile,
				intent.L1ContractsLocator,
				cliCtx.String(VerifierTypeFlagName),
				cliCtx.String(VerifierUrlFlagName),
				cliCtx.String(VerifierAPIKeyFlagName),
			)
		}(); err != nil {
			verify.LogAutoVerifyFailure(l, stateFile, err)
		}
		return nil
	}
}

func Apply(ctx context.Context, cfg ApplyConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for apply: %w", err)
	}

	intent, err := pipeline.ReadIntent(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	st, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	// A state produced by the prepare pipeline MUST not be used with apply.
	if err := st.CheckNotPrepared(); err != nil {
		return err
	}

	if err := ApplyPipeline(ctx, ApplyPipelineOpts{
		L1RPCUrl:           cfg.L1RPCUrl,
		DeploymentTarget:   cfg.DeploymentTarget,
		DeployerPrivateKey: cfg.privateKeyECDSA,
		Intent:             intent,
		State:              st,
		Logger:             cfg.Logger,
		StateWriter:        pipeline.WorkdirStateWriter(cfg.Workdir),
		CacheDir:           cfg.CacheDir,
		Workdir:            cfg.Workdir,
	}); err != nil {
		return err
	}

	return nil
}

// ApplyL1DevGenesis deploys the superchain and the implementations into a new L1 dev genesis, with
// no chains. Unlike a genesis-target ApplyPipeline, it adds the preinstalls before the deployments,
// so that the deployer keeps its nonce and can deploy chains on the running L1 later.
func ApplyL1DevGenesis(ctx context.Context, opts ApplyPipelineOpts) error {
	opts.DeploymentTarget = DeploymentTargetGenesis
	intent := opts.Intent
	st := opts.State
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return err
	}
	bundle, err := artifacts.DownloadBundle(ctx, intent.L1ContractsLocator, intent.L2ContractsLocator, ioutil.NoopProgressor(), opts.CacheDir)
	if err != nil {
		return err
	}
	pEnv, err := newPipelineEnv(ctx, opts, bundle)
	if err != nil {
		return err
	}
	for _, stage := range []func(*pipeline.Env, *state.Intent, *state.State) error{
		pipeline.InitGenesisStrategy, pipeline.PreinstallL1DevGenesis,
		pipeline.DeploySuperchain, pipeline.DeployImplementations,
		pipeline.PrefundL1DevGenesis, pipeline.SealL1DevGenesis,
	} {
		if err := stage(pEnv, intent, st); err != nil {
			return err
		}
	}
	return nil
}

// newPipelineEnv connects the pipeline to the deployment target and to the engine that runs the
// scripts.
func newPipelineEnv(ctx context.Context, opts ApplyPipelineOpts, bundle artifacts.Bundle) (*pipeline.Env, error) {
	deployer := standard.PlaceholderAddress
	var privateKey string
	if opts.DeployerPrivateKey != nil {
		deployer = crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
		privateKey = hexutil.Encode(crypto.FromECDSA(opts.DeployerPrivateKey))
	}
	isGenesis := opts.DeploymentTarget == DeploymentTargetGenesis
	pEnv := &pipeline.Env{
		StateWriter:               opts.StateWriter,
		Logger:                    opts.Logger,
		Deployer:                  deployer,
		L1Artifacts:               &foundry.ArtifactsFS{FS: bundle.L1},
		IsGenesis:                 isGenesis,
		DeployMockSP1Verifier:     opts.DeployMockSP1Verifier,
		AllowUnoptimizedContracts: isGenesis || opts.AllowUnoptimizedContracts,
		Context:                   ctx,
	}

	switch opts.DeploymentTarget {
	case DeploymentTargetLive:
		l1Client, err := ethclient.DialContext(ctx, opts.L1RPCUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
		}
		pEnv.L1Client = l1Client
	case DeploymentTargetGenesis:
	default:
		return nil, fmt.Errorf("invalid deployment target: '%s'", opts.DeploymentTarget)
	}

	forgeClient, err := opcm.NewForgeClient(bundle.L1, opts.CacheDir, opcm.L1ForgeScripts)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 Forge client: %w", err)
	}
	pEnv.Forge = &opcm.ForgeEnv{
		Client:     forgeClient,
		Context:    ctx,
		L1RPCUrl:   opts.L1RPCUrl,
		PrivateKey: privateKey,
	}
	if isGenesis {
		pEnv.Forge.Genesis = opcm.NewGenesisState(deployer, true)
	}
	pEnv.L2Forge, err = opcm.NewForgeClient(bundle.L2, opts.CacheDir, opcm.L2ForgeScripts)
	if err != nil {
		return nil, fmt.Errorf("failed to create L2 Forge client: %w", err)
	}
	return pEnv, nil
}

type pipelineStage struct {
	name  string
	apply func() error
}

type ApplyPipelineOpts struct {
	L1RPCUrl           string
	DeploymentTarget   DeploymentTarget
	DeployerPrivateKey *ecdsa.PrivateKey
	Intent             *state.Intent
	State              *state.State
	Logger             log.Logger
	StateWriter        pipeline.StateWriter
	CacheDir           string
	// DeployMockSP1Verifier is a test-only opt-in for development environments.
	DeployMockSP1Verifier bool
	// AllowUnoptimizedContracts is a test-only opt-in that lets oversized dev-profile
	// artifacts build an L2 genesis. A genesis deployment always allows them.
	AllowUnoptimizedContracts bool
	Workdir                   string
}

func ApplyPipeline(
	ctx context.Context,
	opts ApplyPipelineOpts,
) error {
	if opts.DeployMockSP1Verifier && opts.DeploymentTarget != DeploymentTargetGenesis {
		return fmt.Errorf("mock SP1 verifier deployment is only supported for genesis")
	}

	intent := opts.Intent
	st := opts.State
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return err
	}

	bundle, err := artifacts.DownloadBundle(ctx, intent.L1ContractsLocator, intent.L2ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
	if err != nil {
		return err
	}

	pEnv, err := newPipelineEnv(ctx, opts, bundle)
	if err != nil {
		return err
	}

	pline := []pipelineStage{
		{"init", func() error {
			if opts.DeploymentTarget == DeploymentTargetGenesis {
				return pipeline.InitGenesisStrategy(pEnv, intent, st)
			}
			return pipeline.InitLiveStrategy(ctx, pEnv, intent, st)
		}},
		{"deploy-superchain", func() error {
			return pipeline.DeploySuperchain(pEnv, intent, st)
		}},
		{"deploy-implementations", func() error {
			return pipeline.DeployImplementations(pEnv, intent, st)
		}},
		{"generate-interop-depset", func() error {
			return pipeline.GenerateInteropDepset(ctx, pEnv, intent, st)
		}},
	}

	for _, chain := range intent.Chains {
		chainID := chain.ID
		pline = append(pline, pipelineStage{
			fmt.Sprintf("deploy-opchain-%s", chainID.Hex()),
			func() error {
				return pipeline.DeployOPChain(pEnv, intent, st, chainID)
			},
		}, pipelineStage{
			fmt.Sprintf("deploy-alt-da-%s", chainID.Hex()),
			func() error {
				return pipeline.DeployAltDA(pEnv, intent, st, chainID)
			},
		}, pipelineStage{
			fmt.Sprintf("deploy-additional-dispute-games-%s", chainID.Hex()),
			func() error {
				return pipeline.DeployAdditionalDisputeGames(pEnv, intent, st, chainID)
			},
		}, pipelineStage{
			fmt.Sprintf("generate-l2-genesis-%s", chainID.Hex()),
			func() error {
				return pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chainID)
			},
		})
	}

	if opts.DeploymentTarget == DeploymentTargetGenesis {
		for _, chain := range intent.Chains {
			chainID := chain.ID
			pline = append(pline, pipelineStage{
				"prefund-l2-dev-genesis",
				func() error {
					return pipeline.PrefundL2DevGenesis(pEnv, intent, st, chainID)
				},
			})
		}

		pline = append(pline, pipelineStage{
			"prefund-l1-dev-genesis",
			func() error {
				return pipeline.PrefundL1DevGenesis(pEnv, intent, st)
			},
		})

		pline = append(pline, pipelineStage{
			"preinstall-l1-dev-genesis",
			func() error {
				return pipeline.PreinstallL1DevGenesis(pEnv, intent, st)
			},
		})

		pline = append(pline, pipelineStage{
			"seal-l1-dev-genesis",
			func() error {
				return pipeline.SealL1DevGenesis(pEnv, intent, st)
			},
		})
	}

	// Set start block after all OP chains have been deployed, since the
	// genesis strategy requires all the OP chains to exist in genesis.
	for _, chain := range intent.Chains {
		chainID := chain.ID
		pline = append(pline, pipelineStage{
			fmt.Sprintf("set-start-block-%s", chainID.Hex()),
			func() error {
				if opts.DeploymentTarget == DeploymentTargetGenesis {
					return pipeline.SetStartBlockGenesisStrategy(pEnv, intent, st, chainID)
				}
				return pipeline.SetStartBlockLiveStrategy(ctx, intent, pEnv, st, chainID)
			},
		})
	}

	// Validate that the deployed state renders into a valid L2 genesis and rollup
	// config for every chain, so an invalid intent fails during apply rather than
	// later at inspect time.
	for _, chain := range intent.Chains {
		chainID := chain.ID
		pline = append(pline, pipelineStage{
			fmt.Sprintf("validate-l2-genesis-%s", chainID.Hex()),
			func() error {
				_, _, err := pipeline.RenderGenesisAndRollup(st, chainID, intent)
				return err
			},
		})
	}

	// Run through the pipeline.
	for _, stage := range pline {
		if err := stage.apply(); err != nil {
			return fmt.Errorf("error in pipeline stage apply: %w", err)
		}
		if err := pEnv.StateWriter.WriteState(st); err != nil {
			return fmt.Errorf("failed to write state: %w", err)
		}
	}

	st.AppliedIntent = intent
	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
