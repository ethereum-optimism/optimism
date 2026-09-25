package clean

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/urfave/cli/v2"
)

func CacheCLI(cliCtx *cli.Context) error {
	logCfg := logcli.ReadCLIConfig(cliCtx)
	l := logcli.NewLogger(logcli.AppOut(cliCtx), logCfg)
	logcli.SetGlobalLogHandler(l.Handler())

	cacheDir := cliCtx.String(deployer.CacheDirFlag.Name)
	if cacheDir == "" {
		return fmt.Errorf("cache directory not set")
	}

	return CleanCache(l, cacheDir)
}

func CleanCache(l log.Logger, cacheDir string) error {
	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		l.Warn("failed to recreate cache directory", "err", err)
	}

	return nil
}
