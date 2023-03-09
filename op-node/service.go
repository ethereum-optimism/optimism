package opnode

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"

	"github.com/urfave/cli"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context, log log.Logger) (*node.Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}

	rollupConfig, err := NewRollupConfig(ctx)
	if err != nil {
		return nil, err
	}

	driverConfig, err := NewDriverConfig(ctx)
	if err != nil {
		return nil, err
	}

	p2pSignerSetup, err := p2pcli.LoadSignerSetup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p signer: %w", err)
	}

	p2pConfig, err := p2pcli.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p config: %w", err)
	}

	l1Endpoint, err := NewL1EndpointConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load l1 endpoint info: %w", err)
	}

	l2Endpoint, err := NewL2EndpointConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 endpoints info: %w", err)
	}

	cfg := &node.Config{
		L1:     l1Endpoint,
		L2:     l2Endpoint,
		Rollup: *rollupConfig,
		Driver: *driverConfig,
		RPC: node.RPCConfig{
			ListenAddr:  ctx.GlobalString(flags.RPCListenAddr.Name),
			ListenPort:  ctx.GlobalInt(flags.RPCListenPort.Name),
			EnableAdmin: ctx.GlobalBool(flags.RPCEnableAdmin.Name),
		},
		Metrics: node.MetricsConfig{
			Enabled:    ctx.GlobalBool(flags.MetricsEnabledFlag.Name),
			ListenAddr: ctx.GlobalString(flags.MetricsAddrFlag.Name),
			ListenPort: ctx.GlobalInt(flags.MetricsPortFlag.Name),
		},
		Pprof: oppprof.CLIConfig{
			Enabled:    ctx.GlobalBool(flags.PprofEnabledFlag.Name),
			ListenAddr: ctx.GlobalString(flags.PprofAddrFlag.Name),
			ListenPort: ctx.GlobalInt(flags.PprofPortFlag.Name),
		},
		P2P:                 p2pConfig,
		P2PSigner:           p2pSignerSetup,
		L1EpochPollInterval: ctx.GlobalDuration(flags.L1EpochPollIntervalFlag.Name),
		Heartbeat: node.HeartbeatConfig{
			Enabled: ctx.GlobalBool(flags.HeartbeatEnabledFlag.Name),
			Moniker: ctx.GlobalString(flags.HeartbeatMonikerFlag.Name),
			URL:     ctx.GlobalString(flags.HeartbeatURLFlag.Name),
		},
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
		L1RPCKind:  sources.RPCProviderKind(strings.ToLower(ctx.GlobalString(flags.L1RPCProviderKind.Name))),
	}, nil
}

func NewL2EndpointConfig(ctx *cli.Context, log log.Logger) (*node.L2EndpointConfig, error) {
	l2Addr := ctx.GlobalString(flags.L2EngineAddr.Name)
	fileName := ctx.GlobalString(flags.L2EngineJWTSecret.Name)
	var secret [32]byte
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("file-name of jwt secret is empty")
	}
	if data, err := os.ReadFile(fileName); err == nil {
		jwtSecret := common.FromHex(strings.TrimSpace(string(data)))
		if len(jwtSecret) != 32 {
			return nil, fmt.Errorf("invalid jwt secret in path %s, not 32 hex-formatted bytes", fileName)
		}
		copy(secret[:], jwtSecret)
	} else {
		log.Warn("Failed to read JWT secret from file, generating a new one now. Configure L2 geth with --authrpc.jwt-secret=" + fmt.Sprintf("%q", fileName))
		if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
			return nil, fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		if err := os.WriteFile(fileName, []byte(hexutil.Encode(secret[:])), 0600); err != nil {
			return nil, err
		}
	}

	return &node.L2EndpointConfig{
		L2EngineAddr:      l2Addr,
		L2EngineJWTSecret: secret,
	}, nil
}

func NewDriverConfig(ctx *cli.Context) (*driver.Config, error) {
	return &driver.Config{
		VerifierConfDepth:  ctx.GlobalUint64(flags.VerifierL1Confs.Name),
		SequencerConfDepth: ctx.GlobalUint64(flags.SequencerL1Confs.Name),
		SequencerEnabled:   ctx.GlobalBool(flags.SequencerEnabledFlag.Name),
		SequencerStopped:   ctx.GlobalBool(flags.SequencerStoppedFlag.Name),
	}, nil
}

func NewRollupConfig(ctx *cli.Context) (*rollup.Config, error) {
	network := ctx.GlobalString(flags.Network.Name)
	if network != "" {
		config, err := chaincfg.GetRollupConfig(network)
		if err != nil {
			return nil, err
		}

		return &config, nil
	}

	rollupConfigPath := ctx.GlobalString(flags.RollupConfig.Name)
	file, err := os.Open(rollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollup config: %w", err)
	}
	defer file.Close()

	var rollupConfig rollup.Config
	if err := json.NewDecoder(file).Decode(&rollupConfig); err != nil {
		return nil, fmt.Errorf("failed to decode rollup config: %w", err)
	}
	return &rollupConfig, nil
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
		handler = log.SyncHandler(handler)
	}
	logger := log.New()
	logger.SetHandler(handler)
	return logger, nil
}
