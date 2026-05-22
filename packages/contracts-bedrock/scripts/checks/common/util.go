package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-chain-ops/solc"
	"golang.org/x/sync/errgroup"
)

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
		matches, err := glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("glob pattern error: %w", err)
		}
		for _, match := range matches {
			included[match] = match
		}
	}

	// Get all excluded files
	for _, pattern := range excludes {
		matches, err := glob(pattern)
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

func glob(pattern string) ([]string, error) {
	patterns, err := expandBraces(filepath.ToSlash(pattern))
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		matches, err := globOne(pattern)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			seen[match] = struct{}{}
		}
	}

	matches := make([]string, 0, len(seen))
	for match := range seen {
		matches = append(matches, match)
	}
	sort.Strings(matches)
	return matches, nil
}

func expandBraces(pattern string) ([]string, error) {
	start := strings.IndexByte(pattern, '{')
	if start == -1 {
		if strings.Contains(pattern, "}") {
			return nil, fmt.Errorf("unmatched closing brace in %q", pattern)
		}
		return []string{pattern}, nil
	}
	end := strings.IndexByte(pattern[start+1:], '}')
	if end == -1 {
		return nil, fmt.Errorf("unmatched opening brace in %q", pattern)
	}
	end += start + 1

	options := strings.Split(pattern[start+1:end], ",")
	if len(options) == 0 {
		return nil, fmt.Errorf("empty brace expression in %q", pattern)
	}

	var expanded []string
	for _, option := range options {
		nested, err := expandBraces(pattern[:start] + option + pattern[end+1:])
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, nested...)
	}
	return expanded, nil
}

func globOne(pattern string) ([]string, error) {
	if filepath.IsAbs(pattern) {
		return nil, fmt.Errorf("absolute glob pattern %q is not supported", pattern)
	}

	pattern = strings.TrimPrefix(path.Clean(pattern), "./")
	if pattern == "." {
		pattern = ""
	}

	var matches []string
	segments := strings.Split(pattern, "/")
	if err := globSegments(".", segments, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func globSegments(dir string, segments []string, matches *[]string) error {
	if len(segments) == 0 {
		if dir != "." {
			*matches = append(*matches, dir)
		}
		return nil
	}

	segment := segments[0]
	if segment == "**" {
		if err := globSegments(dir, segments[1:], matches); err != nil {
			return err
		}
		entries, err := os.ReadDir(fromSlashPath(dir))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if err := globSegments(joinSlash(dir, entry.Name()), segments, matches); err != nil {
				return err
			}
		}
		return nil
	}

	// doublestar treats "**.json" as "*.json" within the current directory, not as
	// a recursive match. The contracts checks rely on this for generated artifacts.
	segmentPattern := strings.ReplaceAll(segment, "**", "*")
	if _, err := path.Match(segmentPattern, ""); err != nil {
		return err
	}

	entries, err := os.ReadDir(fromSlashPath(dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	last := len(segments) == 1
	for _, entry := range entries {
		ok, err := path.Match(segmentPattern, entry.Name())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		child := joinSlash(dir, entry.Name())
		if last {
			*matches = append(*matches, child)
			continue
		}
		if entry.IsDir() {
			if err := globSegments(child, segments[1:], matches); err != nil {
				return err
			}
		}
	}
	return nil
}

func fromSlashPath(p string) string {
	if p == "." || p == "" {
		return "."
	}
	return filepath.FromSlash(p)
}

func joinSlash(dir string, name string) string {
	if dir == "." || dir == "" {
		return name
	}
	return dir + "/" + name
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
