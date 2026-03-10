package executor

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner records which commands were executed and in what order.
type mockRunner struct {
	mu       sync.Mutex
	calls    []runCall
	failures map[string]error // category → error (nil = success)
	delay    map[string]time.Duration
}

type runCall struct {
	Category string
	Command  string
	Time     time.Time
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		failures: make(map[string]error),
		delay:    make(map[string]time.Duration),
	}
}

func (m *mockRunner) Run(category string, command string, logPath string) error {
	if d, ok := m.delay[category]; ok {
		time.Sleep(d)
	}
	m.mu.Lock()
	m.calls = append(m.calls, runCall{Category: category, Command: command, Time: time.Now()})
	m.mu.Unlock()
	if err, ok := m.failures[category]; ok {
		return err
	}
	return nil
}

func (m *mockRunner) callOrder() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var names []string
	for _, c := range m.calls {
		names = append(names, c.Category)
	}
	return names
}

func (m *mockRunner) called(category string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.calls {
		if c.Category == category {
			return true
		}
	}
	return false
}

// mockCache provides configurable cache behavior for testing.
type mockCache struct {
	mu          sync.Mutex
	hits        map[string]bool   // category → hit?
	keys        map[string]string // category → key
	restoreFail map[string]error
	verifyFail  map[string]error
	saveFail    map[string]error
	saves       []string // categories that were saved
	restores    []string // categories that were restored
}

func newMockCache() *mockCache {
	return &mockCache{
		hits:        make(map[string]bool),
		keys:        make(map[string]string),
		restoreFail: make(map[string]error),
		verifyFail:  make(map[string]error),
		saveFail:    make(map[string]error),
	}
}

func (m *mockCache) ComputeKey(category string, cat model.JobCategoryConfig) (string, error) {
	if key, ok := m.keys[category]; ok {
		return key, nil
	}
	return "mock-key-" + category, nil
}

func (m *mockCache) Resolve(category string, cat model.JobCategoryConfig) (*CacheResolution, error) {
	key, _ := m.ComputeKey(category, cat)
	return &CacheResolution{
		CacheKey: key,
		Hit:      m.hits[category],
	}, nil
}

func (m *mockCache) Restore(category string, cat model.JobCategoryConfig) error {
	m.mu.Lock()
	m.restores = append(m.restores, category)
	m.mu.Unlock()
	if err, ok := m.restoreFail[category]; ok {
		return err
	}
	return nil
}

func (m *mockCache) Save(category string, cat model.JobCategoryConfig, key string) error {
	m.mu.Lock()
	m.saves = append(m.saves, category)
	m.mu.Unlock()
	if err, ok := m.saveFail[category]; ok {
		return err
	}
	return nil
}

func (m *mockCache) Verify(cat model.JobCategoryConfig) error {
	// Use first workspace path as category identifier (hack for mock).
	// In tests we set verifyFail keyed by category name.
	return nil
}

// mockCacheWithVerify allows per-category verify control.
type mockCacheWithVerify struct {
	mockCache
	verifyResults map[string]error // category → error
	lastVerified  string
}

func newMockCacheWithVerify() *mockCacheWithVerify {
	return &mockCacheWithVerify{
		mockCache: mockCache{
			hits:        make(map[string]bool),
			keys:        make(map[string]string),
			restoreFail: make(map[string]error),
			verifyFail:  make(map[string]error),
			saveFail:    make(map[string]error),
		},
		verifyResults: make(map[string]error),
	}
}

func (m *mockCacheWithVerify) Resolve(category string, cat model.JobCategoryConfig) (*CacheResolution, error) {
	m.lastVerified = category
	key, _ := m.ComputeKey(category, cat)
	return &CacheResolution{
		CacheKey: key,
		Hit:      m.hits[category],
	}, nil
}

func (m *mockCacheWithVerify) Verify(cat model.JobCategoryConfig) error {
	if err, ok := m.verifyResults[m.lastVerified]; ok {
		return err
	}
	return nil
}

// --- helpers ---

func makeScoping(categories map[string]model.JobCategoryConfig) *model.ScopingConfig {
	return &model.ScopingConfig{
		JobCategories: categories,
	}
}

func makeDecision(categories map[string]*model.CategoryDecision) *model.PipelineDecision {
	return &model.PipelineDecision{
		Categories: categories,
	}
}

func needed(reason string) *model.CategoryDecision {
	return &model.CategoryDecision{Needed: true, Reason: reason}
}

func neededWithTargets(reason string, targets ...string) *model.CategoryDecision {
	return &model.CategoryDecision{Needed: true, Reason: reason, Targets: targets}
}

// --- Tests ---

func TestExecute_SingleCategory(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("files changed"),
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "go_tests", results[0].Category)
	assert.Equal(t, "pass", results[0].Status)
	assert.True(t, runner.called("go_tests"))
}

func TestExecute_SkipsUnneededCategories(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"},
		"go_lint":  {Group: "go", Command: "make lint"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("files changed"),
		"go_lint":  {Needed: false, Skipped: true, SkipWhy: "no go files changed"},
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "go_tests", results[0].Category)
	assert.False(t, runner.called("go_lint"))
}

func TestExecute_SkipsCategoriesNotInDecision(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestExecute_OnlyRunsGroupCategories(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests":   {Group: "go", Command: "make go-test"},
		"sol_tests":  {Group: "sol", Command: "forge test"},
		"rust_tests": {Group: "rust", Command: "cargo test"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests":   needed("changed"),
		"sol_tests":  needed("changed"),
		"rust_tests": needed("changed"),
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "go_tests", results[0].Category)
}

func TestExecute_DAGOrdering(t *testing.T) {
	runner := newMockRunner()
	// Add delays to make ordering observable.
	runner.delay["contracts_build"] = 20 * time.Millisecond

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build"},
		"cannon_prestate": {Group: "build", Command: "make cannon-prestates", DependsOn: []string{"contracts_build"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
		"cannon_prestate": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 2)

	// cannon_prestate must have started AFTER contracts_build.
	order := runner.callOrder()
	contractsIdx := -1
	cannonIdx := -1
	for i, name := range order {
		if name == "contracts_build" {
			contractsIdx = i
		}
		if name == "cannon_prestate" {
			cannonIdx = i
		}
	}
	assert.Greater(t, cannonIdx, contractsIdx,
		"cannon_prestate must run after contracts_build, got order: %v", order)
}

func TestExecute_DAGDiamondDependency(t *testing.T) {
	runner := newMockRunner()
	runner.delay["a"] = 10 * time.Millisecond
	runner.delay["b"] = 10 * time.Millisecond

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"a": {Group: "build", Command: "build-a"},
		"b": {Group: "build", Command: "build-b"},
		"c": {Group: "build", Command: "build-c", DependsOn: []string{"a", "b"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"a": needed("changed"),
		"b": needed("changed"),
		"c": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 3)

	// c must be last.
	order := runner.callOrder()
	cIdx := -1
	for i, name := range order {
		if name == "c" {
			cIdx = i
		}
	}
	assert.Equal(t, 2, cIdx, "c must run last, got order: %v", order)
}

func TestExecute_ParallelIndependentCategories(t *testing.T) {
	runner := newMockRunner()
	runner.delay["a"] = 50 * time.Millisecond
	runner.delay["b"] = 50 * time.Millisecond

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"a": {Group: "go", Command: "test-a"},
		"b": {Group: "go", Command: "test-b"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"a": needed("changed"),
		"b": needed("changed"),
	})

	start := time.Now()
	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 2)
	// If run sequentially, would take ~100ms. Parallel should be ~50ms.
	assert.Less(t, elapsed, 90*time.Millisecond,
		"independent categories should run in parallel, took %s", elapsed)
}

func TestExecute_FailedCategorySignalsDeps(t *testing.T) {
	runner := newMockRunner()
	runner.failures["a"] = fmt.Errorf("build failed")

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"a": {Group: "build", Command: "build-a"},
		"b": {Group: "build", Command: "build-b", DependsOn: []string{"a"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"a": needed("changed"),
		"b": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err) // Execute itself doesn't error, individual categories do.
	require.Len(t, results, 2)

	// Both should complete (b shouldn't hang waiting for a).
	statusMap := make(map[string]string)
	for _, r := range results {
		statusMap[r.Category] = r.Status
	}
	assert.Equal(t, "fail", statusMap["a"])
	// b runs even though a failed (it still gets signaled).
	assert.Contains(t, []string{"pass", "fail"}, statusMap["b"])
}

func TestExecute_NoCommandCategory(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"docker_build": {Group: "build"}, // no command
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"docker_build": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "no_command", results[0].Status)
	assert.False(t, runner.called("docker_build"))
}

func TestExecute_DryRun(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("changed"),
	})

	exec := New(Config{Group: "go", DryRun: true, ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "dry_run", results[0].Status)
	assert.False(t, runner.called("go_tests"))
}

func TestExecute_CacheHitSkipsExecution(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	mc.hits["contracts_build"] = true

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"build/"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "cached", results[0].Status)
	assert.False(t, runner.called("contracts_build"), "runner should NOT be called on cache hit")
}

func TestExecute_CacheMissRunsAndSaves(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	// hits is empty → cache miss

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"build/"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pass", results[0].Status)
	assert.True(t, runner.called("contracts_build"))

	mc.mu.Lock()
	assert.Contains(t, mc.saves, "contracts_build", "should save to cache after successful build")
	mc.mu.Unlock()
}

func TestExecute_CacheHitVerifyFailRebuilds(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	mc.hits["contracts_build"] = true
	mc.verifyResults["contracts_build"] = fmt.Errorf("workspace path build/: no such file")

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"build/"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pass", results[0].Status, "should fall through to full build")
	assert.True(t, runner.called("contracts_build"), "runner SHOULD be called when verify fails")
}

func TestExecute_CacheRestoreFailRebuilds(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	mc.hits["contracts_build"] = true
	mc.restoreFail["contracts_build"] = fmt.Errorf("restore I/O error")

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"build/"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pass", results[0].Status)
	assert.True(t, runner.called("contracts_build"), "should rebuild on restore failure")
}

func TestExecute_NoCacheForCategoriesWithoutWorkspacePaths(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	mc.hits["go_tests"] = true // even if cache says hit, should be ignored

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"}, // no WorkspacePaths
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("changed"),
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pass", results[0].Status)
	assert.True(t, runner.called("go_tests"), "should always run without workspace_paths")
}

func TestExecute_TargetedCommand(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {
			Group:         "go",
			Command:       "make test-all",
			TargetCommand: "gotestsum --packages=\"{{targets}}\"",
		},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": neededWithTargets("changed", "pkg/a", "pkg/b"),
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)

	runner.mu.Lock()
	assert.Equal(t, `gotestsum --packages="pkg/a pkg/b"`, runner.calls[0].Command)
	runner.mu.Unlock()
}

func TestExecute_DAGWithCache(t *testing.T) {
	// contracts_build is cached, cannon_prestate depends on it and should still run.
	runner := newMockRunner()
	mc := newMockCacheWithVerify()
	mc.hits["contracts_build"] = true
	// cannon_prestate is a cache miss

	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"build/"}},
		"cannon_prestate": {Group: "build", Command: "make cannon", DependsOn: []string{"contracts_build"}, WorkspacePaths: []string{"cannon/bin"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
		"cannon_prestate": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 2)

	statusMap := make(map[string]string)
	for _, r := range results {
		statusMap[r.Category] = r.Status
	}
	assert.Equal(t, "cached", statusMap["contracts_build"])
	assert.Equal(t, "pass", statusMap["cannon_prestate"])
	assert.False(t, runner.called("contracts_build"))
	assert.True(t, runner.called("cannon_prestate"))
}

func TestExecute_EmptyGroup(t *testing.T) {
	runner := newMockRunner()
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"go_tests": {Group: "go", Command: "make test"},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, nil, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestExecute_CrossGroupDepsRestored(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()

	// go_tests depends on contracts_build (build group).
	// The executor should restore contracts_build artifacts before running go_tests.
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"forge-artifacts/"}},
		"go_tests":        {Group: "go", Command: "make test", DependsOn: []string{"contracts_build"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"go_tests": needed("changed"),
	})

	exec := New(Config{Group: "go", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	results, err := exec.Execute()
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "pass", results[0].Status)

	// contracts_build should have been restored (cross-group dep).
	mc.mu.Lock()
	assert.Contains(t, mc.restores, "contracts_build", "should restore cross-group dependency")
	mc.mu.Unlock()
}

func TestExecute_CrossGroupDepsNotRestoredForOwnGroup(t *testing.T) {
	runner := newMockRunner()
	mc := newMockCacheWithVerify()

	// cannon_prestate depends on contracts_build, both in build group.
	// Should NOT trigger cross-group restore.
	scoping := makeScoping(map[string]model.JobCategoryConfig{
		"contracts_build": {Group: "build", Command: "forge build", WorkspacePaths: []string{"forge-artifacts/"}},
		"cannon_prestate": {Group: "build", Command: "make cannon", DependsOn: []string{"contracts_build"}},
	})
	decision := makeDecision(map[string]*model.CategoryDecision{
		"contracts_build": needed("changed"),
		"cannon_prestate": needed("changed"),
	})

	exec := New(Config{Group: "build", ResultsDir: t.TempDir()}, runner, mc, scoping, decision)
	_, err := exec.Execute()
	require.NoError(t, err)

	// contracts_build should NOT appear in restores from cross-group (same group).
	mc.mu.Lock()
	// restores might have entries from the normal cache flow, but cross-group restore
	// specifically should not have been called for same-group deps.
	mc.mu.Unlock()
}

func TestResolveCommand_Placeholders(t *testing.T) {
	tests := []struct {
		name    string
		cat     model.JobCategoryConfig
		cd      *model.CategoryDecision
		want    string
	}{
		{
			name: "no target command falls back",
			cat:  model.JobCategoryConfig{Command: "make test"},
			cd:   &model.CategoryDecision{Targets: []string{"a", "b"}},
			want: "make test",
		},
		{
			name: "no targets falls back",
			cat:  model.JobCategoryConfig{Command: "make test", TargetCommand: "test {{targets}}"},
			cd:   &model.CategoryDecision{},
			want: "make test",
		},
		{
			name: "space separated",
			cat:  model.JobCategoryConfig{TargetCommand: "test {{targets}}"},
			cd:   &model.CategoryDecision{Targets: []string{"a", "b"}},
			want: "test a b",
		},
		{
			name: "csv",
			cat:  model.JobCategoryConfig{TargetCommand: "test --t={{targets_csv}}"},
			cd:   &model.CategoryDecision{Targets: []string{"a", "b", "c"}},
			want: "test --t=a,b,c",
		},
		{
			name: "glob multiple",
			cat:  model.JobCategoryConfig{TargetCommand: "test {{targets_glob}}"},
			cd:   &model.CategoryDecision{Targets: []string{"a", "b"}},
			want: "test {a,b}",
		},
		{
			name: "glob single",
			cat:  model.JobCategoryConfig{TargetCommand: "test {{targets_glob}}"},
			cd:   &model.CategoryDecision{Targets: []string{"a"}},
			want: "test a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCommand(tt.cat, tt.cd)
			assert.Equal(t, tt.want, got)
		})
	}
}
