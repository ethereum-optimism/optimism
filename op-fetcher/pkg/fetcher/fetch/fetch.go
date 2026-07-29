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
		outputFile := cliCtx.String(OutputFileFlag.Name)
		systemConfigProxy := common.HexToAddress(cliCtx.String(SystemConfigProxyFlag.Name))
		l1StandardBridge := common.HexToAddress(cliCtx.String(L1StandardBridgeProxyFlag.Name))
		l1RPCURL := cliCtx.String(L1RPCURLFlag.Name)

		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)

		fetcher, err := NewFetcher(lgr, l1RPCURL, systemConfigProxy, l1StandardBridge)
		if err != nil {
			return err
		}

		result, err := fetcher.FetchChainInfo(cliCtx.Context)
		if err != nil {
			return fmt.Errorf("failed to validate: %w", err)
		}

		fileData := script.CreateChainConfig(result)
		if outputFile == "" {
			// Write to stdout when no output file is specified
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(fileData); err != nil {
				return fmt.Errorf("failed to write output to stdout: %w", err)
			}
		} else {
			// Write to the specified file
			jsonData, err := json.MarshalIndent(fileData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal output: %w", err)
			}

			err = os.WriteFile(outputFile, jsonData, 0o644)
			if err != nil {
				return fmt.Errorf("failed to write output to file: %w", err)
			}

			fetcher.lgr.Info("completed fetching chain info", "outputFile", outputFile)
		}
		return nil
	}
}

type Fetcher struct {
	L1RPCURL              string
	SystemConfigProxy     common.Address
	L1StandardBridgeProxy common.Address
	lgr                   log.Logger
}

func NewFetcher(lgr log.Logger, l1RPCURL string, systemConfigProxy, l1StandardBridge common.Address) (*Fetcher, error) {
	return &Fetcher{
		L1RPCURL:              l1RPCURL,
		SystemConfigProxy:     systemConfigProxy,
		L1StandardBridgeProxy: l1StandardBridge,
		lgr:                   lgr,
	}, nil
}

func (f *Fetcher) FetchChainInfo(ctx context.Context) (script.FetchChainInfoOutput, error) {
	f.lgr.Info("initializing fetcher", "systemConfigProxy", f.SystemConfigProxy)

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

	return script.FetchChainInfo(l1Host, script.FetchChainInfoInput{
		SystemConfigProxy:     f.SystemConfigProxy,
		L1StandardBridgeProxy: f.L1StandardBridgeProxy,
	})
}
