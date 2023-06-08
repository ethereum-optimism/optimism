package config

import (
	"fmt"

	log "github.com/ethereum/go-ethereum/log"
	cli "github.com/urfave/cli"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

// LoggerFromCLI creates a [log.Logger] from the
// supplied [cli.Context].
func LoggerFromCLI(ctx *cli.Context) (log.Logger, error) {
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		return nil, fmt.Errorf("log config error: %w", err)
	}
	logger := oplog.NewLogger(logCfg)
	return logger, nil
}
