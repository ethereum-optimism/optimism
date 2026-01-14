package sysgo

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// RollupBoostNode is a lightweight sysgo-managed process wrapper around a rollup-boost
// WebSocket stream source. It exposes a stable proxied ws URL and hydrates the L2
// network with a shared WSClient that points at it.
type RollupBoostNode struct {
	mu sync.Mutex

	id         stack.RollupBoostNodeID
	wsProxyURL string
	wsProxy    *tcpproxy.Proxy

	rpcProxyURL string
	rpcProxy    *tcpproxy.Proxy

	header http.Header

	logger log.Logger
	p      devtest.P

	sub *SubProcess

	cfg *RollupBoostConfig
}

var _ hydrator = (*RollupBoostNode)(nil)
var _ stack.Lifecycle = (*RollupBoostNode)(nil)
var _ L2ELNode = (*RollupBoostNode)(nil)

func (r *RollupBoostNode) hydrate(system stack.ExtensibleSystem) {
	elRPC, err := client.NewRPC(system.T().Ctx(), system.Logger(), r.rpcProxyURL, client.WithLazyDial())
	system.T().Require().NoError(err)
	system.T().Cleanup(elRPC.Close)

	// Create a shared websocket client for flashblocks traffic over the proxy.
	wsClient, err := client.DialWS(system.T().Ctx(), client.WSConfig{
		URL:     r.wsProxyURL,
		Headers: r.header,
		Log:     system.Logger(),
	})
	system.T().Require().NoError(err)

	node := shim.NewRollupBoostNode(shim.RollupBoostNodeConfig{
		ID: r.id,
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       elRPC,
			ChainID:      r.id.ChainID(),
		},
		RollupCfg:         system.L2Network(stack.L2NetworkID(r.id.ChainID())).RollupConfig(),
		FlashblocksClient: wsClient,
	})
	system.L2Network(stack.L2NetworkID(r.id.ChainID())).(stack.ExtensibleL2Network).AddRollupBoostNode(node)
}

func (r *RollupBoostNode) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sub != nil {
		r.logger.Warn("rollup-boost already started")
		return
	}

	cfg := r.cfg
	r.p.Require().NotNil(cfg, "rollup-boost config not initialized")

	if r.wsProxy == nil {
		r.wsProxy = tcpproxy.New(r.p.Logger())
		r.p.Require().NoError(r.wsProxy.Start())
		r.wsProxyURL = "ws://" + r.wsProxy.Addr()
		r.p.Cleanup(func() { r.wsProxy.Close() })
	}

	if r.rpcProxy == nil {
		r.rpcProxy = tcpproxy.New(r.p.Logger())
		r.p.Require().NoError(r.rpcProxy.Start())
		r.rpcProxyURL = "http://" + r.rpcProxy.Addr()
		r.p.Cleanup(func() { r.rpcProxy.Close() })
	}

	args, env := cfg.LaunchSpec(r.p)

	// Create channel for discovering flashblocks WS port from process logs.
	// When using port 0, the OS assigns the port at bind time and the process logs it.
	flashblocksWSChan := make(chan string, 1)
	defer close(flashblocksWSChan)

	// Parse Rust-structured logs and forward into Go logger with attributes
	logOut := logpipe.ToLogger(r.logger.New("stream", "stdout"))
	logErr := logpipe.ToLogger(r.logger.New("stream", "stderr"))

	// Log parsing callback to extract bound addresses from process output
	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
		// Flashblocks WS - custom log message from outbound.rs
		if strings.HasPrefix(msg, "Flashblocks WebSocketPublisher listening on ") {
			addr := strings.TrimPrefix(msg, "Flashblocks WebSocketPublisher listening on ")
			select {
			case flashblocksWSChan <- "ws://" + addr:
			default:
			}
		}
	}

	stdOut := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
	})
	stdErr := logpipe.LogCallback(func(line []byte) {
		logErr(logpipe.ParseRustStructuredLogs(line))
	})

	r.sub = NewSubProcess(r.p, stdOut, stdErr)

	execPath, err := EnsureRustBinary(r.p, RustBinarySpec{
		SrcDir:  "rollup-boost",
		Package: "rollup-boost",
		Binary:  "rollup-boost",
	})
	r.p.Require().NoError(err, "prepare rollup-boost binary")
	r.p.Require().NotEmpty(execPath, "rollup-boost binary path resolved")

	err = r.sub.Start(execPath, args, env)
	r.p.Require().NoError(err, "start rollup-boost")

	// RPC port: still uses pre-allocation because rollup-boost doesn't log the actual
	// bound RPC address when using port 0. This requires a Rust change to fix.
	// TODO: Update rollup-boost to log "RPC server listening on {addr}" and parse it here.
	rpcUpstreamURL := "http://" + cfg.RPCHost + ":" + strconv.Itoa(int(cfg.RPCPort))
	waitTCPReady(r.p, rpcUpstreamURL, 5*time.Second)
	r.logger.Info("rollup-boost upstream RPC ready", "rpc", rpcUpstreamURL)
	r.rpcProxy.SetUpstream(ProxyAddr(r.p.Require(), rpcUpstreamURL))
	waitTCPReady(r.p, r.rpcProxyURL, 10*time.Second)
	r.logger.Info("rollup-boost proxy RPC ready", "proxy_rpc", r.rpcProxyURL)

	// Flashblocks WS: discover port from logs, then configure proxy
	if cfg.EnableFlashblocks {
		var flashblocksAddr string
		r.p.Require().NoError(tasks.Await(r.p.Ctx(), flashblocksWSChan, &flashblocksAddr), "need Flashblocks WS address from logs")
		r.logger.Info("rollup-boost upstream WS ready", "upstream_ws", flashblocksAddr)

		r.wsProxy.SetUpstream(ProxyAddr(r.p.Require(), flashblocksAddr))
		r.logger.Info("rollup-boost proxy WS ready", "proxy_ws", r.wsProxyURL)
	}
}

func (r *RollupBoostNode) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sub == nil {
		r.logger.Warn("rollup-boost already stopped")
		return
	}
	r.p.Require().NoError(r.sub.Stop(true))
	r.sub = nil
}

// WithRollupBoost starts a rollup-boost process using the provided options
// and registers a WSClient on the target L2 chain.
// l2ELID is required to link the proxy to the L2 EL it serves.
func WithRollupBoost(id stack.RollupBoostNodeID, l2ELID stack.L2ELNodeID, opts ...RollupBoostOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		logger := p.Logger()

		// Build config from options and derive sensible defaults
		cfg := DefaultRollupBoostConfig()
		RollupBoostOptionBundle(opts).Apply(orch, id, cfg)
		// Source L2 engine/JWT from the L2 EL object (mandatory)
		if l2EL, ok := orch.l2ELs.Get(l2ELID); ok {
			engineRPC := l2EL.EngineRPC()
			switch {
			case strings.HasPrefix(engineRPC, "ws://"):
				engineRPC = "http://" + strings.TrimPrefix(engineRPC, "ws://")
			case strings.HasPrefix(engineRPC, "wss://"):
				engineRPC = "https://" + strings.TrimPrefix(engineRPC, "wss://")
			}
			cfg.L2EngineURL = engineRPC
			cfg.L2JWTPath = l2EL.JWTPath()
		}
		// Normalize builder URL and fallback JWT will be handled after builder link options are applied below.

		r := &RollupBoostNode{
			id:     id,
			logger: logger,
			p:      p,
			cfg:    cfg,
			header: cfg.Headers,
		}
		// Apply any node-level link options
		for _, opt := range opts {
			if linkOpt, ok := opt.(interface {
				applyNode(p devtest.P, id stack.RollupBoostNodeID, r *RollupBoostNode)
			}); ok {
				linkOpt.applyNode(p, id, r)
			}
		}
		logger.Info("Starting rollup-boost")
		r.Start()
		p.Cleanup(r.Stop)
		// Register for hydration
		orch.rollupBoosts.Set(id, r)
	})
}

// RollupBoostConfig configures the rollup-boost process CLI and environment.
type RollupBoostConfig struct {
	// RPC endpoint for rollup-boost itself
	RPCHost string
	RPCPort uint16

	// Flashblocks proxy WebSocket exposure
	EnableFlashblocks bool
	FlashblocksHost   string
	FlashblocksPort   int

	// L2 engine connection details (HTTP(S))
	L2EngineURL string
	L2JWTPath   string

	// Builder engine connection details (HTTP(S))
	BuilderURL            string
	BuilderJWTPath        string
	FlashblocksBuilderURL string // upstream builder WS url (e.g. op-rbuilder ws)

	// Other settings
	ExecutionMode string // e.g. "enabled"
	LogFormat     string // e.g. "json"

	// Debug server
	DebugHost string
	DebugPort int

	// Optional WS headers to expose to clients through the proxy
	Headers http.Header

	// Env variables for the subprocess
	Env []string
	// ExtraArgs appended to the generated CLI (last-flag-wins semantics)
	ExtraArgs []string
}

func DefaultRollupBoostConfig() *RollupBoostConfig {
	return &RollupBoostConfig{
		RPCHost:               "127.0.0.1",
		RPCPort:               0,
		EnableFlashblocks:     true,
		FlashblocksHost:       "127.0.0.1",
		FlashblocksPort:       0,
		FlashblocksBuilderURL: "",
		L2EngineURL:           "",
		L2JWTPath:             "",
		BuilderURL:            "127.0.0.1:8551", // normalized to http:// later
		BuilderJWTPath:        "",
		ExecutionMode:         "enabled",
		LogFormat:             "json",
		DebugHost:             "127.0.0.1",
		DebugPort:             0,
		Headers:               http.Header{},
		Env:                   nil,
		ExtraArgs:             nil,
	}
}

func (cfg *RollupBoostConfig) LaunchSpec(p devtest.P) (args []string, env []string) {
	p.Require().NotNil(cfg, "nil RollupBoostConfig")

	env = append([]string(nil), cfg.Env...)
	args = make([]string, 0, len(cfg.ExtraArgs)+16)

	if cfg.EnableFlashblocks {
		if cfg.FlashblocksHost == "" {
			cfg.FlashblocksHost = "127.0.0.1"
		}
		args = append(args, "--flashblocks", "--flashblocks-host="+cfg.FlashblocksHost)
		if cfg.FlashblocksPort > 0 {
			// Use explicitly configured port
			args = append(args, "--flashblocks-port="+strconv.Itoa(cfg.FlashblocksPort))
		} else {
			// Use port 0 to let the OS assign a port atomically at bind time.
			// The actual port will be discovered by parsing the process logs.
			args = append(args, "--flashblocks-port=0")
		}
		if cfg.FlashblocksBuilderURL != "" {
			args = append(args, "--flashblocks-builder-url="+cfg.FlashblocksBuilderURL)
		}
	}

	if cfg.RPCPort <= 0 {
		portStr, err := getAvailableLocalPort()
		p.Require().NoError(err, "allocate rollup-boost rpc port")
		portVal, err := strconv.ParseUint(portStr, 10, 16)
		p.Require().NoError(err, "parse rollup-boost rpc port")
		cfg.RPCPort = uint16(portVal)
	}
	p.Require().True(cfg.RPCPort > 0, "RPCPort must be > 0")
	args = append(args, "--rpc-host="+cfg.RPCHost, "--rpc-port="+strconv.Itoa(int(cfg.RPCPort)))

	if cfg.L2EngineURL != "" {
		args = append(args, "--l2-url="+ensureHTTPURL(cfg.L2EngineURL))
	}
	if cfg.L2JWTPath != "" {
		args = append(args, "--l2-jwt-path="+cfg.L2JWTPath)
	}
	if cfg.BuilderURL != "" {
		args = append(args, "--builder-url="+ensureHTTPURL(cfg.BuilderURL))
	}
	if cfg.BuilderJWTPath != "" {
		args = append(args, "--builder-jwt-path="+cfg.BuilderJWTPath)
	}

	if cfg.ExecutionMode != "" {
		args = append(args, "--execution-mode="+cfg.ExecutionMode)
	}
	if cfg.LogFormat != "" {
		args = append(args, "--log-format="+cfg.LogFormat)
	}

	if cfg.DebugHost == "" {
		cfg.DebugHost = "127.0.0.1"
	}
	args = append(args, "--debug-host="+cfg.DebugHost)
	if cfg.DebugPort > 0 {
		// Use explicitly configured port
		args = append(args, "--debug-server-port="+strconv.Itoa(cfg.DebugPort))
	} else {
		// Use port 0 to let the OS assign a port atomically at bind time.
		// The debug server logs its bound address, but we don't need to parse it
		// since the debug port is only used for manual debugging.
		args = append(args, "--debug-server-port=0")
	}

	args = append(args, cfg.ExtraArgs...)

	return args, env
}

type RollupBoostOption interface {
	Apply(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig)
}

type RollupBoostOptionFn func(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig)

var _ RollupBoostOption = RollupBoostOptionFn(nil)

func (fn RollupBoostOptionFn) Apply(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
	fn(orch, id, cfg)
}

type RollupBoostOptionBundle []RollupBoostOption

var _ RollupBoostOption = RollupBoostOptionBundle(nil)

func (b RollupBoostOptionBundle) Apply(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
	for _, opt := range b {
		orch.P().Require().NotNil(opt, "cannot Apply nil RollupBoostOption")
		opt.Apply(orch, id, cfg)
	}
}

// Convenience options
func RollupBoostWithExecutionMode(mode string) RollupBoostOption {
	return RollupBoostOptionFn(func(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
		cfg.ExecutionMode = mode
	})
}

func RollupBoostWithEnv(env ...string) RollupBoostOption {
	return RollupBoostOptionFn(func(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
		cfg.Env = append(cfg.Env, env...)
	})
}

func RollupBoostWithExtraArgs(args ...string) RollupBoostOption {
	return RollupBoostOptionFn(func(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
		cfg.ExtraArgs = append(cfg.ExtraArgs, args...)
	})
}

func RollupBoostWithBuilderNode(id stack.OPRBuilderNodeID) RollupBoostOption {
	return RollupBoostOptionFn(func(orch *Orchestrator, rbID stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
		builderNode, ok := orch.oprbuilderNodes.Get(id)
		if !ok {
			orch.P().Require().FailNow("builder node not found")
		}
		cfg.BuilderURL = ensureHTTPURL(builderNode.authProxyURL)
		cfg.BuilderJWTPath = builderNode.cfg.AuthRPCJWTPath
		cfg.FlashblocksBuilderURL = builderNode.wsProxyURL
	})
}

func RollupBoostWithFlashblocksDisabled() RollupBoostOption {
	return RollupBoostOptionFn(func(orch *Orchestrator, id stack.RollupBoostNodeID, cfg *RollupBoostConfig) {
		cfg.EnableFlashblocks = false
	})
}

func ensureHTTPURL(u string) string {
	if strings.Contains(u, "://") {
		return u
	}
	return "http://" + u
}

func (r *RollupBoostNode) EngineRPC() string {
	return r.rpcProxyURL
}

func (r *RollupBoostNode) JWTPath() string {
	return r.cfg.L2JWTPath
}

func (r *RollupBoostNode) UserRPC() string {
	return r.rpcProxyURL
}
