package supervisorv2

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	sv2 "github.com/ethereum-optimism/optimism/op-supervisor-v2/supervisor"
	vnode "github.com/ethereum-optimism/optimism/op-supervisor-v2/supervisor/virtual_node"
)

// Config captures CLI-derived configuration for the SV2 service lifecycle.
type Config struct {
	HTTPAddr    string
	HTTPPort    int
	ProxyOpNode bool
	DataDir     string
	DisableP2P  bool

	// Multi-chain config file path. When set, overrides single-chain flags.
	ConfigPath string

	// Single-chain bootstrap (Milestone 1): retained for backward compatibility
	L1RPC        string
	BeaconAddr   string
	L2AuthRPC    string
	L2UserRPC    string
	JWTPath      string
	RollupPath   string
	PollInterval time.Duration
	ConfirmDepth uint64

	// Cancel enables the service to request the app to stop
	Cancel context.CancelCauseFunc
}

// NewConfig builds a Config from CLI flags.
func NewConfig(ctx *cli.Context, logger log.Logger) (*Config, error) {
	// Read basic server flags
	cfg := &Config{
		HTTPAddr:     ctx.String("http.addr"),
		HTTPPort:     ctx.Int("http.port"),
		ProxyOpNode:  ctx.Bool("proxy.opnode"),
		DataDir:      ctx.String("sv2.data-dir"),
		DisableP2P:   ctx.Bool("disable.p2p"),
		ConfigPath:   ctx.String("sv2.config"),
		L1RPC:        ctx.String("l1.rpc"),
		BeaconAddr:   ctx.String("beacon.addr"),
		L2AuthRPC:    ctx.String("l2.authrpc"),
		L2UserRPC:    ctx.String("l2.userrpc"),
		JWTPath:      ctx.String("jwt.secret"),
		RollupPath:   ctx.String("rollup.config"),
		PollInterval: ctx.Duration("poll.interval"),
		ConfirmDepth: ctx.Uint64("confirm.depth"),
	}
	// No additional validation here; the constructor enforces required fields for single-chain mode
	_ = logger
	return cfg, nil
}

// New constructs the lifecycle for SV2 using the provided config and logger.
// It mirrors the op-node pattern: the returned lifecycle starts HTTP and the supervisor, and stops them gracefully.
func New(_ context.Context, cfg *Config, logger log.Logger, version string, _ any) (cliapp.Lifecycle, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	// Build a supervisor instance
	sup := sv2.NewSupervisor(logger)
	if cfg.DataDir != "" {
		sup.SetDataDir(cfg.DataDir)
	}
	sup.EnableOpNodeProxy(cfg.ProxyOpNode)

	// Build HTTP server (listener created in Start to capture bound address when using port 0)
	httpAddr := fmt.Sprintf("%s:%d", cfg.HTTPAddr, cfg.HTTPPort)
	httpSrv := &http.Server{Handler: sup.HTTPHandler()}

	// Multi-chain config takes precedence over single-chain bootstrap
	if strings.TrimSpace(cfg.ConfigPath) != "" {
		type chainCfg struct {
			L1RPC             string   `json:"l1_rpc"`
			BeaconAddr        string   `json:"beacon_addr"`
			L2AuthRPC         string   `json:"l2_authrpc"`
			L2UserRPC         string   `json:"l2_userrpc"`
			JWTPath           string   `json:"jwt_secret"`
			RollupPath        string   `json:"rollup_config"`
			UserRPCPort       int      `json:"user_rpc_port"`
			UserRPCListenAddr string   `json:"user_rpc_listen_addr"`
			StaticPeers       []string `json:"p2p_static"`
			Bootnodes         []string `json:"p2p_bootnodes"`
			PeerstorePath     string   `json:"p2p_peerstore_path"`
			DiscoveryPath     string   `json:"p2p_discovery_path"`
		}
		type fileCfg struct {
			HTTPAddr     string     `json:"http_addr"`
			HTTPPort     int        `json:"http_port"`
			ProxyOpNode  *bool      `json:"proxy_opnode"`
			SV2DataDir   string     `json:"sv2_data_dir"`
			ConfirmDepth *uint64    `json:"confirm_depth"`
			PollInterval string     `json:"poll_interval"`
			Chains       []chainCfg `json:"chains"`
		}
		raw, err := os.ReadFile(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("read sv2.config: %w", err)
		}
		var fc fileCfg
		if err := json.Unmarshal(raw, &fc); err != nil {
			return nil, fmt.Errorf("parse sv2.config: %w", err)
		}
		// Override server settings if present in file
		if fc.HTTPAddr != "" {
			cfg.HTTPAddr = fc.HTTPAddr
		}
		if fc.HTTPPort != 0 {
			cfg.HTTPPort = fc.HTTPPort
		}
		if fc.ProxyOpNode != nil {
			cfg.ProxyOpNode = *fc.ProxyOpNode
		}
		if fc.SV2DataDir != "" {
			cfg.DataDir = fc.SV2DataDir
		}
		if fc.ConfirmDepth != nil {
			cfg.ConfirmDepth = *fc.ConfirmDepth
		}
		if strings.TrimSpace(fc.PollInterval) != "" {
			if d, err := time.ParseDuration(fc.PollInterval); err == nil {
				cfg.PollInterval = d
			}
		}
		// Recompute HTTP bind address after applying file overrides
		httpAddr = fmt.Sprintf("%s:%d", cfg.HTTPAddr, cfg.HTTPPort)
		// Add each chain
		for i, c := range fc.Chains {
			if c.L1RPC == "" || c.BeaconAddr == "" || c.L2AuthRPC == "" || c.L2UserRPC == "" || c.JWTPath == "" || c.RollupPath == "" {
				return nil, fmt.Errorf("sv2.config chains[%d]: missing required fields", i)
			}
			// Read JWT
			data, err := os.ReadFile(c.JWTPath)
			if err != nil {
				return nil, fmt.Errorf("chains[%d]: read jwt_secret: %w", i, err)
			}
			s := strings.TrimPrefix(strings.TrimSpace(string(data)), "0x")
			b, err := hex.DecodeString(s)
			if err != nil || len(b) != 32 {
				return nil, fmt.Errorf("chains[%d]: invalid jwt_secret", i)
			}
			var jwt [32]byte
			copy(jwt[:], b)
			// Read rollup config
			cfgBytes, err := os.ReadFile(c.RollupPath)
			if err != nil {
				return nil, fmt.Errorf("chains[%d]: read rollup_config: %w", i, err)
			}
			var rcfg rollup.Config
			if err := json.Unmarshal(cfgBytes, &rcfg); err != nil {
				return nil, fmt.Errorf("chains[%d]: parse rollup_config: %w", i, err)
			}
			vCfg := &vnode.VirtualNodeConfig{
				L1RPC:             c.L1RPC,
				BeaconAddr:        c.BeaconAddr,
				L2AuthRPC:         c.L2AuthRPC,
				L2UserRPC:         c.L2UserRPC,
				JwtSecret:         jwt,
				Rcfg:              &rcfg,
				Interval:          cfg.PollInterval,
				ConfirmDepth:      cfg.ConfirmDepth,
				UserRPCListenAddr: c.UserRPCListenAddr,
				UserRPCPort:       c.UserRPCPort,
				DataDir:           cfg.DataDir,
				StaticPeers:       c.StaticPeers,
				Bootnodes:         c.Bootnodes,
				PeerstorePath:     c.PeerstorePath,
				DiscoveryPath:     c.DiscoveryPath,
				DisableP2P:        cfg.DisableP2P,
			}
			if _, err := sup.AddChain(vCfg); err != nil {
				return nil, fmt.Errorf("chains[%d]: add chain: %w", i, err)
			}
		}
	} else if cfg.L1RPC != "" || cfg.BeaconAddr != "" || cfg.L2AuthRPC != "" || cfg.L2UserRPC != "" || cfg.JWTPath != "" || cfg.RollupPath != "" {
		// If single-chain bootstrap fields are set, add one chain
		if cfg.L1RPC == "" || cfg.BeaconAddr == "" || cfg.L2AuthRPC == "" || cfg.L2UserRPC == "" || cfg.JWTPath == "" || cfg.RollupPath == "" {
			return nil, fmt.Errorf("requires --l1.rpc, --beacon.addr, --l2.authrpc, --l2.userrpc, --jwt.secret, --rollup.config for single-chain bootstrap")
		}
		// Read JWT
		data, err := os.ReadFile(cfg.JWTPath)
		if err != nil {
			return nil, fmt.Errorf("read jwt.secret: %w", err)
		}
		s := string(data)
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "0x")
		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, fmt.Errorf("decode jwt.secret: %w", err)
		}
		if len(b) != 32 {
			return nil, fmt.Errorf("jwt.secret must be 32 bytes, got %d", len(b))
		}
		var jwt [32]byte
		copy(jwt[:], b)
		// Read rollup config JSON
		cfgBytes, err := os.ReadFile(cfg.RollupPath)
		if err != nil {
			return nil, fmt.Errorf("read rollup.config: %w", err)
		}
		var rcfg rollup.Config
		if err := json.Unmarshal(cfgBytes, &rcfg); err != nil {
			return nil, fmt.Errorf("parse rollup.config: %w", err)
		}
		vCfg := &vnode.VirtualNodeConfig{
			L1RPC:        cfg.L1RPC,
			BeaconAddr:   cfg.BeaconAddr,
			L2AuthRPC:    cfg.L2AuthRPC,
			L2UserRPC:    cfg.L2UserRPC,
			JwtSecret:    jwt,
			Rcfg:         &rcfg,
			Interval:     cfg.PollInterval,
			ConfirmDepth: cfg.ConfirmDepth,
			DataDir:      cfg.DataDir,
			DisableP2P:   cfg.DisableP2P,
			// No static/bootnodes in single-chain flags mode unless we later add flags
		}
		if _, err := sup.AddChain(vCfg); err != nil {
			return nil, fmt.Errorf("add chain: %w", err)
		}
	}

	// Return a lifecycle that manages HTTP and underlying supervisor shutdown
	return &sv2Lifecycle{srv: httpSrv, sup: sup, logger: logger, version: version, addr: httpAddr}, nil
}

type sv2Lifecycle struct {
	srv     *http.Server
	sup     *sv2.Supervisor
	logger  log.Logger
	version string
	stopped bool
	addr    string
	ln      net.Listener
}

func (l *sv2Lifecycle) Start(ctx context.Context) error {
	// Create listener (allow port 0)
	ln, err := net.Listen("tcp", l.addr)
	if err != nil {
		return err
	}
	l.ln = ln
	// Start HTTP server in background
	go func() {
		l.logger.Info("starting sv2 http server", "addr", ln.Addr().String(), "version", l.version)
		if err := l.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			l.logger.Error("http server error", "err", err)
		}
	}()
	return nil
}

func (l *sv2Lifecycle) Stop(ctx context.Context) error {
	if l.stopped {
		return nil
	}
	l.stopped = true
	// Gracefully stop HTTP
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = l.srv.Shutdown(stopCtx)
	if l.ln != nil {
		_ = l.ln.Close()
		l.ln = nil
	}
	// Stop supervisor chains
	l.sup.Stop()
	return nil
}

func (l *sv2Lifecycle) Stopped() bool { return l.stopped }

// AddChain registers a chain on a running SV2 lifecycle instance. Intended for tests/harnesses.
// It avoids importing internal cmd or supervisor wiring from callers.
func AddChain(lc cliapp.Lifecycle, vCfg *vnode.VirtualNodeConfig) (uint64, error) {
	s, ok := lc.(*sv2Lifecycle)
	if !ok || s == nil || s.sup == nil {
		return 0, fmt.Errorf("unsupported lifecycle instance")
	}
	return s.sup.AddChain(vCfg)
}

// HTTPAddr returns the bound HTTP address (host:port) for a running lifecycle.
func HTTPAddr(lc cliapp.Lifecycle) (string, bool) {
	s, ok := lc.(*sv2Lifecycle)
	if !ok || s == nil || s.ln == nil {
		return "", false
	}
	return s.ln.Addr().String(), true
}

// RollbackChain requests a rollback of a chain managed by the lifecycle's supervisor.
func RollbackChain(lc cliapp.Lifecycle, chainID uint64, toBlock uint64) error {
	s, ok := lc.(*sv2Lifecycle)
	if !ok || s == nil || s.sup == nil {
		return fmt.Errorf("unsupported lifecycle instance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.sup.RollbackChain(ctx, chainID, toBlock)
}
