package metrics

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-batcher/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	txmetrics "github.com/ethereum-optimism/optimism/op-service/txmgr/metrics"
)

const Namespace = "op_batcher"

type Metricer interface {
	RecordInfo(version string)
	RecordUp()

	// Records all L1 and L2 block events
	opmetrics.RefMetricer

	// Record Tx metrics
	txmetrics.TxMetricer

	opmetrics.RPCMetricer

	StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer

	RecordLatestL1Block(l1ref eth.L1BlockRef)
	RecordL2BlocksLoaded(l2ref eth.L2BlockRef)
	RecordChannelOpened(id derive.ChannelID, numPendingBlocks int)
	RecordL2BlocksAdded(l2ref eth.L2BlockRef, numBlocksAdded, numPendingBlocks, inputBytes, outputComprBytes int)
	RecordL2BlockInPendingQueue(block *types.Block)
	RecordL2BlockInChannel(block *types.Block)
	RecordChannelClosed(id derive.ChannelID, numPendingBlocks int, numFrames int, inputBytes int, outputComprBytes int, reason error)
	RecordChannelFullySubmitted(id derive.ChannelID)
	RecordChannelTimedOut(id derive.ChannelID)
	RecordChannelQueueLength(len int)
	RecordThrottleIntensity(intensity float64, controllerType config.ThrottleControllerType)
	RecordThrottleParams(maxTxSize, maxBlockSize uint64)
	RecordThrottleControllerType(controllerType config.ThrottleControllerType)
	RecordPendingBytesVsThreshold(pendingBytes, threshold uint64, controllerType config.ThrottleControllerType)

	// PID Controller specific metrics
	RecordThrottleControllerState(error, integral, derivative float64)
	RecordThrottleResponseTime(duration time.Duration)

	// ClearAllStateMetrics resets any metrics that track current ChannelManager state
	// It should be called when clearing the ChannelManager state.
	ClearAllStateMetrics()

	RecordBatchTxSubmitted()
	RecordBatchTxSuccess()
	RecordBatchTxFailed()

	RecordBlobUsedBytes(num int)

	Document() []opmetrics.DocumentedMetric

	PendingDABytes() float64
}

type Metrics struct {
	ns       string
	registry *prometheus.Registry
	factory  opmetrics.Factory

	opmetrics.RefMetrics
	txmetrics.TxMetrics
	opmetrics.RPCMetrics

	info prometheus.GaugeVec
	up   prometheus.Gauge

	// label by opened, closed, fully_submitted, timed_out
	channelEvs opmetrics.EventVec

	pendingBlocksCount        prometheus.GaugeVec
	pendingBlocksBytesTotal   prometheus.Counter
	pendingBlocksBytesCurrent prometheus.Gauge

	pendingDABytes          int64
	pendingDABytesGaugeFunc prometheus.GaugeFunc

	blocksAddedCount prometheus.Gauge

	channelInputBytes       prometheus.GaugeVec
	channelReadyBytes       prometheus.Gauge
	channelOutputBytes      prometheus.Gauge
	channelClosedReason     prometheus.Gauge
	channelNumFrames        prometheus.Gauge
	channelComprRatio       prometheus.Histogram
	channelInputBytesTotal  prometheus.Counter
	channelOutputBytesTotal prometheus.Counter
	channelQueueLength      prometheus.Gauge

	batcherTxEvs opmetrics.EventVec

	blobUsedBytes prometheus.Histogram

	throttleIntensity      prometheus.GaugeVec
	throttleMaxTxSize      prometheus.Gauge
	throttleMaxBlockSize   prometheus.Gauge
	throttleControllerType prometheus.GaugeVec
	pendingBytesRatio      prometheus.GaugeVec
	throttleHistory        prometheus.Summary

	// PID Controller specific metrics
	pidControllerError      prometheus.Gauge
	pidControllerIntegral   prometheus.Gauge
	pidControllerDerivative prometheus.Gauge
	pidResponseTime         prometheus.Histogram
}

var _ Metricer = (*Metrics)(nil)

// implements the Registry getter, for metrics HTTP server to hook into
var _ opmetrics.RegistryMetricer = (*Metrics)(nil)

func NewMetrics(procName string) *Metrics {
	if procName == "" {
		procName = "default"
	}
	ns := Namespace + "_" + procName

	registry := opmetrics.NewRegistry()
	factory := opmetrics.With(registry)

	m := &Metrics{
		ns:       ns,
		registry: registry,
		factory:  factory,

		RefMetrics: opmetrics.MakeRefMetrics(ns, factory),
		TxMetrics:  txmetrics.MakeTxMetrics(ns, factory),
		RPCMetrics: opmetrics.MakeRPCMetrics(ns, factory),

		info: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "info",
			Help:      "Pseudo-metric tracking version and config info",
		}, []string{
			"version",
		}),
		up: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "up",
			Help:      "1 if the op-batcher has finished starting up",
		}),

		channelEvs: opmetrics.NewEventVec(factory, ns, "", "channel", "Channel", []string{"stage"}),

		pendingBlocksCount: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pending_blocks_count",
			Help:      "Number of pending blocks, not added to a channel yet.",
		}, []string{"stage"}),
		pendingBlocksBytesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "pending_blocks_bytes_total",
			Help:      "Total size of transactions in pending blocks as they are fetched from L2",
		}),
		pendingBlocksBytesCurrent: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pending_blocks_bytes_current",
			Help:      "Current size of transactions in the pending (fetched from L2 but not in a channel) stage.",
		}),
		blocksAddedCount: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "blocks_added_count",
			Help:      "Total number of blocks added to current channel.",
		}),
		channelInputBytes: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "input_bytes",
			Help:      "Number of input bytes to a channel.",
		}, []string{"stage"}),
		channelReadyBytes: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "ready_bytes",
			Help:      "Number of bytes ready in the compression buffer.",
		}),
		channelOutputBytes: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "output_bytes",
			Help:      "Number of compressed output bytes from a channel.",
		}),
		channelClosedReason: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "channel_closed_reason",
			Help:      "Pseudo-metric to record the reason a channel got closed.",
		}),
		channelNumFrames: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "channel_num_frames",
			Help:      "Total number of frames of closed channel.",
		}),
		channelComprRatio: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "channel_compr_ratio",
			Help:      "Compression ratios of closed channel.",
			Buckets:   append([]float64{0.1, 0.2}, prometheus.LinearBuckets(0.3, 0.05, 14)...),
		}),
		channelInputBytesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "input_bytes_total",
			Help:      "Total number of bytes to a channel.",
		}),
		channelOutputBytesTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: ns,
			Name:      "output_bytes_total",
			Help:      "Total number of compressed output bytes from a channel.",
		}),
		channelQueueLength: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "channel_queue_length",
			Help:      "The number of channels currently in memory.",
		}),
		blobUsedBytes: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "blob_used_bytes",
			Help:      "Blob size in bytes (of last blob only for multi-blob txs).",
			Buckets:   prometheus.LinearBuckets(0.0, eth.MaxBlobDataSize/13, 14),
		}),

		batcherTxEvs: opmetrics.NewEventVec(factory, ns, "", "batcher_tx", "BatcherTx", []string{"stage"}),

		throttleIntensity: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "throttle_intensity",
			Help:      "Current throttling intensity (0.0 = no throttling, 1.0 = max throttling)",
		}, []string{"type"}),
		throttleMaxTxSize: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "throttle_max_tx_size",
			Help:      "Current maximum transaction size when throttling (0 = no limit)",
		}),
		throttleMaxBlockSize: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "throttle_max_block_size",
			Help:      "Current maximum block size when throttling",
		}),
		throttleControllerType: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "throttle_controller_type",
			Help:      "Type of throttle controller in use",
		}, []string{"type"}),
		pendingBytesRatio: *factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pending_bytes_ratio",
			Help:      "Ratio of pending bytes to threshold",
		}, []string{"type"}),
		throttleHistory: factory.NewSummary(prometheus.SummaryOpts{
			Namespace: ns,
			Name:      "throttle_intensity_history",
			Help:      "Historical throttle intensity values",
			Objectives: map[float64]float64{
				0.5:  0.05,  // 50th percentile with 5% error
				0.9:  0.01,  // 90th percentile with 1% error
				0.99: 0.001, // 99th percentile with 0.1% error
			},
		}),
		pidControllerError: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pid_controller_error",
			Help:      "Error term of the PID controller",
		}),
		pidControllerIntegral: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pid_controller_integral",
			Help:      "Integral term of the PID controller",
		}),
		pidControllerDerivative: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: ns,
			Name:      "pid_controller_derivative",
			Help:      "Derivative term of the PID controller",
		}),
		pidResponseTime: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: ns,
			Name:      "pid_response_time",
			Help:      "Response time of the PID controller",
			Buckets:   prometheus.DefBuckets,
		}),
	}
	m.pendingDABytesGaugeFunc = factory.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: ns,
		Name:      "pending_da_bytes",
		Help:      "The estimated amount of data currently pending to be written to the DA layer (from blocks fetched from L2 but not yet in a channel).",
	}, m.PendingDABytes)

	return m
}

func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Metrics) Document() []opmetrics.DocumentedMetric {
	return m.factory.Document()
}

// PendingDABytes returns the current number of bytes pending to be written to the DA layer (from blocks fetched from L2
// but not yet in a channel).
func (m *Metrics) PendingDABytes() float64 {
	return float64(atomic.LoadInt64(&m.pendingDABytes))
}

func (m *Metrics) StartBalanceMetrics(l log.Logger, client *ethclient.Client, account common.Address) io.Closer {
	return opmetrics.LaunchBalanceMetrics(l, m.registry, m.ns, client, account)
}

// RecordInfo sets a pseudo-metric that contains versioning and
// config info for the op-batcher.
func (m *Metrics) RecordInfo(version string) {
	m.info.WithLabelValues(version).Set(1)
}

// RecordUp sets the up metric to 1.
func (m *Metrics) RecordUp() {
	m.up.Set(1)
}

const (
	StageLoaded         = "loaded"
	StageOpened         = "opened"
	StageAdded          = "added"
	StageClosed         = "closed"
	StageFullySubmitted = "fully_submitted"
	StageTimedOut       = "timed_out"

	TxStageSubmitted = "submitted"
	TxStageSuccess   = "success"
	TxStageFailed    = "failed"
)

func (m *Metrics) RecordLatestL1Block(l1ref eth.L1BlockRef) {
	m.RecordL1Ref("latest", l1ref)
}

// RecordL2BlocksLoaded should be called when a new L2 block was loaded into the
// channel manager (but not processed yet).
func (m *Metrics) RecordL2BlocksLoaded(l2ref eth.L2BlockRef) {
	m.RecordL2Ref(StageLoaded, l2ref)
}

func (m *Metrics) RecordChannelOpened(id derive.ChannelID, numPendingBlocks int) {
	m.channelEvs.Record(StageOpened)
	m.blocksAddedCount.Set(0) // reset
	m.pendingBlocksCount.WithLabelValues(StageOpened).Set(float64(numPendingBlocks))
}

// RecordL2BlocksAdded should be called when L2 block were added to the channel
// builder, with the latest added block.
func (m *Metrics) RecordL2BlocksAdded(l2ref eth.L2BlockRef, numBlocksAdded, numPendingBlocks, inputBytes, outputComprBytes int) {
	m.RecordL2Ref(StageAdded, l2ref)
	m.blocksAddedCount.Add(float64(numBlocksAdded))
	m.pendingBlocksCount.WithLabelValues(StageAdded).Set(float64(numPendingBlocks))
	m.channelInputBytes.WithLabelValues(StageAdded).Set(float64(inputBytes))
	m.channelReadyBytes.Set(float64(outputComprBytes))
}

func (m *Metrics) RecordChannelClosed(id derive.ChannelID, numPendingBlocks int, numFrames int, inputBytes int, outputComprBytes int, reason error) {
	m.channelEvs.Record(StageClosed)
	m.pendingBlocksCount.WithLabelValues(StageClosed).Set(float64(numPendingBlocks))
	m.channelNumFrames.Set(float64(numFrames))
	m.channelInputBytes.WithLabelValues(StageClosed).Set(float64(inputBytes))
	m.channelOutputBytes.Set(float64(outputComprBytes))
	m.channelInputBytesTotal.Add(float64(inputBytes))
	m.channelOutputBytesTotal.Add(float64(outputComprBytes))

	var comprRatio float64
	if inputBytes > 0 {
		comprRatio = float64(outputComprBytes) / float64(inputBytes)
	}
	m.channelComprRatio.Observe(comprRatio)

	m.channelClosedReason.Set(float64(ClosedReasonToNum(reason)))
}

func (m *Metrics) RecordL2BlockInPendingQueue(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.pendingBlocksBytesTotal.Add(float64(rawSize))
	m.pendingBlocksBytesCurrent.Add(float64(rawSize))
	atomic.AddInt64(&m.pendingDABytes, int64(daSize))
}

func (m *Metrics) RecordL2BlockInChannel(block *types.Block) {
	daSize, rawSize := estimateBatchSize(block)
	m.pendingBlocksBytesCurrent.Add(-1.0 * float64(rawSize))
	atomic.AddInt64(&m.pendingDABytes, -1*int64(daSize))
	// Refer to RecordL2BlocksAdded to see the current + count of bytes added to a channel
}

func ClosedReasonToNum(reason error) int {
	// CLI-3640
	return 0
}

func (m *Metrics) RecordChannelFullySubmitted(id derive.ChannelID) {
	m.channelEvs.Record(StageFullySubmitted)
}

func (m *Metrics) RecordChannelTimedOut(id derive.ChannelID) {
	m.channelEvs.Record(StageTimedOut)
}

func (m *Metrics) RecordBatchTxSubmitted() {
	m.batcherTxEvs.Record(TxStageSubmitted)
}

func (m *Metrics) RecordBatchTxSuccess() {
	m.batcherTxEvs.Record(TxStageSuccess)
}

func (m *Metrics) RecordBatchTxFailed() {
	m.batcherTxEvs.Record(TxStageFailed)
}

func (m *Metrics) RecordBlobUsedBytes(num int) {
	m.blobUsedBytes.Observe(float64(num))
}

func (m *Metrics) RecordChannelQueueLength(len int) {
	m.channelQueueLength.Set(float64(len))
}

func (m *Metrics) RecordThrottleIntensity(intensity float64, controllerType config.ThrottleControllerType) {
	for _, t := range config.ThrottleControllerTypes {
		if t == controllerType {
			m.throttleIntensity.WithLabelValues(string(t)).Set(intensity)
		} else {
			m.throttleIntensity.WithLabelValues(string(t)).Set(0)
		}
	}
	m.throttleHistory.Observe(intensity)
}

func (m *Metrics) RecordThrottleParams(maxTxSize, maxBlockSize uint64) {
	m.throttleMaxTxSize.Set(float64(maxTxSize))
	m.throttleMaxBlockSize.Set(float64(maxBlockSize))
}

func (m *Metrics) RecordThrottleControllerType(controllerType config.ThrottleControllerType) {
	for _, t := range config.ThrottleControllerTypes {
		if t == controllerType {
			m.throttleControllerType.WithLabelValues(string(t)).Set(1)
		} else {
			m.throttleControllerType.WithLabelValues(string(t)).Set(0)
		}
	}
}

func (m *Metrics) RecordPendingBytesVsThreshold(pendingBytes, threshold uint64, controllerType config.ThrottleControllerType) {
	ratio := float64(pendingBytes) / float64(threshold)
	for _, t := range config.ThrottleControllerTypes {
		if t == controllerType {
			m.pendingBytesRatio.WithLabelValues(string(t)).Set(ratio)
		} else {
			m.pendingBytesRatio.WithLabelValues(string(t)).Set(0)
		}
	}
}

// ClearAllStateMetrics clears all state metrics.
//
// This should cover any metric which is a Gauge and is incremented / decremented rather than "set".
// Counter Metrics only ever go up, so they can't be reset and shouldn't be.
// Gauge Metrics which are "set" will get the right value the next time they are updated and don't need to be reset.
func (m *Metrics) ClearAllStateMetrics() {
	m.RecordChannelQueueLength(0)
	atomic.StoreInt64(&m.pendingDABytes, 0)
	m.pendingBlocksBytesCurrent.Set(0)
}

// estimateBatchSize returns the estimated size of the block in a batch both with compression ('daSize') and without
// ('rawSize').
func estimateBatchSize(block *types.Block) (daSize, rawSize uint64) {
	daSize = uint64(70) // estimated overhead of batch metadata
	rawSize = uint64(70)
	for _, tx := range block.Transactions() {
		// Deposit transactions are not included in batches
		if tx.IsDepositTx() {
			continue
		}
		bigSize := tx.RollupCostData().EstimatedDASize()
		if bigSize.IsUint64() { // this should always be true, but if not just ignore
			daSize += bigSize.Uint64()
		}
		// Add 2 for the overhead of encoding the tx bytes in a RLP list
		rawSize += tx.Size() + 2
	}
	return
}

// RecordThrottleControllerState records the state of the PID controller
func (m *Metrics) RecordThrottleControllerState(error, integral, derivative float64) {
	m.pidControllerError.Set(error)
	m.pidControllerIntegral.Set(integral)
	m.pidControllerDerivative.Set(derivative)
}

// RecordThrottleResponseTime records the response time of the PID controller
func (m *Metrics) RecordThrottleResponseTime(duration time.Duration) {
	m.pidResponseTime.Observe(duration.Seconds())
}
