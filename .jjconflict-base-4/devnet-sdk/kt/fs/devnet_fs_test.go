package fs

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDevnetDescriptor(t *testing.T) {
	envContent := `{"name": "test-env", "l1": {"name": "l1", "id": "1", "nodes": []}, "l2": []}`
	expectedEnv := &descriptors.DevnetEnvironment{
		Name: "test-env",
		L1: &descriptors.Chain{
			Name:  "l1",
			ID:    "1",
			Nodes: []descriptors.Node{},
		},
		L2: []*descriptors.L2Chain{},
	}

	tests := []struct {
		name         string
		artifactName string
		artifactPath string
		envContent   string
		wantErr      bool
		expectedEnv  *descriptors.DevnetEnvironment
	}{
		{
			name:         "successful retrieval with default path",
			artifactName: "devnet-descriptor-1",
			artifactPath: DevnetEnvArtifactPath,
			envContent:   envContent,
			expectedEnv:  expectedEnv,
		},
		{
			name:         "successful retrieval with custom path",
			artifactName: "devnet-descriptor-1",
			artifactPath: "custom/path/env.json",
			envContent:   envContent,
			expectedEnv:  expectedEnv,
		},
		{
			name:         "invalid json content",
			artifactName: "devnet-descriptor-1",
			artifactPath: DevnetEnvArtifactPath,
			envContent:   `invalid json`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock context with artifact
			mockCtx := &mockEnclaveContext{
				artifacts: map[string][]byte{
					tt.artifactName: createTarGzArtifact(t, map[string]string{
						tt.artifactPath: tt.envContent,
					}),
				},
				fs: afero.NewMemMapFs(),
			}

			enclaveFS, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
			require.NoError(t, err)

			devnetFS := NewDevnetFS(enclaveFS)

			// Get descriptor with options
			opts := []DevnetFSDescriptorOption{}
			if tt.artifactName != "" {
				opts = append(opts, WithArtifactName(tt.artifactName))
			}
			if tt.artifactPath != DevnetEnvArtifactPath {
				opts = append(opts, WithArtifactPath(tt.artifactPath))
			}

			env, err := devnetFS.GetDevnetDescriptor(context.Background(), opts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedEnv, env)
		})
	}
}

func TestUploadDevnetDescriptor(t *testing.T) {
	env := &descriptors.DevnetEnvironment{
		Name: "test-env",
		L1: &descriptors.Chain{
			Name:  "l1",
			ID:    "1",
			Nodes: []descriptors.Node{},
		},
		L2: []*descriptors.L2Chain{},
	}

	tests := []struct {
		name         string
		artifactName string
		artifactPath string
		env          *descriptors.DevnetEnvironment
		wantErr      bool
	}{
		{
			name:         "successful upload with default path",
			artifactName: "devnet-descriptor-1",
			artifactPath: DevnetEnvArtifactPath,
			env:          env,
		},
		{
			name:         "successful upload with custom path",
			artifactName: "devnet-descriptor-1",
			artifactPath: "custom/path/env.json",
			env:          env,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock context
			mockCtx := &mockEnclaveContext{
				artifacts: make(map[string][]byte),
				fs:        afero.NewMemMapFs(),
			}

			enclaveFS, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
			require.NoError(t, err)

			devnetFS := NewDevnetFS(enclaveFS)

			// Upload descriptor with options
			opts := []DevnetFSDescriptorOption{}
			if tt.artifactName != "" {
				opts = append(opts, WithArtifactName(tt.artifactName))
			}
			if tt.artifactPath != DevnetEnvArtifactPath {
				opts = append(opts, WithArtifactPath(tt.artifactPath))
			}

			err = devnetFS.UploadDevnetDescriptor(context.Background(), tt.env, opts...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify the artifact was uploaded
			require.NotNil(t, mockCtx.uploaded)
			uploaded := mockCtx.uploaded[tt.artifactName]
			require.NotNil(t, uploaded)
			require.Contains(t, uploaded, tt.artifactPath)
		})
	}
}

func TestLoadLatestDevnetDescriptorName(t *testing.T) {
	tests := []struct {
		name          string
		existingNames []string
		expectedName  string
		wantErr       bool
	}{
		{
			name: "single descriptor",
			existingNames: []string{
				"devnet-descriptor-1",
			},
			expectedName: "devnet-descriptor-1",
		},
		{
			name: "multiple descriptors",
			existingNames: []string{
				"devnet-descriptor-1",
				"devnet-descriptor-3",
				"devnet-descriptor-2",
			},
			expectedName: "devnet-descriptor-3",
		},
		{
			name:          "no descriptors",
			existingNames: []string{},
			wantErr:       true,
		},
		{
			name: "invalid descriptor names",
			existingNames: []string{
				"invalid-name",
				"devnet-descriptor-invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock context with artifacts
			mockCtx := &mockEnclaveContext{
				artifacts: make(map[string][]byte),
				fs:        afero.NewMemMapFs(),
			}

			// Add artifacts to the mock context
			for _, name := range tt.existingNames {
				mockCtx.artifacts[name] = []byte{}
			}

			enclaveFS, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
			require.NoError(t, err)

			devnetFS := NewDevnetFS(enclaveFS)

			options := newOptions()
			err = devnetFS.loadLatestDevnetDescriptorName(context.Background(), options)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedName, options.artifactName)
		})
	}
}

func TestLoadNextDevnetDescriptorName(t *testing.T) {
	tests := []struct {
		name          string
		existingNames []string
		expectedName  string
	}{
		{
			name:          "no existing descriptors",
			existingNames: []string{},
			expectedName:  "devnet-descriptor-0",
		},
		{
			name: "single descriptor",
			existingNames: []string{
				"devnet-descriptor-1",
			},
			expectedName: "devnet-descriptor-2",
		},
		{
			name: "multiple descriptors",
			existingNames: []string{
				"devnet-descriptor-1",
				"devnet-descriptor-3",
				"devnet-descriptor-2",
			},
			expectedName: "devnet-descriptor-4",
		},
		{
			name: "with invalid descriptor names",
			existingNames: []string{
				"invalid-name",
				"devnet-descriptor-1",
				"devnet-descriptor-invalid",
			},
			expectedName: "devnet-descriptor-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock context with artifacts
			mockCtx := &mockEnclaveContext{
				artifacts: make(map[string][]byte),
				fs:        afero.NewMemMapFs(),
			}

			// Add artifacts to the mock context
			for _, name := range tt.existingNames {
				mockCtx.artifacts[name] = []byte{}
			}

			enclaveFS, err := NewEnclaveFS(context.Background(), "test-enclave", WithEnclaveCtx(mockCtx), WithFs(mockCtx.fs))
			require.NoError(t, err)

			devnetFS := NewDevnetFS(enclaveFS)

			options := newOptions()
			err = devnetFS.loadNextDevnetDescriptorName(context.Background(), options)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedName, options.artifactName)
		})
	}
}
