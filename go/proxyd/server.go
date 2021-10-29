package proxyd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	httpRequestsCtr = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "http_requests_total",
		Help:      "Count of total HTTP requests.",
	})

	httpRequestDurationHisto = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "proxyd",
		Name:      "http_request_duration_histogram_seconds",
		Help:      "Histogram of HTTP request durations.",
		Buckets: []float64{
			0,
			0.1,
			0.25,
			0.75,
			1,
		},
	})

	rpcRequestsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "rpc_requests_total",
		Help:      "Count of RPC requests.",
	}, []string{
		"method_name",
	})

	blockedRPCsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "blocked_rpc_requests_total",
		Help:      "Count of blocked RPC requests.",
	}, []string{
		"method_name",
	})

	rpcErrorsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "rpc_errors_total",
		Help:      "Count of RPC errors.",
	}, []string{
		"error_code",
	})
)

type RPCReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      *int            `json:"id"`
}

type RPCRes struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCErr     `json:"error,omitempty"`
	ID      *int        `json:"id"`
}

type RPCErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	mappings    *MethodMapping
	maxBodySize int64
}

func NewServer(mappings *MethodMapping, maxBodySize int64) *Server {
	return &Server{
		mappings:    mappings,
		maxBodySize: maxBodySize,
	}
}

func (s *Server) ListenAndServe(host string, port int) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/healthz", s.HandleHealthz).Methods("GET")
	hdlr.HandleFunc("/", s.HandleRPC).Methods("POST")
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Handler: instrumentedHdlr(hdlr),
		Addr:    addr,
	}
	log.Info("starting HTTP server", "addr", addr)
	return server.ListenAndServe()
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *Server) HandleRPC(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, s.maxBodySize))
	if err != nil {
		log.Error("error reading request body", "err", err)
		rpcErrorsCtr.WithLabelValues("-32700").Inc()
		writeRPCError(w, nil, -32700, "could not read request body")
		return
	}

	req := new(RPCReq)
	if err := json.Unmarshal(body, req); err != nil {
		rpcErrorsCtr.WithLabelValues("-32700").Inc()
		writeRPCError(w, nil, -32700, "invalid JSON")
		return
	}

	if req.JSONRPC != JSONRPCVersion {
		rpcErrorsCtr.WithLabelValues("-32600").Inc()
		writeRPCError(w, nil, -32600, "invalid json-rpc version")
		return
	}

	group, err := s.mappings.BackendGroupFor(req.Method)
	if err != nil {
		rpcErrorsCtr.WithLabelValues("-32601").Inc()
		blockedRPCsCtr.WithLabelValues(req.Method).Inc()
		log.Info("blocked request for non-whitelisted method", "method", req.Method)
		writeRPCError(w, req.ID, -32601, "method not found")
		return
	}

	backendRes, err := group.Forward(body)
	if err != nil {
		log.Error("error forwarding RPC request", "group", group.Name, "method", req.Method, "err", err)
		rpcErrorsCtr.WithLabelValues("-32603").Inc()
		msg := "error fetching data from upstream"
		if errors.Is(err, ErrBackendsInconsistent) {
			msg = ErrBackendsInconsistent.Error()
		}
		writeRPCError(w, req.ID, -32603, msg)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(backendRes); err != nil {
		log.Error("error encoding response", "err", err)
		return
	}
	rpcRequestsCtr.WithLabelValues(req.Method).Inc()
	log.Debug("forwarded RPC method", "method", req.Method, "group", group.Name)
}

func writeRPCError(w http.ResponseWriter, id *int, code int, msg string) {
	enc := json.NewEncoder(w)
	w.WriteHeader(200)
	body := &RPCRes{
		ID: id,
		Error: &RPCErr{
			Code:    code,
			Message: msg,
		},
	}
	if err := enc.Encode(body); err != nil {
		log.Error("error writing RPC error", "err", err)
	}
}

func instrumentedHdlr(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpRequestsCtr.Inc()
		start := time.Now()
		h.ServeHTTP(w, r)
		dur := time.Since(start)
		httpRequestDurationHisto.Observe(float64(dur) / float64(time.Second))
	}
}
