package engine

import (
	"log"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// AffectedResult holds the affected targets across all languages.
type AffectedResult struct {
	ByLanguage map[string]*LanguageResult
	ForceAll   bool
}

// LanguageResult holds affected targets for a single language.
type LanguageResult struct {
	Targets         []model.Target
	Configurations  []model.Configuration
	TotalTargets    int
	SelectedTargets int
	SkipRate        float64
	AlwaysRunCount  int
}

// AffectedComputer computes affected targets from changed files.
type AffectedComputer struct {
	registry *adapters.Registry
	scoping  model.ScopingConfig
	repoRoot string
}

// NewAffectedComputer creates a new AffectedComputer.
func NewAffectedComputer(registry *adapters.Registry, scoping model.ScopingConfig, repoRoot string) *AffectedComputer {
	return &AffectedComputer{
		registry: registry,
		scoping:  scoping,
		repoRoot: repoRoot,
	}
}

// Compute determines which targets are affected by the changed files.
func (ac *AffectedComputer) Compute(changedFiles []string, emitter *events.Emitter) (*AffectedResult, error) {
	result := &AffectedResult{
		ByLanguage: make(map[string]*LanguageResult),
	}

	// Check for force-all paths.
	forceAll := ac.checkForceAllPaths(changedFiles)
	result.ForceAll = forceAll

	for lang, adapter := range ac.registry.All() {
		// Check if language is activated.
		if ac.scoping.Activation.Languages != nil {
			if enabled, ok := ac.scoping.Activation.Languages[lang]; ok && !enabled {
				continue
			}
		}

		// Build or load cached graph.
		g, err := adapter.Graph.Build(ac.repoRoot)
		if err != nil {
			// Gracefully degrade: if the toolchain isn't available (e.g., cargo
			// not installed in the setup job), skip graph-based analysis for this
			// language. Categories will fall back to path-based matching.
			log.Printf("WARNING: skipping %s graph (toolchain unavailable): %v", lang, err)
			continue
		}

		// Compute affected targets.
		var affected []model.Target
		if forceAll {
			affected = g.AllTargets()
		} else {
			changed := adapter.Graph.ChangedTargets(changedFiles)
			if changed == nil {
				// nil means "special path triggered, run everything"
				affected = g.AllTargets()
			} else if len(changed) == 0 {
				result.ByLanguage[lang] = &LanguageResult{SkipRate: 1.0}
				if emitter != nil {
					emitter.Emit(model.EventTargetsComputed, model.TargetsComputedPayload{
						Language: lang,
						Selected: 0,
						Total:    len(g.AllTargets()),
						SkipRate: 1.0,
					})
				}
				continue
			} else {
				affected = adapter.Graph.ReverseDeps(g, changed)
			}
		}

		// Filter to test targets.
		testTargets := adapter.Graph.TestTargets(affected)

		// Merge always-run targets.
		alwaysRunIDs := ac.scoping.AlwaysRun[lang]
		alwaysRunTargets := resolveAlwaysRun(g, alwaysRunIDs, lang)
		testTargets = mergeTargets(testTargets, alwaysRunTargets)

		// Apply confidence scores — targets below threshold become always-run.
		testTargets = ac.applyConfidence(testTargets)

		// Compute configurations.
		configs := adapter.Graph.Configurations(testTargets)

		// Compute summary.
		allTargets := adapter.Graph.TestTargets(g.AllTargets())
		skipRate := 0.0
		if len(allTargets) > 0 {
			skipRate = 1.0 - float64(len(testTargets))/float64(len(allTargets))
		}

		lr := &LanguageResult{
			Targets:         testTargets,
			Configurations:  configs,
			TotalTargets:    len(allTargets),
			SelectedTargets: len(testTargets),
			SkipRate:        skipRate,
			AlwaysRunCount:  len(alwaysRunTargets),
		}
		result.ByLanguage[lang] = lr

		if emitter != nil {
			emitter.Emit(model.EventTargetsComputed, model.TargetsComputedPayload{
				Language:  lang,
				Selected:  lr.SelectedTargets,
				Total:     lr.TotalTargets,
				SkipRate:  lr.SkipRate,
				AlwaysRun: lr.AlwaysRunCount,
			})
		}
	}

	return result, nil
}

func (ac *AffectedComputer) checkForceAllPaths(changedFiles []string) bool {
	for _, f := range changedFiles {
		for _, fp := range ac.scoping.ForceAllPaths {
			if strings.HasPrefix(f, fp) {
				return true
			}
		}
	}
	return false
}

func (ac *AffectedComputer) applyConfidence(targets []model.Target) []model.Target {
	threshold := ac.scoping.ConfidenceThreshold
	if threshold <= 0 {
		return targets
	}
	for i := range targets {
		if targets[i].Confidence < threshold && targets[i].Scope != model.ScopeAlways {
			targets[i].Scope = model.ScopeAlways
		}
	}
	return targets
}

func resolveAlwaysRun(g interface{ AllTargets() []model.Target }, ids []string, lang string) []model.Target {
	if len(ids) == 0 {
		return nil
	}
	all := g.AllTargets()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	var result []model.Target
	for _, t := range all {
		for id := range idSet {
			if t.ID == id || strings.HasPrefix(t.ID, id) {
				t.Scope = model.ScopeAlways
				result = append(result, t)
				break
			}
		}
	}
	return result
}

func mergeTargets(a, b []model.Target) []model.Target {
	seen := make(map[string]bool, len(a))
	for _, t := range a {
		seen[t.ID] = true
	}
	merged := append([]model.Target{}, a...)
	for _, t := range b {
		if !seen[t.ID] {
			merged = append(merged, t)
		}
	}
	return merged
}
