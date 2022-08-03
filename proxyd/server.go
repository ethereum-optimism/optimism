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
	"sync"
	"time"

	"github.com/sethvargo/go-limiter"
	"github.com/sethvargo/go-limiter/memorystore"
	"github.com/sethvargo/go-limiter/noopstore"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
)

const (
	ContextKeyAuth              = "authorization"
	ContextKeyReqID             = "req_id"
	ContextKeyXForwardedFor     = "x_forwarded_for"
	MaxBatchRPCCalls            = 100
	cacheStatusHdr              = "X-Proxyd-Cache-Status"
	defaultServerTimeout        = time.Second * 10
	maxRequestBodyLogLen        = 2000
	defaultMaxUpstreamBatchSize = 10
)

var emptyArrayResponse = json.RawMessage("[]")

type Server struct {
	backendGroups        map[string]*BackendGroup
	wsBackendGroup       *BackendGroup
	wsMethodWhitelist    *StringSet
	rpcMethodMappings    map[string]string
	maxBodySize          int64
	enableRequestLog     bool
	maxRequestBodyLogLen int
	authenticatedPaths   map[string]string
	timeout              time.Duration
	maxUpstreamBatchSize int
	upgrader             *websocket.Upgrader
	lim                  limiter.Store
	limConfig            RateLimitConfig
	limExemptOrigins     map[string]bool
	limExemptUserAgents  map[string]bool
	rpcServer            *http.Server
	wsServer             *http.Server
	cache                RPCCache
	srvMu                sync.Mutex
}

func NewServer(
	backendGroups map[string]*BackendGroup,
	wsBackendGroup *BackendGroup,
	wsMethodWhitelist *StringSet,
	rpcMethodMappings map[string]string,
	maxBodySize int64,
	authenticatedPaths map[string]string,
	timeout time.Duration,
	maxUpstreamBatchSize int,
	cache RPCCache,
	rateLimitConfig RateLimitConfig,
	enableRequestLog bool,
	maxRequestBodyLogLen int,
) (*Server, error) {
	if cache == nil {
		cache = &NoopRPCCache{}
	}

	if maxBodySize == 0 {
		maxBodySize = math.MaxInt64
	}

	if timeout == 0 {
		timeout = defaultServerTimeout
	}

	if maxUpstreamBatchSize == 0 {
		maxUpstreamBatchSize = defaultMaxUpstreamBatchSize
	}

	var lim limiter.Store
	limExemptOrigins := make(map[string]bool)
	limExemptUserAgents := make(map[string]bool)
	if rateLimitConfig.RatePerSecond > 0 {
		var err error
		lim, err = memorystore.New(&memorystore.Config{
			Tokens:   uint64(rateLimitConfig.RatePerSecond),
			Interval: time.Second,
		})
		if err != nil {
			return nil, err
		}

		for _, origin := range rateLimitConfig.ExemptOrigins {
			limExemptOrigins[strings.ToLower(origin)] = true
		}
		for _, agent := range rateLimitConfig.ExemptUserAgents {
			limExemptUserAgents[strings.ToLower(agent)] = true
		}
	} else {
		lim, _ = noopstore.New()
	}

	return &Server{
		backendGroups:        backendGroups,
		wsBackendGroup:       wsBackendGroup,
		wsMethodWhitelist:    wsMethodWhitelist,
		rpcMethodMappings:    rpcMethodMappings,
		maxBodySize:          maxBodySize,
		authenticatedPaths:   authenticatedPaths,
		timeout:              timeout,
		maxUpstreamBatchSize: maxUpstreamBatchSize,
		cache:                cache,
		enableRequestLog:     enableRequestLog,
		maxRequestBodyLogLen: maxRequestBodyLogLen,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
		},
		lim:                 lim,
		limConfig:           rateLimitConfig,
		limExemptOrigins:    limExemptOrigins,
		limExemptUserAgents: limExemptUserAgents,
	}, nil
}

func (s *Server) RPCListenAndServe(host string, port int) error {
	s.srvMu.Lock()
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
	s.srvMu.Unlock()
	return s.rpcServer.ListenAndServe()
}

func (s *Server) WSListenAndServe(host string, port int) error {
	s.srvMu.Lock()
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
	s.srvMu.Unlock()
	return s.wsServer.ListenAndServe()
}

func (s *Server) Shutdown() {
	s.srvMu.Lock()
	defer s.srvMu.Unlock()
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, s.timeout)
	defer cancel()

	exemptOrigin := s.limExemptOrigins[strings.ToLower(r.Header.Get("Origin"))]
	exemptUserAgent := s.limExemptUserAgents[strings.ToLower(r.Header.Get("User-Agent"))]
	var ok bool
	if exemptOrigin || exemptUserAgent {
		ok = true
	} else {
		// Use XFF in context since it will automatically be replaced by the remote IP
		xff := stripXFF(GetXForwardedFor(ctx))
		if xff == "" {
			log.Warn("rejecting request without XFF or remote IP")
			ok = false
		} else {
			_, _, _, ok, _ = s.lim.Take(ctx, xff)
		}
	}
	if !ok {
		rpcErr := ErrOverRateLimit.Clone()
		rpcErr.Message = s.limConfig.ErrorMessage
		writeRPCError(ctx, w, nil, rpcErr)
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

	if s.enableRequestLog {
		log.Info("Raw RPC request",
			"body", truncate(string(body), s.maxRequestBodyLogLen),
			"req_id", GetReqID(ctx),
			"auth", GetAuthCtx(ctx),
		)
	}

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

		batchRes, batchContainsCached, err := s.handleBatchRPC(ctx, reqs, true)
		if err == context.DeadlineExceeded {
			writeRPCError(ctx, w, nil, ErrGatewayTimeout)
			return
		}
		if err != nil {
			writeRPCError(ctx, w, nil, ErrInternal)
			return
		}

		setCacheHeader(w, batchContainsCached)
		writeBatchRPCRes(ctx, w, batchRes)
		return
	}

	rawBody := json.RawMessage(body)
	backendRes, cached, err := s.handleBatchRPC(ctx, []json.RawMessage{rawBody}, false)
	if err != nil {
		writeRPCError(ctx, w, nil, ErrInternal)
		return
	}
	setCacheHeader(w, cached)
	writeRPCRes(ctx, w, backendRes[0])
}

func (s *Server) handleBatchRPC(ctx context.Context, reqs []json.RawMessage, isBatch bool) ([]*RPCRes, bool, error) {
	// A request set is transformed into groups of batches.
	// Each batch group maps to a forwarded JSON-RPC batch request (subject to maxUpstreamBatchSize constraints)
	// A groupID is used to decouple Requests that have duplicate ID so they're not part of the same batch that's
	// forwarded to the backend. This is done to ensure that the order of JSON-RPC Responses match the Request order
	// as the backend MAY return Responses out of order.
	// NOTE: Duplicate request ids induces 1-sized JSON-RPC batches
	type batchGroup struct {
		groupID      int
		backendGroup string
	}

	responses := make([]*RPCRes, len(reqs))
	batches := make(map[batchGroup][]batchElem)
	ids := make(map[string]int, len(reqs))

	for i := range reqs {
		parsedReq, err := ParseRPCReq(reqs[i])
		if err != nil {
			log.Info("error parsing RPC call", "source", "rpc", "err", err)
			responses[i] = NewRPCErrorRes(nil, err)
			continue
		}

		if err := ValidateRPCReq(parsedReq); err != nil {
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
			responses[i] = NewRPCErrorRes(nil, err)
			continue
		}

		if parsedReq.Method == "eth_accounts" {
			RecordRPCForward(ctx, BackendProxyd, "eth_accounts", RPCRequestSourceHTTP)
			responses[i] = NewRPCRes(parsedReq.ID, emptyArrayResponse)
			continue
		}

		group := s.rpcMethodMappings[parsedReq.Method]
		if group == "" {
			// use unknown below to prevent DOS vector that fills up memory
			// with arbitrary method names.
			log.Info(
				"blocked request for non-whitelisted method",
				"source", "rpc",
				"req_id", GetReqID(ctx),
				"method", parsedReq.Method,
			)
			RecordRPCError(ctx, BackendProxyd, MethodUnknown, ErrMethodNotWhitelisted)
			responses[i] = NewRPCErrorRes(parsedReq.ID, ErrMethodNotWhitelisted)
			continue
		}

		id := string(parsedReq.ID)
		// If this is a duplicate Request ID, move the Request to a new batchGroup
		ids[id]++
		batchGroupID := ids[id]
		batchGroup := batchGroup{groupID: batchGroupID, backendGroup: group}
		batches[batchGroup] = append(batches[batchGroup], batchElem{parsedReq, i})
	}

	var cached bool
	for group, batch := range batches {
		var cacheMisses []batchElem

		for _, req := range batch {
			backendRes, _ := s.cache.GetRPC(ctx, req.Req)
			if backendRes != nil {
				responses[req.Index] = backendRes
				cached = true
			} else {
				cacheMisses = append(cacheMisses, req)
			}
		}

		// Create minibatches - each minibatch must be no larger than the maxUpstreamBatchSize
		numBatches := int(math.Ceil(float64(len(cacheMisses)) / float64(s.maxUpstreamBatchSize)))
		for i := 0; i < numBatches; i++ {
			if ctx.Err() == context.DeadlineExceeded {
				log.Info("short-circuiting batch RPC",
					"req_id", GetReqID(ctx),
					"auth", GetAuthCtx(ctx),
					"batch_index", i,
				)
				batchRPCShortCircuitsTotal.Inc()
				return nil, false, context.DeadlineExceeded
			}

			start := i * s.maxUpstreamBatchSize
			end := int(math.Min(float64(start+s.maxUpstreamBatchSize), float64(len(cacheMisses))))
			elems := cacheMisses[start:end]
			res, err := s.backendGroups[group.backendGroup].Forward(ctx, createBatchRequest(elems), isBatch)
			if err != nil {
				log.Error(
					"error forwarding RPC batch",
					"batch_size", len(elems),
					"backend_group", group,
					"err", err,
				)
				res = nil
				for _, elem := range elems {
					res = append(res, NewRPCErrorRes(elem.Req.ID, err))
				}
			}

			for i := range elems {
				responses[elems[i].Index] = res[i]

				// TODO(inphi): batch put these
				if res[i].Error == nil && res[i].Result != nil {
					if err := s.cache.PutRPC(ctx, elems[i].Req, res[i]); err != nil {
						log.Warn(
							"cache put error",
							"req_id", GetReqID(ctx),
							"err", err,
						)
					}
				}
			}
		}
	}

	return responses, cached, nil
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
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		ipPort := strings.Split(r.RemoteAddr, ":")
		if len(ipPort) == 2 {
			xff = ipPort[0]
		}
	}
	ctx := context.WithValue(r.Context(), ContextKeyXForwardedFor, xff) // nolint:staticcheck

	if s.authenticatedPaths == nil {
		// handle the edge case where auth is disabled
		// but someone sends in an auth key anyway
		if authorization != "" {
			log.Info("blocked authenticated request against unauthenticated proxy")
			httpResponseCodesTotal.WithLabelValues("404").Inc()
			w.WriteHeader(404)
			return nil
		}
	} else {
		if authorization == "" || s.authenticatedPaths[authorization] == "" {
			log.Info("blocked unauthorized request", "authorization", authorization)
			httpResponseCodesTotal.WithLabelValues("401").Inc()
			w.WriteHeader(401)
			return nil
		}

		ctx = context.WithValue(r.Context(), ContextKeyAuth, s.authenticatedPaths[authorization]) // nolint:staticcheck
	}

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

func truncate(str string, maxLen int) string {
	if maxLen == 0 {
		maxLen = maxRequestBodyLogLen
	}

	if len(str) > maxLen {
		return str[:maxLen] + "..."
	} else {
		return str
	}
}

type batchElem struct {
	Req   *RPCReq
	Index int
}

func createBatchRequest(elems []batchElem) []*RPCReq {
	batch := make([]*RPCReq, len(elems))
	for i := range elems {
		batch[i] = elems[i].Req
	}
	return batch
}
