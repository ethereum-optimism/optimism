package aggregator

import (
	"encoding/json"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// WeeklyReport is the aggregated weekly report.
type WeeklyReport struct {
	Week      string    `json:"week"` // ISO week (e.g., "2026-W10")
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// Top-line metrics.
	TotalPipelines int     `json:"total_pipelines"`
	CatchRate      float64 `json:"catch_rate"`
	FalseNegatives int     `json:"false_negatives"`

	// Performance.
	MedianSpeedup float64 `json:"median_speedup"`
	P95WallTime   float64 `json:"p95_wall_time_seconds"`

	// Flakes.
	FlakesDetected       int      `json:"flakes_detected"`
	UniqueFlakePatterns  int      `json:"unique_flake_patterns"`
	TopFlakes            []FlakeEntry `json:"top_flakes"`

	// Efficiency.
	ByLanguage map[string]LanguageWeekly `json:"by_language"`

	// Graph health.
	GraphGaps      int `json:"graph_gaps"`
	ConfidenceGain int `json:"confidence_gain"`
}

// LanguageWeekly holds per-language weekly stats.
type LanguageWeekly struct {
	AvgSkipRate     float64 `json:"avg_skip_rate"`
	TotalSelected   int     `json:"total_selected"`
	TotalAvailable  int     `json:"total_available"`
	Configurations  int     `json:"configurations"`
}

// FlakeEntry represents a top flake in the report.
type FlakeEntry struct {
	Fingerprint string `json:"fingerprint"`
	Count       int    `json:"count"`
	LastSeen    time.Time `json:"last_seen"`
	Test        string `json:"test"`
}

// DashboardData is the data structure for the static dashboard.
type DashboardData struct {
	GeneratedAt time.Time `json:"generated_at"`

	// Time series (last 30 days).
	CatchRateSeries  []TimePoint `json:"catch_rate_series"`
	SpeedupSeries    []TimePoint `json:"speedup_series"`
	SkipRateSeries   map[string][]TimePoint `json:"skip_rate_series"`

	// Current state.
	FlakeLeaderboard []FlakeEntry `json:"flake_leaderboard"`
	FalseNegativeLog []model.FalseNegativeDetail `json:"false_negative_log"`

	// Summary.
	TotalPipelines    int     `json:"total_pipelines"`
	OverallCatchRate  float64 `json:"overall_catch_rate"`
	OverallSpeedup    float64 `json:"overall_speedup"`
	ActiveFlakes      int     `json:"active_flakes"`
}

// TimePoint is a single data point in a time series.
type TimePoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// Aggregator reads events and produces reports and dashboard data.
type Aggregator struct {
	store events.Store
}

// NewAggregator creates a new Aggregator.
func NewAggregator(store events.Store) *Aggregator {
	return &Aggregator{store: store}
}

// GenerateWeeklyReport produces a weekly report for the given week.
func (a *Aggregator) GenerateWeeklyReport(start, end time.Time) (*WeeklyReport, error) {
	year, week := start.ISOWeek()
	report := &WeeklyReport{
		Week:      formatISOWeek(year, week),
		StartDate: start,
		EndDate:   end,
		ByLanguage: make(map[string]LanguageWeekly),
	}

	// Query comparison events.
	comparisons, err := a.store.Query(events.EventFilter{
		Types: []model.EventType{model.EventComparisonComplete},
		After: start,
		Before: end,
	})
	if err != nil {
		return nil, err
	}

	report.TotalPipelines = len(comparisons)

	var totalCatchRate float64
	var totalSpeedup float64
	for _, evt := range comparisons {
		var comp struct {
			CatchRate      float64 `json:"catch_rate"`
			FalseNegatives int     `json:"false_negatives"`
			Speedup        float64 `json:"speedup"`
		}
		json.Unmarshal(evt.Payload, &comp)

		totalCatchRate += comp.CatchRate
		totalSpeedup += comp.Speedup
		report.FalseNegatives += comp.FalseNegatives
	}

	if report.TotalPipelines > 0 {
		report.CatchRate = totalCatchRate / float64(report.TotalPipelines)
		report.MedianSpeedup = totalSpeedup / float64(report.TotalPipelines)
	}

	// Query flake events.
	flakes, err := a.store.Query(events.EventFilter{
		Types: []model.EventType{model.EventFlakeDetected},
		After: start,
		Before: end,
	})
	if err != nil {
		return nil, err
	}

	report.FlakesDetected = len(flakes)

	// Count unique fingerprints and build top flakes.
	fpCount := make(map[string]int)
	fpLast := make(map[string]time.Time)
	fpTest := make(map[string]string)
	for _, evt := range flakes {
		var fp model.FlakePayload
		json.Unmarshal(evt.Payload, &fp)
		fpCount[fp.Fingerprint]++
		fpLast[fp.Fingerprint] = evt.Timestamp
		fpTest[fp.Fingerprint] = fp.Result.Test.Key()
	}
	report.UniqueFlakePatterns = len(fpCount)

	for fp, count := range fpCount {
		report.TopFlakes = append(report.TopFlakes, FlakeEntry{
			Fingerprint: fp,
			Count:       count,
			LastSeen:    fpLast[fp],
			Test:        fpTest[fp],
		})
	}

	// Sort top flakes by count (simple bubble for small list).
	for i := 0; i < len(report.TopFlakes); i++ {
		for j := i + 1; j < len(report.TopFlakes); j++ {
			if report.TopFlakes[j].Count > report.TopFlakes[i].Count {
				report.TopFlakes[i], report.TopFlakes[j] = report.TopFlakes[j], report.TopFlakes[i]
			}
		}
	}
	if len(report.TopFlakes) > 10 {
		report.TopFlakes = report.TopFlakes[:10]
	}

	// Query graph gap events.
	gaps, err := a.store.Query(events.EventFilter{
		Types: []model.EventType{model.EventGraphGap},
		After: start,
		Before: end,
	})
	if err != nil {
		return nil, err
	}
	report.GraphGaps = len(gaps)

	return report, nil
}

// GenerateDashboard produces the dashboard data structure.
func (a *Aggregator) GenerateDashboard(lookbackDays int) (*DashboardData, error) {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -lookbackDays)

	dash := &DashboardData{
		GeneratedAt:    end,
		SkipRateSeries: make(map[string][]TimePoint),
	}

	// Build daily time series.
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		dayStart := d
		dayEnd := d.AddDate(0, 0, 1)

		comparisons, err := a.store.Query(events.EventFilter{
			Types: []model.EventType{model.EventComparisonComplete},
			After: dayStart,
			Before: dayEnd,
		})
		if err != nil {
			continue
		}

		if len(comparisons) == 0 {
			continue
		}

		dash.TotalPipelines += len(comparisons)

		var dayCatchRate, daySpeedup float64
		for _, evt := range comparisons {
			var comp struct {
				CatchRate float64 `json:"catch_rate"`
				Speedup   float64 `json:"speedup"`
			}
			json.Unmarshal(evt.Payload, &comp)
			dayCatchRate += comp.CatchRate
			daySpeedup += comp.Speedup
		}

		dateStr := d.Format("2006-01-02")
		avgCatchRate := dayCatchRate / float64(len(comparisons))
		avgSpeedup := daySpeedup / float64(len(comparisons))

		dash.CatchRateSeries = append(dash.CatchRateSeries, TimePoint{Date: dateStr, Value: avgCatchRate})
		dash.SpeedupSeries = append(dash.SpeedupSeries, TimePoint{Date: dateStr, Value: avgSpeedup})

		dash.OverallCatchRate += avgCatchRate
		dash.OverallSpeedup += avgSpeedup
	}

	days := len(dash.CatchRateSeries)
	if days > 0 {
		dash.OverallCatchRate /= float64(days)
		dash.OverallSpeedup /= float64(days)
	}

	// Flake leaderboard.
	flakes, _ := a.store.Query(events.EventFilter{
		Types: []model.EventType{model.EventFlakeDetected},
		After: start,
		Before: end,
	})

	fpCount := make(map[string]*FlakeEntry)
	for _, evt := range flakes {
		var fp model.FlakePayload
		json.Unmarshal(evt.Payload, &fp)
		if entry, ok := fpCount[fp.Fingerprint]; ok {
			entry.Count++
			entry.LastSeen = evt.Timestamp
		} else {
			fpCount[fp.Fingerprint] = &FlakeEntry{
				Fingerprint: fp.Fingerprint,
				Count:       1,
				LastSeen:    evt.Timestamp,
				Test:        fp.Result.Test.Key(),
			}
		}
	}
	for _, entry := range fpCount {
		dash.FlakeLeaderboard = append(dash.FlakeLeaderboard, *entry)
	}
	dash.ActiveFlakes = len(fpCount)

	// False negative log.
	fnEvents, _ := a.store.Query(events.EventFilter{
		Types: []model.EventType{model.EventFalseNegative},
		After: start,
		Before: end,
	})
	for _, evt := range fnEvents {
		var detail model.FalseNegativeDetail
		json.Unmarshal(evt.Payload, &detail)
		dash.FalseNegativeLog = append(dash.FalseNegativeLog, detail)
	}

	return dash, nil
}

func formatISOWeek(year, week int) string {
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, (week-1)*7).
		Format("2006") + "-W" + padWeek(week)
}

func padWeek(w int) string {
	if w < 10 {
		return "0" + string(rune('0'+w))
	}
	return string(rune('0'+w/10)) + string(rune('0'+w%10))
}
