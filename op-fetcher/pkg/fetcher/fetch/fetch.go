package fetch

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-fetcher/pkg/fetcher/fetch/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

//go:embed forge-artifacts
var forgeArtifacts embed.FS

func FetchChainInfoCLI() func(ctx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		fetcher, err := NewFetcherFromCli(cliCtx)
		if err != nil {
			return err
		}

		result, err := fetcher.FetchChainInfo(cliCtx.Context)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}

		json, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}

		err = os.WriteFile(fetcher.OutputFile, json, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output to file: %w", err)
		}

		fetcher.lgr.Info("completed fetching chain info", "outputFile", fetcher.OutputFile)
		return nil
	}
}

func (f *Fetcher) FetchChainInfo(ctx context.Context) (script.FetchChainInfoOutput, error) {
	f.lgr.Info("initializing fetcher", "systemConfigProxy", f.SystemConfigProxy, "l1StandardBridgeProxy", f.L1StandardBridgeProxy)

	l1RPC, err := rpc.Dial(f.L1RPCURL)
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
		f.lgr,
		deployerAddress,
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return script.FetchChainInfoOutput{}, fmt.Errorf("failed to create script host: %w", err)
	}

	scriptOutput, err := script.FetchChainInfo(l1Host, script.FetchChainInfoInput{
		SystemConfigProxy:     f.SystemConfigProxy,
		L1StandardBridgeProxy: f.L1StandardBridgeProxy,
	})

	return scriptOutput, nil
}
