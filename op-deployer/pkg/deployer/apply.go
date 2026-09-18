package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"maps"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
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
	UseForge         bool
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
		logCfg := oplog.ReadCLIConfig(cliCtx)
		l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(l.Handler())

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
			UseForge:         cliCtx.Bool(UseForgeFlagName),
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
		UseForge:           cfg.UseForge,
		PrivateKey:         cfg.PrivateKey,
		Workdir:            cfg.Workdir,
	}); err != nil {
		return err
	}

	return nil
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
	UseForge           bool
	// DeployMockSP1Verifier is a test-only opt-in for development environments.
	DeployMockSP1Verifier bool
	PrivateKey            string
	Workdir               string
	ReceiptQueryInterval  time.Duration
	// GenesisAnchorGameType selects the anchor family for a later permissionless game upgrade.
	// Nil keeps permissioned placeholders and derives permissionless anchors from the initial game type.
	GenesisAnchorGameType *uint32
	// L2GenesisAllocMutator runs after prefunds and before any genesis commitment.
	L2GenesisAllocMutator func(common.Hash, *foundry.ForgeAllocs) error
}

func ApplyPipeline(
	ctx context.Context,
	opts ApplyPipelineOpts,
) error {
	if opts.DeployMockSP1Verifier && opts.DeploymentTarget != DeploymentTargetGenesis {
		return fmt.Errorf("mock SP1 verifier deployment is only supported for genesis")
	}
	if opts.DeploymentTarget != DeploymentTargetGenesis &&
		(opts.GenesisAnchorGameType != nil || opts.L2GenesisAllocMutator != nil) {
		return fmt.Errorf("genesis anchor and allocation options require the genesis deployment target")
	}

	intent := opts.Intent
	st := opts.State
	if err := pipeline.ValidateInputs(intent, st); err != nil {
		return err
	}

	prepareGenesis := false
	if opts.DeploymentTarget == DeploymentTargetGenesis {
		prepareGenesis = opts.GenesisAnchorGameType != nil
		if opts.GenesisAnchorGameType != nil {
			requirements, err := pipeline.ResolveInitialDeployRequirements(*opts.GenesisAnchorGameType)
			if err != nil {
				return err
			}
			if !requirements.Permissionless {
				return fmt.Errorf("genesis anchor game type must be permissionless")
			}
		}
		for _, chain := range intent.Chains {
			params, err := pipeline.ResolveChainProofParams(intent, chain)
			if err != nil {
				return err
			}
			requirements, err := pipeline.ResolveInitialDeployRequirements(params.DisputeGameType)
			if err != nil {
				return err
			}
			if requirements.Permissionless && opts.GenesisAnchorGameType != nil &&
				pipeline.IsSuperGameType(params.DisputeGameType) != pipeline.IsSuperGameType(*opts.GenesisAnchorGameType) {
				return fmt.Errorf("chain %s initial game and requested genesis anchor use different root families", chain.ID)
			}
			prepareGenesis = prepareGenesis || requirements.Permissionless
		}
	}

	bundle, err := artifacts.DownloadBundle(ctx, intent.L1ContractsLocator, intent.L2ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
	if err != nil {
		return err
	}

	deployer := standard.PlaceholderAddress
	if opts.DeployerPrivateKey != nil {
		deployer = crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
	}

	var bcaster broadcaster.Broadcaster
	var l1RPC *rpc.Client
	var l1Client *ethclient.Client
	var l1Host *script.Host

	switch opts.DeploymentTarget {
	case DeploymentTargetLive:
		l1RPC, err = rpc.Dial(opts.L1RPCUrl)
		if err != nil {
			return fmt.Errorf("failed to connect to L1 RPC: %w", err)
		}

		l1Client = ethclient.NewClient(l1RPC)

		chainID, err := l1Client.ChainID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get chain ID: %w", err)
		}

		signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(opts.DeployerPrivateKey, chainID))

		bcaster, err = broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
			Logger:               opts.Logger,
			ChainID:              new(big.Int).SetUint64(intent.L1ChainID),
			Client:               l1Client,
			Signer:               signer,
			From:                 deployer,
			ReceiptQueryInterval: opts.ReceiptQueryInterval,
		})
		if err != nil {
			return fmt.Errorf("failed to create broadcaster: %w", err)
		}

		l1Host, err = env.DefaultForkedScriptHost(ctx, bcaster, opts.Logger, deployer, bundle.L1, l1RPC)
		if err != nil {
			return fmt.Errorf("failed to initialize L1 host: %w", err)
		}
	case DeploymentTargetCalldata, DeploymentTargetNoop:
		l1RPC, err = rpc.Dial(opts.L1RPCUrl)
		if err != nil {
			return fmt.Errorf("failed to connect to L1 RPC: %w", err)
		}

		l1Client = ethclient.NewClient(l1RPC)

		bcaster = new(broadcaster.CalldataBroadcaster)

		l1Host, err = env.DefaultForkedScriptHost(ctx, bcaster, opts.Logger, deployer, bundle.L1, l1RPC)
		if err != nil {
			return fmt.Errorf("failed to initialize L1 host: %w", err)
		}
	case DeploymentTargetGenesis:
		bcaster = broadcaster.NoopBroadcaster()
		l1Host, err = env.DefaultScriptHost(
			bcaster,
			opts.Logger,
			deployer,
			bundle.L1,
			script.WithNoMaxCodeSize(), // Allow unoptimized contracts from the forge lite profile in genesis deployments
		)
		if err != nil {
			return fmt.Errorf("failed to create L1 script host: %w", err)
		}
	default:
		return fmt.Errorf("invalid deployment target: '%s'", opts.DeploymentTarget)
	}

	// Now that we have the host, we can load the deployment scripts
	//
	// This step will error out if the ABIs don't match the Go types
	opcmScripts, err := opcm.NewScripts(l1Host)
	if err != nil {
		return fmt.Errorf("failed to load OPCM script: %w", err)
	}

	// Initialize Forge client if UseForge flag is enabled
	var forgeClient *forge.Client
	if opts.UseForge {
		// Forge needs to run from the artifacts directory where foundry.toml is located
		// The workdir is for storing state, not for running forge commands
		artifactsPath := fmt.Sprintf("%v", bundle.L1)
		forgeClient, err = forge.NewStandardClient(artifactsPath)
		if err != nil {
			return fmt.Errorf("failed to create Forge client: %w", err)
		}
	}

	pEnv := &pipeline.Env{
		StateWriter:               opts.StateWriter,
		L1ScriptHost:              l1Host,
		L1Client:                  l1Client,
		Logger:                    opts.Logger,
		Broadcaster:               bcaster,
		Deployer:                  deployer,
		Scripts:                   opcmScripts,
		ForgeClient:               forgeClient,
		UseForge:                  opts.UseForge,
		IsGenesis:                 opts.DeploymentTarget == DeploymentTargetGenesis,
		DeployMockSP1Verifier:     opts.DeployMockSP1Verifier,
		AllowUnoptimizedContracts: opts.DeploymentTarget == DeploymentTargetGenesis,
		L1RPCUrl:                  opts.L1RPCUrl,
		PrivateKey:                opts.PrivateKey,
		Context:                   ctx,
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

	finalizeL2Allocs := func(chainID common.Hash) error {
		if err := pipeline.PrefundL2DevGenesis(pEnv, intent, st, chainID); err != nil {
			return err
		}
		if opts.L2GenesisAllocMutator != nil {
			chain, err := st.Chain(chainID)
			if err != nil {
				return err
			}
			return opts.L2GenesisAllocMutator(chainID, chain.Allocs.Data)
		}
		return nil
	}
	if prepareGenesis {
		pline = append(pline, pipelineStage{"prepare-genesis-anchors", func() error {
			return prepareGenesisAnchors(pEnv, intent, bundle, st, opts.GenesisAnchorGameType, finalizeL2Allocs)
		}})
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
		if !prepareGenesis {
			for _, chain := range intent.Chains {
				chainID := chain.ID
				pline = append(pline, pipelineStage{
					"finalize-l2-dev-genesis",
					func() error { return finalizeL2Allocs(chainID) },
				})
			}
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
		if _, err := pEnv.Broadcaster.Broadcast(ctx); err != nil {
			return fmt.Errorf("failed to broadcast stage %s: %w", stage.name, err)
		}
		if err := pEnv.StateWriter.WriteState(st); err != nil {
			return fmt.Errorf("failed to write state: %w", err)
		}
	}

	if opts.DeploymentTarget == DeploymentTargetCalldata {
		cdCaster := pEnv.Broadcaster.(*broadcaster.CalldataBroadcaster)
		st.DeploymentCalldata, err = cdCaster.Dump()
		if err != nil {
			return fmt.Errorf("failed to dump calldata: %w", err)
		}
	}

	st.AppliedIntent = intent
	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

func prepareGenesisAnchors(
	pEnv *pipeline.Env,
	intent *state.Intent,
	bundle artifacts.Bundle,
	st *state.State,
	anchorGameType *uint32,
	finalizeAllocs func(common.Hash) error,
) error {
	// Prediction must not deploy chain contracts into the final L1 state.
	dump, err := pEnv.L1ScriptHost.StateDump()
	if err != nil {
		return fmt.Errorf("failed to snapshot implementations: %w", err)
	}
	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(), pEnv.Logger, pEnv.Deployer, bundle.L1, script.WithNoMaxCodeSize(),
	)
	if err != nil {
		return err
	}
	host.ImportState(dump)
	deployScript, err := opcm.NewDeployOPChainScript(host)
	if err != nil {
		return err
	}
	if intent.L1DevGenesisParams == nil {
		intent.L1DevGenesisParams = &state.L1DevGenesisParams{}
	}
	if intent.L1DevGenesisParams.BlockParams.Timestamp == 0 {
		intent.L1DevGenesisParams.BlockParams.Timestamp = uint64(time.Now().Unix())
	}
	timestamp := intent.L1DevGenesisParams.BlockParams.Timestamp
	predictionIntent := *intent
	predictionIntent.OPCMAddress = &st.ImplementationsDeployment.OpcmV2Impl
	predictionIntent.SuperchainConfigProxy = &st.SuperchainDeployment.SuperchainConfigProxy
	for _, chain := range intent.Chains {
		if st.IsChainDeployed(chain.ID) {
			continue
		}
		input, err := makePredictionInput(&predictionIntent, st, chain)
		if err != nil {
			return err
		}
		output, err := deployScript.Run(input)
		if err != nil {
			return fmt.Errorf("failed to predict chain %s: %w", chain.ID, err)
		}
		st.SetChainContracts(chain.ID, pipeline.OpChainContractsFromDeployOutput(output), false)
		chainState, err := st.Chain(chain.ID)
		if err != nil {
			return err
		}
		// L2 genesis uses this timestamp, not the L1 block hash. Sealing replaces this reference.
		chainState.StartBlock = &state.L1BlockRefJSON{Time: hexutil.Uint64(timestamp)}
		if err := pipeline.GenerateL2Genesis(pEnv, intent, bundle, st, chain.ID); err != nil {
			return err
		}
		if err := finalizeAllocs(chain.ID); err != nil {
			return err
		}
	}

	rootIntent := intent
	if anchorGameType != nil {
		copyIntent := *intent
		copyIntent.Chains = make([]*state.ChainIntent, len(intent.Chains))
		for i, chain := range intent.Chains {
			copyChain := *chain
			copyChain.DeployOverrides = maps.Clone(chain.DeployOverrides)
			if copyChain.DeployOverrides == nil {
				copyChain.DeployOverrides = make(map[string]any)
			}
			copyChain.DeployOverrides["respectedGameType"] = *anchorGameType
			copyIntent.Chains[i] = &copyChain
		}
		rootIntent = &copyIntent
	}
	return pipeline.ComputeGenesisOutputRoots(pEnv, rootIntent, st)
}
