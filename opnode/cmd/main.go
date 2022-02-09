package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum-optimism/optimistic-specs/opnode/node"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli"
)

var (
	Version = "0.0.0"
	// GitCommit   = ""
	// GitDate     = ""
	VersionMeta = "dev"
)

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func main() {
	// Set up logger with a default INFO level in case we fail to parse flags,
	// otherwise the final critical log won't show what the parsing error was.
	log.Root().SetHandler(
		log.LvlFilterHandler(
			log.LvlInfo,
			log.StreamHandler(os.Stdout, log.TerminalFormat(true)),
		),
	)

	app := cli.NewApp()
	app.Flags = Flags
	app.Version = VersionWithMeta
	app.Name = "opnode"
	app.Usage = "Optimism Rollup Node"
	app.Description = "The deposit only rollup node drives the L2 execution engine based on L1 deposits."

	app.Action = RollupNodeMain
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

func RollupNodeMain(ctx *cli.Context) error {
	log.Info("Initializing Rollup Node")
	cfg, err := NewConfig(ctx)
	if err != nil {
		log.Error("Unable to create the rollup node config", "error", err)
		return err
	}
	n, err := node.New(context.Background(), cfg)
	if err != nil {
		log.Error("Unable to create the rollup node", "error", err)
		return err
	}
	log.Info("Starting rollup node")

	if err := n.Start(); err != nil {
		log.Error("Unable to start L2 Output Submitter", "error", err)
		return err
	}
	defer n.Stop()

	log.Info("Rollup node started")

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel

	return nil

}

// has0xPrefix validates str begins with '0x' or '0X'.
// Copied from geth
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

// HexToHash is copied from Geth, but does not supress the error
func HexToHash(s string) (common.Hash, error) {
	if has0xPrefix(s) {
		s = s[2:]
	}
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return common.Hash{}, fmt.Errorf("Could not decode hex hash: %w", err)
	}
	if len(bytes) != common.HashLength {
		return common.Hash{}, errors.New("Invalid length for Hash")
	}
	return common.BytesToHash(bytes), nil
}

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) (*node.Config, error) {
	L2Hash, err := HexToHash(ctx.GlobalString(GenesisL2Hash.Name))
	if err != nil {
		return nil, fmt.Errorf("Could not decode L2Hash: %w", err)
	}
	L1Hash, err := HexToHash(ctx.GlobalString(GenesisL1Hash.Name))
	if err != nil {
		return nil, fmt.Errorf("Could not decode L1Hash: %w", err)
	}
	logCfg, err := NewLogConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &node.Config{
		/* Required Flags */
		L1NodeAddrs:   ctx.GlobalStringSlice(L1NodeAddrs.Name),
		L2EngineAddrs: ctx.GlobalStringSlice(L2EngineAddrs.Name),
		L2Hash:        L2Hash,
		L1Hash:        L1Hash,
		L1Num:         ctx.GlobalUint64(GenesisL2Hash.Name),
		/* Optional Flags */
		LogCfg: logCfg,
	}, nil
}

// NewLogConfig creates a log config from the provided flags or environment variables.
func NewLogConfig(ctx *cli.Context) (node.LogConfig, error) {
	logCfg := node.DefaultLogConfig() // Done to set color based on terminal type
	logCfg.Level = ctx.GlobalString(LogLevelFlag.Name)
	logCfg.Format = ctx.GlobalString(LogFormatFlag.Name)
	if ctx.IsSet(LogColorFlag.Name) {
		logCfg.Color = ctx.GlobalBool(LogColorFlag.Name)
	}

	if err := logCfg.Check(); err != nil {
		return logCfg, err
	}
	return logCfg, nil
}

// Flags

// Commented out for deadcode lint
// const envVarPrefix = "ROLLUP_NODE_"
// func prefixEnvVar(name string) string {
// 	return envVarPrefix + name
// }

var Flags = []cli.Flag{
	L1NodeAddrs,
	L2EngineAddrs,
	GenesisL2Hash,
	GenesisL1Hash,
	GenesisL1Num,
	LogLevelFlag,
	LogFormatFlag,
	LogColorFlag,
}

var (
	/* Required Flags */
	L1NodeAddrs = cli.StringSliceFlag{
		Name:     "l1",
		Usage:    "Addresses of L1 User JSON-RPC endpoints to use (eth namespace required)",
		Required: true,
	}
	L2EngineAddrs = cli.StringSliceFlag{
		Name:     "l2",
		Usage:    "Addresses of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)",
		Required: true,
	}
	GenesisL2Hash = cli.StringFlag{
		Name:     "genesis.l2-hash",
		Usage:    "Genesis block hash of L2",
		Required: true,
	}
	GenesisL1Hash = cli.StringFlag{
		Name:     "genesis.l1-hash",
		Usage:    "Block hash of L1 after (not incl.) which L1 starts deriving blocks",
		Required: true,
	}
	GenesisL1Num = cli.Uint64Flag{
		Name:     "genesis.l1-num",
		Usage:    "Block number of L1 matching the l1-hash",
		Required: true,
	}

	/* Optional Flags */

	LogLevelFlag = cli.StringFlag{
		Name:  "log.level",
		Usage: "The lowest log level that will be output",
		Value: "info",
	}
	LogFormatFlag = cli.StringFlag{
		Name:  "log.format",
		Usage: "Format the log output. Supported formats: 'text', 'json'",
		Value: "text",
	}
	LogColorFlag = cli.BoolFlag{
		Name:  "log.color",
		Usage: "Color the log output",
	}
)
