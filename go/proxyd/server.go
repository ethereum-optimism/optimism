package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"io"
	"net/http"
	"time"
)

type Server struct {
	backends    *BackendGroup
	maxBodySize int64
	upgrader    *websocket.Upgrader
	server      *http.Server
}

func NewServer(
	backends *BackendGroup,
	maxBodySize int64,
) *Server {
	return &Server{
		backends:    backends,
		maxBodySize: maxBodySize,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) ListenAndServe(host string, port int) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/healthz", s.HandleHealthz).Methods("GET")
	hdlr.HandleFunc("/", s.HandleRPC).Methods("POST")
	hdlr.HandleFunc("/ws", s.HandleWS)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	s.server = &http.Server{
		Handler: instrumentedHdlr(c.Handler(hdlr)),
		Addr:    addr,
	}
	log.Info("starting HTTP server", "addr", addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() {
	s.server.Shutdown(context.Background())
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *Server) HandleRPC(w http.ResponseWriter, r *http.Request) {
	req, err := ParseRPCReq(io.LimitReader(r.Body, s.maxBodySize))
	if err != nil {
		log.Info("rejected request with bad rpc request", "source", "rpc", "err", err)
		RecordRPCError(SourceClient, err)
		writeRPCError(w, nil, err)
		return
	}

	backendRes, err := s.backends.Forward(req)
	if err != nil {
		if errors.Is(err, ErrNoBackends) {
			RecordUnserviceableRequest(RPCRequestSourceHTTP)
			RecordRPCError(SourceProxyd, err)
		} else if errors.Is(err, ErrMethodNotWhitelisted) {
			RecordRPCError(SourceClient, err)
		} else {
			RecordRPCError(SourceBackend, err)
		}
		log.Error("error forwarding RPC request", "method", req.Method, "err", err)
		writeRPCError(w, req.ID, err)
		return
	}
	if backendRes.IsError() {
		RecordRPCError(SourceBackend, backendRes.Error)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(backendRes); err != nil {
		log.Error("error encoding response", "err", err)
		RecordRPCError(SourceProxyd, err)
		writeRPCError(w, req.ID, err)
		return
	}

	log.Debug("forwarded RPC method", "method", req.Method)
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	clientConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("error upgrading client conn", "err", err)
		return
	}

	proxier, err := s.backends.ProxyWS(clientConn)
	if err != nil {
		if errors.Is(err, ErrNoBackends) {
			RecordUnserviceableRequest(RPCRequestSourceWS)
		}
		log.Error("error dialing ws backend", "err", err)
		clientConn.Close()
		return
	}

	activeClientWsConnsGauge.Inc()
	go func() {
		// Below call blocks so run it in a goroutine.
		if err := proxier.Proxy(); err != nil {
			log.Error("error proxying websocket", "err", err)
		}
		activeClientWsConnsGauge.Dec()
	}()
}

func writeRPCError(w http.ResponseWriter, id *int, err error) {
	enc := json.NewEncoder(w)
	w.WriteHeader(200)

	var body *RPCRes
	if r, ok := err.(*RPCErr); ok {
		body = NewRPCErrorRes(id, r)
	} else {
		body = NewRPCErrorRes(id, &RPCErr{
			Code:    JSONRPCErrorInternal,
			Message: "internal error",
		})
	}
	if err := enc.Encode(body); err != nil {
		log.Error("error writing rpc error", "err", err)
	}
}

func instrumentedHdlr(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpRequestsTotal.Inc()
		respTimer := prometheus.NewTimer(httpRequestDurationSumm)
		h.ServeHTTP(w, r)
		respTimer.ObserveDuration()
	}
}
