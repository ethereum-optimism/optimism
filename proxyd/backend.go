package proxyd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sw "github.com/ethereum-optimism/optimism/proxyd/pkg/avg-sliding-window"

	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/semaphore"
)

const (
	JSONRPCVersion       = "2.0"
	JSONRPCErrorInternal = -32000
)

var (
	ErrParseErr = &RPCErr{
		Code:          -32700,
		Message:       "parse error",
		HTTPErrorCode: 400,
	}
	ErrInternal = &RPCErr{
		Code:          JSONRPCErrorInternal,
		Message:       "internal error",
		HTTPErrorCode: 500,
	}
	ErrMethodNotWhitelisted = &RPCErr{
		Code:          JSONRPCErrorInternal - 1,
		Message:       "rpc method is not whitelisted",
		HTTPErrorCode: 403,
	}
	ErrBackendOffline = &RPCErr{
		Code:          JSONRPCErrorInternal - 10,
		Message:       "backend offline",
		HTTPErrorCode: 503,
	}
	ErrNoBackends = &RPCErr{
		Code:          JSONRPCErrorInternal - 11,
		Message:       "no backends available for method",
		HTTPErrorCode: 503,
	}
	ErrBackendOverCapacity = &RPCErr{
		Code:          JSONRPCErrorInternal - 12,
		Message:       "backend is over capacity",
		HTTPErrorCode: 429,
	}
	ErrBackendBadResponse = &RPCErr{
		Code:          JSONRPCErrorInternal - 13,
		Message:       "backend returned an invalid response",
		HTTPErrorCode: 500,
	}
	ErrTooManyBatchRequests = &RPCErr{
		Code:    JSONRPCErrorInternal - 14,
		Message: "too many RPC calls in batch request",
	}
	ErrGatewayTimeout = &RPCErr{
		Code:          JSONRPCErrorInternal - 15,
		Message:       "gateway timeout",
		HTTPErrorCode: 504,
	}
	ErrOverRateLimit = &RPCErr{
		Code:          JSONRPCErrorInternal - 16,
		Message:       "over rate limit",
		HTTPErrorCode: 429,
	}
	ErrOverSenderRateLimit = &RPCErr{
		Code:          JSONRPCErrorInternal - 17,
		Message:       "sender is over rate limit",
		HTTPErrorCode: 429,
	}
	ErrNotHealthy = &RPCErr{
		Code:          JSONRPCErrorInternal - 18,
		Message:       "backend is currently not healthy to serve traffic",
		HTTPErrorCode: 429,
	}

	ErrBackendUnexpectedJSONRPC = errors.New("backend returned an unexpected JSON-RPC response")
)

func ErrInvalidRequest(msg string) *RPCErr {
	return &RPCErr{
		Code:          -32600,
		Message:       msg,
		HTTPErrorCode: 400,
	}
}

func ErrInvalidParams(msg string) *RPCErr {
	return &RPCErr{
		Code:          -32602,
		Message:       msg,
		HTTPErrorCode: 400,
	}
}

type Backend struct {
	Name                 string
	rpcURL               string
	wsURL                string
	authUsername         string
	authPassword         string
	rateLimiter          BackendRateLimiter
	client               *LimitedHTTPClient
	dialer               *websocket.Dialer
	maxRetries           int
	maxResponseSize      int64
	maxRPS               int
	maxWSConns           int
	outOfServiceInterval time.Duration
	stripTrailingXFF     bool
	proxydIP             string

	maxDegradedLatencyThreshold time.Duration
	maxLatencyThreshold         time.Duration
	maxErrorRateThreshold       float64

	latencySlidingWindow         *sw.AvgSlidingWindow
	networkRequestsSlidingWindow *sw.AvgSlidingWindow
	networkErrorsSlidingWindow   *sw.AvgSlidingWindow
}

type BackendOpt func(b *Backend)

func WithBasicAuth(username, password string) BackendOpt {
	return func(b *Backend) {
		b.authUsername = username
		b.authPassword = password
	}
}

func WithTimeout(timeout time.Duration) BackendOpt {
	return func(b *Backend) {
		b.client.Timeout = timeout
	}
}

func WithMaxRetries(retries int) BackendOpt {
	return func(b *Backend) {
		b.maxRetries = retries
	}
}

func WithMaxResponseSize(size int64) BackendOpt {
	return func(b *Backend) {
		b.maxResponseSize = size
	}
}

func WithOutOfServiceDuration(interval time.Duration) BackendOpt {
	return func(b *Backend) {
		b.outOfServiceInterval = interval
	}
}

func WithMaxRPS(maxRPS int) BackendOpt {
	return func(b *Backend) {
		b.maxRPS = maxRPS
	}
}

func WithMaxWSConns(maxConns int) BackendOpt {
	return func(b *Backend) {
		b.maxWSConns = maxConns
	}
}

func WithTLSConfig(tlsConfig *tls.Config) BackendOpt {
	return func(b *Backend) {
		if b.client.Transport == nil {
			b.client.Transport = &http.Transport{}
		}
		b.client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
	}
}

func WithStrippedTrailingXFF() BackendOpt {
	return func(b *Backend) {
		b.stripTrailingXFF = true
	}
}

func WithProxydIP(ip string) BackendOpt {
	return func(b *Backend) {
		b.proxydIP = ip
	}
}

func WithMaxLatencyThreshold(maxLatencyThreshold time.Duration) BackendOpt {
	return func(b *Backend) {
		b.maxLatencyThreshold = maxLatencyThreshold
	}
}

func WithMaxErrorRateThreshold(maxErrorRateThreshold float64) BackendOpt {
	return func(b *Backend) {
		b.maxErrorRateThreshold = maxErrorRateThreshold
	}
}

func NewBackend(
	name string,
	rpcURL string,
	wsURL string,
	rateLimiter BackendRateLimiter,
	rpcSemaphore *semaphore.Weighted,
	opts ...BackendOpt,
) *Backend {
	backend := &Backend{
		Name:            name,
		rpcURL:          rpcURL,
		wsURL:           wsURL,
		rateLimiter:     rateLimiter,
		maxResponseSize: math.MaxInt64,
		client: &LimitedHTTPClient{
			Client:      http.Client{Timeout: 5 * time.Second},
			sem:         rpcSemaphore,
			backendName: name,
		},
		dialer: &websocket.Dialer{},

		maxLatencyThreshold:         10 * time.Second,
		maxDegradedLatencyThreshold: 5 * time.Second,
		maxErrorRateThreshold:       0.5,

		latencySlidingWindow:         sw.NewSlidingWindow(),
		networkRequestsSlidingWindow: sw.NewSlidingWindow(),
		networkErrorsSlidingWindow:   sw.NewSlidingWindow(),
	}

	for _, opt := range opts {
		opt(backend)
	}

	if !backend.stripTrailingXFF && backend.proxydIP == "" {
		log.Warn("proxied requests' XFF header will not contain the proxyd ip address")
	}

	return backend
}

func (b *Backend) Forward(ctx context.Context, reqs []*RPCReq, isBatch bool) ([]*RPCRes, error) {
	if !b.Online() {
		RecordBatchRPCError(ctx, b.Name, reqs, ErrBackendOffline)
		return nil, ErrBackendOffline
	}
	if b.IsRateLimited() {
		RecordBatchRPCError(ctx, b.Name, reqs, ErrBackendOverCapacity)
		return nil, ErrBackendOverCapacity
	}

	var lastError error
	// <= to account for the first attempt not technically being
	// a retry
	for i := 0; i <= b.maxRetries; i++ {
		RecordBatchRPCForward(ctx, b.Name, reqs, RPCRequestSourceHTTP)
		metricLabelMethod := reqs[0].Method
		if isBatch {
			metricLabelMethod = "<batch>"
		}
		timer := prometheus.NewTimer(
			rpcBackendRequestDurationSumm.WithLabelValues(
				b.Name,
				metricLabelMethod,
				strconv.FormatBool(isBatch),
			),
		)

		res, err := b.doForward(ctx, reqs, isBatch)
		switch err {
		case nil: // do nothing
		// ErrBackendUnexpectedJSONRPC occurs because infura responds with a single JSON-RPC object
		// to a batch request whenever any Request Object in the batch would induce a partial error.
		// We don't label the backend offline in this case. But the error is still returned to
		// callers so failover can occur if needed.
		case ErrBackendUnexpectedJSONRPC:
			log.Debug(
				"Received unexpected JSON-RPC response",
				"name", b.Name,
				"req_id", GetReqID(ctx),
				"err", err,
			)
		default:
			lastError = err
			log.Warn(
				"backend request failed, trying again",
				"name", b.Name,
				"req_id", GetReqID(ctx),
				"err", err,
			)
			timer.ObserveDuration()
			RecordBatchRPCError(ctx, b.Name, reqs, err)
			sleepContext(ctx, calcBackoff(i))
			continue
		}
		timer.ObserveDuration()

		MaybeRecordErrorsInRPCRes(ctx, b.Name, reqs, res)
		return res, err
	}

	b.setOffline()
	return nil, wrapErr(lastError, "permanent error forwarding request")
}

func (b *Backend) ProxyWS(clientConn *websocket.Conn, methodWhitelist *StringSet) (*WSProxier, error) {
	if !b.Online() {
		return nil, ErrBackendOffline
	}
	if b.IsWSSaturated() {
		return nil, ErrBackendOverCapacity
	}

	backendConn, _, err := b.dialer.Dial(b.wsURL, nil) // nolint:bodyclose
	if err != nil {
		b.setOffline()
		if err := b.rateLimiter.DecBackendWSConns(b.Name); err != nil {
			log.Error("error decrementing backend ws conns", "name", b.Name, "err", err)
		}
		return nil, wrapErr(err, "error dialing backend")
	}

	activeBackendWsConnsGauge.WithLabelValues(b.Name).Inc()
	return NewWSProxier(b, clientConn, backendConn, methodWhitelist), nil
}

func (b *Backend) Online() bool {
	online, err := b.rateLimiter.IsBackendOnline(b.Name)
	if err != nil {
		log.Warn(
			"error getting backend availability, assuming it is offline",
			"name", b.Name,
			"err", err,
		)
		return false
	}
	return online
}

func (b *Backend) IsRateLimited() bool {
	if b.maxRPS == 0 {
		return false
	}

	usedLimit, err := b.rateLimiter.IncBackendRPS(b.Name)
	if err != nil {
		log.Error(
			"error getting backend used rate limit, assuming limit is exhausted",
			"name", b.Name,
			"err", err,
		)
		return true
	}

	return b.maxRPS < usedLimit
}

func (b *Backend) IsWSSaturated() bool {
	if b.maxWSConns == 0 {
		return false
	}

	incremented, err := b.rateLimiter.IncBackendWSConns(b.Name, b.maxWSConns)
	if err != nil {
		log.Error(
			"error getting backend used ws conns, assuming limit is exhausted",
			"name", b.Name,
			"err", err,
		)
		return true
	}

	return !incremented
}

func (b *Backend) setOffline() {
	err := b.rateLimiter.SetBackendOffline(b.Name, b.outOfServiceInterval)
	if err != nil {
		log.Warn(
			"error setting backend offline",
			"name", b.Name,
			"err", err,
		)
	}
}

// ForwardRPC makes a call directly to a backend and populate the response into `res`
func (b *Backend) ForwardRPC(ctx context.Context, res *RPCRes, id string, method string, params ...any) error {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}

	rpcReq := RPCReq{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  jsonParams,
		ID:      []byte(id),
	}

	slicedRes, err := b.doForward(ctx, []*RPCReq{&rpcReq}, false)
	if err != nil {
		return err
	}

	if len(slicedRes) != 1 {
		return fmt.Errorf("unexpected response len for non-batched request (len != 1)")
	}
	if slicedRes[0].IsError() {
		return fmt.Errorf(slicedRes[0].Error.Error())
	}

	*res = *(slicedRes[0])
	return nil
}

func (b *Backend) doForward(ctx context.Context, rpcReqs []*RPCReq, isBatch bool) ([]*RPCRes, error) {
	// we are concerned about network error rates, so we record 1 request independently of how many are in the batch
	b.networkRequestsSlidingWindow.Incr()

	isSingleElementBatch := len(rpcReqs) == 1

	// Single element batches are unwrapped before being sent
	// since Alchemy handles single requests better than batches.

	var body []byte
	if isSingleElementBatch {
		body = mustMarshalJSON(rpcReqs[0])
	} else {
		body = mustMarshalJSON(rpcReqs)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.rpcURL, bytes.NewReader(body))
	if err != nil {
		b.networkErrorsSlidingWindow.Incr()
		return nil, wrapErr(err, "error creating backend request")
	}

	if b.authPassword != "" {
		httpReq.SetBasicAuth(b.authUsername, b.authPassword)
	}

	xForwardedFor := GetXForwardedFor(ctx)
	if b.stripTrailingXFF {
		xForwardedFor = stripXFF(xForwardedFor)
	} else if b.proxydIP != "" {
		xForwardedFor = fmt.Sprintf("%s, %s", xForwardedFor, b.proxydIP)
	}

	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("X-Forwarded-For", xForwardedFor)

	start := time.Now()
	httpRes, err := b.client.DoLimited(httpReq)
	if err != nil {
		b.networkErrorsSlidingWindow.Incr()
		return nil, wrapErr(err, "error in backend request")
	}

	metricLabelMethod := rpcReqs[0].Method
	if isBatch {
		metricLabelMethod = "<batch>"
	}
	rpcBackendHTTPResponseCodesTotal.WithLabelValues(
		GetAuthCtx(ctx),
		b.Name,
		metricLabelMethod,
		strconv.Itoa(httpRes.StatusCode),
		strconv.FormatBool(isBatch),
	).Inc()

	// Alchemy returns a 400 on bad JSONs, so handle that case
	if httpRes.StatusCode != 200 && httpRes.StatusCode != 400 {
		b.networkErrorsSlidingWindow.Incr()
		return nil, fmt.Errorf("response code %d", httpRes.StatusCode)
	}

	defer httpRes.Body.Close()
	resB, err := io.ReadAll(io.LimitReader(httpRes.Body, b.maxResponseSize))
	if err != nil {
		b.networkErrorsSlidingWindow.Incr()
		return nil, wrapErr(err, "error reading response body")
	}

	var res []*RPCRes
	if isSingleElementBatch {
		var singleRes RPCRes
		if err := json.Unmarshal(resB, &singleRes); err != nil {
			return nil, ErrBackendBadResponse
		}
		res = []*RPCRes{
			&singleRes,
		}
	} else {
		if err := json.Unmarshal(resB, &res); err != nil {
			// Infura may return a single JSON-RPC response if, for example, the batch contains a request for an unsupported method
			if responseIsNotBatched(resB) {
				b.networkErrorsSlidingWindow.Incr()
				return nil, ErrBackendUnexpectedJSONRPC
			}
			b.networkErrorsSlidingWindow.Incr()
			return nil, ErrBackendBadResponse
		}
	}

	if len(rpcReqs) != len(res) {
		b.networkErrorsSlidingWindow.Incr()
		return nil, ErrBackendUnexpectedJSONRPC
	}

	// capture the HTTP status code in the response. this will only
	// ever be 400 given the status check on line 318 above.
	if httpRes.StatusCode != 200 {
		for _, res := range res {
			res.Error.HTTPErrorCode = httpRes.StatusCode
		}
	}
	duration := time.Since(start)
	b.latencySlidingWindow.Add(float64(duration))

	sortBatchRPCResponse(rpcReqs, res)
	return res, nil
}

// IsHealthy checks if the backend is able to serve traffic, based on dynamic parameters
func (b *Backend) IsHealthy() bool {
	errorRate := b.networkErrorsSlidingWindow.Sum() / b.networkRequestsSlidingWindow.Sum()
	avgLatency := time.Duration(b.latencySlidingWindow.Avg())
	if errorRate >= b.maxErrorRateThreshold {
		return false
	}
	if avgLatency >= b.maxLatencyThreshold {
		return false
	}
	return true
}

// IsDegraded checks if the backend is serving traffic in a degraded state (i.e. used as a last resource)
func (b *Backend) IsDegraded() bool {
	avgLatency := time.Duration(b.latencySlidingWindow.Avg())
	return avgLatency >= b.maxDegradedLatencyThreshold
}

func responseIsNotBatched(b []byte) bool {
	var r RPCRes
	return json.Unmarshal(b, &r) == nil
}

// sortBatchRPCResponse sorts the RPCRes slice according to the position of its corresponding ID in the RPCReq slice
func sortBatchRPCResponse(req []*RPCReq, res []*RPCRes) {
	pos := make(map[string]int, len(req))
	for i, r := range req {
		key := string(r.ID)
		if _, ok := pos[key]; ok {
			panic("bug! detected requests with duplicate IDs")
		}
		pos[key] = i
	}

	sort.Slice(res, func(i, j int) bool {
		l := res[i].ID
		r := res[j].ID
		return pos[string(l)] < pos[string(r)]
	})
}

type BackendGroup struct {
	Name      string
	Backends  []*Backend
	Consensus *ConsensusPoller
}

func (b *BackendGroup) Forward(ctx context.Context, rpcReqs []*RPCReq, isBatch bool) ([]*RPCRes, error) {
	if len(rpcReqs) == 0 {
		return nil, nil
	}

	rpcRequestsTotal.Inc()

	for _, back := range b.Backends {
		res, err := back.Forward(ctx, rpcReqs, isBatch)
		if errors.Is(err, ErrMethodNotWhitelisted) {
			return nil, err
		}
		if errors.Is(err, ErrBackendOffline) {
			log.Warn(
				"skipping offline backend",
				"name", back.Name,
				"auth", GetAuthCtx(ctx),
				"req_id", GetReqID(ctx),
			)
			continue
		}
		if errors.Is(err, ErrBackendOverCapacity) {
			log.Warn(
				"skipping over-capacity backend",
				"name", back.Name,
				"auth", GetAuthCtx(ctx),
				"req_id", GetReqID(ctx),
			)
			continue
		}
		if err != nil {
			log.Error(
				"error forwarding request to backend",
				"name", back.Name,
				"req_id", GetReqID(ctx),
				"auth", GetAuthCtx(ctx),
				"err", err,
			)
			continue
		}
		return res, nil
	}

	RecordUnserviceableRequest(ctx, RPCRequestSourceHTTP)
	return nil, ErrNoBackends
}

func (b *BackendGroup) ProxyWS(ctx context.Context, clientConn *websocket.Conn, methodWhitelist *StringSet) (*WSProxier, error) {
	for _, back := range b.Backends {
		proxier, err := back.ProxyWS(clientConn, methodWhitelist)
		if errors.Is(err, ErrBackendOffline) {
			log.Warn(
				"skipping offline backend",
				"name", back.Name,
				"req_id", GetReqID(ctx),
				"auth", GetAuthCtx(ctx),
			)
			continue
		}
		if errors.Is(err, ErrBackendOverCapacity) {
			log.Warn(
				"skipping over-capacity backend",
				"name", back.Name,
				"req_id", GetReqID(ctx),
				"auth", GetAuthCtx(ctx),
			)
			continue
		}
		if err != nil {
			log.Warn(
				"error dialing ws backend",
				"name", back.Name,
				"req_id", GetReqID(ctx),
				"auth", GetAuthCtx(ctx),
				"err", err,
			)
			continue
		}
		return proxier, nil
	}

	return nil, ErrNoBackends
}

func calcBackoff(i int) time.Duration {
	jitter := float64(rand.Int63n(250))
	ms := math.Min(math.Pow(2, float64(i))*1000+jitter, 3000)
	return time.Duration(ms) * time.Millisecond
}

type WSProxier struct {
	backend         *Backend
	clientConn      *websocket.Conn
	backendConn     *websocket.Conn
	methodWhitelist *StringSet
	clientConnMu    sync.Mutex
}

func NewWSProxier(backend *Backend, clientConn, backendConn *websocket.Conn, methodWhitelist *StringSet) *WSProxier {
	return &WSProxier{
		backend:         backend,
		clientConn:      clientConn,
		backendConn:     backendConn,
		methodWhitelist: methodWhitelist,
	}
}

func (w *WSProxier) Proxy(ctx context.Context) error {
	errC := make(chan error, 2)
	go w.clientPump(ctx, errC)
	go w.backendPump(ctx, errC)
	err := <-errC
	w.close()
	return err
}

func (w *WSProxier) clientPump(ctx context.Context, errC chan error) {
	for {
		// Block until we get a message.
		msgType, msg, err := w.clientConn.ReadMessage()
		if err != nil {
			errC <- err
			if err := w.backendConn.WriteMessage(websocket.CloseMessage, formatWSError(err)); err != nil {
				log.Error("error writing backendConn message", "err", err)
			}
			return
		}

		RecordWSMessage(ctx, w.backend.Name, SourceClient)

		// Route control messages to the backend. These don't
		// count towards the total RPC requests count.
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			err := w.backendConn.WriteMessage(msgType, msg)
			if err != nil {
				errC <- err
				return
			}
			continue
		}

		rpcRequestsTotal.Inc()

		// Don't bother sending invalid requests to the backend,
		// just handle them here.
		req, err := w.prepareClientMsg(msg)
		if err != nil {
			var id json.RawMessage
			method := MethodUnknown
			if req != nil {
				id = req.ID
				method = req.Method
			}
			log.Info(
				"error preparing client message",
				"auth", GetAuthCtx(ctx),
				"req_id", GetReqID(ctx),
				"err", err,
			)
			msg = mustMarshalJSON(NewRPCErrorRes(id, err))
			RecordRPCError(ctx, BackendProxyd, method, err)

			// Send error response to client
			err = w.writeClientConn(msgType, msg)
			if err != nil {
				errC <- err
				return
			}
			continue
		}

		// Send eth_accounts requests directly to the client
		if req.Method == "eth_accounts" {
			msg = mustMarshalJSON(NewRPCRes(req.ID, emptyArrayResponse))
			RecordRPCForward(ctx, BackendProxyd, "eth_accounts", RPCRequestSourceWS)
			err = w.writeClientConn(msgType, msg)
			if err != nil {
				errC <- err
				return
			}
			continue
		}

		RecordRPCForward(ctx, w.backend.Name, req.Method, RPCRequestSourceWS)
		log.Info(
			"forwarded WS message to backend",
			"method", req.Method,
			"auth", GetAuthCtx(ctx),
			"req_id", GetReqID(ctx),
		)

		err = w.backendConn.WriteMessage(msgType, msg)
		if err != nil {
			errC <- err
			return
		}
	}
}

func (w *WSProxier) backendPump(ctx context.Context, errC chan error) {
	for {
		// Block until we get a message.
		msgType, msg, err := w.backendConn.ReadMessage()
		if err != nil {
			errC <- err
			if err := w.writeClientConn(websocket.CloseMessage, formatWSError(err)); err != nil {
				log.Error("error writing clientConn message", "err", err)
			}
			return
		}

		RecordWSMessage(ctx, w.backend.Name, SourceBackend)

		// Route control messages directly to the client.
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			err := w.writeClientConn(msgType, msg)
			if err != nil {
				errC <- err
				return
			}
			continue
		}

		res, err := w.parseBackendMsg(msg)
		if err != nil {
			var id json.RawMessage
			if res != nil {
				id = res.ID
			}
			msg = mustMarshalJSON(NewRPCErrorRes(id, err))
			log.Info("backend responded with error", "err", err)
		} else {
			if res.IsError() {
				log.Info(
					"backend responded with RPC error",
					"code", res.Error.Code,
					"msg", res.Error.Message,
					"source", "ws",
					"auth", GetAuthCtx(ctx),
					"req_id", GetReqID(ctx),
				)
				RecordRPCError(ctx, w.backend.Name, MethodUnknown, res.Error)
			} else {
				log.Info(
					"forwarded WS message to client",
					"auth", GetAuthCtx(ctx),
					"req_id", GetReqID(ctx),
				)
			}
		}

		err = w.writeClientConn(msgType, msg)
		if err != nil {
			errC <- err
			return
		}
	}
}

func (w *WSProxier) close() {
	w.clientConn.Close()
	w.backendConn.Close()
	if err := w.backend.rateLimiter.DecBackendWSConns(w.backend.Name); err != nil {
		log.Error("error decrementing backend ws conns", "name", w.backend.Name, "err", err)
	}
	activeBackendWsConnsGauge.WithLabelValues(w.backend.Name).Dec()
}

func (w *WSProxier) prepareClientMsg(msg []byte) (*RPCReq, error) {
	req, err := ParseRPCReq(msg)
	if err != nil {
		return nil, err
	}

	if !w.methodWhitelist.Has(req.Method) {
		return req, ErrMethodNotWhitelisted
	}

	if w.backend.IsRateLimited() {
		return req, ErrBackendOverCapacity
	}

	return req, nil
}

func (w *WSProxier) parseBackendMsg(msg []byte) (*RPCRes, error) {
	res, err := ParseRPCRes(bytes.NewReader(msg))
	if err != nil {
		log.Warn("error parsing RPC response", "source", "ws", "err", err)
		return res, ErrBackendBadResponse
	}
	return res, nil
}

func (w *WSProxier) writeClientConn(msgType int, msg []byte) error {
	w.clientConnMu.Lock()
	err := w.clientConn.WriteMessage(msgType, msg)
	w.clientConnMu.Unlock()
	return err
}

func mustMarshalJSON(in interface{}) []byte {
	out, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	return out
}

func formatWSError(err error) []byte {
	m := websocket.FormatCloseMessage(websocket.CloseNormalClosure, fmt.Sprintf("%v", err))
	if e, ok := err.(*websocket.CloseError); ok {
		if e.Code != websocket.CloseNoStatusReceived {
			m = websocket.FormatCloseMessage(e.Code, e.Text)
		}
	}
	return m
}

func sleepContext(ctx context.Context, duration time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(duration):
	}
}

type LimitedHTTPClient struct {
	http.Client
	sem         *semaphore.Weighted
	backendName string
}

func (c *LimitedHTTPClient) DoLimited(req *http.Request) (*http.Response, error) {
	if err := c.sem.Acquire(req.Context(), 1); err != nil {
		tooManyRequestErrorsTotal.WithLabelValues(c.backendName).Inc()
		return nil, wrapErr(err, "too many requests")
	}
	defer c.sem.Release(1)
	return c.Do(req)
}

func RecordBatchRPCError(ctx context.Context, backendName string, reqs []*RPCReq, err error) {
	for _, req := range reqs {
		RecordRPCError(ctx, backendName, req.Method, err)
	}
}

func MaybeRecordErrorsInRPCRes(ctx context.Context, backendName string, reqs []*RPCReq, resBatch []*RPCRes) {
	log.Info("forwarded RPC request",
		"backend", backendName,
		"auth", GetAuthCtx(ctx),
		"req_id", GetReqID(ctx),
		"batch_size", len(reqs),
	)

	var lastError *RPCErr
	for i, res := range resBatch {
		if res.IsError() {
			lastError = res.Error
			RecordRPCError(ctx, backendName, reqs[i].Method, res.Error)
		}
	}

	if lastError != nil {
		log.Info(
			"backend responded with RPC error",
			"backend", backendName,
			"last_error_code", lastError.Code,
			"last_error_msg", lastError.Message,
			"req_id", GetReqID(ctx),
			"source", "rpc",
			"auth", GetAuthCtx(ctx),
		)
	}
}

func RecordBatchRPCForward(ctx context.Context, backendName string, reqs []*RPCReq, source string) {
	for _, req := range reqs {
		RecordRPCForward(ctx, backendName, req.Method, source)
	}
}

func stripXFF(xff string) string {
	ipList := strings.Split(xff, ", ")
	return strings.TrimSpace(ipList[0])
}
