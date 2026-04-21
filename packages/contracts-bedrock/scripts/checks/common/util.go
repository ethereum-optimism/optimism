package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"golang.org/x/sync/errgroup"
)

// ArtifactsDir returns the forge artifacts directory to read from.
// Defaults to "forge-artifacts" (the foundry.toml [profile.default]
// out-dir) but is overridable via FOUNDRY_ARTIFACTS for callers that
// want to scan a different build (e.g. a lite-profile build at
// forge-artifacts-lite/, or a src-only build at
// forge-artifacts-src-prod/). The ProcessFilesGlob helper rewrites
// any glob pattern whose first segment is "forge-artifacts" to use
// this directory instead, so existing callers don't need to change.
func ArtifactsDir() string {
	if dir := os.Getenv("FOUNDRY_ARTIFACTS"); dir != "" {
		return dir
	}
	return "forge-artifacts"
}

func rewriteArtifactsPatterns(patterns []string) []string {
	dir := ArtifactsDir()
	if dir == "forge-artifacts" {
		return patterns
	}
	out := make([]string, len(patterns))
	for i, p := range patterns {
		if strings.HasPrefix(p, "forge-artifacts/") {
			out[i] = dir + strings.TrimPrefix(p, "forge-artifacts")
		} else {
			out[i] = p
		}
	}
	return out
}

type ErrorReporter struct {
	hasErr atomic.Bool
	outMtx sync.Mutex
}

func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{}
}

func (e *ErrorReporter) Fail(msg string, args ...any) {
	e.outMtx.Lock()
	// Useful for suppressing error reporting in tests
	if os.Getenv("SUPPRESS_ERROR_REPORTER") == "" {
		_, _ = fmt.Fprintf(os.Stderr, "❌  "+msg+"\n", args...)
	}
	e.outMtx.Unlock()
	e.hasErr.Store(true)
}

func (e *ErrorReporter) HasError() bool {
	return e.hasErr.Load()
}

type Void struct{}

type FileProcessor[T any] func(path string) (T, []error)

func ProcessFiles[T any](files map[string]string, processor FileProcessor[T]) (map[string]T, error) {
	g := errgroup.Group{}
	g.SetLimit(runtime.NumCPU())

	reporter := NewErrorReporter()
	results := sync.Map{}

	for _, path := range files {
		path := path // Capture loop variables
		g.Go(func() error {
			result, errs := processor(path)
			if len(errs) > 0 {
				for _, err := range errs {
					reporter.Fail("%s: %v", path, err)
				}
			} else {
				results.Store(path, result)
			}
			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	if reporter.HasError() {
		return nil, fmt.Errorf("processing failed")
	}

	// Convert sync.Map to regular map
	finalResults := make(map[string]T)
	results.Range(func(key, value interface{}) bool {
		finalResults[key.(string)] = value.(T)
		return true
	})

	return finalResults, nil
}

func ProcessFilesGlob[T any](includes, excludes []string, processor FileProcessor[T]) (map[string]T, error) {
	// Transparently rewrite forge-artifacts/ patterns to honor
	// FOUNDRY_ARTIFACTS. Callers keep writing patterns against the
	// canonical "forge-artifacts/" prefix; at runtime those become
	// whatever directory the caller is inspecting.
	includes = rewriteArtifactsPatterns(includes)
	excludes = rewriteArtifactsPatterns(excludes)
	files, err := FindFiles(includes, excludes)
	if err != nil {
		return nil, err
	}
	return ProcessFiles(files, processor)
}

func FindFiles(includes, excludes []string) (map[string]string, error) {
	included := make(map[string]string)
	excluded := make(map[string]struct{})

	// Get all included files
	for _, pattern := range includes {
		matches, err := doublestar.Glob(os.DirFS("."), pattern)
		if err != nil {
			return nil, fmt.Errorf("glob pattern error: %w", err)
		}
		for _, match := range matches {
			included[match] = match
		}
	}

	// Get all excluded files
	for _, pattern := range excludes {
		matches, err := doublestar.Glob(os.DirFS("."), pattern)
		if err != nil {
			return nil, fmt.Errorf("glob pattern error: %w", err)
		}
		for _, match := range matches {
			excluded[match] = struct{}{}
		}
	}

	// Remove excluded files from result
	for name := range excluded {
		delete(included, name)
	}

	return included, nil
}

func ReadForgeArtifact(path string) (*solc.ForgeArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	var artifact solc.ForgeArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse artifact: %w", err)
	}

	return &artifact, nil
}

func WriteJSON(data interface{}, path string) error {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(data)
	if err != nil {
		return fmt.Errorf("failed to encode data: %w", err)
	}
	jsonData := out.Bytes()
	if len(jsonData) > 0 && jsonData[len(jsonData)-1] == '\n' { // strip newline
		jsonData = jsonData[:len(jsonData)-1]
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
