package prestates

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPrestateUnavailable = errors.New("prestate unavailable")
)

type MultiPrestateProvider struct {
	baseUrl *url.URL
	dataDir string
}

func NewMultiPrestateProvider(baseUrl *url.URL, dataDir string) *MultiPrestateProvider {
	return &MultiPrestateProvider{
		baseUrl: baseUrl,
		dataDir: dataDir,
	}
}

func (m *MultiPrestateProvider) PrestatePath(hash common.Hash) (string, error) {
	path := filepath.Join(m.dataDir, hash.Hex()+".json.gz")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := m.fetchPrestate(hash, path); err != nil {
			return "", fmt.Errorf("failed to fetch prestate: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("error checking for existing prestate %v: %w", hash, err)
	}
	return path, nil
}

func (m *MultiPrestateProvider) fetchPrestate(hash common.Hash, dest string) error {
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("error creating prestate dir: %w", err)
	}
	prestateUrl := m.baseUrl.JoinPath(hash.Hex() + ".json")
	resp, err := http.Get(prestateUrl.String())
	if err != nil {
		return fmt.Errorf("failed to fetch prestate from %v: %w", prestateUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w from url %v: status %v", ErrPrestateUnavailable, prestateUrl, resp.StatusCode)
	}
	out, err := ioutil.NewAtomicWriterCompressed(dest, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open atomic writer for %v: %w", dest, err)
	}
	defer func() {
		// If errors occur, try to clean up without renaming the file into its final destination as Close() would do
		_ = out.Abort()
	}()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to write file %v: %w", dest, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close file %v: %w", dest, err)
	}
	return nil
}
