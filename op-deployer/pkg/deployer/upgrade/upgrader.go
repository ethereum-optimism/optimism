package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type Upgrader interface {
	Upgrade(host *script.Host, input json.RawMessage) error
	ArtifactsURL() string
}

func UpgradeCLI(upgrader Upgrader) func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(cliCtx)
		lgr := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
		oplog.SetGlobalLogHandler(lgr.Handler())

		ctx, cancel := context.WithCancel(cliCtx.Context)
		defer cancel()

		l1RPC := cliCtx.String(deployer.L1RPCURLFlag.Name)
		if l1RPC == "" {
			return fmt.Errorf("missing required flag: %s", deployer.L1RPCURLFlag.Name)
		}

		artifactsURL := upgrader.ArtifactsURL()
		overrideArtifactsURL := cliCtx.String(OverrideArtifactsURLFlag.Name)
		if overrideArtifactsURL != "" {
			artifactsURL = overrideArtifactsURL
		}
		artifactsLocator, err := artifacts.NewLocatorFromURL(artifactsURL)
		if err != nil {
			return fmt.Errorf("failed to parse artifacts URL: %w", err)
		}

		rpcClient, err := rpc.Dial(l1RPC)
		if err != nil {
			return fmt.Errorf("failed to dial RPC %s: %w", l1RPC, err)
		}

		bcaster := new(broadcaster.CalldataBroadcaster)
		depAddr := common.Address{'D'}
		cacheDir := cliCtx.String(deployer.CacheDirFlag.Name)

		artifactsFS, err := artifacts.Download(ctx, artifactsLocator, ioutil.BarProgressor(), cacheDir)
		if err != nil {
			return fmt.Errorf("failed to download L1 artifacts: %w", err)
		}

		host, err := env.DefaultForkedScriptHost(
			ctx,
			bcaster,
			lgr,
			depAddr,
			artifactsFS,
			rpcClient,
		)
		if err != nil {
			return fmt.Errorf("failed to create script host: %w", err)
		}

		configFilePath := cliCtx.String(ConfigFlag.Name)
		if configFilePath == "" {
			return fmt.Errorf("missing required flag: %s", ConfigFlag.Name)
		}
		cfgData, err := os.ReadFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := upgrader.Upgrade(host, cfgData); err != nil {
			return fmt.Errorf("failed to upgrade: %w", err)
		}

		dump, err := bcaster.Dump()
		if err != nil {
			return fmt.Errorf("failed to dump calldata: %w", err)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(dump); err != nil {
			return fmt.Errorf("failed to encode calldata: %w", err)
		}

		return nil
	}
}
