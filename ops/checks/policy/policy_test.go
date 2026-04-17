package policy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// TestLoad_Baseline — baseline embeds and loads cleanly with no overrides.
func TestLoad_Baseline(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(p.Stages) == 0 {
		t.Error("baseline missing stages")
	}
	if len(p.Tiers) == 0 {
		t.Error("baseline missing tiers")
	}
	if p.Coverage.SignalFloor != 0.5 {
		t.Errorf("coverage signal_floor = %f, want 0.5", p.Coverage.SignalFloor)
	}
	if p.HighSignal.Threshold != 0.6 {
		t.Errorf("high_signal threshold = %f, want 0.6", p.HighSignal.Threshold)
	}
}

// TestStage_Lookup — StageConfig resolves by name, errors on unknown.
func TestStage_Lookup(t *testing.T) {
	p, _ := Load()

	s, err := p.Stage("pr")
	if err != nil {
		t.Fatalf("Stage(pr): %v", err)
	}
	if s.MissCost != 7200 {
		t.Errorf("pr miss_cost = %f, want 7200", s.MissCost)
	}

	if _, err := p.Stage("nonexistent"); err == nil {
		t.Error("expected error for unknown stage")
	}
}

// TestTierFor — highest-matching tier is returned; below lowest returns nil.
func TestTierFor(t *testing.T) {
	p, _ := Load()

	cases := map[float64]string{
		0.95: "very_high",
		0.7:  "high",
		0.5:  "med",
		0.2:  "low",
	}
	for sig, want := range cases {
		tier := p.TierFor(sig)
		if tier == nil {
			t.Errorf("signal %f: got nil tier, want %q", sig, want)
			continue
		}
		if tier.Label != want {
			t.Errorf("signal %f: tier=%q, want %q", sig, tier.Label, want)
		}
	}

	if tier := p.TierFor(0.05); tier != nil {
		t.Errorf("signal 0.05: got tier %q, want nil", tier.Label)
	}
}

// TestKnobValue_MatrixLookup — forge-test fuzz_runs at pr/high → 128.
func TestKnobValue_MatrixLookup(t *testing.T) {
	p, _ := Load()

	v, ok := p.KnobValue("forge-test", "fuzz_runs", "pr", "high")
	if !ok {
		t.Fatal("expected fuzz_runs value")
	}
	switch n := v.(type) {
	case int:
		if n != 128 {
			t.Errorf("fuzz_runs = %d, want 128", n)
		}
	default:
		t.Errorf("unexpected type %T for fuzz_runs", v)
	}
}

// TestKnobValue_FallsThroughToDefault — low tier at a stage with no
// "low" cell gets `default` from the matrix.
func TestKnobValue_FallsThroughToDefault(t *testing.T) {
	p, _ := Load()

	// go-test.short has default=true, but no by_stage["save"]["med"]
	// cell. Lookup should return the default (true).
	v, ok := p.KnobValue("go-test", "short", "save", "med")
	if !ok {
		t.Fatal("expected short value")
	}
	if v != true {
		t.Errorf("short=%v, want default true", v)
	}
}

// TestPrior_FallsThroughToDefault — unknown kind uses default.
func TestPrior_FallsThroughToDefault(t *testing.T) {
	p, _ := Load()

	if got := p.Prior("test"); got != 0.3 {
		t.Errorf("prior(test) = %f, want 0.3", got)
	}
	if got := p.Prior("unknown-kind"); got != 0.3 {
		t.Errorf("prior(unknown) = %f, want default 0.3", got)
	}
}

// TestKnobConfig_CatalogFallback — knobs without policy entries fall
// back to the catalog-declared Default.
func TestKnobConfig_CatalogFallback(t *testing.T) {
	p, _ := Load()

	ct := &catalog.CheckType{
		ID: "forge-test",
		Knobs: []catalog.Knob{
			{Name: "fuzz_runs", Type: "int", Default: 8},
			{Name: "nonexistent_knob", Type: "string", Default: "hello"},
		},
	}

	cfg := p.KnobConfig(ct, "pr", "high")
	if cfg["fuzz_runs"] == nil {
		t.Error("expected fuzz_runs from policy")
	}
	if cfg["nonexistent_knob"] != "hello" {
		t.Errorf("nonexistent_knob = %v, want catalog default 'hello'", cfg["nonexistent_knob"])
	}
}

// TestBlastRadiusMatch — prefix and exact match both trigger.
func TestBlastRadiusMatch(t *testing.T) {
	p, _ := Load()

	hit, files := p.BlastRadiusMatch([]string{"foundry.toml"})
	if !hit || len(files) == 0 {
		t.Error("expected foundry.toml to trigger blast radius")
	}

	hit, _ = p.BlastRadiusMatch([]string{".circleci/config.yml"})
	if !hit {
		t.Error("expected .circleci/ prefix to trigger blast radius")
	}

	hit, _ = p.BlastRadiusMatch([]string{"src/L1/OptimismPortal.sol"})
	if hit {
		t.Error("source file should not trigger blast radius")
	}
}

// TestLoad_LayeredOverride — override file takes precedence on declared
// scalars, leaves untouched fields intact.
func TestLoad_LayeredOverride(t *testing.T) {
	tmp := t.TempDir()
	overridePath := filepath.Join(tmp, "override.yaml")
	override := []byte(`
priors_by_kind:
  test: 0.45
high_signal:
  threshold: 0.75
knob_policies:
  forge-test:
    fuzz_runs:
      by_stage:
        pr:
          high: 256
`)
	if err := os.WriteFile(overridePath, override, 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	p, err := Load(overridePath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if p.Prior("test") != 0.45 {
		t.Errorf("overridden test prior = %f, want 0.45", p.Prior("test"))
	}
	// Non-overridden prior survives.
	if p.Prior("lint") != 0.8 {
		t.Errorf("baseline lint prior = %f, want 0.8", p.Prior("lint"))
	}
	// Overridden threshold takes effect.
	if p.HighSignal.Threshold != 0.75 {
		t.Errorf("high_signal threshold = %f, want 0.75", p.HighSignal.Threshold)
	}
	// prior_override not in override file → baseline survives.
	if p.HighSignal.PriorOverride != 0.7 {
		t.Errorf("high_signal prior_override = %f, want baseline 0.7", p.HighSignal.PriorOverride)
	}
	// Overridden knob cell replaces that cell only.
	v, _ := p.KnobValue("forge-test", "fuzz_runs", "pr", "high")
	switch n := v.(type) {
	case int:
		if n != 256 {
			t.Errorf("pr.high fuzz_runs = %d, want 256", n)
		}
	}
	// Untouched knob cell survives.
	v, _ = p.KnobValue("forge-test", "fuzz_runs", "pr", "med")
	if v == nil {
		t.Error("pr.med fuzz_runs should survive override")
	}
}

// TestLoad_MissingOverrideIsSilent — nonexistent override paths are ignored.
func TestLoad_MissingOverrideIsSilent(t *testing.T) {
	p, err := Load("/nonexistent/path/that/does/not/exist.yaml")
	if err != nil {
		t.Errorf("expected silent pass for missing override, got: %v", err)
	}
	if p == nil {
		t.Fatal("expected policy")
	}
}
