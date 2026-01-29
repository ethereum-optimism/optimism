package rpc

import (
	"context"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
)

var _ cliapp.Lifecycle = &Service{}

type Service struct {
	log     log.Logger
	srv     *Server
	stopped atomic.Bool
}

func NewService(log log.Logger, srv *Server) *Service {
	return &Service{log: log, srv: srv, stopped: atomic.Bool{}}
}

func (s *Service) Start(_ context.Context) error {
	s.log.Info("starting rpc server")
	return s.srv.Start()
}

func (s *Service) Stop(_ context.Context) error {
	if s.stopped.Load() {
		return nil
	}

	s.log.Info("stopping rpc server")
	err := s.srv.Stop()
	if err == nil {
		s.stopped.Store(true)
	}

	return err
}

func (s *Service) Stopped() bool {
	return s.stopped.Load()
}
