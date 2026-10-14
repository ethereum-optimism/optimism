package sysgo

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shared/rustbin"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
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
	// DisableProofsHistory skips the proofs-history init + runtime flags for this node. The mixed
	// runtime otherwise enables proofs-history on every op-reth node; some CLI-superset binaries
	// reject --proofs-history in the mode under test.
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

// ResolveOpRethBinary locates the binary an op-reth node selected through OpRethWithBinary, or
// this repo's op-reth when binary is empty, through the rustbin env overrides and build rules.
// Resolving a binary before the chain starts keeps a RUST_JIT_BUILD build out of the run.
func ResolveOpRethBinary(t devtest.T, binary string) string {
	if binary == "" {
		binary = "op-reth"
	}
	execPath, err := rustbin.Spec{
		SrcDir:  "rust",
		Package: binary,
		Binary:  binary,
	}.EnsureExists(t.Ctx(), t.Logger())
	t.Require().NoError(err, "%s binary not available (build with 'just build-rust-release', set RUST_JIT_BUILD=1, or set %s for a binary built from another repo)", binary, rustbin.PathOverrideEnvVar(binary))
	return execPath
}

// OpRethWithoutProofsHistory disables the proofs-history subsystem for this node. The mixed runtime
// enables proofs-history on every op-reth node by default; use this for a CLI-superset binary that
// rejects --proofs-history in the mode under test.
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
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.T

	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar

	// logObservers has its own lock because the process's log pipes deliver entries while Start
	// holds mu.
	logObserversMu sync.Mutex
	logObservers   map[int]func(logpipe.LogEntry)
	nextObserverID int
}

var (
	_ L2ELNode                  = (*OpReth)(nil)
	_ stack.BinarySwitchable    = (*OpReth)(nil)
	_ stack.LogObservable       = (*OpReth)(nil)
	_ stack.UngracefulLifecycle = (*OpReth)(nil)
)

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
		n.notifyLogObservers(e)
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
		e := raiseUnstructuredStderr(logpipe.ParseRustStructuredLogs(line))
		logErr(e)
		n.notifyLogObservers(e)
	})
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.args, n.env)
	n.p.Require().NoError(err, "Must start")

	userRPCAddr := awaitAnnouncement(n.p, n.sub, userRPCChan, n.name, "user RPC")
	authRPCAddr := awaitAnnouncement(n.p, n.sub, authRPCChan, n.name, "auth RPC")

	if areMetricsEnabled() {
		metricsTarget := awaitAnnouncement(n.p, n.sub, metricsTargetChan, n.name, "metrics endpoint")
		n.l2MetricsRegistrar.RegisterL2MetricsTargets(n.name, metricsTarget)
	}

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), userRPCAddr))
	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), authRPCAddr))
}

// awaitAnnouncement returns the value a node logs once part of it is up, and fails the test as
// soon as the process exits without logging it, so a binary that aborts at startup does not stall
// the test until its deadline.
func awaitAnnouncement[E any](t devtest.CommonT, sub *SubProcess, announced chan E, name, what string) E {
	var value E
	select {
	case value = <-announced:
	case <-sub.Exited():
		exit := "exit status 0"
		if err := sub.ExitErr(); err != nil {
			exit = err.Error()
		}
		t.Require().Failf("process exited at startup", "%s exited during startup while waiting for its %s: %s", name, what, exit)
	case <-t.Ctx().Done():
		t.Require().NoErrorf(t.Ctx().Err(), "waiting for %s to announce its %s", name, what)
	}
	return value
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

// SwitchBinary resolves binary the way OpRethWithBinary does and makes the next Start run it on
// the node's existing datadir, proofs store and arguments. The node must be stopped.
func (n *OpReth) SwitchBinary(binary string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.p.Require().Truef(n.sub == nil, "op-reth %s must be stopped before its binary is switched", n.name)
	execPath := ResolveOpRethBinary(n.p, binary)
	n.p.Logger().Info("Switching op-reth binary", "name", n.name, "chain", n.chainID, "binary", binary, "from", n.execPath, "to", execPath)
	n.execPath = execPath
}

// ObserveLogs registers observer for every stdout and stderr entry the op-reth process emits,
// across restarts, until the returned function is called.
func (n *OpReth) ObserveLogs(observer func(logpipe.LogEntry)) func() {
	n.logObserversMu.Lock()
	defer n.logObserversMu.Unlock()
	if n.logObservers == nil {
		n.logObservers = make(map[int]func(logpipe.LogEntry))
	}
	id := n.nextObserverID
	n.nextObserverID++
	n.logObservers[id] = observer
	return func() {
		n.logObserversMu.Lock()
		defer n.logObserversMu.Unlock()
		delete(n.logObservers, id)
	}
}

func (n *OpReth) notifyLogObservers(e logpipe.LogEntry) {
	n.logObserversMu.Lock()
	observers := slices.Collect(maps.Values(n.logObservers))
	n.logObserversMu.Unlock()
	for _, observe := range observers {
		observe(e)
	}
}

// raiseUnstructuredStderr turns an unparsable stderr line into an error entry: op-reth writes its
// structured logs to stdout, so anything else on stderr is a panic or an abort message.
func raiseUnstructuredStderr(e logpipe.LogEntry) logpipe.LogEntry {
	if !logpipe.IsUnstructured(e) {
		return e
	}
	r := e.(logpipe.StructuredRustLogEntry)
	r.Level = log.LevelError
	return r
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
