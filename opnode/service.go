package opnode

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimistic-specs/opnode/flags"
	"github.com/ethereum-optimism/optimistic-specs/opnode/node"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context) (*node.Config, error) {
	rollupConfig, err := NewRollupConfig(ctx)
	if err != nil {
		return nil, err
	}

	enableSequencing := ctx.GlobalBool(flags.SequencingEnabledFlag.Name)

	var batchSubmitterKey *ecdsa.PrivateKey
	if enableSequencing {
		keyFile := ctx.GlobalString(flags.BatchSubmitterKeyFlag.Name)
		if keyFile == "" {
			return nil, errors.New("sequencer mode needs batch-submitter key")
		}
		// TODO we should be using encrypted keystores.
		// Mnemonics are bad because they leak *all* keys when they leak
		// Unencrypted keys from file are bad because they are easy to leak (and we are not checking file permissions)
		batchSubmitterKey, err = crypto.LoadECDSA(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read batch submitter key: %v", err)
		}
	}

	cfg := &node.Config{
		L1NodeAddr:       ctx.GlobalString(flags.L1NodeAddr.Name),
		L2EngineAddrs:    ctx.GlobalStringSlice(flags.L2EngineAddrs.Name),
		L2NodeAddr:       ctx.GlobalString(flags.L2EthNodeAddr.Name),
		L1TrustRPC:       ctx.GlobalBool(flags.L1TrustRPC.Name),
		Rollup:           *rollupConfig,
		Sequencer:        enableSequencing,
		SubmitterPrivKey: batchSubmitterKey,
		RPC: node.RPCConfig{
			ListenAddr: ctx.GlobalString(flags.RPCListenAddr.Name),
			ListenPort: ctx.GlobalInt(flags.RPCListenPort.Name),
		},
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
