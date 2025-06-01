package loadtest

import (
	"context"
	"fmt"
	"image/color"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	inFlightMessagesName       = "inflight_messages"
	targetMessagesPerBlockName = "target_messages_per_block"
	messageStatusCountName     = "message_status_count"
)

var (
	inFlightMessages = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      inFlightMessagesName,
		Subsystem: "interop_loadtest",
		Help:      "Number of messages currently in flight between L2 chains",
	})

	targetMessagesPerBlock = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      targetMessagesPerBlockName,
		Subsystem: "interop_loadtest",
		Help:      "Current target messages per block from AIMD scheduler",
	})

	messageStatusCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      messageStatusCountName,
		Subsystem: "interop_loadtest",
		Help:      "Total number of messages by status (success, init_failed, exec_failed)",
	}, []string{"status"})
)

// MetricSample represents a single metric sample at a point in time
type MetricSample struct {
	Timestamp time.Time
	Value     float64
	Labels    []string
}

func (sample *MetricSample) ToPoint(startTime time.Time) plotter.XY {
	return plotter.XY{
		X: sample.Timestamp.Sub(startTime).Seconds(),
		Y: sample.Value,
	}
}

type MetricSamples []MetricSample

func (samples *MetricSamples) ToPoints(startTime time.Time, labels ...string) plotter.XYs {
	pts := make(plotter.XYs, 0)
	for _, sample := range *samples {
		if isSubset(labels, sample.Labels) {
			pts = append(pts, sample.ToPoint(startTime))
		}
	}
	return pts
}

func (samples *MetricSamples) Append(timestamp time.Time, m prometheus.Metric) error {
	sample := &dto.Metric{}
	if err := m.Write(sample); err != nil {
		return fmt.Errorf("write metric to sample: %w", err)
	}
	var value float64
	if sample.Gauge != nil && sample.Gauge.Value != nil {
		value = *sample.Gauge.Value
	} else if sample.Counter != nil && sample.Counter.Value != nil {
		value = *sample.Counter.Value
	}
	labels := make([]string, 0, len(sample.Label))
	for _, labelPair := range sample.Label {
		labels = append(labels, labelPair.GetValue())
	}
	*samples = append(*samples, MetricSample{
		Timestamp: timestamp,
		Value:     value,
		Labels:    labels,
	})
	return nil
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
	inFlightMessagesSamples       MetricSamples
	targetMessagesPerBlockSamples MetricSamples
	messageStatusSamples          MetricSamples
	blockTime                     time.Duration
	startTime                     time.Time
}

// NewMetricsCollector creates a new metrics collector with the given sampling interval.
func NewMetricsCollector(blockTime time.Duration) *MetricsCollector {
	return &MetricsCollector{
		inFlightMessagesSamples:       make([]MetricSample, 0),
		targetMessagesPerBlockSamples: make([]MetricSample, 0),
		messageStatusSamples:          make([]MetricSample, 0),
		blockTime:                     blockTime,
	}
}

// Start begins collecting metrics samples
func (mc *MetricsCollector) Start(ctx context.Context) error {
	lastMessageStatusCounts := make(map[string]float64, 0)
	mc.startTime = time.Now()
	ticker := time.NewTicker(mc.blockTime)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			now := time.Now()
			if err := mc.inFlightMessagesSamples.Append(now, inFlightMessages); err != nil {
				return fmt.Errorf("append to inFlightMessagesSamples: %w", err)
			}
			if err := mc.targetMessagesPerBlockSamples.Append(now, targetMessagesPerBlock); err != nil {
				return fmt.Errorf("append to targetMessagesPerBlockSamples: %w", err)
			}
			for _, status := range []string{"success", "init_failed", "exec_failed"} {
				if err := mc.messageStatusSamples.Append(now, messageStatusCount.WithLabelValues(status)); err != nil {
					return fmt.Errorf("append to messageStatusSamples: %w", err)
				}
				currentCount := mc.messageStatusSamples[len(mc.messageStatusSamples)-1].Value
				delta := currentCount - lastMessageStatusCounts[status]
				mc.messageStatusSamples[len(mc.messageStatusSamples)-1].Value = delta
				lastMessageStatusCounts[status] = currentCount
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
	if err := mc.saveMessageStatusGraph(dir); err != nil {
		return fmt.Errorf("save message status graph: %w", err)
	}
	return nil
}

func (mc *MetricsCollector) saveInFlightMessagesGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "In-Flight Messages"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	line, err := plotter.NewLine(mc.inFlightMessagesSamples.ToPoints(mc.startTime))
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

	line, err := plotter.NewLine(mc.targetMessagesPerBlockSamples.ToPoints(mc.startTime))
	if err != nil {
		return fmt.Errorf("create line plot: %w", err)
	}
	p.Add(line)

	p.Add(plotter.NewGrid())

	return savePlot(p, dir, targetMessagesPerBlockName)
}

func (mc *MetricsCollector) saveMessageStatusGraph(dir string) error {
	p := plot.New()
	p.Title.Text = "Messages by Status"
	p.X.Label.Text = "Time (seconds)"
	p.Y.Label.Text = "Messages"

	successLine, err := plotter.NewLine(mc.messageStatusSamples.ToPoints(mc.startTime, "success"))
	if err != nil {
		return fmt.Errorf("create success line: %w", err)
	}
	successLine.Color = color.RGBA{R: 0, G: 200, B: 0, A: 255} // Green
	successLine.Width = vg.Points(2)
	p.Add(successLine)
	p.Legend.Add("Success", successLine)

	initFailedLine, err := plotter.NewLine(mc.messageStatusSamples.ToPoints(mc.startTime, "init_failed"))
	if err != nil {
		return fmt.Errorf("create init_failed line: %w", err)
	}
	initFailedLine.Color = color.RGBA{R: 200, G: 0, B: 0, A: 255} // Red
	initFailedLine.Width = vg.Points(2)
	p.Add(initFailedLine)
	p.Legend.Add("Init Failed", initFailedLine)

	execFailedLine, err := plotter.NewLine(mc.messageStatusSamples.ToPoints(mc.startTime, "exec_failed"))
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

func savePlot(p *plot.Plot, dir, name string) error {
	filename := filepath.Join(dir, fmt.Sprintf("%s_%s.png", name, time.Now().Format("20060102_150405")))
	if err := p.Save(10*vg.Inch, 6*vg.Inch, filename); err != nil {
		return fmt.Errorf("save plot: %w", err)
	}
	return nil
}
