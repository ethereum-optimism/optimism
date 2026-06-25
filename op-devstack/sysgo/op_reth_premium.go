// Temporary harness for validating the migrated policy engine against the
// op-reth-premium binary. This file is NOT intended to merge: it adapts the
// op-rbuilder acceptance harness to launch op-reth-premium (a single process
// that is both the sequencer EL and the in-process subblocks producer) instead
// of the op-rbuilder + rollup-boost two-process flashblocks topology.
package sysgo

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	yaml "gopkg.in/yaml.v3"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// PremiumNode launches op-reth-premium and exposes it as an L2ELNode (so the CL
// can drive its engine API directly) and as a flashblocks builder (so the rules
// preset can read its subblocks WS stream and rewrite its ruleset file).
type PremiumNode struct {
	mu sync.Mutex

	name      string
	chainID   eth.ChainID
	rollupCfg *rollup.Config

	wsProxyURL string
	wsProxy    *tcpproxy.Proxy

	rpcProxyURL string
	rpcProxy    *tcpproxy.Proxy

	authProxyURL string
	authProxy    *tcpproxy.Proxy

	logger log.Logger
	p      devtest.CommonT

	sub *SubProcess
	cfg *PremiumNodeConfig
}

var _ stack.Lifecycle = (*PremiumNode)(nil)
var _ L2ELNode = (*PremiumNode)(nil)

var (
	errPremiumRulesNotConfigured = errors.New("rules config path is not configured (rules not enabled?)")
	errPremiumNoRuleFiles        = errors.New("no file entries found in rules config")
)

// PremiumNodeConfig configures the op-reth-premium CLI. It mirrors the subset of
// OPRBuilderNodeConfig the rules preset relies on, swapping the flashblocks flags
// for the premium subblocks namespace.
type PremiumNodeConfig struct {
	Chain   string
	DataDir string

	LogStdoutFormat string
	LogFileFormat   string

	// SubblocksPublisherBind is the WS publisher bind address. op-reth-premium
	// defaults this to 0.0.0.0:1111 (op-rbuilder flashblocks-compatible). The
	// publisher bind is not yet a CLI flag, so this only documents the port the
	// harness connects to; see SubblocksPublisherPort.
	SubblocksPublisherPort int

	EnableRPC bool
	RPCAPI    string
	RPCAddr   string
	RPCPort   int

	AuthRPCJWTPath string
	AuthRPCAddr    string
	AuthRPCPort    int

	ChainBlockTime time.Duration

	P2PPort       int
	P2PAddr       string
	P2PNodeKeyHex string
	StaticPeers   []string
	TrustedPeers  []string

	Full bool

	RulesEnabled    bool
	RulesConfigPath string

	ExtraArgs []string
	Env       []string
}

func DefaultPremiumNodeConfig() *PremiumNodeConfig {
	return &PremiumNodeConfig{
		Chain:                  "dev",
		LogStdoutFormat:        "json",
		LogFileFormat:          "json",
		SubblocksPublisherPort: 1111,
		EnableRPC:              true,
		RPCAPI:                 "admin,web3,debug,eth,txpool,net,miner",
		RPCAddr:                "127.0.0.1",
		RPCPort:                0,
		AuthRPCAddr:            "127.0.0.1",
		AuthRPCPort:            0,
		AuthRPCJWTPath:         "",
		P2PAddr:                "127.0.0.1",
		P2PPort:                0,
		ChainBlockTime:         time.Second * 2,
		Full:                   true,
		RulesEnabled:           false,
		RulesConfigPath:        "",
	}
}

func (cfg *PremiumNodeConfig) LaunchSpec(p devtest.CommonT) (args []string, env []string) {
	p.Require().NotNil(cfg, "nil PremiumNodeConfig")

	env = append([]string(nil), cfg.Env...)
	args = make([]string, 0, len(cfg.ExtraArgs)+16)

	args = append(args, "node")

	// Premium subblocks producer is the flashblocks-equivalent path. The WS
	// publisher is auto-spawned at the default bind (port 1111); there is no
	// per-port CLI flag yet, so the harness pins one premium node per run.
	args = append(args, "--subblocks.enable=true")

	if cfg.P2PNodeKeyHex != "" {
		key := strings.TrimPrefix(cfg.P2PNodeKeyHex, "0x")
		_, err := hex.DecodeString(key)
		p.Require().NoError(err, "decode p2p node key")
		keyPath := filepath.Join(p.TempDir(), "premium-nodekey")
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
			args = append(args, "--http.port="+strconv.Itoa(cfg.RPCPort))
		} else {
			args = append(args, "--http.port=0")
		}
		args = append(args, "--http.api="+cfg.RPCAPI)
	}

	if cfg.AuthRPCAddr != "" {
		args = append(args, "--authrpc.addr="+cfg.AuthRPCAddr)
	}
	if cfg.AuthRPCPort > 0 {
		args = append(args, "--authrpc.port="+strconv.Itoa(cfg.AuthRPCPort))
	} else {
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
		// op-reth-premium uses the optimism CLI app, which never calls
		// apply_node_defaults(); reth v2.3.0 then leaves log.file.max-files at
		// None (effective 0 = file logging disabled). op-rbuilder patches the
		// clap default to 5 for the same reason; here we set it explicitly so
		// the node actually writes the JSON log file the rotation watcher tails.
		args = append(args, "--log.file.max-files=5")
	}
	if cfg.Chain != "" {
		args = append(args, "--chain="+cfg.Chain)
	}
	args = append(args, "--disable-discovery")

	if cfg.P2PPort > 0 {
		args = append(args, "--port="+strconv.Itoa(cfg.P2PPort))
	} else {
		args = append(args, "--with-unused-ports")
	}

	if cfg.DataDir == "" {
		tmpDir, err := os.MkdirTemp("", "op-reth-premium-datadir-*")
		p.Require().NoError(err, "create temp datadir for op-reth-premium")
		args = append(args, "--datadir="+tmpDir)
		p.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	} else {
		args = append(args, "--datadir="+cfg.DataDir)
	}

	if cfg.RulesEnabled {
		args = append(args, "--rules.enabled=true")
		args = append(args, "--rules.config-path="+cfg.RulesConfigPath)
	}

	// op-reth-premium has no --rollup.chain-block-time: the subblocks producer
	// derives block time from the CL-set payload-attribute timestamps, and the
	// per-round cadence comes from --subblocks.interval (defaulted). The CL's
	// rollup config (ChainBlockTime) drives the actual block time.
	args = append(args, cfg.ExtraArgs...)

	return args, env
}

func (b *PremiumNode) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub != nil {
		b.logger.Warn("PremiumNode already started")
		return
	}
	cfg := b.cfg
	b.p.Require().NotNil(cfg, "PremiumNode config not initialized")

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

	httpRPCChan := make(chan string, 1)
	authRPCChan := make(chan string, 1)
	defer close(httpRPCChan)
	defer close(authRPCChan)

	logOut := logpipe.ToLoggerWithMinLevel(b.logger.New("component", "op-reth-premium", "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(b.logger.New("component", "op-reth-premium", "src", "stderr"), log.LevelWarn)

	// op-reth-premium reuses reth's standard RPC log lines, so HTTP/auth port
	// discovery matches the op-rbuilder path. The subblocks WS publisher binds a
	// fixed port (1111) and needs no log-derived discovery.
	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
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

	execPath, err := rustbin.Spec{
		SrcDir:  "rust/op-reth-premium",
		Package: "op-reth-premium",
		Binary:  "op-reth-premium",
	}.EnsureExists(b.p.Ctx(), b.p.Logger())
	b.p.Require().NoError(err, "prepare op-reth-premium binary")
	b.p.Require().NotEmpty(execPath, "op-reth-premium binary path resolved")

	err = b.sub.Start(execPath, args, env)
	b.p.Require().NoError(err, "start PremiumNode")

	if cfg.EnableRPC {
		var httpRPCAddr string
		b.p.Require().NoError(tasks.Await(b.p.Ctx(), httpRPCChan, &httpRPCAddr), "need HTTP RPC address from logs")
		b.rpcProxy.SetUpstream(ProxyAddr(b.p.Require(), httpRPCAddr))

		var authRPCAddr string
		b.p.Require().NoError(tasks.Await(b.p.Ctx(), authRPCChan, &authRPCAddr), "need Auth RPC address from logs")
		b.authProxy.SetUpstream(ProxyAddr(b.p.Require(), authRPCAddr))
	}

	// The subblocks WS publisher binds the fixed default port at process start;
	// point the WS proxy straight at it (no log-derived discovery needed).
	// SetUpstream takes a host:port; the publisher binds 0.0.0.0 so connect via
	// loopback.
	b.wsProxy.SetUpstream("127.0.0.1:" + strconv.Itoa(cfg.SubblocksPublisherPort))
}

func (b *PremiumNode) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.sub == nil {
		b.logger.Warn("PremiumNode already stopped")
		return
	}
	b.p.Require().NoError(b.sub.Stop(true))
	b.sub = nil
}

func (b *PremiumNode) EngineRPC() string        { return b.authProxyURL }
func (b *PremiumNode) JWTPath() string          { return b.cfg.AuthRPCJWTPath }
func (b *PremiumNode) UserRPC() string          { return b.rpcProxyURL }
func (b *PremiumNode) FlashblocksWSURL() string { return b.wsProxyURL }

func (b *PremiumNode) UpdateRuleSet(rulesYaml string) error {
	if b.cfg.RulesConfigPath == "" {
		return errPremiumRulesNotConfigured
	}
	rulesConfigContent, err := os.ReadFile(b.cfg.RulesConfigPath)
	if err != nil {
		return err
	}
	var rulesConfig RulesConfig
	if err := yaml.Unmarshal(rulesConfigContent, &rulesConfig); err != nil {
		return err
	}
	if len(rulesConfig.File) == 0 {
		return errPremiumNoRuleFiles
	}
	rulesPath := rulesConfig.File[0].Path
	return os.WriteFile(rulesPath, []byte(rulesYaml), 0o644)
}
