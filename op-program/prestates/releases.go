package prestates

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed standard.json
var standardJSON []byte

type Release struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// Reads the contents of the standard.json file
func GetStandardReleases() ([]Release, error) {
	var releases []Release
	err := json.Unmarshal(standardJSON, &releases)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return releases, nil
}
