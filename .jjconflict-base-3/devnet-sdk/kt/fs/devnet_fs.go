package fs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
)

const (
	DevnetEnvArtifactNamePrefix = "devnet-descriptor-"
	DevnetEnvArtifactPath       = "env.json"
)

type DevnetFS struct {
	*EnclaveFS
}

type DevnetFSDescriptorOption func(*options)

type options struct {
	artifactName string
	artifactPath string
}

func newOptions() *options {
	return &options{
		artifactPath: DevnetEnvArtifactPath,
	}
}

func WithArtifactName(name string) DevnetFSDescriptorOption {
	return func(o *options) {
		o.artifactName = name
	}
}

func WithArtifactPath(path string) DevnetFSDescriptorOption {
	return func(o *options) {
		o.artifactPath = path
	}
}

func NewDevnetFS(fs *EnclaveFS) *DevnetFS {
	return &DevnetFS{EnclaveFS: fs}
}

func (fs *DevnetFS) GetDevnetDescriptor(ctx context.Context, opts ...DevnetFSDescriptorOption) (*descriptors.DevnetEnvironment, error) {
	options := newOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.artifactName == "" {
		if err := fs.loadLatestDevnetDescriptorName(ctx, options); err != nil {
			return nil, err
		}
	}

	artifact, err := fs.GetArtifact(ctx, options.artifactName)
	if err != nil {
		return nil, fmt.Errorf("error getting artifact: %w", err)
	}

	var buf bytes.Buffer
	writer := NewArtifactFileWriter(options.artifactPath, &buf)

	if err := artifact.ExtractFiles(writer); err != nil {
		return nil, fmt.Errorf("error extracting file from artifact: %w", err)
	}

	var env descriptors.DevnetEnvironment
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("error unmarshalling environment: %w", err)
	}

	return &env, nil
}

func (fs *DevnetFS) UploadDevnetDescriptor(ctx context.Context, env *descriptors.DevnetEnvironment, opts ...DevnetFSDescriptorOption) error {
	envBuf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(envBuf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(env); err != nil {
		return fmt.Errorf("error encoding environment: %w", err)
	}

	options := newOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.artifactName == "" {
		if err := fs.loadNextDevnetDescriptorName(ctx, options); err != nil {
			return fmt.Errorf("error getting next devnet descriptor: %w", err)
		}
	}

	if err := fs.PutArtifact(ctx, options.artifactName, NewArtifactFileReader(options.artifactPath, envBuf)); err != nil {
		return fmt.Errorf("error putting environment artifact: %w", err)
	}

	return nil
}

func (fs *DevnetFS) loadLatestDevnetDescriptorName(ctx context.Context, options *options) error {
	names, err := fs.GetAllArtifactNames(ctx)
	if err != nil {
		return fmt.Errorf("error getting artifact names: %w", err)
	}

	var maxSuffix int = -1
	var maxName string
	for _, name := range names {
		_, suffix, found := strings.Cut(name, DevnetEnvArtifactNamePrefix)
		if !found {
			continue
		}

		// Parse the suffix as a number
		var num int
		if _, err := fmt.Sscanf(suffix, "%d", &num); err != nil {
			continue // Skip if suffix is not a valid number
		}

		// Update maxName if this number is larger
		if num > maxSuffix {
			maxSuffix = num
			maxName = name
		}
	}

	if maxName == "" {
		return fmt.Errorf("no descriptor found with valid numerical suffix")
	}

	options.artifactName = maxName
	return nil
}

func (fs *DevnetFS) loadNextDevnetDescriptorName(ctx context.Context, options *options) error {
	artifactNames, err := fs.GetAllArtifactNames(ctx)
	if err != nil {
		return fmt.Errorf("error getting artifact names: %w", err)
	}

	maxNum := -1
	for _, artifactName := range artifactNames {
		if !strings.HasPrefix(artifactName, DevnetEnvArtifactNamePrefix) {
			continue
		}

		numStr := strings.TrimPrefix(artifactName, DevnetEnvArtifactNamePrefix)
		num := 0
		if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
			log.Printf("Warning: invalid devnet descriptor format: %s", artifactName)
			continue
		}

		if num > maxNum {
			maxNum = num
		}
	}

	options.artifactName = fmt.Sprintf("%s%d", DevnetEnvArtifactNamePrefix, maxNum+1)
	return nil
}
