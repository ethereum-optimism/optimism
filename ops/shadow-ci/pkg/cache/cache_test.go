package cache

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// initGitRepo creates a temporary git repo with some tracked files.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %v\n%s", args, err, out)
		}
	}

	run("git", "init")
	run("git", "checkout", "-b", "main")

	// Create some source files.
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "lib.go"), []byte("package main\nfunc helper() {}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("[tools]\ngo = \"1.22\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "go.sum"), []byte("example.com v1.0.0 h1:abc\n"), 0o644)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	return dir
}

func TestComputeKey_Deterministic(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs: []string{"src/"},
	}

	key1, err := r.ComputeKey("test", cat)
	if err != nil {
		t.Fatalf("ComputeKey: %v", err)
	}
	key2, err := r.ComputeKey("test", cat)
	if err != nil {
		t.Fatalf("ComputeKey: %v", err)
	}

	if key1 != key2 {
		t.Errorf("keys not deterministic: %s != %s", key1, key2)
	}
	if len(key1) != 16 {
		t.Errorf("expected 16-char key, got %d: %s", len(key1), key1)
	}
}

func TestComputeKey_DefaultsToTriggerPaths(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	// With explicit cache_inputs.
	catExplicit := model.JobCategoryConfig{
		CacheInputs: []string{"src/"},
	}
	// With trigger_paths fallback.
	catFallback := model.JobCategoryConfig{
		TriggerPaths: []string{"src/"},
	}

	key1, _ := r.ComputeKey("test", catExplicit)
	key2, _ := r.ComputeKey("test", catFallback)

	if key1 != key2 {
		t.Errorf("cache_inputs and trigger_paths should produce same key: %s != %s", key1, key2)
	}
}

func TestComputeKey_SortedInputs(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat1 := model.JobCategoryConfig{
		CacheInputs: []string{"src/", "go.sum"},
	}
	cat2 := model.JobCategoryConfig{
		CacheInputs: []string{"go.sum", "src/"},
	}

	key1, _ := r.ComputeKey("test", cat1)
	key2, _ := r.ComputeKey("test", cat2)

	if key1 != key2 {
		t.Errorf("input order should not affect key: %s != %s", key1, key2)
	}
}

func TestComputeKey_ChangesOnFileModification(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs: []string{"src/"},
	}

	key1, _ := r.ComputeKey("test", cat)

	// Modify a file and commit.
	os.WriteFile(filepath.Join(repo, "src", "main.go"), []byte("package main\nfunc main() { println(\"changed\") }\n"), 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "modify")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	cmd.Run()

	key2, _ := r.ComputeKey("test", cat)

	if key1 == key2 {
		t.Errorf("key should change after file modification: both are %s", key1)
	}
}

func TestResolve_CacheMiss(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs:    []string{"src/"},
		WorkspacePaths: []string{"build/"},
	}

	res, err := r.Resolve("test", cat)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if res.Hit {
		t.Error("expected cache miss, got hit")
	}
	if res.CacheKey == "" {
		t.Error("expected cache key even on miss")
	}
}

func TestResolve_CacheHit(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs:    []string{"src/"},
		WorkspacePaths: []string{"build/"},
	}

	// Compute key and manually create cache entry.
	key, _ := r.ComputeKey("test", cat)
	catDir := filepath.Join(cacheDir, "test")
	os.MkdirAll(catDir, 0o755)
	os.WriteFile(filepath.Join(catDir, "cache.key"), []byte(key), 0o644)

	res, err := r.Resolve("test", cat)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if !res.Hit {
		t.Error("expected cache hit")
	}
}

func TestResolve_CacheHit_StaleKey(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs:    []string{"src/"},
		WorkspacePaths: []string{"build/"},
	}

	// Write a stale key.
	catDir := filepath.Join(cacheDir, "test")
	os.MkdirAll(catDir, 0o755)
	os.WriteFile(filepath.Join(catDir, "cache.key"), []byte("stale_key_value"), 0o644)

	res, err := r.Resolve("test", cat)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if res.Hit {
		t.Error("expected cache miss for stale key")
	}
}

func TestSaveAndRestore(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	// Create build artifacts in the repo.
	buildDir := filepath.Join(repo, "build")
	os.MkdirAll(buildDir, 0o755)
	os.WriteFile(filepath.Join(buildDir, "output.bin"), []byte("binary content"), 0o644)

	cat := model.JobCategoryConfig{
		CacheInputs:    []string{"src/"},
		WorkspacePaths: []string{"build"},
	}

	key, _ := r.ComputeKey("test", cat)

	// Save.
	if err := r.Save("test", cat, key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Remove build artifacts from repo.
	os.RemoveAll(buildDir)

	// Restore.
	if err := r.Restore("test", cat); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	// Verify files exist.
	data, err := os.ReadFile(filepath.Join(repo, "build", "output.bin"))
	if err != nil {
		t.Fatalf("restored file not found: %v", err)
	}
	if string(data) != "binary content" {
		t.Errorf("restored content mismatch: got %q", data)
	}
}

func TestComputeKey_IncludesMiseToml(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{
		CacheInputs: []string{"src/"},
	}

	key1, _ := r.ComputeKey("test", cat)

	// Modify mise.toml and commit.
	os.WriteFile(filepath.Join(repo, "mise.toml"), []byte("[tools]\ngo = \"1.23\"\n"), 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "bump go")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	cmd.Run()

	key2, _ := r.ComputeKey("test", cat)

	if key1 == key2 {
		t.Errorf("key should change when mise.toml changes: both are %s", key1)
	}
}

func TestComputeKey_NoInputs(t *testing.T) {
	repo := initGitRepo(t)
	cacheDir := t.TempDir()
	r := NewResolver(repo, cacheDir)

	cat := model.JobCategoryConfig{} // no inputs

	_, err := r.ComputeKey("test", cat)
	if err == nil {
		t.Error("expected error for category with no inputs")
	}
}

func TestVerify_AllPathsExist(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "build/output"), 0o755)
	os.MkdirAll(filepath.Join(dir, "dist"), 0o755)

	cat := model.JobCategoryConfig{
		WorkspacePaths: []string{"build/output", "dist"},
	}

	if err := Verify(dir, cat); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestVerify_MissingPath(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "build/output"), 0o755)
	// "dist" does not exist

	cat := model.JobCategoryConfig{
		WorkspacePaths: []string{"build/output", "dist"},
	}

	err := Verify(dir, cat)
	if err == nil {
		t.Fatal("expected error for missing workspace path")
	}
	if !contains(err.Error(), "dist") {
		t.Errorf("error should name the missing path, got: %v", err)
	}
}

func TestVerify_EmptyWorkspacePaths(t *testing.T) {
	dir := t.TempDir()
	cat := model.JobCategoryConfig{
		WorkspacePaths: nil,
	}

	if err := Verify(dir, cat); err != nil {
		t.Errorf("expected no error for empty workspace_paths, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
