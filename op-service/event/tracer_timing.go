package event

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/maps"
)

// TimingTracer generates an HTML output with an SVG that shows,
// per deriver, per event-type, bands for event-execution scaled by the execution time.
// This trace gives an idea of patterns between events and where execution-time is spent.
type TimingTracer struct {
	StructTracer
}

var _ Tracer = (*TimingTracer)(nil)

func NewTimingTracer() *TimingTracer {
	return &TimingTracer{}
}

func (st *TimingTracer) Output() string {
	st.l.Lock()
	defer st.l.Unlock()
	out := new(strings.Builder)
	out.WriteString(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Timing trace</title>
</head>
<body>
`)

	var minTime, maxTime time.Time
	denyList := make(map[uint64]struct{})
	for _, e := range st.Entries {
		if e.Kind == TraceDeriveEnd && !e.DeriveEnd.Effect {
			denyList[e.DerivContext] = struct{}{}
		}
		if e.EventTime != (time.Time{}) && (minTime == (time.Time{}) || minTime.After(e.EventTime)) {
			minTime = e.EventTime
		}
		if e.EventTime != (time.Time{}) && (maxTime == (time.Time{}) || e.EventTime.After(maxTime)) {
			maxTime = e.EventTime
		}
	}

	// Time spent on wallclock
	realTime := maxTime.Sub(minTime)

	// Accumulate entries grouped by actor, and then by event-name.
	byActor := make(map[string]map[string][]TraceEntry)
	rows := 0
	for _, e := range st.Entries {
		if e.Kind != TraceDeriveEnd && e.Kind != TraceEmit {
			continue
		}
		// Omit entries which just passed through but did not have any effective processing
		if e.DerivContext != 0 {
			if _, ok := denyList[e.DerivContext]; ok {
				continue
			}
		}
		m, ok := byActor[e.Name]
		if !ok {
			m = make(map[string][]TraceEntry)
			byActor[e.Name] = m
		}
		if len(m[e.EventName]) == 0 {
			rows += 1
		}
		m[e.EventName] = append(m[e.EventName], e)
	}
	// for tick marks
	rows += 2

	// warning: viewbox resolution bounds: 24-bit max resolution, and 8-bit sub-pixel resolution
	leftOffset := float64(300)
	width := float64(2000)
	incrementY := float64(10)

	height := float64(rows) * incrementY

	// min-x, min-y, width, and height
	_, _ = fmt.Fprintf(out, `<svg viewBox="%f %f %f %f" preserveAspectRatio="none" width="2300" height="%d" style="border: 1px solid black">
`,
		-leftOffset, 0.0, leftOffset+width, height, rows*10)

	drawText := func(x, y float64, txt string) {
		_, _ = fmt.Fprintf(out, `<text dy="1em" class="messageText"
	style="font-size: 8px; font-weight: 300;"
	x="%.3f" y="%.3f">%s</text>
`,
			x, y, txt)
	}
	drawBox := func(x, y float64, w, h float64, strokeColor string, color string) {
		strokeTxt := ""
		if strokeColor != "" {
			strokeTxt = `stroke="` + strokeColor + `" stroke-width="0.5px"`
		}
		_, _ = fmt.Fprintf(out, `<rect fill="%s" %s
			x="%.3f" y="%.3f" width="%.3f" height="%.3f"></rect>
`, color, strokeTxt, x, y, w, h)
	}
	drawCircle := func(x, y float64, r float64, strokeColor string, color string) {
		strokeTxt := ""
		if strokeColor != "" {
			strokeTxt = `stroke="` + strokeColor + `" stroke-width="0.5px"`
		}
		_, _ = fmt.Fprintf(out, `<circle fill="%s" %s
			cx="%.3f" cy="%.3f" r="%.3f"></circle>
`, color, strokeTxt, x, y, r)
	}
	drawLine := func(x1, y1, x2, y2 float64, strokeWidth float64) {
		_, _ = fmt.Fprintf(out, `<line stroke="#999" stroke-width="%.2fpx"
			x1="%.3f" y1="%.3f" x2="%.3f" y2="%.3f"></line>
`, strokeWidth, x1, y1, x2, y2)
	}

	timeToX := func(v time.Time) float64 {
		return width * float64(v.Sub(minTime)) / float64(realTime)
	}

	durationToX := func(v time.Duration) float64 {
		return width * float64(v) / float64(realTime)
	}

	// sort the keys, to get deterministic diagram order
	actors := maps.Keys(byActor)
	sort.Strings(actors)

	offsetY := float64(0)
	textX := -leftOffset
	derivCoords := make(map[uint64]struct{ x, y float64 })
	emitCoords := make(map[uint64]struct{ x, y float64 })
	row := 0
	for _, actorName := range actors {

		m := byActor[actorName]
		derived := maps.Keys(m)
		sort.Strings(derived)

		for _, d := range derived {
			if row%2 == 0 {
				drawBox(-leftOffset/2, offsetY, width+leftOffset/2, incrementY, "", "#f4f4f4")
			}
			row += 1

			drawLine(textX+leftOffset/2, offsetY, width, offsetY, 0.5)
			drawText(textX+leftOffset/2, offsetY, d)

			entries := m[d]

			for _, e := range entries {
				if e.Kind != TraceDeriveEnd && e.Kind != TraceEmit {
					continue
				}
				x := timeToX(e.EventTime)
				y := offsetY
				if e.Kind == TraceDeriveEnd {
					derivCoords[e.DerivContext] = struct{ x, y float64 }{x: x, y: y}
					drawBox(x, y, durationToX(e.DeriveEnd.Duration), incrementY, "#aad", "#aad")
				}
				if e.Kind == TraceEmit {
					emitCoords[e.EmitContext] = struct{ x, y float64 }{x: x, y: y}
					// draw tiny point-centered circle to indicate event emission
					r := incrementY / 4
					drawCircle(x, y+(incrementY/2), r, "#daa", "#daa")
				}
			}
			offsetY += incrementY
		}
	}

	offsetY = float64(0)
	for _, actorName := range actors {
		subSectionH := incrementY * float64(len(byActor[actorName]))
		drawText(textX+8.0, offsetY+subSectionH/2-incrementY/2, strings.ToUpper(actorName))
		drawLine(textX, offsetY, width, offsetY, 2) // horizontal separator line to group actors
		offsetY += subSectionH
	}
	drawLine(textX, offsetY, width, offsetY, 2) // horizontal separator line to group actors

	// draw lines between event-emissions and event-execution
	for _, actorName := range actors {
		m := byActor[actorName]
		derived := maps.Keys(m)
		sort.Strings(derived)
		for _, d := range derived {
			entries := m[d]
			for _, e := range entries {
				if e.Kind == TraceDeriveEnd {
					emitFrom, ok := emitCoords[e.EmitContext]
					if !ok {
						continue
					}
					derivTo := derivCoords[e.DerivContext]
					drawLine(emitFrom.x, emitFrom.y+(incrementY/2), derivTo.x, derivTo.y+(incrementY/2), 0.5)
				}
			}
		}
	}
	// draw tick marks
	delta := realTime / 20
	minDelta := time.Millisecond * 10
	for {
		if delta.Truncate(minDelta) == (delta + delta/3).Truncate(minDelta) {
			delta = minDelta
			break
		} else {
			minDelta *= 2
		}
	}
	minTime = minTime.UTC()
	// Round  up to nearest multiple of delta (assuming delta < 1s)
	start := delta - (time.Duration(minTime.Nanosecond()) % delta)
	for x := start; x < realTime; {
		posX := durationToX(x)
		drawLine(posX, offsetY, posX, offsetY+incrementY/4, 2)
		drawText(posX-incrementY, offsetY+incrementY/4, minTime.Add(x).Format("15:04:05.000"))
		x += delta
	}
	// main label <> content separator line
	drawLine(0, 0, 0, height, 1)

	out.WriteString(`
	</svg>
  </body>
</html>
`)
	return out.String()
}
