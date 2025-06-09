package loadtest

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	subsystemName = "interop_loadtest"

	inFlightMessagesName        = "inflight_messages"
	targetMessagesPerBlockName  = "target_messages_per_block"
	messageLatencyName          = "message_latency"
	txSubmissionStatusCountName = "tx_submission_status_count"
)

var (
	inFlightMessages = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      inFlightMessagesName,
		Subsystem: subsystemName,
		Help:      "Number of messages currently in flight between L2 chains",
	})

	targetMessagesPerBlock = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      targetMessagesPerBlockName,
		Subsystem: subsystemName,
		Help:      "Current target messages per block from AIMD scheduler",
	})

	messageLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:      messageLatencyName,
		Subsystem: subsystemName,
		Help:      "Message latencies by stage (init, exec, e2e)",
	}, []string{"stage"})

	txSubmissionStatusCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      txSubmissionStatusCountName,
		Subsystem: subsystemName,
		Help:      "Total number of transaction submission attempts by chain and status",
	}, []string{"chain", "status"})
)

var (
	colors = map[string]color.RGBA{
		"VividRed":   {R: 242, G: 36, B: 36, A: 255},
		"OrangeRed":  {R: 242, G: 98, B: 36, A: 255},
		"Goldenrod":  {R: 242, G: 160, B: 36, A: 255},
		"YellowGold": {R: 242, G: 222, B: 36, A: 255},
		"Chartreuse": {R: 201, G: 242, B: 36, A: 255},
		"Lime":       {R: 139, G: 242, B: 36, A: 255},
		"Spring":     {R: 78, G: 242, B: 36, A: 255},
		"Emerald":    {R: 36, G: 242, B: 57, A: 255},
		"Aqua":       {R: 36, G: 242, B: 119, A: 255},
		"Turquoise":  {R: 36, G: 242, B: 180, A: 255},
		"Cyan":       {R: 36, G: 242, B: 242, A: 255},
		"SkyBlue":    {R: 36, G: 180, B: 242, A: 255},
		"Azure":      {R: 36, G: 119, B: 242, A: 255},
		"RoyalBlue":  {R: 36, G: 57, B: 242, A: 255},
		"Indigo":     {R: 78, G: 36, B: 242, A: 255},
		"Violet":     {R: 139, G: 36, B: 242, A: 255},
		"Purple":     {R: 201, G: 36, B: 242, A: 255},
		"Magenta":    {R: 242, G: 36, B: 222, A: 255},
		"Fuchsia":    {R: 242, G: 36, B: 160, A: 255},
		"Crimson":    {R: 242, G: 36, B: 98, A: 255},
	}

	// colorOrder is a slice of color names that maximizes contrast between consecutive colors by
	// jumping across the color wheel.
	colorOrder = []string{
		"VividRed", "Cyan", "YellowGold", "RoyalBlue", "Lime", "Magenta", "OrangeRed", "Turquoise",
		"Purple", "Spring", "Crimson", "SkyBlue", "Chartreuse", "Indigo", "Goldenrod", "Aqua",
		"Fuchsia", "Emerald", "Violet", "Azure",
	}
)

var (
	e2eColor  = colors["RoyalBlue"]
	initColor = colors["YellowGold"]
	execColor = colors["VividRed"]
)

// MetricSample represents a single metric sample at a point in time
type MetricSample struct {
	Timestamp time.Time
	Value     float64
	Count     uint64 // Count is only used in histograms.
	Labels    []string
}

type MetricSamples []MetricSample

func (samples MetricSamples) UniqueLabels(i int) []string {
	var labels []string
Outer:
	for _, sample := range samples {
		l := sample.Labels[i]
		for _, label := range labels {
			if l == label {
				continue Outer
			}
		}
		labels = append(labels, l)
	}
	return labels
}

func (samples MetricSamples) WithLabels(labels ...string) MetricSamples {
	newSamples := make([]MetricSample, 0)
	for _, sample := range samples {
		if isSubset(labels, sample.Labels) {
			newSamples = append(newSamples, sample)
		}
	}
	return newSamples
}

func (samples MetricSamples) ToPoints(startTime time.Time) plotter.XYs {
	pts := make(plotter.XYs, 0, len(samples))
	for _, sample := range samples {
		pts = append(pts, plotter.XY{
			X: sample.Timestamp.Sub(startTime).Seconds(),
			Y: sample.Value,
		})
	}
	return pts
}

func (samples MetricSamples) ToValuePerIntervalPoints(startTime time.Time) plotter.XYs {
	pts := make(plotter.XYs, 0, len(samples))
	var prevValue float64
	for _, sample := range samples {
		pts = append(pts, plotter.XY{
			X: sample.Timestamp.Sub(startTime).Seconds(),
			Y: sample.Value - prevValue,
		})
		prevValue = sample.Value
	}
	return pts
}

func (samples MetricSamples) ToHistogramPoints(startTime time.Time) plotter.XYs {
	pts := make(plotter.XYs, 0, len(samples))
	var prevValue float64
	var prevCount uint64
	for _, sample := range samples {
		if count := sample.Count - prevCount; count > 0 {
			pts = append(pts, plotter.XY{
				X: sample.Timestamp.Sub(startTime).Seconds(),
				Y: (sample.Value - prevValue) / float64(count), // Average over the sample interval.
			})
		}
		prevCount = sample.Count
		prevValue = sample.Value
	}
	return pts
}

func isSubset[T comparable](xs []T, ys []T) bool {
	if len(xs) > len(ys) {
		return false
	}
Outer:
	for _, x := range xs {
		for _, y := range ys {
			if x == y {
				continue Outer
			}
		}
		return false
	}
	return true
}

// MetricsCollector collects metrics samples over time
type MetricsCollector struct {
	samples   map[string]MetricSamples
	blockTime time.Duration
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector with the given sampling interval.
func NewMetricsCollector(blockTime time.Duration) *MetricsCollector {
	return &MetricsCollector{
		samples:   make(map[string]MetricSamples),
		blockTime: blockTime,
	}
}

// Start begins collecting metrics samples
func (mc *MetricsCollector) Start(ctx context.Context) error {
	mc.startTime = time.Now()
	ticker := time.NewTicker(mc.blockTime)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			metricFamilies, err := prometheus.DefaultGatherer.Gather()
			if err != nil {
				return fmt.Errorf("gather metrics: %w", err)
			}
			for _, metricFamily := range metricFamilies {
				name, hasPrefix := strings.CutPrefix(metricFamily.GetName(), subsystemName+"_")
				if !hasPrefix {
					continue // Skip metrics we don't care about.
				}
				for _, metric := range metricFamily.GetMetric() {
					var value float64
					var count uint64
					if metric.Gauge != nil && metric.Gauge.Value != nil {
						value = *metric.Gauge.Value
					} else if metric.Counter != nil && metric.Counter.Value != nil {
						value = *metric.Counter.Value
					} else if metric.Histogram != nil {
						count = metric.Histogram.GetSampleCount()
						value = metric.Histogram.GetSampleSum()
					}
					labels := make([]string, 0, len(metric.Label))
					for _, labelPair := range metric.Label {
						labels = append(labels, labelPair.GetValue())
					}
					mc.samples[name] = append(mc.samples[name], MetricSample{
						Timestamp: now,
						Value:     value,
						Count:     count,
						Labels:    labels,
					})
				}
			}
		}
	}
}

// SaveGraphs generates and saves graphs of collected metrics over time
func (mc *MetricsCollector) SaveGraphs(dir string) error {
	if err := mc.saveInFlightMessagesGraph(dir); err != nil {
		return fmt.Errorf("save in-flight messages graph: %w", err)
	}
	if err := mc.saveTargetMessagesPerBlockGraph(dir); err != nil {
		return fmt.Errorf("save target messages per block graph: %w", err)
	}
	if err := mc.saveMessageCountGraph(dir); err != nil {
		return fmt.Errorf("save message count graph: %w", err)
	}
	if err := mc.saveMessageLatencyGraph(dir); err != nil {
		return fmt.Errorf("save message latency graph: %w", err)
	}
	if err := mc.saveTxSubmissionStatusCountGraphs(dir); err != nil {
		return fmt.Errorf("save tx submission status count graphs: %w", err)
	}
	return nil
}

func (mc *MetricsCollector) saveInFlightMessagesGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "In-Flight Messages"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	if _, err := addLine(p, mc.samples[inFlightMessagesName].ToPoints(mc.startTime), colors[colorOrder[0]]); err != nil {
		return err
	}

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, inFlightMessagesName)
}

func (mc *MetricsCollector) saveTargetMessagesPerBlockGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Target Messages Per Block Time"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Target"

	if _, err := addLine(p, mc.samples[targetMessagesPerBlockName].ToPoints(mc.startTime), colors[colorOrder[0]]); err != nil {
		return err
	}

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, targetMessagesPerBlockName)
}

func (mc *MetricsCollector) saveMessageCountGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Messages per Block Time"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	latencySamples := mc.samples[messageLatencyName].WithLabels("e2e")
	countSamples := make(MetricSamples, 0, len(latencySamples))
	for _, latencySample := range latencySamples {
		countSamples = append(countSamples, MetricSample{
			Timestamp: latencySample.Timestamp,
			Value:     float64(latencySample.Count),
			Labels:    latencySample.Labels,
		})
	}

	if _, err := addLine(p, countSamples.ToValuePerIntervalPoints(mc.startTime), colors[colorOrder[0]]); err != nil {
		return fmt.Errorf("create line plot: %w", err)
	}

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, "message_count")
}

func (mc *MetricsCollector) saveMessageLatencyGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Message Latency by Stage"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Latency"

	samples := mc.samples[messageLatencyName]

	e2eLine, err := addLine(p, samples.WithLabels("e2e").ToHistogramPoints(mc.startTime), e2eColor)
	if err != nil {
		return fmt.Errorf("success: %w", err)
	}
	p.Legend.Add("e2e", e2eLine)

	initLine, err := addLine(p, samples.WithLabels("init").ToHistogramPoints(mc.startTime), initColor)
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	p.Legend.Add("init", initLine)

	execLine, err := addLine(p, samples.WithLabels("exec").ToHistogramPoints(mc.startTime), execColor)
	if err != nil {
		return fmt.Errorf("exec: %w", err)
	}
	p.Legend.Add("exec", execLine)

	p.Add(plotter.NewGrid())
	p.Legend.Top = true

	return savePlot(p, dir, messageLatencyName)
}

func (mc *MetricsCollector) saveTxSubmissionStatusCountGraphs(dir string) error {
	samples := mc.samples[txSubmissionStatusCountName]
	for _, chain := range samples.UniqueLabels(0) {
		p := plot.New()
		p.Title.Text = "Transaction Submission Count by Status on Chain " + chain
		p.X.Label.Text = "Time (seconds)"
		p.Y.Label.Text = "Count"

		chainSamples := samples.WithLabels(chain)
		for i, status := range chainSamples.UniqueLabels(1) {
			statusPoints := chainSamples.WithLabels(status).ToValuePerIntervalPoints(mc.startTime)
			// Prometheus's Gatherer interface guarantees the statuses are sorted,
			// so we will always assign them the same colors.
			line, err := addLine(p, statusPoints, colors[colorOrder[i%len(colorOrder)]])
			if err != nil {
				return fmt.Errorf("%s: %w", status, err)
			}
			p.Legend.Add(status, line)
		}

		p.Add(plotter.NewGrid())
		p.Legend.Top = true

		if err := savePlot(p, dir, txSubmissionStatusCountName+"_"+chain); err != nil {
			return err
		}
	}
	return nil
}

func addLine(p *plot.Plot, points plotter.XYs, c color.Color) (*plotter.Line, error) {
	line, err := plotter.NewLine(points)
	if err != nil {
		return nil, fmt.Errorf("create line: %w", err)
	}
	line.Color = c
	line.Width = vg.Points(2)
	p.Add(line)
	return line, nil
}

func savePlot(p *plot.Plot, dir, name string) error {
	filename := filepath.Join(dir, name+".png")
	if err := p.Save(10*vg.Inch, 6*vg.Inch, filename); err != nil {
		return fmt.Errorf("save plot: %w", err)
	}
	return nil
}

type ResubmitterObserver string

var _ txinclude.ResubmitterObserver = (*ResubmitterObserver)(nil)

func (m ResubmitterObserver) SubmissionError(err error) {
	var status string
	if err == nil {
		status = "success"
	} else {
		status = sanitizePrometheusLabel(err.Error())
	}
	txSubmissionStatusCount.WithLabelValues(string(m), status).Add(1)
}

var (
	notAllowedRegex = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	startingRegex   = regexp.MustCompile(`^[a-zA-Z_]`)
)

// sanitizePrometheusLabel transforms a string into a valid Prometheus label.
func sanitizePrometheusLabel(s string) string {
	// Replace any non-alphanumeric/underscore characters with underscores.
	sanitized := notAllowedRegex.ReplaceAllString(s, "_")
	// Ensure it starts with letter or underscore.
	if len(sanitized) > 0 && !startingRegex.MatchString(sanitized) {
		sanitized = "_" + sanitized
	}
	// Remove trailing underscores for cleanliness.
	return strings.TrimRight(sanitized, "_")
}
