package metrics

import (
	"fmt"
	"io"
	"math/big"

	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
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

type CreditExpectation uint8

const (
	// Max Duration reached
	CreditBelowMaxDuration CreditExpectation = iota
	CreditEqualMaxDuration
	CreditAboveMaxDuration

	// Max Duration not reached
	CreditBelowNonMaxDuration
	CreditEqualNonMaxDuration
	CreditAboveNonMaxDuration
)

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

type ClaimStatus uint8

const (
	// Claims where the game is in the first half
	FirstHalfExpiredResolved ClaimStatus = iota
	FirstHalfExpiredUnresolved
	FirstHalfNotExpiredResolved
	FirstHalfNotExpiredUnresolved

	// Claims where the game is in the second half
	SecondHalfExpiredResolved
	SecondHalfExpiredUnresolved
	SecondHalfNotExpiredResolved
	SecondHalfNotExpiredUnresolved
)

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	RecordUnexpectedClaimResolution(address common.Address, count int)

	RecordGameResolutionStatus(complete bool, maxDurationReached bool, count int)

	RecordCredit(expectation CreditExpectation, count int)

	RecordClaims(status ClaimStatus, count int)

	RecordWithdrawalRequests(delayedWeth common.Address, matches bool, count int)

	RecordClaimResolutionDelayMax(delay float64)

	RecordOutputFetchTime(timestamp float64)

	RecordGameAgreement(status GameAgreementStatus, count int)

	RecordBondCollateral(addr common.Address, required *big.Int, available *big.Int)

	caching.Metrics
	contractMetrics.ContractMetricer
}

// Metrics implementation must implement RegistryMetricer to allow the metrics server to work.
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	*opmetrics.CacheMetrics
	*contractMetrics.ContractMetrics

	resolutionStatus prometheus.GaugeVec

	claims prometheus.GaugeVec

	unexpectedClaimResolutions prometheus.GaugeVec

	withdrawalRequests prometheus.GaugeVec

	info prometheus.GaugeVec
	up   prometheus.Gauge

	credits prometheus.GaugeVec

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

		CacheMetrics:    opmetrics.NewCacheMetrics(factory, Namespace, "provider_cache", "Provider cache"),
		ContractMetrics: contractMetrics.MakeContractMetrics(Namespace, factory),

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
		unexpectedClaimResolutions: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "unexpected_claim_resolutions",
			Help:      "Total number of unexpected claim resolutions against an honest actor",
		}, []string{
			"honest_actor_address",
		}),
		resolutionStatus: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "resolution_status",
			Help:      "Number of games categorised by whether the game is complete and whether the maximum duration has been reached",
		}, []string{
			"completion",
			"max_duration",
		}),
		credits: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "credits",
			Help:      "Cumulative credits",
		}, []string{
			"credit",
			"max_duration",
		}),
		claims: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "claims",
			Help:      "Claims broken down by whether they were resolved, whether the clock expired, and the game time period",
		}, []string{
			"resolved",
			"clock",
			"game_time_period",
		}),
		withdrawalRequests: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: Namespace,
			Name:      "withdrawal_requests",
			Help:      "Number of withdrawal requests categorised by the source DelayedWETH contract and whether the withdrawal request amount matches or diverges from its fault dispute game credits",
		}, []string{
			"delayedWETH",
			"credits",
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

func (m *Metrics) RecordUnexpectedClaimResolution(address common.Address, count int) {
	m.unexpectedClaimResolutions.WithLabelValues(address.Hex()).Set(float64(count))
}

func (m *Metrics) RecordGameResolutionStatus(complete bool, maxDurationReached bool, count int) {
	completion := "complete"
	if !complete {
		completion = "in_progress"
	}
	maxDuration := "reached"
	if !maxDurationReached {
		maxDuration = "not_reached"
	}
	m.resolutionStatus.WithLabelValues(completion, maxDuration).Set(float64(count))
}

func (m *Metrics) RecordCredit(expectation CreditExpectation, count int) {
	asLabels := func(expectation CreditExpectation) []string {
		switch expectation {
		case CreditBelowMaxDuration:
			return []string{"below", "max_duration"}
		case CreditEqualMaxDuration:
			return []string{"expected", "max_duration"}
		case CreditAboveMaxDuration:
			return []string{"above", "max_duration"}
		case CreditBelowNonMaxDuration:
			return []string{"below", "non_max_duration"}
		case CreditEqualNonMaxDuration:
			return []string{"expected", "non_max_duration"}
		case CreditAboveNonMaxDuration:
			return []string{"above", "non_max_duration"}
		default:
			panic(fmt.Errorf("unknown credit expectation: %v", expectation))
		}
	}
	m.credits.WithLabelValues(asLabels(expectation)...).Set(float64(count))
}

func (m *Metrics) RecordClaims(status ClaimStatus, count int) {
	asLabels := func(status ClaimStatus) []string {
		switch status {
		case FirstHalfExpiredResolved:
			return []string{"resolved", "expired", "first_half"}
		case FirstHalfExpiredUnresolved:
			return []string{"unresolved", "expired", "first_half"}
		case FirstHalfNotExpiredResolved:
			return []string{"resolved", "not_expired", "first_half"}
		case FirstHalfNotExpiredUnresolved:
			return []string{"unresolved", "not_expired", "first_half"}
		case SecondHalfExpiredResolved:
			return []string{"resolved", "expired", "second_half"}
		case SecondHalfExpiredUnresolved:
			return []string{"unresolved", "expired", "second_half"}
		case SecondHalfNotExpiredResolved:
			return []string{"resolved", "not_expired", "second_half"}
		case SecondHalfNotExpiredUnresolved:
			return []string{"unresolved", "not_expired", "second_half"}
		default:
			panic(fmt.Errorf("unknown claim status: %v", status))
		}
	}
	m.claims.WithLabelValues(asLabels(status)...).Set(float64(count))
}

func (m *Metrics) RecordWithdrawalRequests(delayedWeth common.Address, matches bool, count int) {
	credits := "matching"
	if !matches {
		credits = "divergent"
	}
	m.withdrawalRequests.WithLabelValues(delayedWeth.Hex(), credits).Set(float64(count))
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
