package sysgo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

// OpRethConfig holds the configurable knobs applied to an op-reth node before it is started.
type OpRethConfig struct {
	// ExtraArgs are appended to the generated CLI args.
	ExtraArgs []string
	// ProofsHistoryWindow overrides the default proofs-history retention window when non-zero.
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

// OpRethWithProofsHistoryWindow overrides the proofs-history retention window.
func OpRethWithProofsHistoryWindow(window uint64) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		p.Require().Greater(window, uint64(0), "proofs-history window must be non-zero")
		cfg.ProofsHistoryWindow = window
	})
}

// OpRethWithSDMEnabled enables Sequencer-Defined Metering on the op-reth node.
func OpRethWithSDMEnabled() OpRethOption {
	return OpRethWithExtraArgs("--rollup.sdm-enabled")
}

// OpRethWithSupervisorURL wires the op-reth node to the given supervisor HTTP endpoint.
// An empty supervisorURL is a no-op so callers can pass the value unconditionally.
func OpRethWithSupervisorURL(supervisorURL string) OpRethOption {
	return OpRethOptionFn(func(p devtest.T, _ ComponentTarget, cfg *OpRethConfig) {
		if supervisorURL == "" {
			return
		}
		cfg.ExtraArgs = append(cfg.ExtraArgs, "--rollup.supervisor-http="+supervisorURL)
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
}

var _ L2ELNode = (*OpReth)(nil)

func (n *OpReth) Start() {
	n.p.Require().NoError(n.start(n.p.Ctx()), "Must start")
}

func (n *OpReth) StartWithTimeout(timeout time.Duration) error {
	ctx := n.p.Ctx()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return n.start(ctx)
}

func (n *OpReth) start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		n.p.Logger().Warn("op-reth already started")
		return nil
	}
	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.p.Logger())
		if err := n.authProxy.Start(); err != nil {
			return fmt.Errorf("start auth proxy: %w", err)
		}
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.p.Logger())
		if err := n.userProxy.Start(); err != nil {
			return fmt.Errorf("start user proxy: %w", err)
		}
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "ws://" + n.userProxy.Addr()
	}
	logOut := logpipe.ToLoggerWithMinLevel(n.p.Logger().New("component", "op-reth", "src", "stdout", "name", n.name, "chain", n.chainID), log.LevelInfo)
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
	})
	stdErrLogs := logpipe.LogCallback(func(line []byte) {
		e := logpipe.ParseRustStructuredLogs(line)
		logErr(e)
	})
	n.sub = NewSubProcess(n.p, stdOutLogs, stdErrLogs)

	err := n.sub.Start(n.execPath, n.args, n.env)
	if err != nil {
		return fmt.Errorf("start op-reth subprocess: %w", err)
	}

	var userRPCAddr, authRPCAddr string
	if err := tasks.Await(ctx, userRPCChan, &userRPCAddr); err != nil {
		return fmt.Errorf("need user RPC: %w", err)
	}
	if err := tasks.Await(ctx, authRPCChan, &authRPCAddr); err != nil {
		return fmt.Errorf("need auth RPC: %w", err)
	}

	if areMetricsEnabled() {
		var metricsTarget PrometheusMetricsTarget
		if err := tasks.Await(ctx, metricsTargetChan, &metricsTarget); err != nil {
			return fmt.Errorf("need metrics endpoint: %w", err)
		}
		n.l2MetricsRegistrar.RegisterL2MetricsTargets(n.name, metricsTarget)
	}

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), userRPCAddr))
	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), authRPCAddr))
	return nil
}

// Stop stops the op-reth node.
// warning: no restarts supported yet, since the RPC port is not remembered.
func (n *OpReth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	err := n.sub.Stop(true)
	n.p.Require().NoError(err, "Must stop")
	n.sub = nil
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
