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
	"strconv"
	"time"
)

const (
	ContextKeyAuth  = "authorization"
	ContextKeyReqID = "req_id"
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
}

func NewServer(
	backendGroups map[string]*BackendGroup,
	wsBackendGroup *BackendGroup,
	wsMethodWhitelist *StringSet,
	rpcMethodMappings map[string]string,
	maxBodySize int64,
	authenticatedPaths map[string]string,
) *Server {
	return &Server{
		backendGroups:      backendGroups,
		wsBackendGroup:     wsBackendGroup,
		wsMethodWhitelist:  wsMethodWhitelist,
		rpcMethodMappings:  rpcMethodMappings,
		maxBodySize:        maxBodySize,
		authenticatedPaths: authenticatedPaths,
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
		s.rpcServer.Shutdown(context.Background())
	}
	if s.wsServer != nil {
		s.wsServer.Shutdown(context.Background())
	}
}

func (s *Server) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
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

	req, err := ParseRPCReq(io.LimitReader(r.Body, s.maxBodySize))
	if err != nil {
		log.Info("rejected request with bad rpc request", "source", "rpc", "err", err)
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
		writeRPCError(ctx, w, nil, err)
		return
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
		writeRPCError(ctx, w, req.ID, ErrMethodNotWhitelisted)
		return
	}

	backendRes, err := s.backendGroups[group].Forward(ctx, req)
	if err != nil {
		log.Error(
			"error forwarding RPC request",
			"method", req.Method,
			"req_id", GetReqID(ctx),
			"err", err,
		)
		writeRPCError(ctx, w, req.ID, err)
		return
	}

	writeRPCRes(ctx, w, backendRes)
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
			ContextKeyReqID,
			randStr(10),
		)
	}

	if authorization == "" || s.authenticatedPaths[authorization] == "" {
		log.Info("blocked unauthorized request", "authorization", authorization)
		httpResponseCodesTotal.WithLabelValues("401").Inc()
		w.WriteHeader(401)
		return nil
	}

	ctx := context.WithValue(r.Context(), ContextKeyAuth, s.authenticatedPaths[authorization])
	return context.WithValue(
		ctx,
		ContextKeyReqID,
		randStr(10),
	)
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
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	if err := enc.Encode(res); err != nil {
		log.Error("error writing rpc response", "err", err)
		RecordRPCError(ctx, BackendProxyd, MethodUnknown, err)
		return
	}
	httpResponseCodesTotal.WithLabelValues(strconv.Itoa(statusCode)).Inc()
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
