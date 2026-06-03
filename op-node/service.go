package opnode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/superchain"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/cliiface"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// NewConfig creates a Config from the provided flags or environment variables.
func NewConfig(ctx cliiface.Context, log log.Logger) (*config.Config, error) {
	if err := flags.CheckRequired(ctx); err != nil {
		return nil, err
	}

	rollupConfig, err := NewRollupConfigFromCLI(log, ctx)
	if err != nil {
		return nil, err
	}

	l1ChainConfig, err := NewL1ChainConfig(rollupConfig.L1ChainID, ctx, log)
	if err != nil {
		return nil, err
	}

	depSet, err := NewDependencySetFromCLI(ctx, eth.ChainIDFromBig(rollupConfig.L2ChainID))
	if err != nil {
		return nil, err
	}

	configPersistence := NewConfigPersistence(ctx)

	driverConfig := NewDriverConfig(ctx)

	p2pSignerSetup, err := p2pcli.LoadSignerSetup(ctx, log)
	if err != nil {
		return nil, fmt.Errorf("failed to load p2p signer: %w", err)
	}

	p2pConfig, err := p2pcli.NewConfig(ctx, rollupConfig.BlockTime)
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

	if ctx.IsSet(flags.HeartbeatEnabledFlag.Name) ||
		ctx.IsSet(flags.HeartbeatMonikerFlag.Name) ||
		ctx.IsSet(flags.HeartbeatURLFlag.Name) {
		log.Warn("Heartbeat functionality is not supported anymore, CLI flags will be removed in following release.")
	}
	conductorRPCEndpoint := ctx.String(flags.ConductorRpcFlag.Name)
	cfg := &config.Config{
		L1:                          l1Endpoint,
		L2:                          l2Endpoint,
		L1ChainConfig:               l1ChainConfig,
		Rollup:                      *rollupConfig,
		DependencySet:               depSet,
		Driver:                      *driverConfig,
		Beacon:                      NewBeaconEndpointConfig(ctx),
		RPC:                         rpc.ReadCLIConfig(ctx),
		Metrics:                     opmetrics.ReadCLIConfig(ctx),
		Pprof:                       oppprof.ReadCLIConfig(ctx),
		P2P:                         p2pConfig,
		P2PSigner:                   p2pSignerSetup,
		L1EpochPollInterval:         ctx.Duration(flags.L1EpochPollIntervalFlag.Name),
		RuntimeConfigReloadInterval: ctx.Duration(flags.RuntimeConfigReloadIntervalFlag.Name),
		ConfigPersistence:           configPersistence,
		SafeDBPath:                  ctx.String(flags.SafeDBPath.Name),
		Sync:                        *syncConfig,
		L2FollowSource:              NewL2FollowSourceConfig(ctx),

		ConductorEnabled: ctx.Bool(flags.ConductorEnabledFlag.Name),
		ConductorRpc: func(context.Context) (string, error) {
			return conductorRPCEndpoint, nil
		},
		ConductorRpcTimeout: ctx.Duration(flags.ConductorRpcTimeoutFlag.Name),

		AltDA: altda.ReadCLIConfig(ctx),

		IgnoreMissingPectraBlobSchedule: ctx.Bool(flags.IgnoreMissingPectraBlobSchedule.Name),
		FetchWithdrawalRootFromState:    ctx.Bool(flags.FetchWithdrawalRootFromState.Name),

		ExperimentalOPStackAPI: ctx.Bool(flags.ExperimentalOPStackAPI.Name),
	}

	if err := cfg.LoadPersisted(log); err != nil {
		return nil, fmt.Errorf("failed to load driver config: %w", err)
	}

	// conductor controls the sequencer state
	if cfg.ConductorEnabled {
		cfg.Driver.SequencerStopped = true
	}

	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewBeaconEndpointConfig(ctx cliiface.Context) config.L1BeaconEndpointSetup {
	return &config.L1BeaconEndpointConfig{
		BeaconAddr:                 ctx.String(flags.BeaconAddr.Name),
		BeaconHeader:               ctx.String(flags.BeaconHeader.Name),
		BeaconFallbackAddrs:        ctx.StringSlice(flags.BeaconFallbackAddrs.Name),
		BeaconCheckIgnore:          ctx.Bool(flags.BeaconCheckIgnore.Name),
		BeaconFetchAllSidecars:     ctx.Bool(flags.BeaconFetchAllSidecars.Name),
		BeaconSlotDurationOverride: ctx.Uint64(flags.BeaconSlotDurationOverride.Name),
	}
}

func NewL1EndpointConfig(ctx cliiface.Context) *config.L1EndpointConfig {
	return &config.L1EndpointConfig{
		L1NodeAddr:       ctx.String(flags.L1NodeAddr.Name),
		L1TrustRPC:       ctx.Bool(flags.L1TrustRPC.Name),
		L1RPCKind:        sources.RPCProviderKind(strings.ToLower(ctx.String(flags.L1RPCProviderKind.Name))),
		RateLimit:        ctx.Float64(flags.L1RPCRateLimit.Name),
		BatchSize:        ctx.Int(flags.L1RPCMaxBatchSize.Name),
		HttpPollInterval: ctx.Duration(flags.L1HTTPPollInterval.Name),
		MaxConcurrency:   ctx.Int(flags.L1RPCMaxConcurrency.Name),
		CacheSize:        ctx.Uint(flags.L1CacheSize.Name),
	}
}

func NewL2EndpointConfig(ctx cliiface.Context, logger log.Logger) (*config.L2EndpointConfig, error) {
	l2Addr := ctx.String(flags.L2EngineAddr.Name)
	fileName := ctx.String(flags.L2EngineJWTSecret.Name)
	secret, err := rpc.ObtainJWTSecret(logger, fileName, true)
	if err != nil {
		return nil, err
	}
	l2RpcTimeout := ctx.Duration(flags.L2EngineRpcTimeout.Name)
	return &config.L2EndpointConfig{
		L2EngineAddr:        l2Addr,
		L2EngineJWTSecret:   secret,
		L2EngineCallTimeout: l2RpcTimeout,
	}, nil
}

func NewL2FollowSourceConfig(ctx cliiface.Context) *config.L2FollowSourceConfig {
	l2Addr := ctx.String(flags.L2FollowSource.Name)
	l2RpcTimeout := ctx.Duration(flags.L2FollowSourceRpcTimeout.Name)
	return &config.L2FollowSourceConfig{
		L2RPCAddr:        l2Addr,
		L2RPCCallTimeout: l2RpcTimeout,
	}
}

func NewConfigPersistence(ctx cliiface.Context) config.ConfigPersistence {
	stateFile := ctx.String(flags.RPCAdminPersistence.Name)
	if stateFile == "" {
		return config.DisabledConfigPersistence{}
	}
	return config.NewConfigPersistence(stateFile)
}

func NewDriverConfig(ctx cliiface.Context) *driver.Config {
	cfg := &driver.Config{
		VerifierConfDepth:        ctx.Uint64(flags.VerifierL1Confs.Name),
		SequencerConfDepth:       ctx.Uint64(flags.SequencerL1Confs.Name),
		SequencerEnabled:         ctx.Bool(flags.SequencerEnabledFlag.Name),
		SequencerStopped:         ctx.Bool(flags.SequencerStoppedFlag.Name),
		SequencerMaxSafeLag:      ctx.Uint64(flags.SequencerMaxSafeLagFlag.Name),
		RecoverMode:              ctx.Bool(flags.SequencerRecoverMode.Name),
		SequencerSealingDuration: ctx.Duration(flags.SequencerSealingDurationFlag.Name),
	}

	// Populate finality config from flags. A finality config with null fields
	// is handled the same way as a null finality config.
	cfg.Finalizer = &finality.Config{}
	if ctx.IsSet(flags.FinalityLookbackFlag.Name) {
		lookback := ctx.Uint64(flags.FinalityLookbackFlag.Name)
		cfg.Finalizer.FinalityLookback = &lookback
	}
	if ctx.IsSet(flags.FinalityDelayFlag.Name) {
		delay := ctx.Uint64(flags.FinalityDelayFlag.Name)
		cfg.Finalizer.FinalityDelay = &delay
	}

	return cfg
}

func NewRollupConfigFromCLI(log log.Logger, ctx cliiface.Context) (*rollup.Config, error) {
	network := ctx.String(opflags.NetworkFlagName)
	rollupConfigPath := ctx.String(opflags.RollupConfigFlagName)
	if ctx.Bool(flags.BetaExtraNetworks.Name) {
		log.Warn("The beta.extra-networks flag is deprecated and can be omitted safely.")
	}
	rollupConfig, err := NewRollupConfig(log, network, rollupConfigPath)
	if err != nil {
		return nil, err
	}
	applyOverrides(ctx, rollupConfig)
	return rollupConfig, nil
}

func NewRollupConfig(log log.Logger, network string, rollupConfigPath string) (*rollup.Config, error) {
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
		return rollupConfig, nil
	}

	file, err := os.Open(rollupConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rollup config: %w", err)
	}
	defer file.Close()

	var rollupConfig rollup.Config
	return &rollupConfig, rollupConfig.ParseRollupConfig(file)
}

func applyOverrides(ctx cliiface.Context, rollupConfig *rollup.Config) {
	for _, fork := range opflags.OverridableForks {
		flagName := opflags.OverrideName(fork)
		if ctx.IsSet(flagName) {
			timestamp := ctx.Uint64(flagName)
			rollupConfig.SetActivationTime(fork, &timestamp)
		}
	}
}

func NewL1ChainConfig(chainId *big.Int, ctx cliiface.Context, log log.Logger) (*params.ChainConfig, error) {
	if chainId == nil {
		panic("l1 chain id is nil")
	}

	if cfg := eth.L1ChainConfigByChainID(eth.ChainIDFromBig(chainId)); cfg != nil {
		return cfg, nil
	}

	// if the chain id is not known, we fallback to the CLI config
	cf, err := NewL1ChainConfigFromCLI(log, ctx)
	if err != nil {
		return nil, err
	}
	if cf.ChainID.Cmp(chainId) != 0 {
		return nil, fmt.Errorf("l1 chain config chain ID mismatch: %v != %v", cf.ChainID, chainId)
	}
	if !cf.IsOptimism() && cf.BlobScheduleConfig == nil {
		// No error if the chain config is an OP-Stack chain and doesn't have a blob schedule config
		return nil, fmt.Errorf("L1 chain config does not have a blob schedule config")
	}
	return cf, nil
}

func NewL1ChainConfigFromCLI(log log.Logger, ctx cliiface.Context) (*params.ChainConfig, error) {
	l1ChainConfigPath := ctx.String(flags.L1ChainConfig.Name)
	file, err := os.Open(l1ChainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain spec: %w", err)
	}
	defer file.Close()

	// Attempt to decode directly as a ChainConfig
	var chainConfig params.ChainConfig
	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&chainConfig); err == nil {
		return &chainConfig, nil
	}

	// If that fails, try to load the config from the .config property.
	// This should work if the provided file is a genesis file / chainspec
	return jsonutil.LoadJSONFieldStrict[params.ChainConfig](l1ChainConfigPath, "config")
}

// NewDependencySetFromCLI returns the dep set from --interop.dependency-set if
// set, otherwise from the superchain-registry. An unknown chain yields
// (nil, nil); config.Check then errors iff InteropTime is set.
func NewDependencySetFromCLI(cli cliiface.Context, chainID eth.ChainID) (depset.DependencySet, error) {
	if cli.IsSet(flags.InteropDependencySet.Name) {
		loader := &depset.JSONDependencySetLoader{Path: cli.Path(flags.InteropDependencySet.Name)}
		return loader.LoadDependencySet()
	}
	ds, err := depset.FromRegistry(chainID)
	if err != nil {
		if errors.Is(err, superchain.ErrUnknownChain) {
			return nil, nil
		}
		return nil, fmt.Errorf("load dependency set from superchain-registry: %w", err)
	}
	return ds, nil
}

func NewSyncConfig(ctx cliiface.Context, log log.Logger) (*sync.Config, error) {
	if ctx.IsSet(flags.L2EngineSyncEnabled.Name) && ctx.IsSet(flags.SyncModeFlag.Name) {
		return nil, errors.New("cannot set both --l2.engine-sync and --syncmode at the same time")
	} else if ctx.IsSet(flags.L2EngineSyncEnabled.Name) {
		log.Error("l2.engine-sync is deprecated and will be removed in a future release. Use --syncmode=execution-layer instead.")
	}
	l2FollowSourceEndpoint := ctx.String(flags.L2FollowSource.Name)
	rrSyncEnabled := ctx.Bool(flags.SyncModeReqRespFlag.Name)
	// p2p.sync.req-resp=false && syncmode.req-resp=true is not allowed
	if !ctx.Bool(flags.SyncReqRespName) && rrSyncEnabled {
		return nil, errors.New("cannot set --p2p.sync.req-resp=false and --syncmode.req-resp=true at the same time")
	}
	mode, err := sync.StringToMode(ctx.String(flags.SyncModeFlag.Name))
	if err != nil {
		return nil, err
	}
	engineKind := engine.Kind(ctx.String(flags.L2EngineKind.Name))
	offsetELSafe := ctx.Duration(flags.SyncModeOffsetELSafeFlag.Name)
	cfg := &sync.Config{
		SyncMode:                       mode,
		SyncModeReqResp:                ctx.Bool(flags.SyncModeReqRespFlag.Name),
		SkipSyncStartCheck:             ctx.Bool(flags.SkipSyncStartCheck.Name),
		SupportsPostFinalizationELSync: engineKind.SupportsPostFinalizationELSync(),
		L2FollowSourceEndpoint:         l2FollowSourceEndpoint,
		// Sequencer needs a manual initial reset when follow source
		NeedInitialResetEngine: ctx.Bool(flags.SequencerEnabledFlag.Name) && l2FollowSourceEndpoint != "",
		OffsetELSafe:           offsetELSafe,
	}
	if ctx.Bool(flags.L2EngineSyncEnabled.Name) {
		cfg.SyncMode = sync.ELSync
	}
	if cfg.OffsetELSafe > 0 && cfg.SyncMode != sync.ELSync {
		log.Warn("syncmode.offset-el-safe is ineffective unless --syncmode=execution-layer; ignoring configured value", "syncmode", cfg.SyncMode.String(), "configured_offset", cfg.OffsetELSafe)
		cfg.OffsetELSafe = 0
	}
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return cfg, nil
}
