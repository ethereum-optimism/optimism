package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
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
	PreStateBuilder  pipeline.PreStateBuilder
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
		opProgramSvcUrl := cliCtx.String(OpProgramSvcUrlFlag.Name)

		var preStateBuilder pipeline.PreStateBuilder
		if opProgramSvcUrl != "" {
			preStateBuilder = prestate.NewPrestateBuilderClient(opProgramSvcUrl)
		}

		if err != nil {
			return fmt.Errorf("failed to parse deployment target: %w", err)
		}

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

		return Apply(ctx, ApplyConfig{
			L1RPCUrl:         l1RPCUrl,
			Workdir:          workdir,
			PrivateKey:       privateKey,
			DeploymentTarget: depTarget,
			Logger:           l,
			CacheDir:         cacheDir,
			PreStateBuilder:  preStateBuilder,
		})
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
		PreStateBuilder:    cfg.PreStateBuilder,
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
	PreStateBuilder    pipeline.PreStateBuilder
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
		l2Afs, err := artifacts.Download(ctx, intent.L2ContractsLocator, ioutil.BarProgressor(), opts.CacheDir)
		if err != nil {
			return fmt.Errorf("failed to download L2 artifacts: %w", err)
		}
		l2ArtifactsFS = l2Afs
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

	initForkHost := func() error {
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

		latest, err := l1Client.HeaderByNumber(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to get latest block: %w", err)
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
		bcaster = broadcaster.NoopBroadcaster()
		l1Host, err = env.DefaultScriptHost(
			bcaster,
			opts.Logger,
			deployer,
			bundle.L1,
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

	pEnv := &pipeline.Env{
		StateWriter:  opts.StateWriter,
		L1ScriptHost: l1Host,
		L1Client:     l1Client,
		Logger:       opts.Logger,
		Broadcaster:  bcaster,
		Deployer:     deployer,
		Scripts:      opcmScripts,
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

	// Generate the prestate for all chains
	pline = append(pline, pipelineStage{
		"deploy-pre-state",
		func() error {
			return pipeline.GeneratePreState(ctx, pEnv, intent, st, opts.PreStateBuilder)
		},
	})

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
