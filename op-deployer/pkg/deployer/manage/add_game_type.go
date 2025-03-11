package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type AddGameTypeConfig struct {
	L1RPCUrl         string
	Logger           log.Logger
	ArtifactsLocator *artifacts.Locator
	Input            opcm.AddGameTypeInput
	CacheDir         string
}

func (c *AddGameTypeConfig) Check() error {
	if c.L1RPCUrl == "" {
		return fmt.Errorf("l1RPCUrl must be specified")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger must be specified")
	}

	if c.ArtifactsLocator == nil {
		return fmt.Errorf("artifacts locator must be specified")
	}

	return nil
}

func AddGameTypeCLI(cliCtx *cli.Context) error {
	logCfg := oplog.ReadCLIConfig(cliCtx)
	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l1RPCUrl := cliCtx.String(deployer.L1RPCURLFlagName)
	configFile := cliCtx.String(ConfigFlag.Name)
	artifactsLocatorStr := cliCtx.String(deployer.ArtifactsLocatorFlag.Name)
	cacheDir := cliCtx.String(deployer.CacheDirFlag.Name)

	artifactsLocator := new(artifacts.Locator)
	if err := artifactsLocator.UnmarshalText([]byte(artifactsLocatorStr)); err != nil {
		return fmt.Errorf("failed to parse artifacts locator: %w", err)
	}

	// Read the input configuration from file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var input opcm.AddGameTypeInput
	if err := json.Unmarshal(configData, &input); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	ctx := ctxinterrupt.WithCancelOnInterrupt(cliCtx.Context)

	_, calldata, err := AddGameType(ctx, AddGameTypeConfig{
		L1RPCUrl:         l1RPCUrl,
		Logger:           l,
		ArtifactsLocator: artifactsLocator,
		Input:            input,
		CacheDir:         cacheDir,
	})
	if err != nil {
		return fmt.Errorf("failed to add game type: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(calldata); err != nil {
		return fmt.Errorf("failed to encode calldata: %w", err)
	}

	return nil
}

func AddGameType(ctx context.Context, cfg AddGameTypeConfig) (opcm.AddGameTypeOutput, []broadcaster.CalldataDump, error) {
	var output opcm.AddGameTypeOutput
	if err := cfg.Check(); err != nil {
		return output, nil, fmt.Errorf("invalid config for AddGameType: %w", err)
	}

	lgr := cfg.Logger

	artifactsFS, err := artifacts.Download(ctx, cfg.ArtifactsLocator, artifacts.BarProgressor(), cfg.CacheDir)
	if err != nil {
		return output, nil, fmt.Errorf("failed to download artifacts: %w", err)
	}

	bcaster := new(broadcaster.CalldataBroadcaster)

	l1RPC, err := rpc.Dial(cfg.L1RPCUrl)
	if err != nil {
		return output, nil, fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	l1Host, err := env.DefaultForkedScriptHost(
		ctx,
		bcaster,
		lgr,
		common.Address{'D'},
		artifactsFS,
		l1RPC,
	)
	if err != nil {
		return output, nil, fmt.Errorf("failed to create script host: %w", err)
	}

	script, err := opcm.NewAddGameTypeScript(l1Host)
	if err != nil {
		return output, nil, fmt.Errorf("failed to create L2 genesis script: %w", err)
	}

	output, err = script.Run(cfg.Input)
	if err != nil {
		return output, nil, fmt.Errorf("error adding game type: %w", err)
	}

	// Get the calldata
	calldata, err := bcaster.Dump()
	if err != nil {
		return output, nil, fmt.Errorf("failed to get calldata: %w", err)
	}

	return output, calldata, nil
}
