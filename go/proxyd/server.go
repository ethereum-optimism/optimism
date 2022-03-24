package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
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
	ctx := s.populateContext(w, r)
	if ctx == nil {
		return
	}

	log.Info(
		"received RPC request",
		"req_id", GetReqID(ctx),
		"auth", GetAuthCtx(ctx),
		"user_agent", r.Header.Get("user-agent"),
	)

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, s.maxBodySize))
	if err != nil {
		log.Error("error reading request body", "err", err)
		writeRPCError(ctx, w, nil, ErrInternal)
		return
	}
	RecordRequestPayloadSize(ctx, len(body))

	if IsBatch(body) {
		reqs, err := ParseBatchRPCReq(body)
		if err != nil {
			log.Error("error parsing batch RPC request", "err", err)
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
			writeRPCError(ctx, w, nil, ErrParseErr)
			return
		}

		if len(reqs) > MaxBatchRPCCalls {
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, ErrTooManyBatchRequests)
			writeRPCError(ctx, w, nil, ErrTooManyBatchRequests)
			return
		}

		if len(reqs) == 0 {
			writeRPCError(ctx, w, nil, ErrInvalidRequest("must specify at least one batch call"))
			return
		}

		batchRes := make([]*RPCRes, len(reqs))
		var batchContainsCached bool
		for i := 0; i < len(reqs); i++ {
			req, err := ParseRPCReq(reqs[i])
			if err != nil {
				log.Info("error parsing RPC call", "source", "rpc", "err", err)
				batchRes[i] = NewRPCErrorRes(nil, err)
				continue
			}

			var cached bool
			batchRes[i], cached = s.handleSingleRPC(ctx, req)
			if cached {
				batchContainsCached = true
			}
		}

		setCacheHeader(w, batchContainsCached)
		writeBatchRPCRes(ctx, w, batchRes)
		return
	}

	req, err := ParseRPCReq(body)
	if err != nil {
		log.Info("error parsing RPC call", "source", "rpc", "err", err)
		writeRPCError(ctx, w, nil, err)
		return
	}

	backendRes, cached := s.handleSingleRPC(ctx, req)
	setCacheHeader(w, cached)
	writeRPCRes(ctx, w, backendRes)
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
	ctx := s.populateContext(w, r)
	if ctx == nil {
		return
	}

	log.Info("received WS connection", "req_id", GetReqID(ctx))

	clientConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("error upgrading client conn", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
		return
	}

	proxier, err := s.wsBackendGroup.ProxyWS(ctx, clientConn, s.wsMethodWhitelist)
	if err != nil {
		if errors.Is(err, ErrNoBackends) {
			RecordUnserviceableRequest(ctx, RPCRequestSourceWS)
		}
		log.Error("error dialing ws backend", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
		clientConn.Close()
		return
	}

	activeClientWsConnsGauge.WithLabelValues(GetAuthCtx(ctx)).Inc()
	go func() {
		// Below call blocks so run it in a goroutine.
		if err := proxier.Proxy(ctx); err != nil {
			log.Error("error proxying websocket", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx), "err", err)
		}
		activeClientWsConnsGauge.WithLabelValues(GetAuthCtx(ctx)).Dec()
	}()

	log.Info("accepted WS connection", "auth", GetAuthCtx(ctx), "req_id", GetReqID(ctx))
}

func (s *Server) populateContext(w http.ResponseWriter, r *http.Request) context.Context {
	vars := mux.Vars(r)
	authorization := vars["authorization"]

	if s.authenticatedPaths == nil {
		// handle the edge case where auth is disabled
		// but someone sends in an auth key anyway
		if authorization != "" {
			log.Info("blocked authenticated request against unauthenticated proxy")
			httpResponseCodesTotal.WithLabelValues("404").Inc()
			w.WriteHeader(404)
			return nil
		}
		return context.WithValue(
			r.Context(),
			ContextKeyReqID, // nolint:staticcheck
			randStr(10),
		)
	}

	if authorization == "" || s.authenticatedPaths[authorization] == "" {
		log.Info("blocked unauthorized request", "authorization", authorization)
		httpResponseCodesTotal.WithLabelValues("401").Inc()
		w.WriteHeader(401)
		return nil
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		ipPort := strings.Split(r.RemoteAddr, ":")
		if len(ipPort) == 2 {
			xff = ipPort[0]
		}
	}

	ctx := context.WithValue(r.Context(), ContextKeyAuth, s.authenticatedPaths[authorization]) // nolint:staticcheck
	ctx = context.WithValue(ctx, ContextKeyXForwardedFor, xff)                                 // nolint:staticcheck
	return context.WithValue(
		ctx,
		ContextKeyReqID, // nolint:staticcheck
		randStr(10),
	)
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
