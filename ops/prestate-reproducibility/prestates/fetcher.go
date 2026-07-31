package prestates

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// standardPrestatesUrl is the URL to the TOML file in superchain registry that defines the list of standard prestates
// Note that this explicitly points to the main branch and is not pinned to a specific version.
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
		d, err := fetchStandardPrestates()
		if err != nil {
			return nil, err
		}
		data = d
	}
	var standardPrestates Prestates
	err := toml.Unmarshal(data, &standardPrestates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standard prestates from %v: %w", standardPrestatesUrl, err)
	}
	return &standardPrestates, nil
}

func fetchStandardPrestates() ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return retry.Do(context.Background(), 3, retry.Fixed(2*time.Second), func() ([]byte, error) {
		resp, err := client.Get(standardPrestatesUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to download standard prestates from %v: %w", standardPrestatesUrl, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to download standard prestates from %v: received status code %d", standardPrestatesUrl, resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read standard prestates from %v: %w", standardPrestatesUrl, err)
		}
		return data, nil
	})
}

type Prestates struct {
	Prestates map[string][]Prestate `toml:"prestates"`
}

type Prestate struct {
	Type string `toml:"type"`
	Hash string `toml:"hash"`
}
