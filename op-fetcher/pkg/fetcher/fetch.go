package fetcher

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/script"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

//go:embed forge-artifacts
var forgeArtifacts embed.FS

func FetchChainInfoCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		cfg, err := NewConfig(cliCtx)
		if err != nil {
			return err
		}

		output, err := FetchChainInfo(cliCtx.Context, lgr, cfg)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}
		lgr.Info("successfully fetched chain info")

		fileOutput := script.ChainConfig{
			Addresses: output.Addresses,
			Roles:     output.Roles,
			FaultProofStatus: script.FaultProofStatus{
				Permissioned:   output.FaultProofPermissioned,
				Permissionless: output.FaultProofPermissionless,
			},
		}

		json, err := json.MarshalIndent(fileOutput, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}

		if err := os.MkdirAll("./.fetcher", 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		err = os.WriteFile("./.fetcher/chain-info.json", json, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output to file: %w", err)
		}
		lgr.Info("wrote chain info to file")
		return nil
	}
}

func FetchChainInfo(ctx context.Context, lgr log.Logger, cfg *Config) (script.FetchChainInfoOutput, error) {
	lgr.Info("fetching chain info", "systemConfig", cfg.SystemConfig, "l1StandardBridge", cfg.L1StandardBridge)

	l1RPC, err := rpc.Dial(cfg.L1RPCURL)
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	bcaster := broadcaster.NoopBroadcaster()
	deployerAddress := common.Address{0x01}

	artifactsFS := &foundry.EmbedFS{
		FS:      forgeArtifacts,
		RootDir: "forge-artifacts",
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		deployerAddress,
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to create script host: %w", err)
	}

	return script.FetchChainInfo(l1Host, script.FetchChainInfoInput{
		SystemConfigProxy:     cfg.SystemConfig,
		L1StandardBridgeProxy: cfg.L1StandardBridge,
	})
}
