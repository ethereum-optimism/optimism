package depset

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
)

const (
	depsetFileNamePrefix = "dependency_set"
)

// extractor implements the interfaces.DepsetExtractor interface
type extractor struct {
	enclave string
}

// NewExtractor creates a new dependency set extractor
func NewExtractor(enclave string) *extractor {
	return &extractor{
		enclave: enclave,
	}
}

// ExtractData extracts dependency set from its respective artifact
func (e *extractor) ExtractData(ctx context.Context) (map[string]descriptors.DepSet, error) {
	fs, err := ktfs.NewEnclaveFS(ctx, e.enclave)
	if err != nil {
		return nil, err
	}

	return extractDepsetsFromArtifacts(ctx, fs)
}

func extractDepsetsFromArtifacts(ctx context.Context, fs *ktfs.EnclaveFS) (map[string]descriptors.DepSet, error) {
	allArtifacts, err := fs.GetAllArtifactNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all artifact names: %w", err)
	}

	depsetArtifacts := make([]string, 0)
	for _, artifactName := range allArtifacts {
		if strings.HasPrefix(artifactName, depsetFileNamePrefix) {
			depsetArtifacts = append(depsetArtifacts, artifactName)
		}
	}

	depsets := make(map[string]descriptors.DepSet)
	for _, artifactName := range depsetArtifacts {
		a, err := fs.GetArtifact(ctx, artifactName)
		if err != nil {
			return nil, fmt.Errorf("failed to get artifact: %w", err)
		}

		buffer := &bytes.Buffer{}
		if err := a.ExtractFiles(ktfs.NewArtifactFileWriter(artifactName, buffer)); err != nil {
			return nil, fmt.Errorf("failed to extract dependency set: %w", err)
		}

		depsetName := strings.TrimSuffix(strings.TrimPrefix(artifactName, depsetFileNamePrefix+"-"), ".json")
		depsets[depsetName] = descriptors.DepSet(buffer.Bytes())
	}

	return depsets, nil
}
