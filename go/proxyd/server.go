package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
)

const (
	ContextKeyAuth          = "authorization"
	ContextKeyReqID         = "req_id"
	ContextKeyXForwardedFor = "x_forwarded_for"
	MaxBatchRPCCalls        = 100
	cacheStatusHdr          = "X-Proxyd-Cache-Status"
)

type Server struct {
	backendGroups      map[string]*BackendGroup
	wsBackendGroup     *BackendGroup
	wsMethodWhitelist  *StringSet
	rpcMethodMappings  map[string]string
	maxBodySize        int64
	authenticatedPaths map[string]string
	upgrader           *websocket.Upgrader
	rpcServer          *http.Server
	wsServer           *http.Server
	cache              RPCCache
}

func NewServer(
	backendGroups map[string]*BackendGroup,
	wsBackendGroup *BackendGroup,
	wsMethodWhitelist *StringSet,
	rpcMethodMappings map[string]string,
	maxBodySize int64,
	authenticatedPaths map[string]string,
	cache RPCCache,
) *Server {
	if cache == nil {
		cache = &NoopRPCCache{}
	}

	if maxBodySize == 0 {
		maxBodySize = math.MaxInt64
	}

	return &Server{
		backendGroups:      backendGroups,
		wsBackendGroup:     wsBackendGroup,
		wsMethodWhitelist:  wsMethodWhitelist,
		rpcMethodMappings:  rpcMethodMappings,
		maxBodySize:        maxBodySize,
		authenticatedPaths: authenticatedPaths,
		cache:              cache,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) RPCListenAndServe(host string, port int) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/healthz", s.HandleHealthz).Methods("GET")
	hdlr.HandleFunc("/", s.HandleRPC).Methods("POST")
	hdlr.HandleFunc("/{authorization}", s.HandleRPC).Methods("POST")
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	s.rpcServer = &http.Server{
		Handler: instrumentedHdlr(c.Handler(hdlr)),
		Addr:    addr,
	}
	log.Info("starting HTTP server", "addr", addr)
	return s.rpcServer.ListenAndServe()
}

func (s *Server) WSListenAndServe(host string, port int) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/", s.HandleWS)
	hdlr.HandleFunc("/{authorization}", s.HandleWS)
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	s.wsServer = &http.Server{
		Handler: instrumentedHdlr(c.Handler(hdlr)),
		Addr:    addr,
	}
	log.Info("starting WS server", "addr", addr)
	return s.wsServer.ListenAndServe()
}

func (s *Server) Shutdown() {
	if s.rpcServer != nil {
		_ = s.rpcServer.Shutdown(context.Background())
	}
	if s.wsServer != nil {
		_ = s.wsServer.Shutdown(context.Background())
	}
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

func (s *Server) HandleRPC(w http.ResponseWriter, r *http.Request) {
	ctx := populateContext(w, r, s.authenticatedPaths)
	if ctx == nil {
		return
	}

	handleRPC(ctx, w, r, s.maxBodySize, s.handleSingleRPC)
}

func (s *Server) handleSingleRPC(ctx context.Context, req *RPCReq) (*RPCRes, bool) {
	if err := ValidateRPCReq(req); err != nil {
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
		return NewRPCErrorRes(nil, err), false
	}

	group := s.rpcMethodMappings[req.Method]
	if group == "" {
		// use unknown below to prevent DOS vector that fills up memory
		// with arbitrary method names.
		log.Info(
			"blocked request for non-whitelisted method",
			"source", "rpc",
			"req_id", GetReqID(ctx),
			"method", req.Method,
		)
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, ErrMethodNotWhitelisted)
		return NewRPCErrorRes(req.ID, ErrMethodNotWhitelisted), false
	}

	var backendRes *RPCRes
	backendRes, err := s.cache.GetRPC(ctx, req)
	if err != nil {
		log.Warn(
			"cache lookup error",
			"req_id", GetReqID(ctx),
			"err", err,
		)
	}
	if backendRes != nil {
		return backendRes, true
	}

	backendRes, err = s.backendGroups[group].Forward(ctx, req)
	if err != nil {
		log.Error(
			"error forwarding RPC request",
			"method", req.Method,
			"req_id", GetReqID(ctx),
			"err", err,
		)
		return NewRPCErrorRes(req.ID, err), false
	}

	if backendRes.Error == nil && backendRes.Result != nil {
		if err = s.cache.PutRPC(ctx, req, backendRes); err != nil {
			log.Warn(
				"cache put error",
				"req_id", GetReqID(ctx),
				"err", err,
			)
		}
	}

	return backendRes, false
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	ctx := populateContext(w, r, s.authenticatedPaths)
	if ctx == nil {
		return
	}

	getProxier := func() (*WSProxier, error) {
		clientConn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("error upgrading client conn", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
			return nil, err
		}

		proxier, err := s.wsBackendGroup.ProxyWS(ctx, clientConn, s.wsMethodWhitelist)
		if err != nil {
			if errors.Is(err, ErrNoBackends) {
				RecordUnserviceableRequest(ctx, RPCRequestSourceWS)
			}
			log.Error("error dialing ws backend", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
			clientConn.Close()
			return nil, err
		}
		return proxier, nil
	}

	handleWS(ctx, w, r, getProxier)
}

func setCacheHeader(w http.ResponseWriter, cached bool) {
	if cached {
		w.Header().Set(cacheStatusHdr, "HIT")
	} else {
		w.Header().Set(cacheStatusHdr, "MISS")
	}
}

func writeRPCError(ctx context.Context, w http.ResponseWriter, id json.RawMessage, err error) {
	var res *RPCRes
	if r, ok := err.(*RPCErr); ok {
		res = NewRPCErrorRes(id, r)
	} else if errors.Is(err, ErrParseErr) {
		res = NewRPCErrorRes(id, ErrParseErr)
	} else {
		res = NewRPCErrorRes(id, ErrInternal)
	}
	writeRPCRes(ctx, w, res)
}

func writeRPCRes(ctx context.Context, w http.ResponseWriter, res *RPCRes) {
	statusCode := 200
	if res.IsError() && res.Error.HTTPErrorCode != 0 {
		statusCode = res.Error.HTTPErrorCode
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(statusCode)
	ww := &recordLenWriter{Writer: w}
	enc := json.NewEncoder(ww)
	if err := enc.Encode(res); err != nil {
		log.Error("error writing rpc response", "err", err)
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
		return
	}
	httpResponseCodesTotal.WithLabelValues(strconv.Itoa(statusCode)).Inc()
	RecordResponsePayloadSize(ctx, ww.Len)
}

func writeBatchRPCRes(ctx context.Context, w http.ResponseWriter, res []*RPCRes) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	ww := &recordLenWriter{Writer: w}
	enc := json.NewEncoder(ww)
	if err := enc.Encode(res); err != nil {
		log.Error("error writing batch rpc response", "err", err)
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
		return
	}
	RecordResponsePayloadSize(ctx, ww.Len)
}

func instrumentedHdlr(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respTimer := prometheus.NewTimer(httpRequestDurationSumm)
		h.ServeHTTP(w, r)
		respTimer.ObserveDuration()
	}
}

func GetAuthCtx(ctx context.Context) string {
	authUser, ok := ctx.Value(ContextKeyAuth).(string)
	if !ok {
		return "none"
	}

	return authUser
}

func GetReqID(ctx context.Context) string {
	reqId, ok := ctx.Value(ContextKeyReqID).(string)
	if !ok {
		return ""
	}
	return reqId
}

func GetXForwardedFor(ctx context.Context) string {
	xff, ok := ctx.Value(ContextKeyXForwardedFor).(string)
	if !ok {
		return ""
	}
	return xff
}

type recordLenWriter struct {
	io.Writer
	Len int
}

func (w *recordLenWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	w.Len += n
	return
}

type NoopRPCCache struct{}

func (n *NoopRPCCache) GetRPC(context.Context, *RPCReq) (*RPCRes, error) {
	return nil, nil
}

func (n *NoopRPCCache) PutRPC(context.Context, *RPCReq, *RPCRes) error {
	return nil
}
