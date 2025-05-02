package build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	ktfs "github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/enclave"
	"github.com/spf13/afero"
)

// ContractBuilder handles building smart contracts using just commands
type ContractBuilder struct {
	// Base directory where the build commands should be executed
	baseDir string
	// Template for the build command
	cmdTemplate *template.Template

	// Dry run mode
	dryRun bool

	builtContracts map[string]string

	enclave string

	// Command factory for testing
	cmdFactory cmdFactory
	// Enclave manager for testing
	enclaveManager *enclave.KurtosisEnclaveManager
	// Enclave filesystem for testing
	enclaveFS *ktfs.EnclaveFS
	// Filesystem for operations
	fs afero.Fs
}

const (
	contractsCmdTemplateStr = "just {{ .ContractsPath }}/build-no-tests"
	relativeContractsPath   = "../packages/contracts-bedrock"
	solidityCachePath       = "cache/solidity-files-cache.json"
)

var defaultContractTemplate *template.Template

func init() {
	defaultContractTemplate = template.Must(template.New("contract_build_cmd").Parse(contractsCmdTemplateStr))
}

type ContractBuilderOptions func(*ContractBuilder)

func WithContractBaseDir(baseDir string) ContractBuilderOptions {
	return func(b *ContractBuilder) {
		b.baseDir = baseDir
	}
}

func WithContractDryRun(dryRun bool) ContractBuilderOptions {
	return func(b *ContractBuilder) {
		b.dryRun = dryRun
	}
}

func WithContractEnclave(enclave string) ContractBuilderOptions {
	return func(b *ContractBuilder) {
		b.enclave = enclave
	}
}

func WithContractEnclaveManager(manager *enclave.KurtosisEnclaveManager) ContractBuilderOptions {
	return func(b *ContractBuilder) {
		b.enclaveManager = manager
	}
}

func WithContractFS(fs afero.Fs) ContractBuilderOptions {
	return func(b *ContractBuilder) {
		b.fs = fs
	}
}

// NewContractBuilder creates a new ContractBuilder instance
func NewContractBuilder(opts ...ContractBuilderOptions) *ContractBuilder {
	b := &ContractBuilder{
		baseDir:        ".",
		cmdTemplate:    defaultContractTemplate,
		dryRun:         false,
		builtContracts: make(map[string]string),
		cmdFactory:     defaultCmdFactory,
		enclaveManager: nil,
		enclaveFS:      nil,
		fs:             afero.NewOsFs(), // Default to OS filesystem
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Build executes the contract build command
func (b *ContractBuilder) Build(_ string) (string, error) {
	// since we ignore layer for now, we can skip the build if the file already
	// exists: it'll be the same file!
	if url, ok := b.builtContracts[""]; ok {
		return url, nil
	}

	log.Println("Building contracts bundle")

	// Execute template to get command string
	var cmdBuf bytes.Buffer
	if err := b.cmdTemplate.Execute(&cmdBuf, map[string]string{
		"ContractsPath": relativeContractsPath,
	}); err != nil {
		return "", fmt.Errorf("failed to execute command template: %w", err)
	}

	// Create command using the factory
	cmd := b.cmdFactory("sh", "-c", cmdBuf.String())
	cmd.SetDir(b.baseDir)

	if b.dryRun {
		return "artifact://contracts", nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("contract build command failed: %w\nOutput: %s", err, string(output))
	}

	name, err := b.createContractsArtifact()
	if err != nil {
		return "", fmt.Errorf("failed to create contracts artifact: %w", err)
	}

	url := fmt.Sprintf("artifact://%s", name)
	b.builtContracts[""] = url
	return url, nil
}

func (b *ContractBuilder) GetContractUrl() string {
	if b.dryRun {
		return "artifact://contracts"
	}
	return fmt.Sprintf("artifact://%s", b.getBuiltContractName())
}

func (b *ContractBuilder) getBuiltContractName() string {
	return fmt.Sprintf("contracts-%s", b.buildHash())
}

func (b *ContractBuilder) buildHash() string {
	// the solidity cache file contains up-to-date information about the current
	// state of the build, so it's suitable to provide a unique hash.
	cacheFilePath := filepath.Join(b.baseDir, relativeContractsPath, solidityCachePath)

	fileData, err := afero.ReadFile(b.fs, cacheFilePath)
	if err != nil {
		log.Printf("Error reading solidity cache file: %v", err)
		return "error"
	}

	hash := sha256.Sum256(fileData)
	return hex.EncodeToString(hash[:])
}

func (b *ContractBuilder) createContractsArtifact() (name string, retErr error) {
	ctx := context.TODO()
	name = b.getBuiltContractName()

	// Ensure the enclave exists
	var err error
	if b.enclaveManager == nil {
		return "", fmt.Errorf("enclave manager not set")
	}

	// TODO: this is not ideal, we should feed the resulting enclave into the
	// EnclaveFS constructor. As it is, we're making sure the enclave exists,
	// and we recreate an enclave context for NewEnclaveFS..
	_, err = b.enclaveManager.GetEnclave(ctx, b.enclave)
	if err != nil {
		return "", fmt.Errorf("failed to get or create enclave: %w", err)
	}

	// Create a new Kurtosis filesystem for the specified enclave
	var enclaveFS *ktfs.EnclaveFS
	if b.enclaveFS != nil {
		enclaveFS = b.enclaveFS
	} else {
		enclaveFS, err = ktfs.NewEnclaveFS(ctx, b.enclave)
		if err != nil {
			return "", fmt.Errorf("failed to create enclave filesystem: %w", err)
		}
	}

	// Check if artifact already exists
	artifactNames, err := enclaveFS.GetAllArtifactNames(ctx)
	if err != nil {
		log.Printf("Warning: Failed to retrieve artifact names: %v", err)
	} else {
		for _, existingName := range artifactNames {
			if existingName == name {
				log.Printf("Artifact '%s' already exists, skipping creation", name)
				return name, nil
			}
		}
	}

	// Check the base contracts directory
	contractsDir := filepath.Join(b.baseDir, relativeContractsPath)
	exists, err := afero.DirExists(b.fs, contractsDir)
	if err != nil || !exists {
		return "", fmt.Errorf("contracts directory not found at %s: %w", contractsDir, err)
	}

	// Create temp directory to hold the files we want to include
	tempDir, err := afero.TempDir(b.fs, "", "contracts-artifacts-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if err := b.fs.RemoveAll(tempDir); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to cleanup temporary directory: %w", err)
		}
	}()

	// Populate the temp directory with the necessary files
	if err := b.populateContractsArtifact(contractsDir, tempDir); err != nil {
		return "", fmt.Errorf("failed to populate contracts artifact: %w", err)
	}

	// Create file readers for the artifact
	readers, err := b.createArtifactReaders(tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to create artifact readers: %w", err)
	}

	// Upload the artifact with all file readers
	err = enclaveFS.PutArtifact(ctx, name, readers...)
	if err != nil {
		return "", fmt.Errorf("failed to upload artifact: %w", err)
	}

	return
}

// createArtifactReaders creates file readers for all files in the directory
func (b *ContractBuilder) createArtifactReaders(dir string) ([]*ktfs.ArtifactFileReader, error) {
	var readers []*ktfs.ArtifactFileReader
	var openFiles []io.Closer

	err := afero.Walk(b.fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories themselves
		if info.IsDir() {
			return nil
		}

		// Get relative path from base dir
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Open the file
		file, err := b.fs.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		openFiles = append(openFiles, file)

		// Add file reader to the list
		readers = append(readers, ktfs.NewArtifactFileReader(relPath, file))

		return nil
	})

	if err != nil {
		// Close any open files
		for _, file := range openFiles {
			file.Close()
		}
		return nil, fmt.Errorf("failed to prepare artifact files: %w", err)
	}

	return readers, nil
}

// populateContractsArtifact populates the temporary directory with required contract files
func (b *ContractBuilder) populateContractsArtifact(contractsDir, tempDir string) error {
	// Handle forge-artifacts directories - exclude *.t.sol directories as we don't need tests.
	// op-deployer will need contracts and scripts.
	forgeArtifactsPath := filepath.Join(contractsDir, "forge-artifacts")
	exists, err := afero.DirExists(b.fs, forgeArtifactsPath)
	if err != nil {
		return fmt.Errorf("failed to check forge-artifacts directory: %w", err)
	}
	if !exists {
		return nil
	}

	entries, err := afero.ReadDir(b.fs, forgeArtifactsPath)
	if err != nil {
		return fmt.Errorf("failed to read forge-artifacts directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip test directories
		if strings.HasSuffix(entry.Name(), ".t.sol") {
			continue
		}

		srcPath := filepath.Join(forgeArtifactsPath, entry.Name())
		destPath := filepath.Join(tempDir, entry.Name())

		// Create destination directory
		if err := b.fs.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destPath, err)
		}

		// Copy directory contents
		err = afero.Walk(b.fs, srcPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Get path relative to source directory
			relPath, err := filepath.Rel(srcPath, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// Skip root directory
			if relPath == "." {
				return nil
			}

			destPath := filepath.Join(destPath, relPath)

			if info.IsDir() {
				return b.fs.MkdirAll(destPath, info.Mode())
			}

			// Copy file contents
			srcFile, err := b.fs.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open source file: %w", err)
			}
			defer srcFile.Close()

			destFile, err := b.fs.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create destination file: %w", err)
			}
			defer destFile.Close()

			if _, err := io.Copy(destFile, srcFile); err != nil {
				return fmt.Errorf("failed to copy file contents: %w", err)
			}

			return b.fs.Chmod(destPath, info.Mode())
		})

		if err != nil {
			return fmt.Errorf("failed to copy directory %s: %w", srcPath, err)
		}
	}

	return nil
}
