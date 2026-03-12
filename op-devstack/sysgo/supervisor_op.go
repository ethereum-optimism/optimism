package sysgo

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/log"

	supervisorConfig "github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor"
)

type OpSupervisor struct {
	mu sync.Mutex

	name    string
	userRPC string

	cfg    *supervisorConfig.Config
	p      devtest.CommonT
	logger log.Logger

	service *supervisor.SupervisorService

	proxy *tcpproxy.Proxy
}

var _ stack.Lifecycle = (*OpSupervisor)(nil)

func (s *OpSupervisor) UserRPC() string {
	return s.userRPC
}

func (s *OpSupervisor) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.service != nil {
		s.logger.Warn("Supervisor already started")
		return
	}

	if s.proxy == nil {
		s.proxy = tcpproxy.New(s.logger.New("proxy", "supervisor"))
		s.p.Require().NoError(s.proxy.Start())
		s.p.Cleanup(func() {
			s.proxy.Close()
		})
		s.userRPC = "http://" + s.proxy.Addr()
	}

	super, err := supervisor.SupervisorFromConfig(context.Background(), s.cfg, s.logger)
	s.p.Require().NoError(err)

	s.service = super
	s.logger.Info("Starting supervisor")
	err = super.Start(context.Background())
	s.p.Require().NoError(err, "supervisor failed to start")
	s.logger.Info("Started supervisor")
	s.proxy.SetUpstream(ProxyAddr(s.p.Require(), super.RPC()))
}

func (s *OpSupervisor) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.service == nil {
		s.logger.Warn("Supervisor already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	s.logger.Info("Closing supervisor")
	closeErr := s.service.Stop(ctx)
	s.logger.Info("Closed supervisor", "err", closeErr)

	s.service = nil
}
