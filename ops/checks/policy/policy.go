// Package policy owns every tunable the checks engine reasons about.
//
// Policy is the single source of truth for stage miss-costs, kind priors,
// tier boundaries, blast-radius patterns, coverage signal floors, and
// per-check knob matrices. A baseline.yaml ships embedded in the binary
// so the tool works with zero configuration; operators can layer
// overrides on top via files, and tweak (4) will auto-generate
// policy.learned.yaml from CI history as a second layer.
//
// Layering (first → last, later wins on scalar conflicts):
//
//	baseline.yaml (embedded)
//	policy.learned.yaml    (optional, auto-written)
//	--policy <path>        (optional, operator-provided)
package policy

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"gopkg.in/yaml.v3"
)

//go:embed baseline.yaml
var baselineYAML []byte

// Policy holds every selector-tunable value.
type Policy struct {
	Stages       []StageConfig                    `yaml:"stages"`
	Coverage     CoverageConfig                   `yaml:"coverage"`
	Freshness    FreshnessConfig                  `yaml:"freshness"`
	PriorsByKind map[string]float64               `yaml:"priors_by_kind"`
	HighSignal   HighSignalConfig                 `yaml:"high_signal"`
	Tiers        []Tier                           `yaml:"tiers"`
	Schedule     ScheduleConfig                   `yaml:"schedule"`
	KnobPolicies map[string]map[string]KnobMatrix `yaml:"knob_policies"` // checkID → knobName → matrix
}

// StageConfig is one development-lifecycle stage with its miss cost.
type StageConfig struct {
	Name     string  `yaml:"name"`
	MissCost float64 `yaml:"miss_cost"`
}

// CoverageConfig holds coverage-aggregation constants.
type CoverageConfig struct {
	SignalFloor float64 `yaml:"signal_floor"`
}

// FreshnessConfig controls how aggressively stale evidence is down-
// weighted. StaleMultiplier applies when a stamped SHA does not match
// the current file content. MaxAgeDays is the fallback time decay for
// edges without content stamps.
type FreshnessConfig struct {
	StaleMultiplier float64 `yaml:"stale_multiplier"`
	MaxAgeDays      int     `yaml:"max_age_days"`
}

// HighSignalConfig overrides the prior and forces run when aggregated
// signal exceeds Threshold.
type HighSignalConfig struct {
	Threshold     float64 `yaml:"threshold"`
	PriorOverride float64 `yaml:"prior_override"`
}

// Tier is one signal band.
type Tier struct {
	MinSignal float64 `yaml:"min_signal"`
	Label     string  `yaml:"label"`
}

// ScheduleConfig holds scheduler-level tunables.
type ScheduleConfig struct {
	MaxParallelism int `yaml:"max_parallelism"`
}

// KnobMatrix is the stage×tier lookup table for one knob on one check.
// Missing cells fall back to Default.
type KnobMatrix struct {
	Default interface{}                       `yaml:"default,omitempty"`
	ByStage map[string]map[string]interface{} `yaml:"by_stage,omitempty"`
}

// Load reads the embedded baseline, then deep-merges each overridePath
// in order (later files override earlier). Missing override files are
// silently ignored so callers can pass optional paths unconditionally.
func Load(overridePaths ...string) (*Policy, error) {
	p, err := parse(baselineYAML)
	if err != nil {
		return nil, fmt.Errorf("baseline policy: %w", err)
	}
	for _, path := range overridePaths {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading policy override %q: %w", path, err)
		}
		over, err := parse(data)
		if err != nil {
			return nil, fmt.Errorf("parsing policy override %q: %w", path, err)
		}
		p = merge(p, over)
	}
	if err := p.validate(); err != nil {
		return nil, fmt.Errorf("policy validation: %w", err)
	}
	return p, nil
}

func parse(data []byte) (*Policy, error) {
	var p Policy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Stage looks up a stage by name.
func (p *Policy) Stage(name string) (StageConfig, error) {
	for _, s := range p.Stages {
		if s.Name == name {
			return s, nil
		}
	}
	valid := make([]string, 0, len(p.Stages))
	for _, s := range p.Stages {
		valid = append(valid, s.Name)
	}
	return StageConfig{}, fmt.Errorf("unknown stage %q (valid: %s)", name, strings.Join(valid, ", "))
}

// AllStageNames returns stage names in declaration order.
func (p *Policy) AllStageNames() []string {
	names := make([]string, len(p.Stages))
	for i, s := range p.Stages {
		names[i] = s.Name
	}
	return names
}

// Prior returns the prior for a check kind, falling back to "default".
func (p *Policy) Prior(kind string) float64 {
	if v, ok := p.PriorsByKind[kind]; ok {
		return v
	}
	return p.PriorsByKind["default"]
}

// TierFor returns the highest-min_signal tier the signal satisfies, or
// nil if signal is below the lowest band.
func (p *Policy) TierFor(signal float64) *Tier {
	for i := range p.Tiers {
		if signal >= p.Tiers[i].MinSignal {
			return &p.Tiers[i]
		}
	}
	return nil
}

// KnobValue looks up (checkID, knobName, stage, tierLabel) → value.
// Returns (value, true) when an explicit cell is set, (default, true)
// when only Default is set, or (nil, false) when no policy is declared.
func (p *Policy) KnobValue(checkID, knobName, stageName, tierLabel string) (interface{}, bool) {
	check, ok := p.KnobPolicies[checkID]
	if !ok {
		return nil, false
	}
	matrix, ok := check[knobName]
	if !ok {
		return nil, false
	}
	if row, ok := matrix.ByStage[stageName]; ok {
		if v, ok := row[tierLabel]; ok {
			return v, true
		}
	}
	if matrix.Default != nil {
		return matrix.Default, true
	}
	return nil, false
}

// KnobConfig resolves every knob for a check at a given stage and
// tier. Knobs without a policy entry fall back to Knob.Default from
// the catalog.
func (p *Policy) KnobConfig(ct *catalog.CheckType, stageName, tierLabel string) map[string]any {
	out := make(map[string]any, len(ct.Knobs))
	for _, knob := range ct.Knobs {
		if v, ok := p.KnobValue(ct.ID, knob.Name, stageName, tierLabel); ok {
			out[knob.Name] = v
			continue
		}
		out[knob.Name] = knob.Default
	}
	return out
}

// MaxParallelism returns the scheduler concurrency cap, defaulting to 8.
func (p *Policy) MaxParallelism() int {
	if p.Schedule.MaxParallelism <= 0 {
		return 8
	}
	return p.Schedule.MaxParallelism
}

// DefaultOverridePaths returns the conventional layered-override paths
// relative to the repo root, in the order they should be loaded:
// learned first (generated), then operator-provided (committed). The
// embedded baseline is always the first layer.
func DefaultOverridePaths(root string) []string {
	return []string{
		filepath.Join(root, "ops", "checks", "policy", "learned.yaml"),
		filepath.Join(root, "ops", "checks", "policy", "policy.yaml"),
	}
}

func (p *Policy) validate() error {
	if len(p.Stages) == 0 {
		return fmt.Errorf("no stages declared")
	}
	if len(p.Tiers) == 0 {
		return fmt.Errorf("no tiers declared")
	}
	// Tiers must be sorted highest-first for TierFor.
	for i := 1; i < len(p.Tiers); i++ {
		if p.Tiers[i-1].MinSignal < p.Tiers[i].MinSignal {
			return fmt.Errorf("tiers must be declared highest min_signal first (got %q=%f before %q=%f)",
				p.Tiers[i-1].Label, p.Tiers[i-1].MinSignal, p.Tiers[i].Label, p.Tiers[i].MinSignal)
		}
	}
	if p.HighSignal.Threshold <= 0 {
		return fmt.Errorf("high_signal.threshold must be > 0")
	}
	if p.Coverage.SignalFloor < 0 || p.Coverage.SignalFloor > 1 {
		return fmt.Errorf("coverage.signal_floor must be in [0, 1], got %f", p.Coverage.SignalFloor)
	}
	return nil
}

// merge deep-merges over onto base. Scalar and map values: later wins.
// Slices: later replaces base entirely (predictable; operators who want
// to extend a list must redeclare it). KnobPolicies merges three levels
// deep (check → knob → stage cells).
func merge(base, over *Policy) *Policy {
	out := *base // shallow copy

	if len(over.Stages) > 0 {
		out.Stages = over.Stages
	}
	if over.Coverage.SignalFloor != 0 {
		out.Coverage.SignalFloor = over.Coverage.SignalFloor
	}
	if over.Freshness.StaleMultiplier != 0 {
		out.Freshness.StaleMultiplier = over.Freshness.StaleMultiplier
	}
	if over.Freshness.MaxAgeDays != 0 {
		out.Freshness.MaxAgeDays = over.Freshness.MaxAgeDays
	}
	if over.HighSignal.Threshold != 0 {
		out.HighSignal.Threshold = over.HighSignal.Threshold
	}
	if over.HighSignal.PriorOverride != 0 {
		out.HighSignal.PriorOverride = over.HighSignal.PriorOverride
	}
	if len(over.Tiers) > 0 {
		out.Tiers = over.Tiers
	}
	if over.Schedule.MaxParallelism != 0 {
		out.Schedule.MaxParallelism = over.Schedule.MaxParallelism
	}

	// priors_by_kind: map merge (per-key override).
	if len(over.PriorsByKind) > 0 {
		merged := make(map[string]float64, len(out.PriorsByKind)+len(over.PriorsByKind))
		for k, v := range out.PriorsByKind {
			merged[k] = v
		}
		for k, v := range over.PriorsByKind {
			merged[k] = v
		}
		out.PriorsByKind = merged
	}

	// knob_policies: merge by checkID → knobName → (default scalar, by_stage map).
	if len(over.KnobPolicies) > 0 {
		merged := make(map[string]map[string]KnobMatrix)
		for checkID, knobs := range out.KnobPolicies {
			dup := make(map[string]KnobMatrix, len(knobs))
			for k, v := range knobs {
				dup[k] = v
			}
			merged[checkID] = dup
		}
		for checkID, knobs := range over.KnobPolicies {
			dst, ok := merged[checkID]
			if !ok {
				dst = make(map[string]KnobMatrix)
			}
			for knobName, overMatrix := range knobs {
				baseMatrix := dst[knobName]
				dst[knobName] = mergeKnobMatrix(baseMatrix, overMatrix)
			}
			merged[checkID] = dst
		}
		out.KnobPolicies = merged
	}

	return &out
}

func mergeKnobMatrix(base, over KnobMatrix) KnobMatrix {
	out := base
	if over.Default != nil {
		out.Default = over.Default
	}
	if len(over.ByStage) > 0 {
		merged := make(map[string]map[string]interface{})
		for stage, cells := range base.ByStage {
			row := make(map[string]interface{}, len(cells))
			for k, v := range cells {
				row[k] = v
			}
			merged[stage] = row
		}
		for stage, cells := range over.ByStage {
			row, ok := merged[stage]
			if !ok {
				row = make(map[string]interface{})
			}
			for k, v := range cells {
				row[k] = v
			}
			merged[stage] = row
		}
		out.ByStage = merged
	}
	return out
}
