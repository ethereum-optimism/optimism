package release

import (
	"fmt"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/ethereum-optimism/optimism/oprm/components"
)

type ChangeDetection struct {
	ComparedRef  string
	Changed      bool
	MatchedFiles []string
	ScannedFiles int
}

func DetectComponentChanges(spec components.ComponentSpec, comparedRef string, changedFiles []string) (*ChangeDetection, error) {
	matched, err := MatchChangedFiles(spec.ChangeScope, changedFiles)
	if err != nil {
		return nil, fmt.Errorf("match changed files for %s: %w", spec.ID, err)
	}
	return &ChangeDetection{
		ComparedRef:  comparedRef,
		Changed:      len(matched) > 0,
		MatchedFiles: matched,
		ScannedFiles: len(changedFiles),
	}, nil
}

func MatchChangedFiles(scope []string, changedFiles []string) ([]string, error) {
	matchedSet := make(map[string]struct{})
	for _, file := range changedFiles {
		for _, pattern := range scope {
			ok, err := doublestar.PathMatch(pattern, file)
			if err != nil {
				return nil, fmt.Errorf("pattern %q: %w", pattern, err)
			}
			if ok {
				matchedSet[file] = struct{}{}
				break
			}
		}
	}
	matched := make([]string, 0, len(matchedSet))
	for file := range matchedSet {
		matched = append(matched, file)
	}
	sort.Strings(matched)
	return matched, nil
}
