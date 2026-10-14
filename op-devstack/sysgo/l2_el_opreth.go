package sysgo

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// OpRethConfig holds the configurable knobs applied to an op-reth node before it is started.
type OpRethConfig struct {
	// ExtraArgs are appended to the generated CLI args.
	ExtraArgs []string
	// Binary selects the EL binary to launch. Empty means "op-reth". A CLI-compatible superset
	// (any binary that accepts every op-reth subcommand/flag, plus optionally its own additive
	// flags) may be selected via OpRethWithBinary.
	Binary string
	// DisableProofsHistory builds the node without a seeded proofs-history store and without the
	// proofs-history runtime flags. The mixed runtime otherwise enables proofs history on every
	// op-reth node; some CLI-superset binaries reject --proofs-history in the mode under test, and
	// a test of the operator path seeds the store later with InitProofsHistory.
	DisableProofsHistory bool
	// ProofsHistoryWindow is the number of blocks the proofs-history store keeps. Zero keeps the
	// mixed runtime's default window.
	ProofsHistoryWindow uint64
}

// DefaultOpRethConfig returns a zero-valued OpRethConfig that callers can mutate via OpRethOptions.
func DefaultOpRethConfig() *OpRethConfig {
	return &OpRethConfig{}
}

// OpRethOption customises an OpRethConfig for a specific component target.
type OpRethOption interface {
	Apply(p devtest.T, target ComponentTarget, cfg *OpRethConfig)
}

// OpRethOptionFn adapts a plain function into an OpRethOption.
type OpRethOptionFn func(p devtest.T, target ComponentTarget, cfg *OpRethConfig)

var _ OpRethOption = OpRethOptionFn(nil)

// Apply invokes the underlying function against the supplied config.
func (fn OpRethOptionFn) Apply(p devtest.T, target ComponentTarget, cfg *OpRethConfig) {
	fn(p, target, cfg)
}

// OpRethOptionBundle applies multiple OpRethOptions in order.
type OpRethOptionBundle []OpRethOption

var _ OpRethOption = OpRethOptionBundle(nil)

// Apply runs each contained option against cfg, failing the test on a nil entry.
func (b OpRethOptionBundle) Apply(p devtest.T, target ComponentTarget, cfg *OpRethConfig) {
	for _, opt := range b {
		p.Require().NotNil(opt, "cannot Apply nil OpRethOption")
		opt.Apply(p, target, cfg)
	}
}

// OpRethWithExtraArgs appends raw CLI arguments to the op-reth invocation.
func OpRethWithExtraArgs(args ...string) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		cfg.ExtraArgs = append(cfg.ExtraArgs, args...)
	})
}

// OpRethWithBinary selects the EL binary to launch instead of the default "op-reth". The binary
// must be a CLI superset of op-reth (it is invoked with op-reth's subcommands and flags). A binary
// that lives outside this repo is resolved via the rustbin env overrides keyed off its name —
// RUST_BINARY_PATH_<NAME> (or RUST_SRC_DIR_<NAME> + RUST_JIT_BUILD), with <NAME> the upper-snake-cased
// binary name.
func OpRethWithBinary(binary string) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		cfg.Binary = binary
	})
}

// OpRethWithoutProofsHistory builds the node without proofs history. The mixed runtime enables it
// on every op-reth node by default; use this for a CLI-superset binary that rejects
// --proofs-history in the mode under test, or for a node whose store a test seeds later with
// InitProofsHistory.
func OpRethWithoutProofsHistory() OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		cfg.DisableProofsHistory = true
	})
}

// OpRethWithProofsHistoryWindow sets the number of blocks the proofs-history store keeps. The mixed
// runtime passes --proofs-history.window itself and op-reth rejects a repeated flag, so the window
// cannot be set through OpRethWithExtraArgs.
func OpRethWithProofsHistoryWindow(blocks uint64) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		cfg.ProofsHistoryWindow = blocks
	})
}

// OpRethWithEthProofWindow lets eth_getProof answer for blocks up to the given distance behind the
// head from reth's own historical state. reth's default window is zero, so a node without proofs
// history rejects every historical proof request unless it gets this option.
func OpRethWithEthProofWindow(blocks uint64) OpRethOption {
	return OpRethWithExtraArgs(fmt.Sprintf("--rpc.eth-proof-window=%d", blocks))
}

// OpRethWithInteropURL wires the op-reth node to the given interop filter HTTP endpoint.
// An empty interopURL is a no-op so callers can pass the value unconditionally.
func OpRethWithInteropURL(interopURL string) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		if interopURL == "" {
			return
		}
		cfg.ExtraArgs = append(cfg.ExtraArgs, "--rollup.interop-http="+interopURL)
	})
}

type OpReth struct {
	mu sync.Mutex

	name      string
	chainID   eth.ChainID
	jwtPath   string
	jwtSecret [32]byte
	authRPC   string
	userRPC   string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy

	execPath string
	// args are the `node` arguments without the proofs-history flags, which Start appends while
	// proofsHistoryEnabled is set.
	args []string
	// Each entry is of the form "key=value".
	env []string

	chainConfigPath string
	dataDirPath     string

	// proofsHistoryEnabled says the store at proofsHistoryStoragePath is seeded, so the node runs
	// with proofs history from its next start.
	proofsHistoryEnabled     bool
	proofsHistoryStoragePath string
	// proofsHistoryWindow is the window proofs init backfills and the node keeps; zero means the
	// mixed runtime's default.
	proofsHistoryWindow uint64

	p devtest.T

	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar

	// logRecords outlives the sub-process so a record registered before a restart keeps
	// collecting afterwards.
	logRecords logpipe.LogRecords
}

var (
	_ L2ELNode                       = (*OpReth)(nil)
	_ stack.LogRecorder              = (*OpReth)(nil)
	_ stack.ProofsHistoryInitializer = (*OpReth)(nil)
	_ stack.UngracefulLifecycle      = (*OpReth)(nil)
)

// RecordLogLines returns a record of every line op-reth writes to stdout or stderr from now on,
// across restarts, whose message, or whose raw text for a line that is not a structured log
// entry, contains substring. Lines below the level the test log keeps are recorded too.
func (n *OpReth) RecordLogLines(substring string) *logpipe.LogRecord {
	return n.logRecords.Record(substring)
}

func (n *OpReth) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.p.Logger().Warn("op-reth already started")
		return
	}
	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.p.Logger())
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.p.Logger())
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "ws://" + n.userProxy.Addr()
	}
	stdoutLogger := n.p.Logger().New("component", "op-reth", "src", "stdout", "name", n.name, "chain", n.chainID)
	stdoutInfo := logpipe.ToLoggerWithMinLevel(stdoutLogger, log.LevelInfo)
	// Peer-disconnect reasons are logged below INFO under net::session / net::peers.
	// Raise those entries to INFO (original level kept as an attribute) so they
	// survive the devtest INFO log filter and peer drops stay diagnosable.
	stdoutNetPeers := logpipe.ToLoggerRaisedToLevel(stdoutLogger, log.LevelInfo)
	logOut := func(e logpipe.LogEntry) {
		if r, ok := e.(logpipe.StructuredRustLogEntry); ok &&
			(strings.HasPrefix(r.Target, "net::session") || strings.HasPrefix(r.Target, "net::peers")) {
			stdoutNetPeers(e)
			return
		}
		stdoutInfo(e)
	}
	logErr := logpipe.ToLoggerWithMinLevel(n.p.Logger().New("component", "op-reth", "src", "stderr", "name", n.name, "chain", n.chainID), log.LevelWarn)

	authRPCChan := make(chan string, 1)
	defer close(authRPCChan)

	metricsTargetChan := make(chan PrometheusMetricsTarget, 1)
	defer close(metricsTargetChan)

	userRPCChan := make(chan string, 1)
	defer close(userRPCChan)
	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
		if msg == "RPC WS server started" {
			select {
			case userRPCChan <- "ws://" + e.FieldValue("url").(string):
			default:
			}
		} else if msg == "RPC auth server started" {
			select {
			case authRPCChan <- "ws://" + e.FieldValue("url").(string):
			default:
			}
		} else if metricsUrl, found := strings.CutPrefix(msg, "Starting metrics endpoint at "); found {
			// expected format: "Starting metrics endpoint at 127.0.0.1:9091"
			if !strings.HasPrefix(metricsUrl, "http") {
				metricsUrl = fmt.Sprintf("http://%s", metricsUrl)
			}
			parsedUrl, err := url.Parse(metricsUrl)
			n.p.Require().NoError(err, "invalid metrics url output to logs", "log", msg)
			n.p.Require().NotEmpty(parsedUrl.Port(), "empty port in logged metrics url", "log", msg)
			metricsTargetChan <- NewPrometheusMetricsTarget(parsedUrl.Hostname(), parsedUrl.Port(), false)
		}
	}
	stdOutLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logOut(e)
		onLogEntry(e)
		n.logRecords.Observe(e)
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
		n.logRecords.Observe(e)
	})
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.nodeArgs(), n.env)
	n.p.Require().NoError(err, "Must start")

	var userRPCAddr, authRPCAddr string
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), userRPCChan, &userRPCAddr), "need user RPC")
	n.p.Require().NoError(tasks.Await(n.p.Ctx(), authRPCChan, &authRPCAddr), "need auth RPC")

	if areMetricsEnabled() {
		var metricsTarget PrometheusMetricsTarget
		n.p.Require().NoError(tasks.Await(n.p.Ctx(), metricsTargetChan, &metricsTarget), "need metrics endpoint")
		n.l2MetricsRegistrar.RegisterL2MetricsTargets(n.name, metricsTarget)
	}

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), userRPCAddr))
	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), authRPCAddr))
}

// Stop stops the op-reth node.
func (n *OpReth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		n.p.Logger().Warn("op-reth already stopped")
		return
	}
	n.clearProxyUpstreams()
	err := n.sub.Stop(true)
	n.p.Require().NoError(err, "Must stop")
	n.sub = nil
}

// StopUngracefully ends the op-reth process with SIGKILL, the way a crash would, so whatever it
// had not yet handed to the OS for its datadir is lost. Start brings the node back on the same
// datadir.
func (n *OpReth) StopUngracefully() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		n.p.Logger().Warn("op-reth already stopped")
		return
	}
	n.p.Logger().Info("Stopping op-reth ungracefully", "name", n.name, "chain", n.chainID)
	n.clearProxyUpstreams()
	n.p.Require().NoError(n.sub.Kill(), "Must kill")
	n.sub = nil
}

// nodeArgs are the `node` arguments of the next start. Callers must hold n.mu.
func (n *OpReth) nodeArgs() []string {
	if !n.proofsHistoryEnabled {
		return n.args
	}
	args := append([]string{}, n.args...)
	return append(args, proofsHistoryNodeArgs(n.proofsHistoryWindow, n.proofsHistoryStoragePath)...)
}

// InitProofsHistory runs `op-reth proofs init` against this node's datadir and, when it succeeds,
// makes the next Start run with proofs history on the seeded store. It does not stop the node:
// what init does to a datadir the node is still writing to is the caller's concern.
func (n *OpReth) InitProofsHistory(init stack.ProofsHistoryInit) (stack.ProofsHistoryInitReport, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	window := n.proofsHistoryWindow
	if init.Window != 0 {
		window = init.Window
	}
	args := proofsHistoryInitArgs(n.chainConfigPath, n.dataDirPath, n.proofsHistoryStoragePath, window, init.UseSnapshot)
	logger := n.p.Logger().New("name", n.name, "chain", n.chainID, "running", n.sub != nil)
	logger.Info("Running op-reth proofs init", "cmd", n.execPath, "args", args)
	started := time.Now()
	output, err := exec.Command(n.execPath, args...).CombinedOutput()
	report := stack.ProofsHistoryInitReport{Duration: time.Since(started), Output: string(output)}
	if err != nil {
		logger.Warn("op-reth proofs init failed", "duration", report.Duration, "err", err, "output", report.Output)
		return report, fmt.Errorf("op-reth proofs init failed after %s: %w\n%s", report.Duration, err, report.Output)
	}
	n.proofsHistoryEnabled = true
	n.proofsHistoryWindow = window
	logger.Info("op-reth proofs init succeeded; proofs history is on from the next start",
		"duration", report.Duration, "window", resolveProofsHistoryWindow(window), "storage_path", n.proofsHistoryStoragePath)
	logger.Debug("op-reth proofs init output", "output", report.Output)
	return report, nil
}

// defaultMixedProofsHistoryWindow is the proofs-history window an op-reth node gets unless
// OpRethWithProofsHistoryWindow or a ProofsHistoryInit overrides it.
const defaultMixedProofsHistoryWindow uint64 = 10_000

func resolveProofsHistoryWindow(window uint64) uint64 {
	if window == 0 {
		return defaultMixedProofsHistoryWindow
	}
	return window
}

// proofsHistoryNodeArgs returns the node flags that enable proofs history on the store at
// storagePath.
func proofsHistoryNodeArgs(window uint64, storagePath string) []string {
	return []string{
		"--proofs-history",
		"--proofs-history.storage-path=" + storagePath,
		"--proofs-history.window=" + strconv.FormatUint(resolveProofsHistoryWindow(window), 10),
	}
}

// proofsHistoryInitArgs returns the `proofs init` invocation that seeds the store at storagePath
// from the datadir and backfills it window blocks behind the datadir's tip. op-reth backfills
// through a trie-state snapshot unless told not to, so the snapshot path gets the operator's
// invocation with no flag.
func proofsHistoryInitArgs(chainConfigPath, dataDirPath, storagePath string, window uint64, useSnapshot bool) []string {
	args := []string{
		"proofs",
		"init",
		"--chain=" + chainConfigPath,
		"--datadir=" + dataDirPath,
		"--proofs-history.storage-path=" + storagePath,
		"--proofs-history.window=" + strconv.FormatUint(resolveProofsHistoryWindow(window), 10),
	}
	if !useSnapshot {
		args = append(args, "--proofs-history.use-snapshot=false")
	}
	return args
}

// Callers must hold n.mu.
func (n *OpReth) clearProxyUpstreams() {
	if n.userProxy != nil {
		n.userProxy.ClearUpstream()
	}
	if n.authProxy != nil {
		n.authProxy.ClearUpstream()
	}
}

func (n *OpReth) StartControlled(ctx context.Context) error {
	return runControlStart(ctx, n.Running, n.Start)
}

func (n *OpReth) StopControlled(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		return nil
	}
	n.clearProxyUpstreams()
	if err := n.sub.StopControlled(ctx, controlledInterruptWait, controlledKillWait); err != nil {
		return err
	}
	n.sub = nil
	return nil
}

func (n *OpReth) Running() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.sub != nil
}

func (n *OpReth) UserRPC() string {
	return n.userRPC
}

func (n *OpReth) EngineRPC() string {
	return n.authRPC
}

func (n *OpReth) JWTPath() string {
	return n.jwtPath
}
