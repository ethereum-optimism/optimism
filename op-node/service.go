package opnode

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx *cli.Context, log log.Logger) (*node.Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}

	rollupConfig, err := NewRollupConfig(log, ctx)
	if err != nil {
		return nil, err
	}

	if !ctx.Bool(flags.RollupLoadProtocolVersions.Name) {
		log.Info("Not opted in to ProtocolVersions signal loading, disabling ProtocolVersions contract now.")
		rollupConfig.ProtocolVersionsAddress = common.Address{}
	}

	configPersistence := NewConfigPersistence(ctx)

	driverConfig := NewDriverConfig(ctx)

	p2pSignerSetup, err := p2pcli.LoadSignerSetup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p signer: %w", err)
	}

	p2pConfig, err := p2pcli.NewConfig(ctx, rollupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p config: %w", err)
	}

	l1Endpoint := NewL1EndpointConfig(ctx)

	l2Endpoint, err := NewL2EndpointConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 endpoints info: %w", err)
	}

	syncConfig, err := NewSyncConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create the sync config: %w", err)
	}

	haltOption := ctx.String(flags.RollupHalt.Name)
	if haltOption == "none" {
		haltOption = ""
	}

	cfg := &node.Config{
		L1:     l1Endpoint,
		L2:     l2Endpoint,
		Rollup: *rollupConfig,
		Driver: *driverConfig,
		RPC: node.RPCConfig{
			ListenAddr:  ctx.String(flags.RPCListenAddr.Name),
			ListenPort:  ctx.Int(flags.RPCListenPort.Name),
			EnableAdmin: ctx.Bool(flags.RPCEnableAdmin.Name),
		},
		Metrics: node.MetricsConfig{
			Enabled:    ctx.Bool(flags.MetricsEnabledFlag.Name),
			ListenAddr: ctx.String(flags.MetricsAddrFlag.Name),
			ListenPort: ctx.Int(flags.MetricsPortFlag.Name),
		},
		Pprof: oppprof.CLIConfig{
			Enabled:    ctx.Bool(flags.PprofEnabledFlag.Name),
			ListenAddr: ctx.String(flags.PprofAddrFlag.Name),
			ListenPort: ctx.Int(flags.PprofPortFlag.Name),
		},
		P2P:                         p2pConfig,
		P2PSigner:                   p2pSignerSetup,
		L1EpochPollInterval:         ctx.Duration(flags.L1EpochPollIntervalFlag.Name),
		RuntimeConfigReloadInterval: ctx.Duration(flags.RuntimeConfigReloadIntervalFlag.Name),
		Heartbeat: node.HeartbeatConfig{
			Enabled: ctx.Bool(flags.HeartbeatEnabledFlag.Name),
			Moniker: ctx.String(flags.HeartbeatMonikerFlag.Name),
			URL:     ctx.String(flags.HeartbeatURLFlag.Name),
		},
		ConfigPersistence: configPersistence,
		Sync:              *syncConfig,
		RollupHalt:        haltOption,
		RethDBPath:        ctx.String(flags.L1RethDBPath.Name),
	}

	if err := cfg.LoadPersisted(log); err != nil {
		return nil, fmt.Errorf("failed to load driver config: %w", err)
	}

	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewL1EndpointConfig(ctx *cli.Context) *node.L1EndpointConfig {
	return &node.L1EndpointConfig{
		L1NodeAddr:       ctx.String(flags.L1NodeAddr.Name),
		L1TrustRPC:       ctx.Bool(flags.L1TrustRPC.Name),
		L1RPCKind:        sources.RPCProviderKind(strings.ToLower(ctx.String(flags.L1RPCProviderKind.Name))),
		RateLimit:        ctx.Float64(flags.L1RPCRateLimit.Name),
		BatchSize:        ctx.Int(flags.L1RPCMaxBatchSize.Name),
		HttpPollInterval: ctx.Duration(flags.L1HTTPPollInterval.Name),
		MaxConcurrency:   ctx.Int(flags.L1RPCMaxConcurrency.Name),
	}
}

func NewL2EndpointConfig(ctx *cli.Context, log log.Logger) (*node.L2EndpointConfig, error) {
	l2Addr := ctx.String(flags.L2EngineAddr.Name)
	fileName := ctx.String(flags.L2EngineJWTSecret.Name)
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

func NewConfigPersistence(ctx *cli.Context) node.ConfigPersistence {
	stateFile := ctx.String(flags.RPCAdminPersistence.Name)
	if stateFile == "" {
		return node.DisabledConfigPersistence{}
	}
	return node.NewConfigPersistence(stateFile)
}

func NewDriverConfig(ctx *cli.Context) *driver.Config {
	return &driver.Config{
		VerifierConfDepth:   ctx.Uint64(flags.VerifierL1Confs.Name),
		SequencerConfDepth:  ctx.Uint64(flags.SequencerL1Confs.Name),
		SequencerEnabled:    ctx.Bool(flags.SequencerEnabledFlag.Name),
		SequencerStopped:    ctx.Bool(flags.SequencerStoppedFlag.Name),
		SequencerMaxSafeLag: ctx.Uint64(flags.SequencerMaxSafeLagFlag.Name),
	}
}

func NewRollupConfig(log log.Logger, ctx *cli.Context) (*rollup.Config, error) {
	network := ctx.String(flags.Network.Name)
	rollupConfigPath := ctx.String(flags.RollupConfig.Name)
	if ctx.Bool(flags.BetaExtraNetworks.Name) {
		log.Warn("The beta.extra-networks flag is deprecated and can be omitted safely.")
	}
	if network != "" {
		if rollupConfigPath != "" {
			log.Error(`Cannot configure network and rollup-config at the same time.
Startup will proceed to use the network-parameter and ignore the rollup config.
Conflicting configuration is deprecated, and will stop the op-node from starting in the future.
`, "network", network, "rollup_config", rollupConfigPath)
		}
		rollupConfig, err := chaincfg.GetRollupConfig(network)
		if err != nil {
			return nil, err
		}
		applyOverrides(ctx, rollupConfig)
		return rollupConfig, nil
	}

	file, err := os.Open(rollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollup config: %w", err)
	}
	defer file.Close()

	var rollupConfig rollup.Config
	if err := json.NewDecoder(file).Decode(&rollupConfig); err != nil {
		return nil, fmt.Errorf("failed to decode rollup config: %w", err)
	}
	applyOverrides(ctx, &rollupConfig)
	return &rollupConfig, nil
}

func applyOverrides(ctx *cli.Context, rollupConfig *rollup.Config) {
	if ctx.IsSet(flags.CanyonOverrideFlag.Name) {
		canyon := ctx.Uint64(flags.CanyonOverrideFlag.Name)
		rollupConfig.CanyonTime = &canyon
	}
	if ctx.IsSet(flags.DeltaOverrideFlag.Name) {
		delta := ctx.Uint64(flags.DeltaOverrideFlag.Name)
		rollupConfig.DeltaTime = &delta
	}
}

func NewSnapshotLogger(ctx *cli.Context) (log.Logger, error) {
	snapshotFile := ctx.String(flags.SnapshotLog.Name)
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

func NewSyncConfig(ctx *cli.Context, log log.Logger) (*sync.Config, error) {
	if ctx.IsSet(flags.L2EngineSyncEnabled.Name) && ctx.IsSet(flags.SyncModeFlag.Name) {
		return nil, errors.New("cannot set both --l2.engine-sync and --syncmode at the same time.")
	} else if ctx.IsSet(flags.L2EngineSyncEnabled.Name) {
		log.Error("l2.engine-sync is deprecated and will be removed in a future release. Use --syncmode=execution-layer instead.")
	}
	mode, err := sync.StringToMode(ctx.String(flags.SyncModeFlag.Name))
	if err != nil {
		return nil, err
	}
	cfg := &sync.Config{
		SyncMode:           mode,
		SkipSyncStartCheck: ctx.Bool(flags.SkipSyncStartCheck.Name),
	}
	if ctx.Bool(flags.L2EngineSyncEnabled.Name) {
		cfg.SyncMode = sync.ELSync
	}

	return cfg, nil
}
