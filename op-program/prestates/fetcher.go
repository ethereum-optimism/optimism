package prestates

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
)

// standardPrestatesUrl is the URL to the TOML file in superchain registry that defines the list of standard prestates
// Note that this explicitly points to the main branch and is not pinned to a specific version. The verification check
// intends to
const standardPrestatesUrl = "https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/refs/heads/main/validation/standard/standard-prestates.toml"

func LoadReleases(overrideFile string) (*Prestates, error) {
	var data []byte
	if overrideFile != "" {
		d, err := os.ReadFile(overrideFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read override file (%v): %w", overrideFile, err)
		}
		data = d
	} else {
		resp, err := http.Get(standardPrestatesUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to download standard prestates from %v: %w", standardPrestatesUrl, err)
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read standard prestates from %v: %w", standardPrestatesUrl, err)
		}
	}
	var standardPrestates Prestates
	err := toml.Unmarshal(data, &standardPrestates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standard prestates from %v: %w", standardPrestatesUrl, err)
	}
	return &standardPrestates, nil
}

type Prestates struct {
	Prestates map[string][]Prestate `toml:"prestates"`
}

type Prestate struct {
	Type string `toml:"type"`
	Hash string `toml:"hash"`
}
