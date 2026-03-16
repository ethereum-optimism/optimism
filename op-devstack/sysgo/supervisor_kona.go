package sysgo

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/tasks"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

type KonaSupervisor struct {
	mu sync.Mutex

	name    string
	userRPC string

	userProxy *tcpproxy.Proxy

	execPath string
	args     []string
	// Each entry is of the form "key=value".
	env []string

	p devtest.CommonT

	sub *SubProcess
}

var _ stack.Lifecycle = (*KonaSupervisor)(nil)

func (s *KonaSupervisor) UserRPC() string {
	return s.userRPC
}

func (s *KonaSupervisor) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub != nil {
		s.p.Logger().Warn("Kona-supervisor already started")
		return
	}

	// Create a proxy for the user RPC,
	// so other services can connect, and stay connected, across restarts.
	if s.userProxy == nil {
		s.userProxy = tcpproxy.New(s.p.Logger())
		s.p.Require().NoError(s.userProxy.Start())
		s.p.Cleanup(func() {
			s.userProxy.Close()
		})
		s.userRPC = "http://" + s.userProxy.Addr()
	}

	// Create the sub-process.
	// We pipe sub-process logs to the test-logger.
	// And inspect them along the way, to get the RPC server address.
	logOut := logpipe.ToLoggerWithMinLevel(s.p.Logger().New("src", "stdout"), log.LevelWarn)
	logErr := logpipe.ToLoggerWithMinLevel(s.p.Logger().New("src", "stderr"), log.LevelWarn)
	userRPC := make(chan string, 1)
	onLogEntry := func(e logpipe.LogEntry) {
		switch e.LogMessage() {
		case "RPC server bound to address":
			userRPC <- "http://" + e.FieldValue("addr").(string)
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

	s.sub = NewSubProcess(s.p, stdOutLogs, stdErrLogs)
	err := s.sub.Start(s.execPath, s.args, s.env)
	s.p.Require().NoError(err, "Must start")

	var userRPCAddr string
	s.p.Require().NoError(tasks.Await(s.p.Ctx(), userRPC, &userRPCAddr), "need user RPC")

	s.userProxy.SetUpstream(ProxyAddr(s.p.Require(), userRPCAddr))
}

func (s *KonaSupervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sub == nil {
		s.p.Logger().Warn("kona-supervisor already stopped")
		return
	}
	err := s.sub.Stop(true)
	s.p.Require().NoError(err, "Must stop")
	s.sub = nil
}
