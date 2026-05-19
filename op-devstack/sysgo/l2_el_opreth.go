package sysgo

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"

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

	// On-disk state — tracked so tests can wipe and re-init before restart.
	dataDirPath     string
	chainConfigPath string
	proofHistoryDir string
	proofStorageVer string

	p devtest.T

	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar
}

var _ L2ELNode = (*OpReth)(nil)

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

// Stop stops the op-reth node. The user/auth RPC proxy addresses survive so
// Start may be called again to bring the process back up.
func (n *OpReth) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub == nil {
		return
	}
	err := n.sub.Stop(true)
	n.p.Require().NoError(err, "Must stop")
	n.sub = nil
}

// initStorage runs op-reth's `init` and (when configured) `proofs init`
// against the node's data dirs. Used at first start and after WipeOnDiskState.
func (n *OpReth) initStorage() error {
	if out, err := exec.Command(n.execPath, "init", "--datadir="+n.dataDirPath, "--chain="+n.chainConfigPath).CombinedOutput(); err != nil {
		return fmt.Errorf("op-reth %s: init: %w: %s", n.name, err, string(out))
	}
	if n.proofHistoryDir != "" && n.proofStorageVer != "" {
		out, err := exec.Command(n.execPath, "proofs", "init",
			"--datadir="+n.dataDirPath,
			"--chain="+n.chainConfigPath,
			"--proofs-history.storage-path="+n.proofHistoryDir,
			"--proofs-history.storage-version="+n.proofStorageVer,
		).CombinedOutput()
		if err != nil {
			return fmt.Errorf("op-reth %s: proofs init: %w: %s", n.name, err, string(out))
		}
	}
	return nil
}

// WipeOnDiskState removes and re-initialises the op-reth data dir and
// proof-history dir. Callers must Stop the node first.
func (n *OpReth) WipeOnDiskState() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sub != nil {
		return fmt.Errorf("op-reth %s: cannot wipe while running", n.name)
	}
	if n.dataDirPath == "" || n.chainConfigPath == "" {
		return fmt.Errorf("op-reth %s: data dir not tracked", n.name)
	}
	if err := os.RemoveAll(n.dataDirPath); err != nil {
		return fmt.Errorf("op-reth %s: remove datadir: %w", n.name, err)
	}
	if err := os.MkdirAll(n.dataDirPath, 0o755); err != nil {
		return fmt.Errorf("op-reth %s: recreate datadir: %w", n.name, err)
	}
	if n.proofHistoryDir != "" {
		if err := os.RemoveAll(n.proofHistoryDir); err != nil {
			return fmt.Errorf("op-reth %s: remove proof history: %w", n.name, err)
		}
	}
	return n.initStorage()
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
