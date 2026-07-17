package metrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	clientmodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const defaultFetchTimeout = 30 * time.Second

type MetricsClient struct {
	client       client.HTTP
	fetchTimeout time.Duration
}

type Option func(*MetricsClient)

func WithFetchTimeout(timeout time.Duration) Option {
	return func(client *MetricsClient) {
		client.fetchTimeout = timeout
	}
}

func NewMetricsClient(httpClient client.HTTP, options ...Option) *MetricsClient {
	metricsClient := &MetricsClient{client: httpClient, fetchTimeout: defaultFetchTimeout}
	for _, option := range options {
		if option != nil {
			option(metricsClient)
		}
	}
	return metricsClient
}

func (c *MetricsClient) validate() error {
	if c.client == nil {
		return fmt.Errorf("fetch metrics: HTTP client must not be nil")
	}
	if c.fetchTimeout <= 0 {
		return fmt.Errorf("fetch metrics: fetch timeout must be positive")
	}
	return nil
}

type Snapshot struct {
	families map[string]*clientmodel.MetricFamily
	payload  string
}

func (s *Snapshot) Payload() string {
	return s.payload
}

func (c *MetricsClient) Fetch(ctx context.Context) (*Snapshot, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	fetchCtx, cancel := context.WithTimeout(ctx, c.fetchTimeout)
	defer cancel()

	response, err := c.client.Get(fetchCtx, "/metrics", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch metrics: %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("fetch metrics: HTTP client returned a nil response")
	}
	if response.Body == nil {
		return nil, fmt.Errorf("fetch metrics: HTTP response has a nil body")
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read metrics response: %w", err)
	}
	payloadText := string(payload)
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics endpoint returned HTTP %d (%s): %s", response.StatusCode, response.Status, payloadText)
	}
	return parseSnapshot(payloadText)
}

func parseSnapshot(payload string) (*Snapshot, error) {
	parser := expfmt.TextParser{}
	families, err := parser.TextToMetricFamilies(strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("parse metrics response: %w", err)
	}
	return &Snapshot{families: families, payload: payload}, nil
}

func (s *Snapshot) Gauge(name string, labels map[string]string) (float64, error) {
	family, ok := s.families[name]
	if !ok {
		return 0, fmt.Errorf("metric family %s not found", name)
	}
	if family.GetType() != clientmodel.MetricType_GAUGE {
		return 0, fmt.Errorf("metric %s is not a gauge", name)
	}

	var matches []*clientmodel.Metric
	for _, metric := range family.Metric {
		if metricHasLabels(metric, labels) {
			matches = append(matches, metric)
		}
	}
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("metric %s with labels %v not found", name, labels)
	case 1:
	default:
		return 0, fmt.Errorf("metric %s with labels %v matched multiple series", name, labels)
	}

	gauge := matches[0].GetGauge()
	if gauge == nil {
		return 0, fmt.Errorf("metric %s with labels %v has no gauge value", name, labels)
	}
	return gauge.GetValue(), nil
}

func metricHasLabels(metric *clientmodel.Metric, expected map[string]string) bool {
	for name, value := range expected {
		matched := false
		for _, label := range metric.Label {
			if label.GetName() == name && label.GetValue() == value {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

type GaugeDefinition struct {
	Name     string
	Labels   map[string]string
	Expected float64
}

func (c *MetricsClient) WaitForGauge(ctx context.Context, definition GaugeDefinition, pollInterval time.Duration) error {
	if err := c.validate(); err != nil {
		return err
	}
	if pollInterval <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		snapshot, err := c.Fetch(ctx)
		if err != nil && lastErr != nil && ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return fmt.Errorf(
				"metric %s did not reach expected value: %w: %w",
				definition.Name,
				lastErr,
				ctx.Err(),
			)
		}
		if err == nil {
			observed, gaugeErr := snapshot.Gauge(definition.Name, definition.Labels)
			if gaugeErr == nil && observed == definition.Expected {
				return nil
			}
			if gaugeErr == nil {
				gaugeErr = fmt.Errorf(
					"metric %s expected %v but observed %v",
					definition.Name,
					definition.Expected,
					observed,
				)
			}
			err = fmt.Errorf("%w\nmetrics payload:\n%s", gaugeErr, snapshot.Payload())
		}
		lastErr = err

		if ctx.Err() != nil {
			return fmt.Errorf(
				"metric %s did not reach expected value: %w: %w",
				definition.Name,
				lastErr,
				ctx.Err(),
			)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"metric %s did not reach expected value: %w: %w",
				definition.Name,
				lastErr,
				ctx.Err(),
			)
		case <-ticker.C:
		}
	}
}
