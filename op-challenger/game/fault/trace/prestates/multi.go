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

	// supportedFileTypes lists, in preferred order, the prestate file types to attempt to download
	supportedFileTypes = []string{".bin.gz", ".json.gz", ".json"}
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
	// First try to find a previously downloaded prestate
	for _, fileType := range supportedFileTypes {
		path := filepath.Join(m.dataDir, hash.Hex()+fileType)
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue // File doesn't exist, try the next file type
		} else if err != nil {
			return "", fmt.Errorf("error checking for existing prestate %v in file %v: %w", hash, path, err)
		}
		return path, nil // Found an existing file so use it
	}

	// Didn't find any available files, try to download one
	var combinedErr error // Keep a track of each download attempt so we can report them if none work
	for _, fileType := range supportedFileTypes {
		path := filepath.Join(m.dataDir, hash.Hex()+fileType)
		if err := m.fetchPrestate(hash, fileType, path); errors.Is(err, ErrPrestateUnavailable) {
			combinedErr = errors.Join(combinedErr, err)
			continue // Didn't find prestate in this format, try the next
		} else if err != nil {
			return "", fmt.Errorf("error downloading prestate %v to file %v: %w", hash, path, err)
		}
		return path, nil // Successfully downloaded a prestate so use it
	}
	return "", errors.Join(ErrPrestateUnavailable, combinedErr)
}

func (m *MultiPrestateProvider) fetchPrestate(hash common.Hash, fileType string, dest string) error {
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("error creating prestate dir: %w", err)
	}
	prestateUrl := m.baseUrl.JoinPath(hash.Hex() + fileType)
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
