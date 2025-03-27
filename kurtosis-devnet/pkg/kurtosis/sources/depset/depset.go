package depset

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
)

const (
	depsetFileName = "dependency_set.json"
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
func (e *extractor) ExtractData(ctx context.Context) (json.RawMessage, error) {
	fs, err := ktfs.NewEnclaveFS(ctx, e.enclave)
	if err != nil {
		return nil, err
	}

	return extractDepsetFromArtifact(ctx, fs, depsetFileName)
}

func extractDepsetFromArtifact(ctx context.Context, fs *ktfs.EnclaveFS, artifactName string) (json.RawMessage, error) {
	a, err := fs.GetArtifact(ctx, artifactName)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	buffer := &bytes.Buffer{}
	if err := a.ExtractFiles(ktfs.NewArtifactFileWriter(depsetFileName, buffer)); err != nil {
		return nil, fmt.Errorf("failed to extract dependency set: %w", err)
	}

	return json.RawMessage(buffer.Bytes()), nil
}
