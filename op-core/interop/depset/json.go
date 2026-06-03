package depset

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// JSONDependencySetLoader loads a dependency set from a file-path.
type JSONDependencySetLoader struct {
	Path string
}

var _ DependencySetSource = (*JSONDependencySetLoader)(nil)

func (j *JSONDependencySetLoader) LoadDependencySet() (DependencySet, error) {
	f, err := os.Open(j.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dependency set: %w", err)
	}
	defer f.Close()
	return ParseJSONDependencySet(f)
}

func ParseJSONDependencySet(f io.Reader) (DependencySet, error) {
	dec := json.NewDecoder(f)
	var out StaticConfigDependencySet
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode dependency set: %w", err)
	}
	return &out, nil
}
