package resources

import (
	"context"
	"net"
	"net/http"
	"strconv"

	gethlog "github.com/ethereum/go-ethereum/log"
)

// MetricsService encapsulates an HTTP server that serves the MetricsRouter.
type MetricsService struct {
	log    gethlog.Logger
	server *http.Server
}

// NewMetricsService constructs a metrics HTTP server bound to the given address/port using the provided handler.
func NewMetricsService(log gethlog.Logger, listenAddr string, port int, handler http.Handler) *MetricsService {
	addr := net.JoinHostPort(listenAddr, strconv.Itoa(port))
	return &MetricsService{
		log:    log,
		server: &http.Server{Addr: addr, Handler: handler},
	}
}

// Start begins serving metrics in a background goroutine. If the server exits with an error,
// the optional onError callback is invoked.
func (s *MetricsService) Start(onError func(error)) {
	if s.server == nil {
		return
	}
	go func() {
		s.log.Info("starting metrics router server", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("metrics server error", "error", err)
			if onError != nil {
				onError(err)
			}
		}
	}()
}

// Stop gracefully shuts down the metrics server.
func (s *MetricsService) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
