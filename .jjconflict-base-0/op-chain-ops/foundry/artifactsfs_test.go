package foundry

import (
	"embed"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestArtifacts(t *testing.T) {
	logger := testlog.Logger(t, log.LevelWarn) // lower this log level to get verbose test dump of all artifacts
	af := OpenArtifactsDir("./testdata/forge-artifacts")
	artifacts, err := af.ListArtifacts()
	require.NoError(t, err)
	require.NotEmpty(t, artifacts)
	for _, name := range artifacts {
		contracts, err := af.ListContracts(name)
		require.NoError(t, err, "failed to list %s", name)
		require.NotEmpty(t, contracts)
		for _, contract := range contracts {
			artifact, err := af.ReadArtifact(name, contract)
			if err != nil {
				if errors.Is(err, ErrLinkingUnsupported) {
					logger.Info("linking not supported", "name", name, "contract", contract, "err", err)
					continue
				}
				require.NoError(t, err, "failed to read artifact %s / %s", name, contract)
			}
			logger.Info("artifact",
				"name", name,
				"contract", contract,
				"compiler", artifact.Metadata.Compiler.Version,
				"sources", len(artifact.Metadata.Sources),
				"evmVersion", artifact.Metadata.Settings.EVMVersion,
			)
		}
	}
}

//go:embed testdata
var testFS embed.FS

func TestEmbedFS(t *testing.T) {
	embedFS := &EmbedFS{
		FS:      testFS,
		RootDir: "testdata",
	}

	t.Run("Open", func(t *testing.T) {
		// Test opening an existing file
		file, err := embedFS.Open("forge-artifacts/Owned.sol/Owned.json")
		require.NoError(t, err)
		defer file.Close()

		// Verify file stats
		info, err := file.Stat()
		require.NoError(t, err)
		require.Equal(t, "Owned.json", info.Name())
		require.False(t, info.IsDir())
	})

	t.Run("Stat", func(t *testing.T) {
		// Test stat on an existing file
		info, err := embedFS.Stat("forge-artifacts/Owned.sol/Owned.json")
		require.NoError(t, err)
		require.Equal(t, "Owned.json", info.Name())
		require.False(t, info.IsDir())

		// Test stat on a directory
		dirInfo, err := embedFS.Stat("forge-artifacts/Owned.sol")
		require.NoError(t, err)
		require.Equal(t, "Owned.sol", dirInfo.Name())
		require.True(t, dirInfo.IsDir())
	})

	t.Run("ReadDir", func(t *testing.T) {
		// Test reading a directory
		entries, err := embedFS.ReadDir("forge-artifacts/Owned.sol")
		require.NoError(t, err)
		require.Equal(t, 3, len(entries))

		// Check that all expected files exist
		expectedFiles := map[string]bool{
			"Owned.json":        false,
			"Owned.0.8.15.json": false,
			"Owned.0.8.25.json": false,
		}

		// Verify each entry matches an expected file and is not a directory
		for _, entry := range entries {
			name := entry.Name()
			require.Contains(t, expectedFiles, name, "Unexpected file found: %s", name)
			require.False(t, entry.IsDir(), "File should not be a directory: %s", name)
			expectedFiles[name] = true
		}

		// Verify all expected files were found
		for name, found := range expectedFiles {
			require.True(t, found, "Expected file %s was not found", name)
		}
	})
}

func TestEmbedFSWithArtifacts(t *testing.T) {
	embedFS := &EmbedFS{
		FS:      testFS,
		RootDir: "testdata/forge-artifacts",
	}

	// Create an ArtifactsFS using the EmbedFS
	af := &ArtifactsFS{
		FS: embedFS,
	}

	// Test listing artifacts
	artifacts, err := af.ListArtifacts()
	require.NoError(t, err)
	require.Contains(t, artifacts, "ERC20.sol")

	// Test listing contracts for an artifact
	contracts, err := af.ListContracts("ERC20.sol")
	require.NoError(t, err)
	require.Contains(t, contracts, "ERC20")

	// Test reading an artifact
	artifact, err := af.ReadArtifact("ERC20.sol", "ERC20")
	require.NoError(t, err)

	// Verify artifact contents
	require.NotEmpty(t, artifact.ABI)
	require.NotEmpty(t, artifact.Bytecode.Object)
	require.NotEmpty(t, artifact.DeployedBytecode.Object)
	require.NotEmpty(t, artifact.Metadata.Compiler.Version)
}
