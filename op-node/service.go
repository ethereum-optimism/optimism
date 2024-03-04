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
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	"github.com/ethereum-optimism/optimism/op-node/node"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
)

// NewConfig creates a new configuration for the Optimistic Rollup node using CLI flags and environment variables.
func NewConfig(ctx *cli.Context, log log.Logger) (*node.Config, error) {
	// Check if required flags are set
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}

	// Create rollup config from CLI flags
	rollupConfig, err := NewRollupConfigFromCLI(log, ctx)
	if err != nil {
		return nil, err
	}

	// Disable ProtocolVersions contract if not opted in to ProtocolVersions signal loading
	if !ctx.Bool(flags.RollupLoadProtocolVersions.Name) {
		log.Info("Not opted in to ProtocolVersions signal loading, disabling ProtocolVersions contract now.")
		rollupConfig.ProtocolVersionsAddress = common.Address{}
	}

	// Create a new ConfigPersistence object
	configPersistence := NewConfigPersistence(ctx)

	// Create a new DriverConfig object
	driverConfig := NewDriverConfig(ctx)

	// Load p2p signer setup
	p2pSignerSetup, err := p2pcli.LoadSignerSetup(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p signer: %w", err)
	}

	// Create a new p2p config object
	p2pConfig, err := p2pcli.NewConfig(ctx, rollupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p config: %w", err)
	}

	// Create a new L1 endpoint config object
	l1Endpoint := NewL1EndpointConfig(ctx)

	// Create a new L2 endpoint config object
	l2Endpoint, err := NewL2EndpointConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to load l2 endpoints info: %w", err)
	}

	// Create a new SyncConfig object
	syncConfig, err := NewSyncConfig(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create the sync config: %w", err)
	}

	// Set the halt option for the rollup
	haltOption := ctx.String(flags.RollupHalt.Name)
	if haltOption == "none" {
		haltOption = ""
	}

	// Create a new node.Config object with the created config objects
	cfg := &node.Config{
		L1:     l1Endpoint,
		L2:     l2Endpoint,
		Rollup: *rollupConfig,
		Driver: *driverConfig,
		Beacon: NewBeaconEndpointConfig(ctx),
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
		Pprof:                       oppprof.ReadCLIConfig(ctx),
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
		SafeDBPath:        ctx.String(flags.SafeDBPath.Name),
		Sync:              *syncConfig,
		RollupHalt:        haltOption,
		RethDBPath:        ctx.String(flags.L1RethDBPath.Name),

		ConductorEnabled:    ctx.Bool(flags.ConductorEnabledFlag.Name),
		ConductorRpc:        ctx.String(flags.ConductorRpcFlag.Name),
		ConductorRpcTimeout: ctx.Duration(flags.ConductorRpcTimeoutFlag.Name),

		Plasma: plasma.ReadCLIConfig(ctx),
	}


	// Load persisted config from disk
	if err := cfg.LoadPersisted(log); err != nil {
		return nil, fmt.Errorf("failed to load driver config: %w", err)
	}

	// Set the sequencer stopped flag if the conductor is enabled
	// conductor controls the sequencer state
	if cfg.ConductorEnabled {
		cfg.Driver.SequencerStopped = true
	}

	// Check the config for errors
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// NewBeaconEndpointConfig creates a new L1 beacon endpoint config object from CLI flags.
func NewBeaconEndpointConfig(ctx *cli.Context) node.L1BeaconEndpointSetup {
	return &node.L1BeaconEndpointConfig{
		BeaconAddr:             ctx.String(flags.BeaconAddr.Name),
		BeaconHeader:           ctx.String(flags.BeaconHeader.Name),
		BeaconArchiverAddr:     ctx.String(flags.BeaconArchiverAddr.Name),
		BeaconCheckIgnore:      ctx.Bool(flags.BeaconCheckIgnore.Name),
		BeaconFetchAllSidecars: ctx.Bool(flags.BeaconFetchAllSidecars.Name),
	}
}

// NewL1EndpointConfig creates a new L1 endpoint config object from CLI flags.
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

// NewL2EndpointConfig creates a new L2 endpoint config object from CLI flags.
func NewL2EndpointConfig(ctx *cli.Context, log log.Logger) (*node.L2EndpointConfig, error) {
	// Get the L2 engine address and JWT secret file name from CLI flags
	l2Addr := ctx.String(flags.L2EngineAddr.Name)
	fileName := ctx.String(flags.L2EngineJWTSecret.Name)
	var secret [32]byte  // Create a new secret byte array

	// Trim the file name and check if it's empty
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("file-name of jwt secret is empty")
	}

	// Read the JWT secret from the file
	if data, err := os.ReadFile(fileName); err == nil {
		jwtSecret := common.FromHex(strings.TrimSpace(string(data)))
		if len(jwtSecret) != 32 {
			return nil, fmt.Errorf("invalid jwt secret in path %s, not 32 hex-formatted bytes", fileName)
		}
		copy(secret[:], jwtSecret)
	} else {
		// If the JWT secret file can't be read, generate a new secret and write it to the file
		log.Warn("Failed to read JWT secret from file, generating a new one now. Configure L2 geth with --authrpc.jwt-secret=" + fmt.Sprintf("%q", fileName))
		if _, err := io.ReadFull(rand.Reader, secret[:]); err != nil {
			return nil, fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		if err := os.WriteFile(fileName, []byte(hexutil.Encode(secret[:])), 0o600); err != nil {
			return nil, err
		}
	}

	return &node.L2EndpointConfig{
		L2EngineAddr:      l2Addr,
		L2EngineJWTSecret: secret,
	}, nil
}
// NewConfigPersistence creates a new ConfigPersistence object from CLI flags.
func NewConfigPersistence(ctx *cli.Context) node.ConfigPersistence {
	stateFile := ctx.String(flags.RPCAdminPersistence.Name)
	if stateFile == "" {
		return node.DisabledConfigPersistence{}
	}
	return node.NewConfigPersistence(stateFile)
}

// NewDriverConfig creates a new DriverConfig object from CLI flags.
func NewDriverConfig(ctx *cli.Context) *driver.Config {
	return &driver.Config{
		VerifierConfDepth:   ctx.Uint64(flags.VerifierL1Confs.Name),
		SequencerConfDepth:  ctx.Uint64(flags.SequencerL1Confs.Name),
		SequencerEnabled:    ctx.Bool(flags.SequencerEnabledFlag.Name),
		SequencerStopped:    ctx.Bool(flags.SequencerStoppedFlag.Name),
		SequencerMaxSafeLag: ctx.Uint64(flags.SequencerMaxSafeLagFlag.Name),
	}
}

// NewRollupConfigFromCLI creates a new RollupConfig object from CLI flags.
func NewRollupConfigFromCLI(log log.Logger, ctx *cli.Context) (*rollup.Config, error) {
	// Get the network and rollup config path from CLI flags
	network := ctx.String(opflags.NetworkFlagName)
	rollupConfigPath := ctx.String(opflags.RollupConfigFlagName)

	// Warn about deprecated flags
	if ctx.Bool(flags.BetaExtraNetworks.Name) {
		log.Warn("The beta.extra-networks flag is deprecated and can be omitted safely.")
	}

	// Create a new RollupConfig object with the network and rollup config path
	rollupConfig, err := NewRollupConfig(log, network, rollupConfigPath)
	if err != nil {
		return nil, err
	}

	// Apply overrides to the rollup config
	applyOverrides(ctx, rollupConfig)
	return rollupConfig, nil
}

// NewRollupConfig creates a new RollupConfig object with the specified network and rollup config path.
func NewRollupConfig(log log.Logger, network string, rollupConfigPath string) (*rollup.Config, error) {
	// Check if both network and rollup config path are specified
	if network != "" {
		if rollupConfigPath != "" {
			log.Error(`Cannot configure network and rollup-config at the same time.
Startup will proceed to use the network-parameter and ignore the rollup config.
Conflicting configuration is deprecated, and will stop the op-node from starting in the future.
`, "network", network, "rollup_config", rollupConfigPath)
		}
		// Get the rollup config for the specified network
		rollupConfig, err := chaincfg.GetRollupConfig(network)
		if err != nil {
			return nil, err
		}
		return rollupConfig, nil
	}

	// Read the rollup config from the specified file path
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

// applyOverrides applies overrides to the rollup config from CLI flags.
func applyOverrides(ctx *cli.Context, rollupConfig *rollup.Config) {
	// Set the canyon time override
	if ctx.IsSet(opflags.CanyonOverrideFlagName) {
		canyon := ctx.Uint64(opflags.CanyonOverrideFlagName)
		rollupConfig.CanyonTime = &canyon
	}
	// Set the delta time override
	if ctx.IsSet(opflags.DeltaOverrideFlagName) {
		delta := ctx.Uint64(opflags.DeltaOverrideFlagName)
		rollupConfig.DeltaTime = &delta
	}
	// Set the ecotone time override
	if ctx.IsSet(opflags.EcotoneOverrideFlagName) {
		ecotone := ctx.Uint64(opflags.EcotoneOverrideFlagName)
		rollupConfig.EcotoneTime = &ecotone
	}
}

// NewSnapshotLogger creates a new logger for snapshot logs.
func NewSnapshotLogger(ctx *cli.Context) (log.Logger, error) {
	// Get the snapshot log file name from CLI flags
	snapshotFile := ctx.String(flags.SnapshotLog.Name)
	if snapshotFile == "" {
		// If the snapshot log file name is not specified, return a logger that discards logs
		return log.NewLogger(log.DiscardHandler()), nil
	}

	// Open the snapshot log file for writing
	sf, err := os.OpenFile(snapshotFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	// Create a new logger that writes to the snapshot log file
	handler := log.JSONHandler(sf)
	return log.NewLogger(handler), nil
}

// NewSyncConfig creates a new SyncConfig object from CLI flags.
func NewSyncConfig(ctx *cli.Context, log log.Logger) (*sync.Config, error) {
	// Check if both l2.engine-sync and syncmode flags are set
	if ctx.IsSet(flags.L2EngineSyncEnabled.Name) && ctx.IsSet(flags.SyncModeFlag.Name) {
		return nil, errors.New("cannot set both --l2.engine-sync and --syncmode at the same time.")
	} else if ctx.IsSet(flags.L2EngineSyncEnabled.Name) {
		// Warn about deprecated flag
		log.Error("l2.engine-sync is deprecated and will be removed in a future release. Use --syncmode=execution-layer instead.")
	}
	// Get the sync mode from CLI flags
	mode, err := sync.StringToMode(ctx.String(flags.SyncModeFlag.Name))
	if err != nil {
		return nil, err
	}
	// Create a new SyncConfig object with the specified sync mode
	cfg := &sync.Config{
		SyncMode:           mode,
		SkipSyncStartCheck: ctx.Bool(flags.SkipSyncStartCheck.Name),
	}
	// Set the sync mode to execution-layer if the l2.engine-sync flag is set
	if ctx.Bool(flags.L2EngineSyncEnabled.Name) {
		cfg.SyncMode = sync.ELSync
	}

	return cfg, nil
}
