package sysgo

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

type KonaNode struct {
	mu sync.Mutex

	name    string
	chainID eth.ChainID

	userRPC string

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.T

	sub *SubProcess
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

	onLogEntry := func(e logpipe.LogEntry) {
		if e.LogMessage() == "RPC server bound to address" {
			userRPCChan <- "http://" + e.FieldValue("addr").(string)
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

	// Wait for kona-node to log its RPC address, but fail fast if the process exits first
	// (e.g. a crash on boot) rather than blocking on the context until the test times out.
	var userRPCAddr string
	select {
	case userRPCAddr = <-userRPCChan:
	case <-k.sub.Exited():
		// Re-check the RPC channel in case the address was logged in the same instant the
		// process exited; otherwise the process died before becoming ready.
		select {
		case userRPCAddr = <-userRPCChan:
		default:
			k.p.Require().FailNow("kona-node exited before its RPC server became ready")
		}
	case <-k.p.Ctx().Done():
		k.p.Require().NoError(k.p.Ctx().Err(), "need user RPC")
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
	k.clearProxyUpstreams()
	err := k.sub.Stop(true)
	k.p.Require().NoError(err, "Must stop")
	k.sub = nil
}

func (k *KonaNode) StartControlled(ctx context.Context) error {
	return runControlStart(ctx, k.Running, k.Start)
}

func (k *KonaNode) StopControlled(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.sub == nil {
		return nil
	}
	k.clearProxyUpstreams()
	if err := k.sub.StopControlled(ctx, controlledInterruptWait, controlledKillWait); err != nil {
		return err
	}
	k.sub = nil
	return nil
}

// clearProxyUpstreams unsets the proxy upstreams. It must run before the
// process is asked to stop: the process frees its OS-assigned ports early in
// shutdown, and they may be rebound by another process, so a stale upstream
// would silently cross-wire this node's endpoints to it. Callers must hold k.mu.
func (k *KonaNode) clearProxyUpstreams() {
	if k.userProxy != nil {
		k.userProxy.ClearUpstream()
	}
}

func (k *KonaNode) Running() bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.sub != nil
}

func (k *KonaNode) UserRPC() string {
	return k.userRPC
}

var _ L2CLNode = (*KonaNode)(nil)
