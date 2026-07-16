package presets

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/common"
	clientmodel "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

const (
	disputeMonMetricWaitTimeout  = 30 * time.Second
	disputeMonMetricPollInterval = 100 * time.Millisecond
)

type DisputeMon struct {
	t          devtest.T
	metricsURL string
	client     *http.Client
}

func newDisputeMon(t devtest.T, metricsURL string) *DisputeMon {
	return &DisputeMon{
		t:          t,
		metricsURL: strings.TrimRight(metricsURL, "/") + "/metrics",
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (d *DisputeMon) VerifyGameCount(gameType gameTypes.GameType, expected int) {
	d.t.Helper()
	ctx, cancel := context.WithTimeout(d.t.Ctx(), disputeMonMetricWaitTimeout)
	defer cancel()
	err := waitForGauge(
		ctx,
		d.client,
		d.metricsURL,
		"op_dispute_mon_games",
		map[string]string{"game_type": gameType.String()},
		float64(expected),
		disputeMonMetricPollInterval,
	)
	d.t.Require().NoError(err, "expected dispute monitor to export %d games of type %s", expected, gameType)
}

func waitForGauge(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	name string,
	labels map[string]string,
	expected float64,
	pollInterval time.Duration,
) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	var lastErr error
	for {
		observed, payload, err := fetchGauge(ctx, client, endpoint, name, labels)
		if err == nil && observed == expected {
			return nil
		}
		if err == nil {
			err = fmt.Errorf("metric %s expected %v but observed %v", name, expected, observed)
		} else if ctx.Err() != nil && lastErr != nil {
			return fmt.Errorf("metric %s did not reach expected value: %w: %v", name, lastErr, ctx.Err())
		}
		lastErr = fmt.Errorf("%w\nmetrics payload:\n%s", err, payload)
		if ctx.Err() != nil {
			return fmt.Errorf("metric %s did not reach expected value: %w: %v", name, lastErr, ctx.Err())
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("metric %s did not reach expected value: %w: %v", name, lastErr, ctx.Err())
		case <-ticker.C:
		}
	}
}

func fetchGauge(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	name string,
	labels map[string]string,
) (float64, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, "", fmt.Errorf("create metrics request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("fetch metrics: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read metrics response: %w", err)
	}
	payloadText := string(payload)
	if resp.StatusCode != http.StatusOK {
		return 0, payloadText, fmt.Errorf("metrics endpoint returned %s", resp.Status)
	}
	parser := expfmt.TextParser{}
	families, err := parser.TextToMetricFamilies(strings.NewReader(payloadText))
	if err != nil {
		return 0, payloadText, fmt.Errorf("parse metrics response: %w", err)
	}
	family, ok := families[name]
	if !ok {
		return 0, payloadText, fmt.Errorf("metric family %s not found", name)
	}
	for _, metric := range family.Metric {
		if metricHasLabels(metric, labels) {
			if metric.Gauge == nil {
				return 0, payloadText, fmt.Errorf("metric %s is not a gauge", name)
			}
			return metric.GetGauge().GetValue(), payloadText, nil
		}
	}
	return 0, payloadText, fmt.Errorf("metric %s with labels %v not found", name, labels)
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

type disputeMonOptions struct {
	rollupRPCs    []string
	supernodeRPCs []string
}

type DisputeMonOption func(*disputeMonOptions)

func WithDisputeMonRollupNodes(nodes ...*dsl.L2CLNode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.rollupRPCs = append(opts.rollupRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func WithDisputeMonSupernodes(nodes ...*dsl.Supernode) DisputeMonOption {
	return func(opts *disputeMonOptions) {
		for _, node := range nodes {
			if node != nil {
				opts.supernodeRPCs = append(opts.supernodeRPCs, node.Escape().UserRPC())
			}
		}
	}
}

func StartDisputeMon(
	t devtest.T,
	l1EL *dsl.L1ELNode,
	factory common.Address,
	options ...DisputeMonOption,
) *DisputeMon {
	t.Helper()
	t.Require().NotNil(l1EL, "L1 EL is required to start dispute monitor")
	opts := &disputeMonOptions{}
	for _, option := range options {
		if option != nil {
			option(opts)
		}
	}
	t.Require().NotEmpty(
		append(append([]string{}, opts.rollupRPCs...), opts.supernodeRPCs...),
		"at least one rollup node or supernode is required to start dispute monitor",
	)
	runtime := sysgo.StartDisputeMon(t, sysgo.DisputeMonConfig{
		L1RPC:              l1EL.Escape().UserRPC(),
		GameFactoryAddress: factory,
		RollupRPCs:         opts.rollupRPCs,
		SupernodeRPCs:      opts.supernodeRPCs,
	})
	return newDisputeMon(t, runtime.MetricsURL())
}
