package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/httputil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpLabels = []string{
	"method",
	"response_code",
}

type HTTPParams struct {
	Method     string
	StatusCode int
}

type HTTPRecorder interface {
	RecordHTTPRequestDuration(params *HTTPParams, dur time.Duration)
	RecordHTTPResponseSize(params *HTTPParams, size int)
	RecordInflightRequest(params *HTTPParams, quantity int)
	RecordHTTPRequest(params *HTTPParams)
	RecordHTTPResponse(params *HTTPParams)
}

type noopHTTPRecorder struct{}

var NoopHTTPRecorder = new(noopHTTPRecorder)

func (n *noopHTTPRecorder) RecordHTTPRequestDuration(*HTTPParams, time.Duration) {}

func (n *noopHTTPRecorder) RecordHTTPResponseSize(*HTTPParams, int) {}

func (n *noopHTTPRecorder) RecordInflightRequest(*HTTPParams, int) {}

func (n *noopHTTPRecorder) RecordHTTPRequest(*HTTPParams) {}

func (n *noopHTTPRecorder) RecordHTTPResponse(*HTTPParams) {}

type PromHTTPRecorder struct {
	// HTTPRequestDuration is the old metric for request latency
	// it was created with too tight buckets: [.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]
	// in order to preserve backward compatibility we are keeping this metric for now
	// and it will be removed when services opt in to HTTPRequestLatency
	// Deprecated: HTTPRequestDuration is deprecated
	HTTPRequestDuration *prometheus.HistogramVec

	// HTTPRequestLatency measures request execution latency in *seconds*
	// buckets are: [.025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50]
	HTTPRequestLatency *prometheus.HistogramVec

	HTTPResponseSize     *prometheus.HistogramVec
	HTTPInflightRequests *prometheus.GaugeVec
	HTTPRequests         *prometheus.CounterVec
	HTTPResponses        *prometheus.CounterVec
}

var LatencyBuckets = []float64{.025, .05, .1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100}

func NewPromHTTPRecorder(r *prometheus.Registry, ns string) HTTPRecorder {
	return &PromHTTPRecorder{
		// TODO: remove this in the future when services opted in to HTTPRequestLatency
		HTTPRequestDuration: promauto.With(r).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "http_request_duration_ms",
			Help:      "Tracks HTTP request durations, in ms",
			Buckets:   prometheus.DefBuckets,
		}, httpLabels),
		HTTPRequestLatency: promauto.With(r).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "http_request_latency_seconds",
			Help:      "Tracks HTTP request execution latency, in seconds",
			Buckets:   LatencyBuckets,
		}, httpLabels),

		HTTPResponseSize: promauto.With(r).NewHistogramVec(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "http_response_size",
			Help:      "Tracks HTTP response sizes",
			Buckets:   prometheus.DefBuckets,
		}, httpLabels),
		HTTPInflightRequests: promauto.With(r).NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "http_inflight_requests_count",
			Help:      "Tracks currently in-flight requests",
		}, []string{"method"}),
		HTTPRequests: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "http_requests_count_total",
			Help:      "Tracks total HTTP requests",
		}, []string{"method"}),
		HTTPResponses: promauto.With(r).NewCounterVec(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "http_responses_count_total",
			Help:      "Tracks total HTTP responses",
		}, httpLabels),
	}
}

func (p *PromHTTPRecorder) RecordHTTPRequestDuration(params *HTTPParams, dur time.Duration) {
	// TODO: remove this in the future when services opted in to new metric
	p.HTTPRequestDuration.WithLabelValues(params.Method, strconv.Itoa(params.StatusCode)).
		Observe(float64(dur.Milliseconds()))

	p.HTTPRequestLatency.WithLabelValues(params.Method, strconv.Itoa(params.StatusCode)).
		Observe(dur.Seconds())
}

func (p *PromHTTPRecorder) RecordHTTPResponseSize(params *HTTPParams, size int) {
	p.HTTPResponseSize.WithLabelValues(params.Method, strconv.Itoa(params.StatusCode)).Observe(float64(size))
}

func (p *PromHTTPRecorder) RecordInflightRequest(params *HTTPParams, quantity int) {
	p.HTTPInflightRequests.WithLabelValues(params.Method).Add(float64(quantity))
}

func (p *PromHTTPRecorder) RecordHTTPRequest(params *HTTPParams) {
	p.HTTPRequests.WithLabelValues(params.Method).Inc()
}

func (p *PromHTTPRecorder) RecordHTTPResponse(params *HTTPParams) {
	p.HTTPResponses.WithLabelValues(params.Method, strconv.Itoa(params.StatusCode)).Inc()
}

func NewHTTPRecordingMiddleware(rec HTTPRecorder, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := httputil.NewWrappedResponseWriter(w)
		params := &HTTPParams{
			Method: r.Method,
		}
		rec.RecordInflightRequest(params, 1)
		rec.RecordHTTPRequest(params)
		start := time.Now()
		next.ServeHTTP(ww, r)
		params.StatusCode = ww.StatusCode
		dur := time.Since(start)
		rec.RecordHTTPResponse(params)
		rec.RecordHTTPResponseSize(params, ww.ResponseLen)
		rec.RecordHTTPRequestDuration(params, dur)
		rec.RecordInflightRequest(params, -1)
	})
}
