package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/script"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

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

		json, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
		err = os.WriteFile("chain-info.json", json, 0644)
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

	l1ContractsLocator := "file:///Users/samuel/repos/op-labs/optimism/packages/contracts-bedrock/forge-artifacts"
	locator, err := artifacts.NewLocatorFromURL(l1ContractsLocator)
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to parse l1 contracts release locator: %w", err)
	}
	artifactsFS, err := artifacts.Download(ctx, locator, nil, deployer.GetDefaultCacheDir())
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to download artifacts: %w", err)
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
