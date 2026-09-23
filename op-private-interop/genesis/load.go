package genesis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core"
)

// maxGenesisBytes bounds a downloaded genesis. A stock op-deployer genesis is around 10 MB; a
// response far beyond that is a misconfigured URL, not a genesis.
const maxGenesisBytes = 256 << 20

// LoadPrivateChainGenesis reads the private-chain genesis from a local path or an http(s) URL.
//
// Deployments stage a chain's genesis as a downloadable artifact and hand execution clients its
// URL; the batcher and the supernode take the same reference so every consumer of the projection
// reads one artifact. Loading is the only I/O: the projection itself stays a pure function of the
// decoded genesis.
func LoadPrivateChainGenesis(ctx context.Context, pathOrURL string) (*core.Genesis, error) {
	if pathOrURL == "" {
		return nil, fmt.Errorf("private-chain genesis path or URL is required")
	}
	var data []byte
	var err error
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		data, err = fetchGenesis(ctx, pathOrURL)
	} else {
		data, err = os.ReadFile(pathOrURL)
	}
	if err != nil {
		return nil, fmt.Errorf("reading private-chain genesis %q: %w", pathOrURL, err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return nil, fmt.Errorf("decoding private-chain genesis %q: %w", pathOrURL, err)
	}
	return &genesis, nil
}

func fetchGenesis(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		data, err := fetchOnce(ctx, url)
		if err == nil {
			return data, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}
	return nil, lastErr
}

func fetchOnce(ctx context.Context, url string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxGenesisBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxGenesisBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxGenesisBytes)
	}
	return data, nil
}
