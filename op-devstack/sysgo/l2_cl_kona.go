package sysgo

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/common"
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
	// Start sequencing through the admin API after each process startup.
	startSequencer bool

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
	if k.startSequencer {
		rollupClient, err := dial.DialRollupClientWithTimeout(k.p.Ctx(), k.p.Logger(), k.userRPC)
		k.p.Require().NoError(err, "failed to dial sequencer node")
		defer rollupClient.Close()
		// kona-node serves RPC before its engine reads the heads from the execution layer, and
		// reports a zero unsafe head until then; it starts sequencing only on its own head, which
		// may also move between reading it and starting.
		ctx, cancel := context.WithTimeout(k.p.Ctx(), time.Minute)
		defer cancel()
		err = retry.Do0(ctx, 120, retry.Fixed(500*time.Millisecond), func() error {
			status, err := rollupClient.SyncStatus(ctx)
			if err != nil {
				return err
			}
			if status.UnsafeL2.Hash == (common.Hash{}) {
				return errors.New("kona-node has not read its unsafe head yet")
			}
			return rollupClient.StartSequencer(ctx, status.UnsafeL2.Hash)
		})
		k.p.Require().NoError(err, "failed to start sequencing")
	}
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

// Callers must hold k.mu.
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
