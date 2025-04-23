package env

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/spf13/afero"
)

type fileFetcher struct {
	fs afero.Fs
}

// fetchFileData reads data from a local file
func (f *fileFetcher) fetchFileData(u *url.URL) (*descriptors.DevnetEnvironment, error) {
	body, err := afero.ReadFile(f.fs, u.Path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	basename := u.Path
	if lastSlash := strings.LastIndex(basename, "/"); lastSlash >= 0 {
		basename = basename[lastSlash+1:]
	}
	if lastDot := strings.LastIndex(basename, "."); lastDot >= 0 {
		basename = basename[:lastDot]
	}

	var config descriptors.DevnetEnvironment
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	config.Name = basename

	return &config, nil
}

func fetchFileData(u *url.URL) (*descriptors.DevnetEnvironment, error) {
	fetcher := &fileFetcher{
		fs: afero.NewOsFs(),
	}
	return fetcher.fetchFileData(u)
}
