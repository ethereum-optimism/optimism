package env

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
)

// DevnetFS is an interface that both our mock and the real implementation satisfy
type DevnetFS interface {
	GetDevnetDescriptor(ctx context.Context, opts ...ktfs.DevnetFSDescriptorOption) (*descriptors.DevnetEnvironment, error)
}

type devnetFSFactory func(ctx context.Context, enclave string) (DevnetFS, error)

type kurtosisFetcher struct {
	devnetFSFactory devnetFSFactory
}

func newDevnetFS(ctx context.Context, enclave string) (DevnetFS, error) {
	fs, err := ktfs.NewEnclaveFS(ctx, enclave)
	if err != nil {
		return nil, err
	}
	return ktfs.NewDevnetFS(fs), nil
}

// parseKurtosisURL parses a Kurtosis URL of the form kt://enclave/artifact/file
// If artifact is omitted, it defaults to ""
// If file is omitted, it defaults to "env.json"
func (f *kurtosisFetcher) parseKurtosisURL(u *url.URL) (enclave, artifactName, fileName string) {
	enclave = u.Host
	artifactName = ""
	fileName = ktfs.DevnetEnvArtifactPath

	// Trim both prefix and suffix slashes before splitting
	trimmedPath := strings.Trim(u.Path, "/")
	parts := strings.Split(trimmedPath, "/")
	if len(parts) > 0 && parts[0] != "" {
		artifactName = parts[0]
	}
	if len(parts) > 1 && parts[1] != "" {
		fileName = parts[1]
	}

	return
}

// fetchKurtosisData reads data from a Kurtosis artifact
func (f *kurtosisFetcher) fetchKurtosisData(u *url.URL) (*descriptors.DevnetEnvironment, error) {
	enclave, artifactName, fileName := f.parseKurtosisURL(u)

	devnetFS, err := f.devnetFSFactory(context.Background(), enclave)
	if err != nil {
		return nil, fmt.Errorf("error creating enclave fs: %w", err)
	}

	env, err := devnetFS.GetDevnetDescriptor(context.Background(), ktfs.WithArtifactName(artifactName), ktfs.WithArtifactPath(fileName))
	if err != nil {
		return nil, fmt.Errorf("error getting devnet descriptor: %w", err)
	}

	env.Name = enclave
	return env, nil
}
