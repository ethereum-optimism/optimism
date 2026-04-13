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

type KonaNode struct {
	mu sync.Mutex

	name    string
	chainID eth.ChainID

	userRPC          string
	interopEndpoint  string // warning: currently not fully supported
	interopJwtSecret eth.Bytes32

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.T

	sub *SubProcess

	l2MetricsRegistrar L2MetricsRegistrar
}

func (k *KonaNode) Start() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sub != nil {
		k.p.Logger().Warn("Kona-node already started")
		return
	}
	// Create a proxy for the user RPC,
	// so other services can connect, and stay connected, across restarts.
	if k.userProxy == nil {
		k.userProxy = tcpproxy.New(k.p.Logger())
		k.p.Require().NoError(k.userProxy.Start())
		k.p.Cleanup(func() {
			k.userProxy.Close()
		})
		k.userRPC = "http://" + k.userProxy.Addr()
	}
	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	// And inspect them along the way, to get the RPC server address.
	logOut := logpipe.ToLoggerWithMinLevel(k.p.Logger().New("component", "kona-node", "src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(k.p.Logger().New("component", "kona-node", "src", "stderr"), log.LevelWarn)
	userRPCChan := make(chan string, 1)
	defer close(userRPCChan)

	metricsTargetChan := make(chan PrometheusMetricsTarget, 1)
	defer close(metricsTargetChan)

	onLogEntry := func(e logpipe.LogEntry) {
		msg := e.LogMessage()
		if msg == "RPC server bound to address" {
			userRPCChan <- "http://" + e.FieldValue("addr").(string)
		} else if metricsUrl, found := strings.CutPrefix(msg, "Serving metrics at: "); found {
			// Matching messages like "Serving metrics at: http://0.0.0.0:9091"
			if !strings.HasPrefix(metricsUrl, "http") {
				metricsUrl = fmt.Sprintf("http://%s", metricsUrl)
			}
			parsedUrl, err := url.Parse(metricsUrl)
			k.p.Require().NoError(err, "invalid metrics url output to logs", "log", msg)
			k.p.Require().NotEmpty(parsedUrl.Port(), "empty port in logged metrics url", "log", msg)
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
	k.sub = NewSubProcess(k.p, stdOutLogs, stdErrLogs)

	err := k.sub.Start(k.execPath, k.args, k.env)
	k.p.Require().NoError(err, "Must start")

	var userRPCAddr string
	k.p.Require().NoError(tasks.Await(k.p.Ctx(), userRPCChan, &userRPCAddr), "need user RPC")

	if areMetricsEnabled() {
		var metricsTarget PrometheusMetricsTarget
		k.p.Require().NoError(tasks.Await(k.p.Ctx(), metricsTargetChan, &metricsTarget), "need metrics endpoint")
		k.l2MetricsRegistrar.RegisterL2MetricsTargets(k.name, metricsTarget)
	}

	k.userProxy.SetUpstream(ProxyAddr(k.p.Require(), userRPCAddr))
}

// Stop stops the kona node.
// warning: no restarts supported yet, since the RPC port is not remembered.
func (k *KonaNode) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sub == nil {
		k.p.Logger().Warn("kona-node already stopped")
		return
	}
	err := k.sub.Stop(true)
	k.p.Require().NoError(err, "Must stop")
	k.sub = nil
}

func (k *KonaNode) UserRPC() string {
	return k.userRPC
}

func (k *KonaNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return k.interopEndpoint, k.interopJwtSecret
}

var _ L2CLNode = (*KonaNode)(nil)
