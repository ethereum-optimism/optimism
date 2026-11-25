package prestate

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-prestate/util"
)

// standardPrestatesURL points to the TOML file in the superchain-registry that defines the list of standard prestates.
// It explicitly tracks the main branch and is not pinned to a specific version.
const standardPrestatesURL = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/refs/heads/main/validation/standard/standard-prestates.toml"

// LoadReleases loads the standard prestates list from the superchain-registry, or from overrideFile if provided.
func LoadReleases(overrideFile string) (*Releases, error) {
	var data []byte
	if overrideFile != "" {
		d, err := os.ReadFile(overrideFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read override file (%v): %w", overrideFile, err)
		}
		data = d
	} else {
		d, err := util.Fetch(standardPrestatesURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download standard prestates: %w", err)
		}
		data = d
	}
	var releases Releases
	if err := toml.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse standard prestates from %v: %w", standardPrestatesURL, err)
	}
	return &releases, nil
}

type Releases struct {
	Prestates map[string][]Release `toml:"prestates"`
}

type Release struct {
	Type string `toml:"type"`
	Hash string `toml:"hash"`
}
