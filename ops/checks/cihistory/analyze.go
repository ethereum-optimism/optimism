package cihistory

import (
	"sort"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// Analysis is the output of rolling events through Analyze — everything
// the writer needs to emit correlation edges and learned priors.
type Analysis struct {
	WindowStart time.Time
	WindowEnd   time.Time

	// Correlation entries, one per (file, check) pair that met the
	// minimum-observation threshold.
	Correlations []Correlation

	// Per-kind base failure rate over the window. nil for kinds with
	// fewer than MinObservationsForPrior samples (avoids noisy priors).
	PriorsByKind map[string]float64

	// Per-check base failure rate. Same threshold rule.
	PriorsByCheck map[string]float64
}

// Correlation is the historical precision of one check given a change
// to one file.
type Correlation struct {
	File         string  // repo-relative path
	CheckID      string  // catalog check ID
	Observations int     // how many events had File in diff and CheckID ran
	Failures     int     // how many of those had CheckID failed
	Precision    float64 // Failures / Observations
}

// Options controls Analyze.
type Options struct {
	// MinObservations filters correlation edges: fewer observations than
	// this → the signal is too noisy to trust, drop it. Default 3.
	MinObservations int

	// MinPrecision filters correlation edges by precision. Below this,
	// the file/check co-occurrence is noise. Default 0.1.
	MinPrecision float64

	// MinObservationsForPrior — kinds/checks with fewer total samples
	// than this don't get a learned prior. Default 20.
	MinObservationsForPrior int
}

func (o *Options) withDefaults() Options {
	out := Options{
		MinObservations:         o.MinObservations,
		MinPrecision:            o.MinPrecision,
		MinObservationsForPrior: o.MinObservationsForPrior,
	}
	if out.MinObservations <= 0 {
		out.MinObservations = 3
	}
	if out.MinPrecision <= 0 {
		out.MinPrecision = 0.1
	}
	if out.MinObservationsForPrior <= 0 {
		out.MinObservationsForPrior = 20
	}
	return out
}

// Analyze rolls a list of events into correlations and priors. Events
// that reference checks not in the catalog are ignored (can't attribute
// a kind for the prior).
func Analyze(events []Event, cat *catalog.Catalog, opts Options) *Analysis {
	o := opts.withDefaults()

	type fcKey struct {
		file, check string
	}
	ran := make(map[fcKey]int)
	failed := make(map[fcKey]int)

	checkKindByID := make(map[string]string, len(cat.CheckTypes))
	for _, ct := range cat.CheckTypes {
		checkKindByID[ct.ID] = ct.Kind
	}

	kindRan := make(map[string]int)
	kindFailed := make(map[string]int)
	checkRan := make(map[string]int)
	checkFailed := make(map[string]int)

	var earliest, latest time.Time
	for _, e := range events {
		if !e.MergedAt.IsZero() {
			if earliest.IsZero() || e.MergedAt.Before(earliest) {
				earliest = e.MergedAt
			}
			if e.MergedAt.After(latest) {
				latest = e.MergedAt
			}
		}

		for _, cr := range e.Checks {
			kind, known := checkKindByID[cr.ID]
			if !known {
				continue
			}
			kindRan[kind]++
			checkRan[cr.ID]++
			if cr.Failed {
				kindFailed[kind]++
				checkFailed[cr.ID]++
			}

			for _, f := range e.Files {
				k := fcKey{f, cr.ID}
				ran[k]++
				if cr.Failed {
					failed[k]++
				}
			}
		}
	}

	var corrs []Correlation
	for k, n := range ran {
		if n < o.MinObservations {
			continue
		}
		fails := failed[k]
		p := float64(fails) / float64(n)
		if p < o.MinPrecision {
			continue
		}
		corrs = append(corrs, Correlation{
			File:         k.file,
			CheckID:      k.check,
			Observations: n,
			Failures:     fails,
			Precision:    p,
		})
	}

	// Deterministic output order.
	sort.Slice(corrs, func(i, j int) bool {
		if corrs[i].File != corrs[j].File {
			return corrs[i].File < corrs[j].File
		}
		return corrs[i].CheckID < corrs[j].CheckID
	})

	priorsByKind := make(map[string]float64)
	for kind, n := range kindRan {
		if n < o.MinObservationsForPrior {
			continue
		}
		priorsByKind[kind] = float64(kindFailed[kind]) / float64(n)
	}

	priorsByCheck := make(map[string]float64)
	for id, n := range checkRan {
		if n < o.MinObservationsForPrior {
			continue
		}
		priorsByCheck[id] = float64(checkFailed[id]) / float64(n)
	}

	return &Analysis{
		WindowStart:   earliest,
		WindowEnd:     latest,
		Correlations:  corrs,
		PriorsByKind:  priorsByKind,
		PriorsByCheck: priorsByCheck,
	}
}
