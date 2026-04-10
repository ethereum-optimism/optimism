package journal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/oprm/release"
	"gopkg.in/yaml.v3"
)

const (
	frontmatterDelimiter = "---"
)

// Store manages persisted release run journals.
type Store struct {
	runsDir string
}

func NewStore(runsDir string) *Store {
	return &Store{runsDir: runsDir}
}

func (s *Store) RunsDir() string {
	return s.runsDir
}

func (s *Store) PathFor(runID string) string {
	return filepath.Join(s.runsDir, runID+".md")
}

func (s *Store) ResolvePath(identifier string) string {
	if strings.HasSuffix(identifier, ".md") || strings.ContainsRune(identifier, os.PathSeparator) {
		return identifier
	}
	return s.PathFor(identifier)
}

func (s *Store) Save(run *release.Run) (string, error) {
	if err := os.MkdirAll(s.runsDir, 0o755); err != nil {
		return "", fmt.Errorf("create runs dir %q: %w", s.runsDir, err)
	}
	path := s.PathFor(run.RunID)
	content, err := Marshal(run)
	if err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return "", fmt.Errorf("write run journal: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("rename run journal into place: %w", err)
	}
	return path, nil
}

func (s *Store) Load(identifier string) (*release.Run, string, error) {
	path := s.ResolvePath(identifier)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("read run journal %q: %w", path, err)
	}
	run, err := Unmarshal(data)
	if err != nil {
		return nil, path, fmt.Errorf("parse run journal %q: %w", path, err)
	}
	return run, path, nil
}

func Marshal(run *release.Run) ([]byte, error) {
	frontmatter, err := yaml.Marshal(run)
	if err != nil {
		return nil, fmt.Errorf("marshal run frontmatter: %w", err)
	}
	var out bytes.Buffer
	out.WriteString(frontmatterDelimiter)
	out.WriteByte('\n')
	out.Write(frontmatter)
	if len(frontmatter) == 0 || frontmatter[len(frontmatter)-1] != '\n' {
		out.WriteByte('\n')
	}
	out.WriteString(frontmatterDelimiter)
	out.WriteString("\n\n")
	out.WriteString("# OPRM Release Run ")
	out.WriteString(run.RunID)
	out.WriteString("\n\n")
	out.WriteString("## Timeline\n\n")
	if len(run.Timeline) == 0 {
		out.WriteString("_No entries yet._\n")
		return out.Bytes(), nil
	}
	for _, entry := range run.Timeline {
		out.WriteString("- ")
		out.WriteString(entry.At.UTC().Format(time.RFC3339))
		out.WriteByte(' ')
		out.WriteString(entry.Summary)
		out.WriteByte('\n')
		for _, detail := range entry.Details {
			out.WriteString("  - ")
			out.WriteString(detail)
			out.WriteByte('\n')
		}
	}
	return out.Bytes(), nil
}

func Unmarshal(data []byte) (*release.Run, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != frontmatterDelimiter {
		return nil, fmt.Errorf("missing markdown frontmatter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == frontmatterDelimiter {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, fmt.Errorf("unterminated markdown frontmatter")
	}
	frontmatter := strings.Join(lines[1:end], "\n")
	var run release.Run
	if err := yaml.Unmarshal([]byte(frontmatter), &run); err != nil {
		return nil, fmt.Errorf("decode frontmatter: %w", err)
	}
	return &run, nil
}
