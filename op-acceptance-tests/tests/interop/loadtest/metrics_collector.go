package loadtest

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	subsystemName = "interop_loadtest"

	inFlightMessagesName       = "inflight_messages"
	targetMessagesPerBlockName = "target_messages_per_block"
	messageStatusCountName     = "message_status_count"
	messageLatencyName         = "message_latency"
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

	messageStatusCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      messageStatusCountName,
		Subsystem: subsystemName,
		Help:      "Total number of messages by status (success, init_failed, exec_failed)",
	}, []string{"status"})

	messageLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:      messageLatencyName,
		Subsystem: subsystemName,
		Help:      "Message latencies by stage (init, exec, e2e)",
	}, []string{"stage"})
)

var (
	e2eColor  = color.RGBA{R: 0, G: 200, B: 0, A: 255} // Green
	initColor = color.RGBA{R: 200, G: 0, B: 0, A: 255} // Red
	execColor = color.RGBA{R: 0, G: 0, B: 200, A: 255} // Blue
)

// MetricSample represents a single metric sample at a point in time
type MetricSample struct {
	Timestamp time.Time
	Value     float64
	Count     uint64 // Count is only used in histograms.
	Labels    []string
}

type MetricSamples []MetricSample

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

// SaveGraph generates and saves graphs of collected metrics over time
func (mc *MetricsCollector) SaveGraph(dir string) error {
	if err := mc.saveInFlightMessagesGraph(dir); err != nil {
		return fmt.Errorf("save in-flight messages graph: %w", err)
	}
	if err := mc.saveTargetMessagesPerBlockGraph(dir); err != nil {
		return fmt.Errorf("save target messages per block graph: %w", err)
	}
	if err := mc.saveMessageStatusCountGraph(dir); err != nil {
		return fmt.Errorf("save message status count graph: %w", err)
	}
	if err := mc.saveMessageLatencyGraph(dir); err != nil {
		return fmt.Errorf("save message latency graph: %w", err)
	}
	return nil
}

func (mc *MetricsCollector) saveInFlightMessagesGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "In-Flight Messages"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	line, err := plotter.NewLine(mc.samples[inFlightMessagesName].ToPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create line plot: %w", err)
	}
	p.Add(line)

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, inFlightMessagesName)
}

func (mc *MetricsCollector) saveTargetMessagesPerBlockGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Target Messages Per Block Time"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Target"

	line, err := plotter.NewLine(mc.samples[targetMessagesPerBlockName].ToPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create line plot: %w", err)
	}
	p.Add(line)

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, targetMessagesPerBlockName)
}

func (mc *MetricsCollector) saveMessageStatusCountGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Messages by Status"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	samples := mc.samples[messageStatusCountName]

	successLine, err := plotter.NewLine(samples.WithLabels("success").ToValuePerIntervalPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create success line: %w", err)
	}
	successLine.Color = color.RGBA{R: 0, G: 200, B: 0, A: 255} // Green
	successLine.Width = vg.Points(2)
	p.Add(successLine)
	p.Legend.Add("Success", successLine)

	initFailedLine, err := plotter.NewLine(samples.WithLabels("init_failed").ToValuePerIntervalPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create init_failed line: %w", err)
	}
	initFailedLine.Color = color.RGBA{R: 200, G: 0, B: 0, A: 255} // Red
	initFailedLine.Width = vg.Points(2)
	p.Add(initFailedLine)
	p.Legend.Add("Init Failed", initFailedLine)

	execFailedLine, err := plotter.NewLine(samples.WithLabels("exec_failed").ToValuePerIntervalPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create exec_failed line: %w", err)
	}
	execFailedLine.Color = color.RGBA{R: 0, G: 0, B: 200, A: 255} // Blue
	execFailedLine.Width = vg.Points(2)
	p.Add(execFailedLine)
	p.Legend.Add("Exec Failed", execFailedLine)

	p.Add(plotter.NewGrid())
	p.Legend.Top = true

	return savePlot(p, dir, messageStatusCountName)
}

func (mc *MetricsCollector) saveMessageLatencyGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Message Latency by Stage"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Latency"

	samples := mc.samples[messageLatencyName]

	e2eLine, err := plotter.NewLine(samples.WithLabels("e2e").ToHistogramPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create success line: %w", err)
	}
	e2eLine.Color = e2eColor
	e2eLine.Width = vg.Points(2)
	p.Add(e2eLine)
	p.Legend.Add("E2E", e2eLine)

	initLine, err := plotter.NewLine(samples.WithLabels("init").ToHistogramPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create init_failed line: %w", err)
	}
	initLine.Color = initColor
	initLine.Width = vg.Points(2)
	p.Add(initLine)
	p.Legend.Add("Init", initLine)

	execLine, err := plotter.NewLine(samples.WithLabels("exec").ToHistogramPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create exec_failed line: %w", err)
	}
	execLine.Color = execColor
	execLine.Width = vg.Points(2)
	p.Add(execLine)
	p.Legend.Add("Exec", execLine)

	p.Add(plotter.NewGrid())
	p.Legend.Top = true

	return savePlot(p, dir, messageLatencyName)
}

func savePlot(p *plot.Plot, dir, name string) error {
	filename := filepath.Join(dir, fmt.Sprintf("%s_%s.png", name, time.Now().Format("20060102_150405")))
	if err := p.Save(10*vg.Inch, 6*vg.Inch, filename); err != nil {
		return fmt.Errorf("save plot: %w", err)
	}
	return nil
}
