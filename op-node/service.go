package opnode

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/urfave/cli"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context, log log.Logger) (*node.Config, error) {
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

	l1Endpoint, err := NewL1EndpointConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load l1 endpoint info: %v", err)
	}

	l2Endpoints, err := NewL2EndpointsConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 endpoints info: %v", err)
	}

	cfg := &node.Config{
		L1:        l1Endpoint,
		L2s:       l2Endpoints,
		Rollup:    *rollupConfig,
		Sequencer: enableSequencing,
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

func NewL1EndpointConfig(ctx *cli.Context) (*node.L1EndpointConfig, error) {
	return &node.L1EndpointConfig{
		L1NodeAddr: ctx.GlobalString(flags.L1NodeAddr.Name),
		L1TrustRPC: ctx.GlobalBool(flags.L1TrustRPC.Name),
	}, nil
}

func NewL2EndpointsConfig(ctx *cli.Context, log log.Logger) (*node.L2EndpointsConfig, error) {
	l2Addrs := ctx.GlobalStringSlice(flags.L2EngineAddrs.Name)
	engineJWTSecrets := ctx.GlobalStringSlice(flags.L2EngineJWTSecret.Name)
	var secrets [][32]byte
	for i, fileName := range engineJWTSecrets {
		fileName = strings.TrimSpace(fileName)
		if fileName == "" {
			return nil, fmt.Errorf("file-name of jwt secret %d is empty", i)
		}
		if data, err := os.ReadFile(fileName); err == nil {
			jwtSecret := common.FromHex(strings.TrimSpace(string(data)))
			if len(jwtSecret) != 32 {
				return nil, fmt.Errorf("invalid jwt secret in path %s, not 32 hex-formatted bytes", fileName)
			}
			var secret [32]byte
			copy(secret[:], jwtSecret)
			secrets = append(secrets, secret)
		} else {
			log.Warn("Failed to read JWT secret from file, generating a new one now. Configure L2 geth with --authrpc.jwt-secret=" + fmt.Sprintf("%q", fileName))
			var secret [32]byte
			if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
				return nil, fmt.Errorf("failed to generate jwt secret: %v", err)
			}
			secrets = append(secrets, secret)
			if err := os.WriteFile(fileName, []byte(hexutil.Encode(secret[:])), 0600); err != nil {
				return nil, err
			}
		}
	}
	return &node.L2EndpointsConfig{
		L2EngineAddrs:      l2Addrs,
		L2EngineJWTSecrets: secrets,
	}, nil
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
