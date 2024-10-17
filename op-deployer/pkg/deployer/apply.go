package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

type ApplyConfig struct {
	L1RPCUrl   string
	Workdir    string
	PrivateKey string
	Logger     log.Logger

	privateKeyECDSA *ecdsa.PrivateKey
}

func (a *ApplyConfig) Check() error {
	if a.L1RPCUrl == "" {
		return fmt.Errorf("l1RPCUrl must be specified")
	}

	if a.Workdir == "" {
		return fmt.Errorf("workdir must be specified")
	}

	if a.PrivateKey == "" {
		return fmt.Errorf("private key must be specified")
	}

	privECDSA, err := crypto.HexToECDSA(strings.TrimPrefix(a.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	a.privateKeyECDSA = privECDSA

	if a.Logger == nil {
		return fmt.Errorf("logger must be specified")
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

		ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

		return Apply(ctx, ApplyConfig{
			L1RPCUrl:   l1RPCUrl,
			Workdir:    workdir,
			PrivateKey: privateKey,
			Logger:     l,
		})
	}
}

func Apply(ctx context.Context, cfg ApplyConfig) error {
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid config for apply: %w", err)
	}

	l1Client, err := ethclient.Dial(cfg.L1RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	chainID, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(cfg.privateKeyECDSA, chainID))
	deployer := crypto.PubkeyToAddress(cfg.privateKeyECDSA.PublicKey)

	intent, err := pipeline.ReadIntent(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	st, err := pipeline.ReadState(cfg.Workdir)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	env := &pipeline.Env{
		Workdir:  cfg.Workdir,
		L1Client: l1Client,
		Logger:   cfg.Logger,
		Signer:   signer,
		Deployer: deployer,
	}

	if err := ApplyPipeline(ctx, env, intent, st); err != nil {
		return err
	}

	return nil
}

type pipelineStage struct {
	name  string
	apply pipeline.Stage
}

func ApplyPipeline(
	ctx context.Context,
	env *pipeline.Env,
	intent *state.Intent,
	st *state.State,
) error {
	progressor := func(curr, total int64) {
		env.Logger.Info("artifacts download progress", "current", curr, "total", total)
	}

	l1ArtifactsFS, cleanupL1, err := pipeline.DownloadArtifacts(ctx, intent.L1ContractsLocator, progressor)
	if err != nil {
		return fmt.Errorf("failed to download L1 artifacts: %w", err)
	}
	defer func() {
		if err := cleanupL1(); err != nil {
			env.Logger.Warn("failed to clean up L1 artifacts", "err", err)
		}
	}()

	l2ArtifactsFS, cleanupL2, err := pipeline.DownloadArtifacts(ctx, intent.L2ContractsLocator, progressor)
	if err != nil {
		return fmt.Errorf("failed to download L2 artifacts: %w", err)
	}
	defer func() {
		if err := cleanupL2(); err != nil {
			env.Logger.Warn("failed to clean up L2 artifacts", "err", err)
		}
	}()

	bundle := pipeline.ArtifactsBundle{
		L1: l1ArtifactsFS,
		L2: l2ArtifactsFS,
	}

	pline := []pipelineStage{
		{"init", pipeline.Init},
		{"deploy-superchain", pipeline.DeploySuperchain},
		{"deploy-implementations", pipeline.DeployImplementations},
	}

	for _, chain := range intent.Chains {
		chainID := chain.ID
		pline = append(pline, pipelineStage{
			fmt.Sprintf("deploy-opchain-%s", chainID.Hex()),
			func(ctx context.Context, env *pipeline.Env, bundle pipeline.ArtifactsBundle, intent *state.Intent, st *state.State) error {
				return pipeline.DeployOPChain(ctx, env, bundle, intent, st, chainID)
			},
		}, pipelineStage{
			fmt.Sprintf("generate-l2-genesis-%s", chainID.Hex()),
			func(ctx context.Context, env *pipeline.Env, bundle pipeline.ArtifactsBundle, intent *state.Intent, st *state.State) error {
				return pipeline.GenerateL2Genesis(ctx, env, bundle, intent, st, chainID)
			},
		})
	}

	for _, stage := range pline {
		if err := stage.apply(ctx, env, bundle, intent, st); err != nil {
			return fmt.Errorf("error in pipeline stage apply: %w", err)
		}
		if err := pipeline.WriteState(env.Workdir, st); err != nil {
			return fmt.Errorf("failed to write state: %w", err)
		}
	}

	st.AppliedIntent = intent
	if err := pipeline.WriteState(env.Workdir, st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
