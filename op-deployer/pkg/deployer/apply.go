package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/rustengine"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/verify"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
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
	ScriptEngine     env.ScriptEngineKind
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

		scriptEngine, err := env.ParseScriptEngine(cliCtx.String(ScriptEngineFlagName))
		if err != nil {
			return err
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
			ScriptEngine:     scriptEngine,
		}); err != nil {
			return err
		}

		if !cliCtx.Bool(AutoVerifyFlag.Name) {
			return nil
		}

		stateFile := fmt.Sprintf("%s/state.json", workdir)
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
			intent.L1ContractsLocator,
			cliCtx.String(VerifierTypeFlagName),
			cliCtx.String(VerifierUrlFlagName),
			cliCtx.String(VerifierAPIKeyFlagName),
		)
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
		ScriptEngine:       cfg.ScriptEngine,
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
	ScriptEngine       env.ScriptEngineKind
	PrivateKey         string
	Workdir            string
}

func ApplyPipeline(
	ctx context.Context,
	opts ApplyPipelineOpts,
) error {
	intent := opts.Intent
	if err := intent.Check(); err != nil {
		return err
	}
	st := opts.State

	l1ArtifactsFS, err := artifacts.Download(ctx, intent.L1ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to download L1 artifacts: %w", err)
	}

	var l2ArtifactsFS foundry.StatDirFs
	if intent.L1ContractsLocator.Equal(intent.L2ContractsLocator) {
		l2ArtifactsFS = l1ArtifactsFS
	} else {
		l2ArtifactsFS, err = artifacts.Download(ctx, intent.L2ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
		if err != nil {
			return fmt.Errorf("failed to download L2 artifacts: %w", err)
		}
	}

	bundle := pipeline.ArtifactsBundle{
		L1: l1ArtifactsFS,
		L2: l2ArtifactsFS,
	}

	deployer := common.Address{0x01}
	if opts.DeployerPrivateKey != nil {
		deployer = crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
	}

	var bcaster broadcaster.Broadcaster
	var l1RPC *rpc.Client
	var l1Client *ethclient.Client
	var l1Host *script.Host
	// l1Engine, when set, routes the L1 deploy stages through the out-of-process Rust
	// op-script-engine instead of l1Host (which is then nil). It backs both the non-forked
	// DeploymentTargetGenesis L1 deploy and the forked Live/Calldata/Noop targets on the rust
	// engine; --script-engine=go keeps every target on the in-process Go script.Host.
	var l1Engine *rustengine.Engine
	var l1EngineFA *foundry.ArtifactsFS
	var opcmScripts *opcm.Scripts

	// Engine selection is now flag-only (never host-kind): the resolved ScriptEngine governs both
	// the non-forked genesis host and the forked Live/Calldata/Noop hosts. rust (the default) runs
	// the L1 deploy on the Rust op-script-engine's fork mode; --script-engine=go keeps the Go host.
	useRustEngine := opts.ScriptEngine.Resolve() == env.ScriptEngineRust

	// initForkHost builds the forked L1 host for the Live/Calldata/Noop targets: the Rust engine's
	// fork mode (CreateSelectFork against opts.L1RPCUrl) on the rust default, or the in-process Go
	// fork-backed script.Host on --script-engine=go.
	initForkHost := func() error {
		latest, err := l1Client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block: %w", err)
		}

		if useRustEngine {
			eng, scripts, fa, eerr := initForkedL1Engine(ctx, opts, bundle, deployer, latest.Number.Uint64())
			if eerr != nil {
				return fmt.Errorf("failed to initialize forked L1 script engine: %w", eerr)
			}
			l1Engine, opcmScripts, l1EngineFA = eng, scripts, fa
			return nil
		}

		l1Host, err = env.DefaultScriptHost(
			bcaster,
			opts.Logger,
			deployer,
			bundle.L1,
			script.WithForkHook(func(cfg *script.ForkConfig) (forking.ForkSource, error) {
				src, err := forking.RPCSourceByNumber(cfg.URLOrAlias, l1RPC, *cfg.BlockNumber)
				if err != nil {
					return nil, fmt.Errorf("failed to create RPC fork source: %w", err)
				}
				return forking.Cache(src), nil
			}),
		)
		if err != nil {
			return fmt.Errorf("failed to create L1 script host: %w", err)
		}

		if _, err := l1Host.CreateSelectFork(
			script.ForkWithURLOrAlias("main"),
			script.ForkWithBlockNumberU256(latest.Number),
		); err != nil {
			return fmt.Errorf("failed to select fork: %w", err)
		}

		return nil
	}

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
			Logger:  opts.Logger,
			ChainID: new(big.Int).SetUint64(intent.L1ChainID),
			Client:  l1Client,
			Signer:  signer,
			From:    deployer,
		})
		if err != nil {
			return fmt.Errorf("failed to create broadcaster: %w", err)
		}

		if err := initForkHost(); err != nil {
			return fmt.Errorf("failed to initialize L1 host: %w", err)
		}
	case DeploymentTargetCalldata, DeploymentTargetNoop:
		l1RPC, err = rpc.Dial(opts.L1RPCUrl)
		if err != nil {
			return fmt.Errorf("failed to connect to L1 RPC: %w", err)
		}

		l1Client = ethclient.NewClient(l1RPC)

		bcaster = new(broadcaster.CalldataBroadcaster)

		if err := initForkHost(); err != nil {
			return fmt.Errorf("failed to initialize L1 host: %w", err)
		}
	case DeploymentTargetGenesis:
		// Non-forked genesis L1 deploy host. Engine selection is flag-only: rust (default) runs the
		// L1 deploy on the Rust op-script-engine, --script-engine=go falls back to the in-process Go
		// script.Host. The forked Live/Calldata/Noop targets (above) route the same way.
		bcaster = broadcaster.NoopBroadcaster()
		if useRustEngine {
			l1Engine, opcmScripts, l1EngineFA, err = initGenesisL1Engine(ctx, opts, bundle, deployer)
			if err != nil {
				return fmt.Errorf("failed to initialize L1 script engine: %w", err)
			}
		} else {
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
		}
	default:
		return fmt.Errorf("invalid deployment target: '%s'", opts.DeploymentTarget)
	}

	// The L1 engine (genesis or forked) owns a subprocess; close it when the pipeline returns.
	defer func() {
		if l1Engine != nil {
			l1Engine.Close()
		}
	}()

	// Now that we have the host, we can load the deployment scripts (unless the L1 engine already
	// built engine-backed scripts above).
	//
	// This step will error out if the ABIs don't match the Go types
	if opcmScripts == nil {
		opcmScripts, err = opcm.NewScripts(l1Host)
		if err != nil {
			return fmt.Errorf("failed to load OPCM script: %w", err)
		}
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
		StateWriter:  opts.StateWriter,
		L1ScriptHost: l1Host,
		L1Engine:     l1Engine,
		L1Artifacts:  l1EngineFA,
		L1Client:     l1Client,
		Logger:       opts.Logger,
		Broadcaster:  bcaster,
		Deployer:     deployer,
		Scripts:      opcmScripts,
		ForgeClient:  forgeClient,
		UseForge:     opts.UseForge,
		ScriptEngine: opts.ScriptEngine,
		L1RPCUrl:     opts.L1RPCUrl,
		PrivateKey:   opts.PrivateKey,
		Context:      ctx,
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
	}

	// Deploy all OP Chains first.
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

	// Generate the interop dependency set
	pline = append(pline, pipelineStage{
		"generate-interop-depset",
		func() error {
			return pipeline.GenerateInteropDepset(ctx, pEnv, intent, st)
		},
	})

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

	// drainForkedBroadcasts moves the broadcasts a forked engine captured during a stage into the
	// Go broadcaster, mirroring the Go host's synchronous WithBroadcastHook delivery. It runs only
	// for the forked-target engine path: the Go host feeds the broadcaster directly, and the
	// non-forked genesis engine uses a Noop broadcaster (no broadcasts to deliver).
	drainForkedBroadcasts := func() error {
		if l1Engine == nil || opts.DeploymentTarget == DeploymentTargetGenesis {
			return nil
		}
		bcasts, err := l1Engine.TakeBroadcasts()
		if err != nil {
			return fmt.Errorf("failed to take engine broadcasts: %w", err)
		}
		for _, b := range bcasts {
			bcaster.Hook(b)
		}
		return nil
	}

	// Run through the pipeline.
	for _, stage := range pline {
		if err := stage.apply(); err != nil {
			return fmt.Errorf("error in pipeline stage apply: %w", err)
		}
		if err := drainForkedBroadcasts(); err != nil {
			return fmt.Errorf("failed to drain broadcasts for stage %s: %w", stage.name, err)
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

// initGenesisL1Engine spawns the out-of-process Rust op-script-engine for the non-forked L1 deploy
// stages of a genesis deployment and builds the engine-backed OPCM Scripts bundle. The engine's
// host context mirrors the Go env.DefaultScriptHost used for the same path: the default script
// chain/context, the CREATE2 deployer preloaded, and the EIP-170/3860 code-size limits lifted (the
// forge lite profile emits unoptimized contracts). The returned engine must be Closed by the caller.
func initGenesisL1Engine(
	ctx context.Context,
	opts ApplyPipelineOpts,
	bundle pipeline.ArtifactsBundle,
	deployer common.Address,
) (*rustengine.Engine, *opcm.Scripts, *foundry.ArtifactsFS, error) {
	artifactsDir, err := rustengine.ArtifactsDir(bundle.L1)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve L1 artifacts directory: %w", err)
	}

	binPath, err := rustengine.EngineBinary(ctx, opts.Logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to provision op-script-engine binary: %w", err)
	}

	eng, err := rustengine.Spawn(binPath, rustengine.SpawnOpts{
		ArtifactsDir:       artifactsDir,
		ChainID:            script.DefaultContext.ChainID.Uint64(),
		Create2Deployer:    true,
		NoMaxCodeSize:      true,
		IsolatedBroadcasts: true, // mirrors env.DefaultScriptHost's script.WithIsolatedBroadcasts
		BlockNum:           script.DefaultContext.BlockNum,
		Timestamp:          script.DefaultContext.Timestamp,
		PrevRandao:         script.DefaultContext.PrevRandao,
	}, rustengine.NewLogWriter(opts.Logger))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to spawn op-script-engine for L1: %w", err)
	}

	fa := &foundry.ArtifactsFS{FS: bundle.L1}
	origin := func() common.Address { return deployer }
	scripts, err := pipeline.NewEngineScripts(eng, fa, origin)
	if err != nil {
		eng.Close()
		return nil, nil, nil, fmt.Errorf("failed to load engine-backed OPCM scripts: %w", err)
	}
	return eng, scripts, fa, nil
}

// initForkedL1Engine spawns the out-of-process Rust op-script-engine for the forked L1 deploy of a
// Live/Calldata/Noop deployment and installs an RPC-backed fork of the live L1 pinned to forkBlock,
// mirroring the Go forked host (env.DefaultScriptHost + WithForkHook + CreateSelectFork(latest)).
// The engine dials opts.L1RPCUrl directly (Option A, the unidirectional transport). Unlike the
// genesis engine it does NOT lift the code-size limits: forked deploys use the optimized production
// artifacts. The returned engine must be Closed by the caller.
func initForkedL1Engine(
	ctx context.Context,
	opts ApplyPipelineOpts,
	bundle pipeline.ArtifactsBundle,
	deployer common.Address,
	forkBlock uint64,
) (*rustengine.Engine, *opcm.Scripts, *foundry.ArtifactsFS, error) {
	artifactsDir, err := rustengine.ArtifactsDir(bundle.L1)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve L1 artifacts directory: %w", err)
	}

	binPath, err := rustengine.EngineBinary(ctx, opts.Logger)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to provision op-script-engine binary: %w", err)
	}

	eng, err := rustengine.Spawn(binPath, rustengine.SpawnOpts{
		ArtifactsDir:       artifactsDir,
		ChainID:            script.DefaultContext.ChainID.Uint64(),
		Create2Deployer:    true,
		IsolatedBroadcasts: true, // mirrors env.DefaultScriptHost's script.WithIsolatedBroadcasts
		BlockNum:           script.DefaultContext.BlockNum,
		Timestamp:          script.DefaultContext.Timestamp,
		PrevRandao:         script.DefaultContext.PrevRandao,
	}, rustengine.NewLogWriter(opts.Logger))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to spawn op-script-engine for forked L1: %w", err)
	}

	fb := forkBlock
	if _, err := eng.CreateSelectFork(opts.L1RPCUrl, &fb); err != nil {
		eng.Close()
		return nil, nil, nil, fmt.Errorf("failed to select fork: %w", err)
	}

	fa := &foundry.ArtifactsFS{FS: bundle.L1}
	origin := func() common.Address { return deployer }
	scripts, err := pipeline.NewEngineScripts(eng, fa, origin)
	if err != nil {
		eng.Close()
		return nil, nil, nil, fmt.Errorf("failed to load engine-backed OPCM scripts: %w", err)
	}
	return eng, scripts, fa, nil
}
