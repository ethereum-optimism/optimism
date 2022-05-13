package opnode

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/urfave/cli"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) (*node.Config, error) {
	rollupConfig, err := NewRollupConfig(ctx)
	if err != nil {
		return nil, err
	}

	enableSequencing := ctx.GlobalBool(flags.SequencingEnabledFlag.Name)

	p2pSignerSetup, err := p2p.LoadSignerSetup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p signer: %v", err)
	}

	p2pConfig, err := p2p.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p config: %v", err)
	}

	cfg := &node.Config{
		L1NodeAddr:    ctx.GlobalString(flags.L1NodeAddr.Name),
		L2EngineAddrs: ctx.GlobalStringSlice(flags.L2EngineAddrs.Name),
		L2NodeAddr:    ctx.GlobalString(flags.L2EthNodeAddr.Name),
		L1TrustRPC:    ctx.GlobalBool(flags.L1TrustRPC.Name),
		Rollup:        *rollupConfig,
		Sequencer:     enableSequencing,
		RPC: node.RPCConfig{
			ListenAddr: ctx.GlobalString(flags.RPCListenAddr.Name),
			ListenPort: ctx.GlobalInt(flags.RPCListenPort.Name),
		},
		P2P:       p2pConfig,
		P2PSigner: p2pSignerSetup,
	}
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewRollupConfig(ctx *cli.Context) (*rollup.Config, error) {
	rollupConfigPath := ctx.GlobalString(flags.RollupConfig.Name)
	file, err := os.Open(rollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollup config: %v", err)
	}
	defer file.Close()

	var rollupConfig rollup.Config
	if err := json.NewDecoder(file).Decode(&rollupConfig); err != nil {
		return nil, fmt.Errorf("failed to decode rollup config: %v", err)
	}
	return &rollupConfig, nil
}

// NewLogConfig creates a log config from the provided flags or environment variables.
func NewLogConfig(ctx *cli.Context) (node.LogConfig, error) {
	cfg := node.DefaultLogConfig() // Done to set color based on terminal type
	cfg.Level = ctx.GlobalString(flags.LogLevelFlag.Name)
	cfg.Format = ctx.GlobalString(flags.LogFormatFlag.Name)
	if ctx.IsSet(flags.LogColorFlag.Name) {
		cfg.Color = ctx.GlobalBool(flags.LogColorFlag.Name)
	}

	if err := cfg.Check(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func NewSnapshotLogger(ctx *cli.Context) (log.Logger, error) {
	snapshotFile := ctx.GlobalString(flags.SnapshotLog.Name)
	handler := log.DiscardHandler()
	if snapshotFile != "" {
		var err error
		handler, err = log.FileHandler(snapshotFile, log.JSONFormat())
		if err != nil {
			return nil, err
		}
	}
	handler = log.SyncHandler(handler)
	logger := log.New()
	logger.SetHandler(handler)
	return logger, nil
}
