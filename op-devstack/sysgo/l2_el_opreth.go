package sysgo

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

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
	logOut := logpipe.ToLoggerWithMinLevel(n.p.Logger().New("component", "op-reth", "src", "stdout", "name", n.name, "chain", n.chainID), log.LevelWarn)
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
