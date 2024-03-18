package metrics

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const Namespace = "op_dispute_mon"

type GameAgreementStatus uint8

const (
	// In progress
	AgreeChallengerAhead GameAgreementStatus = iota
	DisagreeChallengerAhead
	AgreeDefenderAhead
	DisagreeDefenderAhead

	// Completed
	AgreeDefenderWins
	DisagreeDefenderWins
	AgreeChallengerWins
	DisagreeChallengerWins
)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	RecordClaimResolutionDelayMax(delay float64)

	RecordOutputFetchTime(timestamp float64)

	RecordGameAgreement(status GameAgreementStatus, count int)

	RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int)

	caching.Metrics
}

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	*opmetrics.CacheMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	lastOutputFetch prometheus.Gauge

	claimResolutionDelayMax prometheus.Gauge

	gamesAgreement prometheus.GaugeVec

	requiredCollateral  prometheus.GaugeVec
	availableCollateral prometheus.GaugeVec
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

var _ Metricer = (*Metrics)(nil)

func NewMetrics() *Metrics {
	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	return &Metrics{
		ns:       Namespace,
		registry: registry,
		factory:  factory,

		CacheMetrics: opmetrics.NewCacheMetrics(factory, Namespace, "provider_cache", "Provider cache"),

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "up",
			Help:      "1 if the op-challenger has finished starting up",
		}),
		lastOutputFetch: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "last_output_fetch",
			Help:      "Timestamp of the last output fetch",
		}),
		claimResolutionDelayMax: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "claim_resolution_delay_max",
			Help:      "Maximum claim resolution delay in seconds",
		}),
		gamesAgreement: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "games_agreement",
			Help:      "Number of games broken down by whether the result agrees with the reference node",
		}, []string{
			"status",
			"completion",
			"result_correctness",
			"root_agreement",
		}),
		requiredCollateral: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "bond_collateral_required",
			Help:      "Required collateral (ETH) to cover outstanding bonds and credits",
		}, []string{
			// Address of the DelayedWETH contract in use. This is a limited set as only permissioned actors can deploy
			// additional DelayedWETH contracts to be used by dispute games
			"delayedWETH",
			"balance",
		}),
		availableCollateral: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "bond_collateral_available",
			Help:      "Available collateral (ETH) to cover outstanding bonds and credits",
		}, []string{
			// Address of the DelayedWETH contract in use. This is a limited set as only permissioned actors can deploy
			// additional DelayedWETH contracts to be used by dispute games
			"delayedWETH",
			"balance",
		}),
	}
}

func (m *Metrics) Start(host string, port int) (*httputil.HTTPServer, error) {
	return opmetrics.StartServer(m.registry, host, port)
}

func (m *Metrics) StartBalanceMetrics(
	l log.Logger,
	client *ethclient.Client,
	account common.Address,
) io.Closer {
	return opmetrics.LaunchBalanceMetrics(l, m.registry, m.ns, client, account)
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the op-proposer.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	prometheus.MustRegister()
	m.up.Set(1)
}

func (m *Metrics) RecordClaimResolutionDelayMax(delay float64) {
	m.claimResolutionDelayMax.Set(delay)
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

func (m *Metrics) RecordOutputFetchTime(timestamp float64) {
	m.lastOutputFetch.Set(timestamp)
}

func (m *Metrics) RecordGameAgreement(status GameAgreementStatus, count int) {
	m.gamesAgreement.WithLabelValues(labelValuesFor(status)...).Set(float64(count))
}

func (m *Metrics) RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int) {
	balance := "sufficient"
	if required.Cmp(available) > 0 {
		balance = "insufficient"
	}
	m.requiredCollateral.WithLabelValues(addr.Hex(), balance).Set(weiToEther(required))
	m.availableCollateral.WithLabelValues(addr.Hex(), balance).Set(weiToEther(available))
}

const (
	inProgress = true
	correct    = true
	agree      = true
)

func labelValuesFor(status GameAgreementStatus) []string {
	asStrings := func(status string, inProgress bool, correct bool, agree bool) []string {
		inProgressStr := "in_progress"
		if !inProgress {
			inProgressStr = "complete"
		}
		correctStr := "correct"
		if !correct {
			correctStr = "incorrect"
		}
		agreeStr := "agree"
		if !agree {
			agreeStr = "disagree"
		}
		return []string{status, inProgressStr, correctStr, agreeStr}
	}
	switch status {
	case AgreeChallengerAhead:
		return asStrings("agree_challenger_ahead", inProgress, !correct, agree)
	case DisagreeChallengerAhead:
		return asStrings("disagree_challenger_ahead", inProgress, correct, !agree)
	case AgreeDefenderAhead:
		return asStrings("agree_defender_ahead", inProgress, correct, agree)
	case DisagreeDefenderAhead:
		return asStrings("disagree_defender_ahead", inProgress, !correct, !agree)

	// Completed
	case AgreeDefenderWins:
		return asStrings("agree_defender_wins", !inProgress, correct, agree)
	case DisagreeDefenderWins:
		return asStrings("disagree_defender_wins", !inProgress, !correct, !agree)
	case AgreeChallengerWins:
		return asStrings("agree_challenger_wins", !inProgress, !correct, agree)
	case DisagreeChallengerWins:
		return asStrings("disagree_challenger_wins", !inProgress, correct, !agree)
	default:
		panic(fmt.Errorf("unknown game agreement status: %v", status))
	}
}

// weiToEther divides the wei value by 10^18 to get a number in ether as a float64
func weiToEther(wei *big.Int) float64 {
	num := new(big.Rat).SetInt(wei)
	denom := big.NewRat(params.Ether, 1)
	num = num.Quo(num, denom)
	f, _ := num.Float64()
	return f
}
