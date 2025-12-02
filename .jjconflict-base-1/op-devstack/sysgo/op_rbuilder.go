// Moved from fb_OPRbuilderNode_real.go
package sysgo

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type OPRBuilderNode struct {
	mu sync.Mutex

	id        stack.OPRBuilderNodeID
	rollupCfg *rollup.Config

	wsProxyURL string
	wsProxy    *tcpproxy.Proxy

	rpcProxyURL string
	rpcProxy    *tcpproxy.Proxy

	authProxyURL string
	authProxy    *tcpproxy.Proxy

	logger log.Logger
	p      devtest.P

	sub *SubProcess
	cfg *OPRBuilderNodeConfig //nolint:unused,structcheck // configuration retained for restarts and JWT lookups
}

var _ hydrator = (*OPRBuilderNode)(nil)
var _ stack.Lifecycle = (*OPRBuilderNode)(nil)
var _ L2ELNode = (*OPRBuilderNode)(nil)

// OPRBuilderNodeConfig contains configuration used to generate the op-OPRbuilderNode CLI.
// Callers can modify the defaults via OPRbuilderNodeOption functions.
type OPRBuilderNodeConfig struct {
	// Chain selector (defaults to "dev" to avoid mainnet imports during tests)
	Chain string

	// DataDir for op-OPRbuilderNode. If empty, a temp dir is created and cleaned up.
	DataDir string

	// Logging formats
	LogStdoutFormat string // e.g. "json"
	LogFileFormat   string // e.g. "json"

	// Flashblocks websocket bind address (host)
	FlashblocksAddr string
	// Flashblocks websocket port. 0 means auto-allocate an available local port.
	FlashblocksPort int
	// EnableFlashblocks enables the flashblocks feature.
	EnableFlashblocks bool

	// --http
	EnableRPC  bool
	RPCAPI     string
	RPCAddr    string
	RPCPort    int
	RPCJWTPath string

	AuthRPCJWTPath string
	AuthRPCAddr    string
	AuthRPCPort    int

	// P2P
	P2PPort       int
	P2PAddr       string
	P2PNodeKeyHex string
	StaticPeers   []string
	TrustedPeers  []string

	// Misc process toggles
	WithUnusedPorts  bool // choose unused ports for subsystems
	DisableDiscovery bool // avoid discv5 UDP socket collisions

	Full bool

	// ExtraArgs are appended to the generated CLI allowing callers to override defaults
	// if the binary respects "last flag wins".
	ExtraArgs []string
	// Env is passed to the subprocess environment.
	Env []string
}

func DefaultOPRbuilderNodeConfig() *OPRBuilderNodeConfig {
	return &OPRBuilderNodeConfig{
		EnableFlashblocks: true,
		FlashblocksAddr:   "127.0.0.1",
		FlashblocksPort:   0,
		EnableRPC:         true,
		RPCAPI:            "admin,web3,debug,eth,txpool,net,miner",
		RPCAddr:           "127.0.0.1",
		RPCPort:           0,
		RPCJWTPath:        "",
		AuthRPCAddr:       "127.0.0.1",
		AuthRPCPort:       0,
		AuthRPCJWTPath:    "",
		P2PAddr:           "127.0.0.1",
		P2PPort:           0,
		P2PNodeKeyHex:     "",
		StaticPeers:       nil,
		TrustedPeers:      nil,
		Full:              true,
		LogStdoutFormat:   "json",
		LogFileFormat:     "json",
		Chain:             "dev",
		WithUnusedPorts:   false,
		DisableDiscovery:  true,
		DataDir:           "",
		ExtraArgs:         nil,
		Env:               nil,
	}
}

func (cfg *OPRBuilderNodeConfig) LaunchSpec(p devtest.P) (args []string, env []string) {
	p.Require().NotNil(cfg, "nil OPRbuilderNodeConfig")

	env = append([]string(nil), cfg.Env...)
	args = make([]string, 0, len(cfg.ExtraArgs)+8)

	args = append(args, "node")

	if cfg.EnableFlashblocks {
		if cfg.FlashblocksAddr == "" {
			cfg.FlashblocksAddr = "127.0.0.1"
		}
		args = append(args, "--flashblocks.enabled")
		args = append(args, "--flashblocks.addr="+cfg.FlashblocksAddr)
		if cfg.FlashblocksPort > 0 {
			// Use explicitly configured port
			args = append(args, "--flashblocks.port="+strconv.Itoa(cfg.FlashblocksPort))
		} else {
			// Use port 0 to let the OS assign a port atomically at bind time.
			// The actual port will be discovered by parsing the process logs.
			args = append(args, "--flashblocks.port=0")
		}
	}

	// P2P configuration: enforce deterministic identity and static peering to the sequencer EL.
	if cfg.P2PNodeKeyHex != "" {
		key := strings.TrimPrefix(cfg.P2PNodeKeyHex, "0x")
		_, err := hex.DecodeString(key)
		p.Require().NoError(err, "decode p2p node key")
		keyPath := filepath.Join(p.TempDir(), "oprbuilder-nodekey")
		p.Require().NoError(os.WriteFile(keyPath, []byte(key), 0o600), "write p2p node key")
		args = append(args, "--p2p-secret-key", keyPath)
	}
	if cfg.P2PAddr != "" {
		args = append(args, "--addr", cfg.P2PAddr)
	}
	if len(cfg.StaticPeers) > 0 {
		args = append(args, "--bootnodes", strings.Join(cfg.StaticPeers, ","))
	}
	if len(cfg.TrustedPeers) > 0 {
		args = append(args, "--trusted-peers", strings.Join(cfg.TrustedPeers, ","))
	}

	if cfg.EnableRPC {
		args = append(args, "--http")
		args = append(args, "--http.addr="+cfg.RPCAddr)
		if cfg.RPCPort > 0 {
			// Use explicitly configured port
			args = append(args, "--http.port="+strconv.Itoa(cfg.RPCPort))
		} else {
			// Use port 0 to let the OS assign a port atomically at bind time.
			// The actual port will be discovered by parsing the process logs.
			args = append(args, "--http.port=0")
		}
		args = append(args, "--http.api="+cfg.RPCAPI)
	}

	if cfg.AuthRPCAddr != "" {
		args = append(args, "--authrpc.addr="+cfg.AuthRPCAddr)
	}
	if cfg.AuthRPCPort > 0 {
		// Use explicitly configured port
		args = append(args, "--authrpc.port="+strconv.Itoa(cfg.AuthRPCPort))
	} else {
		// Use port 0 to let the OS assign a port atomically at bind time.
		// The actual port will be discovered by parsing the process logs.
		args = append(args, "--authrpc.port=0")
	}
	if cfg.AuthRPCJWTPath != "" {
		args = append(args, "--authrpc.jwtsecret="+cfg.AuthRPCJWTPath)
	}

	if cfg.Full {
		args = append(args, "--full")
	}

	if cfg.LogStdoutFormat != "" {
		args = append(args, "--log.stdout.format="+cfg.LogStdoutFormat)
	}
	if cfg.LogFileFormat != "" {
		args = append(args, "--log.file.format="+cfg.LogFileFormat)
	}
	if cfg.Chain != "" {
		args = append(args, "--chain="+cfg.Chain)
	}
	if cfg.DisableDiscovery {
		args = append(args, "--disable-discovery")
	}

	if cfg.P2PPort > 0 {
		// Use explicitly configured P2P port
		args = append(args, "--port="+strconv.Itoa(cfg.P2PPort))
	} else {
		// Use --with-unused-ports to let reth assign P2P port atomically at bind time.
		args = append(args, "--with-unused-ports")
	}

	if cfg.DataDir == "" {
		tmpDir, err := os.MkdirTemp("", "op-OPRBuilderNode-datadir-*")
		p.Require().NoError(err, "create temp datadir for op-OPRBuilderNode")
		args = append(args, "--datadir="+tmpDir)
		p.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	} else {
		args = append(args, "--datadir="+cfg.DataDir)
	}

	args = append(args, cfg.ExtraArgs...)

	return args, env
}

type OPRBuilderNodeOption interface {
	Apply(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig)
}

type OPRBuilderNodeOptionFn func(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig)

var _ OPRBuilderNodeOption = OPRBuilderNodeOptionFn(nil)

func (fn OPRBuilderNodeOptionFn) Apply(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
	fn(p, id, cfg)
}

// OPRBuilderNodeOptionBundle applies multiple OPRBuilderNodeOptions in order.
type OPRBuilderNodeOptionBundle []OPRBuilderNodeOption

var _ OPRBuilderNodeOption = OPRBuilderNodeOptionBundle(nil)

func (b OPRBuilderNodeOptionBundle) Apply(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
	for _, opt := range b {
		p.Require().NotNil(opt, "cannot Apply nil OPRBuilderNodeOption")
		opt.Apply(p, id, cfg)
	}
}

// OPRBuilderWithP2PConfig sets deterministic P2P identity and static peers for the builder EL.
func OPRBuilderWithP2PConfig(addr string, port int, nodeKeyHex string, staticPeers, trustedPeers []string) OPRBuilderNodeOption {
	return OPRBuilderNodeOptionFn(func(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
		cfg.P2PAddr = addr
		cfg.P2PPort = port
		cfg.P2PNodeKeyHex = nodeKeyHex
		cfg.StaticPeers = staticPeers
		cfg.TrustedPeers = trustedPeers
	})
}

// OPRBuilderWithNodeIdentity applies an ELNodeIdentity directly to the builder EL.
func OPRBuilderWithNodeIdentity(identity *ELNodeIdentity, addr string, staticPeers, trustedPeers []string) OPRBuilderNodeOption {
	return OPRBuilderNodeOptionFn(func(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
		cfg.P2PAddr = addr
		cfg.P2PPort = identity.Port
		cfg.P2PNodeKeyHex = identity.KeyHex()
		cfg.StaticPeers = staticPeers
		cfg.TrustedPeers = trustedPeers
	})
}

func OPRBuilderNodeWithExtraArgs(args ...string) OPRBuilderNodeOption {
	return OPRBuilderNodeOptionFn(func(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
		cfg.ExtraArgs = append(cfg.ExtraArgs, args...)
	})
}

func OPRBuilderNodeWithEnv(env ...string) OPRBuilderNodeOption {
	return OPRBuilderNodeOptionFn(func(p devtest.P, id stack.OPRBuilderNodeID, cfg *OPRBuilderNodeConfig) {
		cfg.Env = append(cfg.Env, env...)
	})
}

func (b *OPRBuilderNode) hydrate(system stack.ExtensibleSystem) {
	elRPC, err := client.NewRPC(system.T().Ctx(), system.Logger(), b.rpcProxyURL, client.WithLazyDial())
	system.T().Require().NoError(err)
	system.T().Cleanup(elRPC.Close)

	// Create a shared websocket client for flashblocks traffic over the proxy.
	wsClient, err := client.DialWS(system.T().Ctx(), client.WSConfig{
		URL: b.wsProxyURL,
		Log: system.Logger(),
	})
	system.T().Require().NoError(err)

	node := shim.NewOPRBuilderNode(shim.OPRBuilderNodeConfig{
		ID: b.id,
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       elRPC,
			ChainID:      b.id.ChainID(),
		},
		RollupCfg:         b.rollupCfg,
		FlashblocksClient: wsClient,
	})
	system.L2Network(stack.L2NetworkID(b.id.ChainID())).(stack.ExtensibleL2Network).AddOPRBuilderNode(node)
}

func (b *OPRBuilderNode) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub != nil {
		b.logger.Warn("OPRbuilderNode already started")
		return
	}
	cfg := b.cfg
	b.p.Require().NotNil(cfg, "OPRbuilderNode config not initialized")

	if b.wsProxy == nil {
		b.wsProxy = tcpproxy.New(b.p.Logger())
		b.p.Require().NoError(b.wsProxy.Start())
		b.wsProxyURL = "ws://" + b.wsProxy.Addr()
		b.p.Cleanup(func() { b.wsProxy.Close() })
	}

	if b.rpcProxy == nil {
		b.rpcProxy = tcpproxy.New(b.p.Logger())
		b.p.Require().NoError(b.rpcProxy.Start())
		b.rpcProxyURL = "http://" + b.rpcProxy.Addr()
		b.p.Cleanup(func() { b.rpcProxy.Close() })
	}

	if cfg.EnableRPC && b.authProxy == nil {
		b.authProxy = tcpproxy.New(b.p.Logger())
		b.p.Require().NoError(b.authProxy.Start())
		b.authProxyURL = "http://" + b.authProxy.Addr()
		b.p.Cleanup(func() { b.authProxy.Close() })
	}

	args, env := cfg.LaunchSpec(b.p)

	// Create channels for discovering ports from process logs.
	// When using port 0, the OS assigns ports at bind time and the process logs them.
	flashblocksWSChan := make(chan string, 1)
	httpRPCChan := make(chan string, 1)
	authRPCChan := make(chan string, 1)
	defer close(flashblocksWSChan)
	defer close(httpRPCChan)
	defer close(authRPCChan)

	// Forward structured logs to Go logger and parse for port discovery
	logOut := logpipe.ToLogger(b.logger.New("component", "op-OPRbuilderNode", "src", "stdout"))
	logErr := logpipe.ToLogger(b.logger.New("component", "op-OPRbuilderNode", "src", "stderr"))

	// Log parsing callback to extract bound addresses from process output
	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
		// Flashblocks WS - custom log message from wspub.rs
		if strings.HasPrefix(msg, "Flashblocks WebSocketPublisher listening on ") {
			addr := strings.TrimPrefix(msg, "Flashblocks WebSocketPublisher listening on ")
			if validURL := parseAndValidateAddr(addr, "ws"); validURL != "" {
				select {
				case flashblocksWSChan <- validURL:
				default:
				}
			}
		}
		// HTTP RPC - standard reth log message
		if msg == "RPC HTTP server started" {
			if addr, ok := e.FieldValue("url").(string); ok {
				if validURL := parseAndValidateAddr(addr, "http"); validURL != "" {
					select {
					case httpRPCChan <- validURL:
					default:
					}
				}
			}
		}
		// Auth RPC - standard reth log message
		if msg == "RPC auth server started" {
			if addr, ok := e.FieldValue("url").(string); ok {
				if validURL := parseAndValidateAddr(addr, "http"); validURL != "" {
					select {
					case authRPCChan <- validURL:
					default:
					}
				}
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

	b.sub = NewSubProcess(b.p, stdOut, stdErr)

	execPath, err := EnsureRustBinary(b.p, RustBinarySpec{
		SrcDir:  "op-rbuilder",
		Package: "op-rbuilder",
		Binary:  "op-rbuilder",
	})
	b.p.Require().NoError(err, "prepare op-rbuilder binary")
	b.p.Require().NotEmpty(execPath, "op-rbuilder binary path resolved")

	err = b.sub.Start(execPath, args, env)
	b.p.Require().NoError(err, "start OPRBuilderNode")

	// Wait for ports to be discovered from logs, then configure proxies
	if cfg.EnableRPC {
		var httpRPCAddr string
		b.p.Require().NoError(tasks.Await(b.p.Ctx(), httpRPCChan, &httpRPCAddr), "need HTTP RPC address from logs")
		b.logger.Info("OPRBuilderNode upstream RPC ready", "rpc", httpRPCAddr)
		b.rpcProxy.SetUpstream(ProxyAddr(b.p.Require(), httpRPCAddr))
		b.logger.Info("OPRBuilderNode proxy RPC ready", "proxy_rpc", b.rpcProxyURL)

		var authRPCAddr string
		b.p.Require().NoError(tasks.Await(b.p.Ctx(), authRPCChan, &authRPCAddr), "need Auth RPC address from logs")
		b.logger.Info("OPRBuilderNode upstream auth RPC ready", "auth_rpc", authRPCAddr)
		b.authProxy.SetUpstream(ProxyAddr(b.p.Require(), authRPCAddr))
		b.logger.Info("OPRBuilderNode proxy auth RPC ready", "proxy_auth_rpc", b.authProxyURL)
	}

	if cfg.EnableFlashblocks {
		var flashblocksAddr string
		b.p.Require().NoError(tasks.Await(b.p.Ctx(), flashblocksWSChan, &flashblocksAddr), "need Flashblocks WS address from logs")
		b.logger.Info("OPRBuilderNode upstream WS ready", "ws", flashblocksAddr)
		b.wsProxy.SetUpstream(ProxyAddr(b.p.Require(), flashblocksAddr))
		b.logger.Info("OPRBuilderNode proxy WS ready", "proxy_ws", b.wsProxyURL)
	}
}

func (b *OPRBuilderNode) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub == nil {
		b.logger.Warn("OPRbuilderNode already stopped")
		return
	}
	b.p.Require().NoError(b.sub.Stop(true))
	b.sub = nil
}

// WithOPRBuilderNode constructs and starts an OPRbuilderNode using the provided options.
func WithOPRBuilderNode(id stack.OPRBuilderNodeID, opts ...OPRBuilderNodeOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		l2Net, ok := orch.l2Nets.Get(id.ChainID())
		p.Require().True(ok, "l2 network required")

		tempDir := p.TempDir()
		data, err := json.Marshal(l2Net.genesis)
		p.Require().NoError(err, "must json-encode genesis")
		chainConfigPath := filepath.Join(tempDir, "genesis.json")
		p.Require().NoError(os.WriteFile(chainConfigPath, data, 0o644), "must write genesis file")

		// Build config from options
		cfg := DefaultOPRbuilderNodeConfig()
		cfg.AuthRPCJWTPath, _ = orch.writeDefaultJWT()
		cfg.Chain = chainConfigPath
		OPRBuilderNodeOptionBundle(opts).Apply(orch.P(), id, cfg)

		rb := &OPRBuilderNode{
			id:        id,
			logger:    p.Logger(),
			p:         p,
			rollupCfg: l2Net.rollupCfg,
			cfg:       cfg,
		}
		p.Logger().Info("Starting OPRbuilderNode")
		rb.Start()
		p.Cleanup(rb.Stop)
		orch.oprbuilderNodes.Set(id, rb)
	})
}

func (b *OPRBuilderNode) EngineRPC() string {
	return b.authProxyURL
}

func (b *OPRBuilderNode) JWTPath() string {
	return b.cfg.AuthRPCJWTPath
}

func (b *OPRBuilderNode) UserRPC() string {
	return b.rpcProxyURL
}
