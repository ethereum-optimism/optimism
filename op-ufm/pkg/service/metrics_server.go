package service

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	ctx    context.Context
	server *http.Server
}

func (m *MetricsServer) Start(ctx context.Context, addr string) error {
	server := &http.Server{
		Handler: promhttp.Handler(),
		Addr:    addr,
	}
	m.server = server
	m.ctx = ctx
	return m.server.ListenAndServe()
}

func (m *MetricsServer) Shutdown() error {
	return m.server.Shutdown(m.ctx)
}
