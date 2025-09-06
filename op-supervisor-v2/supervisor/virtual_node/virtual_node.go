package virtual_node

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	e2eopnode "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	opNodeConfig "github.com/ethereum-optimism/optimism/op-node/config"
	opNodeFlags "github.com/ethereum-optimism/optimism/op-node/flags"
	p2pcli "github.com/ethereum-optimism/optimism/op-node/p2p/cli"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/urfave/cli/v2"

	// derive sequencer mode from env to avoid struct coupling
	"os"
	"path/filepath"
	"strings"
)

type VirtualNodeConfig struct {
	L1RPC             string
	BeaconAddr        string
	L2AuthRPC         string
	L2UserRPC         string
	JwtSecret         [32]byte
	Rcfg              *rollup.Config
	Interval          time.Duration
	ConfirmDepth      uint64
	UserRPCListenAddr string
	UserRPCPort       int
	DataDir           string
	// Optional: P2P configuration passthroughs
	StaticPeers   []string
	Bootnodes     []string
	PeerstorePath string
	DiscoveryPath string
	DisableP2P    bool
}

// StartVirtualNode starts a virtual op-node in-process with minimal configuration and returns the user-RPC URL
// and a function to stop the virtual op-node. The node is configured as a sequencer with local RPCs and no external P2P.
func StartVirtualNode(
	cfg *VirtualNodeConfig,
	logger log.Logger,
) (string, func(context.Context) error, error) {
	// Minimal P2P config (memory-only, local addresses)
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	if !cfg.DisableP2P {
		for _, f := range opNodeFlags.P2PFlags(opNodeFlags.EnvVarPrefix) {
			_ = f.Apply(fs)
		}
	}
	// Configure P2P if not disabled
	if !cfg.DisableP2P {
		// Prefer a stable P2P identity per chain when a data-dir is provided; otherwise, use a unique temp key.
		keyPath := ""
		if cfg.DataDir != "" && cfg.Rcfg != nil && cfg.Rcfg.L2ChainID != nil {
			base := filepath.Join(cfg.DataDir, "p2p", fmt.Sprintf("%d", cfg.Rcfg.L2ChainID.Uint64()))
			_ = os.MkdirAll(base, 0o755)
			keyPath = filepath.Join(base, "priv.txt")
			_ = fs.Set(opNodeFlags.P2PPrivPathName, keyPath)
		} else {
			// ensure distinct peer IDs across virtual nodes by using a unique p2p privkey path per instance
			tmpDir, _ := os.MkdirTemp("", "sv2-p2p-")
			keyPath = filepath.Join(tmpDir, "p2p_priv.txt")
			_ = fs.Set(opNodeFlags.P2PPrivPathName, keyPath)
		}
		_ = fs.Set(opNodeFlags.AdvertiseIPName, "127.0.0.1")
		_ = fs.Set(opNodeFlags.AdvertiseTCPPortName, "0")
		_ = fs.Set(opNodeFlags.AdvertiseUDPPortName, "0")
		_ = fs.Set(opNodeFlags.ListenIPName, "127.0.0.1")
		_ = fs.Set(opNodeFlags.ListenTCPPortName, "0")
		_ = fs.Set(opNodeFlags.ListenUDPPortName, "0")
		// Configure discovery/peerstore paths: prefer explicit paths, then DataDir defaults, else memory
		peerstorePath := "memory"
		if cfg.PeerstorePath != "" {
			peerstorePath = cfg.PeerstorePath
		} else if cfg.DataDir != "" && cfg.Rcfg != nil && cfg.Rcfg.L2ChainID != nil {
			peerstorePath = filepath.Join(cfg.DataDir, "p2p", fmt.Sprintf("%d", cfg.Rcfg.L2ChainID.Uint64()), "peerstore")
		}
		_ = fs.Set(opNodeFlags.PeerstorePathName, peerstorePath)
		discoveryPath := "memory"
		if cfg.DiscoveryPath != "" {
			discoveryPath = cfg.DiscoveryPath
		} else if cfg.DataDir != "" && cfg.Rcfg != nil && cfg.Rcfg.L2ChainID != nil {
			discoveryPath = filepath.Join(cfg.DataDir, "p2p", fmt.Sprintf("%d", cfg.Rcfg.L2ChainID.Uint64()), "discovery")
		}
		_ = fs.Set(opNodeFlags.DiscoveryPathName, discoveryPath)
		if peerstorePath == "memory" {
			logger.Warn("op-node peerstore is in-memory; use unique sv2_data_dir to persist per-node state")
		}
		if discoveryPath == "memory" {
			logger.Warn("op-node discovery DB is in-memory; use unique sv2_data_dir to persist per-node state")
		}
		logger.Info("configured op-node p2p storage", "key", keyPath, "peerstore", peerstorePath, "discovery", discoveryPath)
		// Bootnodes / static peers: remain isolated unless configured
		if len(cfg.Bootnodes) > 0 {
			_ = fs.Set(opNodeFlags.BootnodesName, strings.Join(cfg.Bootnodes, ","))
		} else {
			_ = fs.Set(opNodeFlags.BootnodesName, "")
		}
		if len(cfg.StaticPeers) > 0 {
			_ = fs.Set(opNodeFlags.StaticPeersName, strings.Join(cfg.StaticPeers, ","))
		}
	} else {
		logger.Info("P2P networking disabled for virtual op-node")
	}
	cliCtx := cli.NewContext(&cli.App{}, fs, nil)
	p2pConfig, _ := p2pcli.NewConfig(cliCtx, cfg.Rcfg.BlockTime)

	// Build op-node config
	enabled := true
	if v := strings.ToLower(os.Getenv("SV2_SEQUENCER_ENABLED")); v != "" {
		if v == "0" || v == "false" || v == "no" || v == "off" {
			enabled = false
		}
	}
	listenAddr := cfg.UserRPCListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	listenPort := cfg.UserRPCPort
	if listenPort < 0 {
		listenPort = 0
	}
	nodeCfg := &opNodeConfig.Config{
		L1: &opNodeConfig.L1EndpointConfig{
			L1NodeAddr: cfg.L1RPC,
			L1TrustRPC: false,
			// Use debug geth RPC kind to expose extra endpoints in tests
			L1RPCKind:        sources.RPCKindDebugGeth,
			RateLimit:        0,
			BatchSize:        20,
			HttpPollInterval: 100 * time.Millisecond,
			MaxConcurrency:   10,
			CacheSize:        0,
		},
		L2: &opNodeConfig.L2EndpointConfig{
			L2EngineAddr:      cfg.L2AuthRPC,
			L2EngineJWTSecret: cfg.JwtSecret,
		},
		Beacon:        &opNodeConfig.L1BeaconEndpointConfig{BeaconAddr: cfg.BeaconAddr},
		Driver:        driver.Config{SequencerEnabled: enabled, SequencerConfDepth: cfg.ConfirmDepth},
		Rollup:        *cfg.Rcfg,
		RPC:           oprpc.CLIConfig{ListenAddr: listenAddr, ListenPort: listenPort, EnableAdmin: true},
		InteropConfig: &interop.Config{},
		P2P:           p2pConfig,
		Sync: nodeSync.Config{
			SyncMode: nodeSync.CLSync,
		},
		ConfigPersistence:               opNodeConfig.DisabledConfigPersistence{},
		Metrics:                         opmetrics.CLIConfig{},
		Pprof:                           oppprof.CLIConfig{},
		SafeDBPath:                      "",
		RollupHalt:                      "",
		AltDA:                           altda.CLIConfig{},
		IgnoreMissingPectraBlobSchedule: false,
		ExperimentalOPStackAPI:          true,
	}

	opNode, err := e2eopnode.NewOpnode(logger, nodeCfg, func(err error) {
		if err != nil {
			logger.Error("virtual op-node error", "err", err)
		}
	})
	if err != nil {
		return "", nil, fmt.Errorf("start virtual op-node: %w", err)
	}

	stopFn := func(ctx context.Context) error { return opNode.Stop(ctx) }
	return opNode.UserRPC().RPC(), stopFn, nil
}
