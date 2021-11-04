package proxyd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	JSONRPCVersion = "2.0"
)

var (
	ErrNoBackend            = errors.New("no backend available for method")
	ErrBackendsInconsistent = errors.New("backends inconsistent, try again")
	ErrBackendOffline       = errors.New("backend offline")

	backendRequestsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "backend_requests_total",
		Help:      "Count of backend requests.",
	}, []string{
		"backend_name",
		"method_name",
	})

	backendErrorsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "backend_errors_total",
		Help:      "Count of backend errors.",
	}, []string{
		"backend_name",
		"method_name",
	})

	backendMethodErrorsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "backend_method_errors_total",
		Help:      "Count of backend method errors.",
	}, []string{
		"error_code",
	})

	backendMethodErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "backend_method_errors",
		Help:      "Backend method errors.",
	}, []string{
		"error_type",
	})
	backendPermanentErrorsCtr = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "proxyd",
		Name:      "backend_permanent_errors_total",
		Help:      "Count of backend errors that mark a backend as offline.",
	}, []string{
		"backend_name",
		"method_name",
	})

	backendResponseTimeSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "proxyd",
		Name:       "backend_response_time_seconds",
		Help:       "Summary of backend response times broken down by backend and method name.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	}, []string{
		"backend_name",
		"method_name",
	})
)

type Backend struct {
	Name                   string
	authUsername           string
	authPassword           string
	baseURL                string
	client                 *http.Client
	maxRetries             int
	maxResponseSize        int64
	lastPermError          int64
	unhealthyRetryInterval int64
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

func WithUnhealthyRetryInterval(interval int64) BackendOpt {
	return func(b *Backend) {
		b.unhealthyRetryInterval = interval
	}
}

func NewBackend(name, baseURL string, opts ...BackendOpt) *Backend {
	backend := &Backend{
		Name:            name,
		baseURL:         baseURL,
		maxResponseSize: math.MaxInt64,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(backend)
	}

	return backend
}

func (b *Backend) Forward(req *RPCReq) (*RPCRes, error) {
	if time.Now().Unix()-atomic.LoadInt64(&b.lastPermError) < b.unhealthyRetryInterval {
		return nil, ErrBackendOffline
	}

	var lastError error
	// <= to account for the first attempt not technically being
	// a retry
	for i := 0; i <= b.maxRetries; i++ {
		resB, err := b.doForward(req)
		if err != nil {
			lastError = err
			log.Warn("backend request failed, trying again", "err", err, "name", b.Name)
			time.Sleep(calcBackoff(i))
			continue
		}

		res := new(RPCRes)
		// don't mark the backend down if they give us a bad response body
		if err := json.Unmarshal(resB, res); err != nil {
			return nil, wrapErr(err, "error unmarshaling JSON")
		}

		return res, nil
	}

	atomic.StoreInt64(&b.lastPermError, time.Now().Unix())
	backendPermanentErrorsCtr.WithLabelValues(b.Name, req.Method).Inc()
	return nil, wrapErr(lastError, "permanent error forwarding request")
}

func (b *Backend) doForward(rpcReq *RPCReq) ([]byte, error) {
	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, wrapErr(err, "error marshaling request in forward")
	}

	httpReq, err := http.NewRequest("POST", b.baseURL, bytes.NewReader(body))

	// Required when connecting directly to geth
	httpReq.Header.Add("Content-type", "application/json")

	if err != nil {
		backendErrorsCtr.WithLabelValues(b.Name, rpcReq.Method).Inc()
		return nil, wrapErr(err, "error creating backend request")
	}

	if b.authPassword != "" {
		httpReq.SetBasicAuth(b.authUsername, b.authPassword)
	}

	timer := prometheus.NewTimer(backendResponseTimeSummary.WithLabelValues(b.Name, rpcReq.Method))
	defer timer.ObserveDuration()
	defer backendRequestsCtr.WithLabelValues(b.Name, rpcReq.Method).Inc()
	res, err := b.client.Do(httpReq)
	if err != nil {
		backendErrorsCtr.WithLabelValues(b.Name, rpcReq.Method).Inc()
		return nil, wrapErr(err, "error in backend request")
	}

	if res.StatusCode != 200 {
		backendErrorsCtr.WithLabelValues(b.Name, rpcReq.Method).Inc()
		return nil, fmt.Errorf("response code %d", res.StatusCode)
	}

	defer res.Body.Close()
	resB, err := ioutil.ReadAll(io.LimitReader(res.Body, b.maxResponseSize))
	if err != nil {
		backendErrorsCtr.WithLabelValues(b.Name, rpcReq.Method).Inc()
		return nil, wrapErr(err, "error reading response body")
	}
	var responseBody RPCResponse
	err = json.Unmarshal(resB, &responseBody)
	if err != nil {
		return nil, wrapErr(err, "error unmarshaling response from forward")
	}
	go processReponse(responseBody)

	return resB, nil
}
func processReponse(response RPCResponse) {
	noneTooLowRegex := regexp.MustCompile(`nonce too low`)
	gasPriceTooHighRegex := regexp.MustCompile(`gas price too high`)
	gasPriceTooLowRegex := regexp.MustCompile(`gas price too low`)
	invalidParametersRegex := regexp.MustCompile(`invalid parameters`)

	if response.Error.Code < 0 {
		log.Warn("Upstream returned an error", "message", response.Error.Message, "error", response.Error.Code)
		backendMethodErrorsCtr.WithLabelValues(strconv.Itoa(response.Error.Code)).Inc()
		switch {
		case noneTooLowRegex.MatchString(response.Error.Message):
			backendMethodErrors.WithLabelValues("low-nonce").Inc()
		case gasPriceTooHighRegex.MatchString(response.Error.Message):
			backendMethodErrors.WithLabelValues("high-gas").Inc()
		case gasPriceTooLowRegex.MatchString(response.Error.Message):
			backendMethodErrors.WithLabelValues("low-gas").Inc()
		case invalidParametersRegex.MatchString(response.Error.Message):
			backendMethodErrors.WithLabelValues("invalid-parameters").Inc()
		}
	}
}

type BackendGroup struct {
	Name     string
	backends []*Backend
	i        int64
}

type RPCResponse struct {
	Error struct {
		Code    int
		Message string
	}
	Result string
}

func (b *BackendGroup) Forward(rpcReq *RPCReq) (*RPCRes, error) {
	var outRes *RPCRes
	for _, back := range b.backends {
		res, err := back.Forward(rpcReq)
		if err == ErrBackendOffline {
			log.Debug("skipping offline backend", "name", back.Name)
			continue
		}
		if err != nil {
			log.Error("error forwarding request to backend", "err", err, "name", b.Name)
			continue
		}
		outRes = res
		break
	}

	if outRes == nil {
		return nil, errors.New("no backends available")
	}

	return outRes, nil
}

type MethodMapping struct {
	methods map[string]*BackendGroup
}

func NewMethodMapping(methods map[string]*BackendGroup) *MethodMapping {
	return &MethodMapping{methods: methods}
}

func (m *MethodMapping) BackendGroupFor(method string) (*BackendGroup, error) {
	group := m.methods[method]
	if group == nil {
		return nil, ErrNoBackend
	}
	return group, nil
}

func calcBackoff(i int) time.Duration {
	jitter := float64(rand.Int63n(250))
	ms := math.Min(math.Pow(2, float64(i))*1000+jitter, 10000)
	return time.Duration(ms) * time.Millisecond
}
