package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/pipeline"
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

	env := &pipeline.Env{
		Workdir:  cfg.Workdir,
		L1Client: l1Client,
		Logger:   cfg.Logger,
		Signer:   signer,
		Deployer: deployer,
	}

	intent, err := env.ReadIntent()
	if err != nil {
		return err
	}

	if err := intent.Check(); err != nil {
		return fmt.Errorf("invalid intent: %w", err)
	}

	st, err := env.ReadState()
	if err != nil {
		return err
	}

	if err := ApplyPipeline(ctx, env, intent, st); err != nil {
		return err
	}

	st.AppliedIntent = intent
	if err := env.WriteState(st); err != nil {
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

	artifactsFS, cleanup, err := pipeline.DownloadArtifacts(ctx, intent.ContractArtifactsURL, progressor)
	if err != nil {
		return fmt.Errorf("failed to download artifacts: %w", err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			env.Logger.Warn("failed to clean up artifacts", "err", err)
		}
	}()

	pline := []pipelineStage{
		{"init", pipeline.Init},
		{"deploy-superchain", pipeline.DeploySuperchain},
		{"deploy-implementations", pipeline.DeployImplementations},
	}

	for _, chain := range intent.Chains {
		chainID := chain.ID
		pline = append(pline, pipelineStage{
			fmt.Sprintf("deploy-opchain-%s", chainID.Hex()),
			func(ctx context.Context, env *pipeline.Env, artifactsFS foundry.StatDirFs, intent *state.Intent, st *state.State) error {
				return pipeline.DeployOPChain(ctx, env, artifactsFS, intent, st, chainID)
			},
		}, pipelineStage{
			fmt.Sprintf("generate-l2-genesis-%s", chainID.Hex()),
			func(ctx context.Context, env *pipeline.Env, artifactsFS foundry.StatDirFs, intent *state.Intent, st *state.State) error {
				return pipeline.GenerateL2Genesis(ctx, env, artifactsFS, intent, st, chainID)
			},
		})
	}

	for _, stage := range pline {
		if err := stage.apply(ctx, env, artifactsFS, intent, st); err != nil {
			return fmt.Errorf("error in pipeline stage apply: %w", err)
		}
	}

	st.AppliedIntent = intent
	if err := env.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
