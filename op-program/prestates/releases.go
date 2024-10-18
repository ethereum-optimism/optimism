package prestates

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Release struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// Reads the contents of the standard.json file
func GetStandardReleases() ([]Release, error) {
	filepath := "standard.json"
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	var releases []Release
	err = json.Unmarshal(byteValue, &releases)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return releases, nil
}
