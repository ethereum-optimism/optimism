package hardhat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-bindings/solc"
)

var (
	ErrCannotFindDeployment = errors.New("cannot find deployment")
	ErrCannotFindArtifact   = errors.New("cannot find artifact")
)

// `Hardhat` encapsulates all of the functionality required to interact
// with hardhat style artifacts.
type Hardhat struct {
	ArtifactPaths   []string
	DeploymentPaths []string

	network string

	amu sync.Mutex
	dmu sync.Mutex
	bmu sync.Mutex

	artifacts   []*Artifact
	deployments []*Deployment
	buildInfos  []*BuildInfo //nolint:unused
}

// New creates a new `Hardhat` struct and reads all of the files from
// disk so that they are cached for the end user. A network is passed
// that corresponds to the network that they deployments are associated
// with. A slice of artifact paths and deployment paths are passed
// so that a single `Hardhat` instance can operate on multiple sets
// of artifacts and deployments. The deployments paths should be
// the root of the deployments directory that contains additional
// directories for each particular network.
func New(network string, artifacts, deployments []string) (*Hardhat, error) {
	hh := &Hardhat{
		network:         network,
		ArtifactPaths:   artifacts,
		DeploymentPaths: deployments,
	}

	if err := hh.init(); err != nil {
		return nil, err
	}

	return hh, nil
}

// init is called in the constructor and will cache required files to disk.
func (h *Hardhat) init() error {
	h.amu.Lock()
	defer h.amu.Unlock()
	h.dmu.Lock()
	defer h.dmu.Unlock()

	if err := h.initArtifacts(); err != nil {
		return err
	}
	if err := h.initDeployments(); err != nil {
		return err
	}
	return nil
}

// initDeployments reads all of the deployment json files from disk and then
// caches the deserialized `Deployment` structs.
func (h *Hardhat) initDeployments() error {
	for _, deploymentPath := range h.DeploymentPaths {
		fileSystem := os.DirFS(filepath.Join(deploymentPath, h.network))
		err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.Contains(path, "solcInputs") {
				return nil
			}
			if !strings.HasSuffix(path, ".json") {
				return nil
			}

			name := filepath.Join(deploymentPath, h.network, path)
			file, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			var deployment Deployment
			if err := json.Unmarshal(file, &deployment); err != nil {
				return err
			}

			deployment.Name = filepath.Base(name[:len(name)-5])
			h.deployments = append(h.deployments, &deployment)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// initArtifacts reads all of the artifact json files from disk and then caches
// the deserialized `Artifact` structs.
func (h *Hardhat) initArtifacts() error {
	for _, artifactPath := range h.ArtifactPaths {
		fileSystem := os.DirFS(artifactPath)
		err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			name := filepath.Join(artifactPath, path)

			if strings.Contains(name, "build-info") {
				return nil
			}
			if strings.HasSuffix(name, ".dbg.json") {
				return nil
			}
			file, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			var artifact Artifact
			if err := json.Unmarshal(file, &artifact); err != nil {
				return err
			}

			h.artifacts = append(h.artifacts, &artifact)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetArtifact returns the artifact that corresponds to the contract.
// This method supports just the contract name and the fully qualified
// contract name.
func (h *Hardhat) GetArtifact(name string) (*Artifact, error) {
	h.amu.Lock()
	defer h.amu.Unlock()

	if IsFullyQualifiedName(name) {
		fqn := ParseFullyQualifiedName(name)
		for _, artifact := range h.artifacts {
			contractNameMatches := artifact.ContractName == fqn.ContractName
			sourceNameMatches := artifact.SourceName == fqn.SourceName

			if contractNameMatches && sourceNameMatches {
				return artifact, nil
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrCannotFindArtifact, name)
	}

	for _, artifact := range h.artifacts {
		if name == artifact.ContractName {
			return artifact, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrCannotFindArtifact, name)
}

// GetDeployment returns the deployment that corresponds to the contract.
// It does not support fully qualified contract names.
func (h *Hardhat) GetDeployment(name string) (*Deployment, error) {
	h.dmu.Lock()
	defer h.dmu.Unlock()

	fqn := ParseFullyQualifiedName(name)
	for _, deployment := range h.deployments {
		if deployment.Name == fqn.ContractName {
			return deployment, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrCannotFindDeployment, name)
}

// GetBuildInfo returns the build info that corresponds to the contract.
// It does not support fully qualified contract names.
func (h *Hardhat) GetBuildInfo(name string) (*BuildInfo, error) {
	h.bmu.Lock()
	defer h.bmu.Unlock()

	fqn := ParseFullyQualifiedName(name)
	buildInfos := make([]*BuildInfo, 0)

	for _, artifactPath := range h.ArtifactPaths {
		fileSystem := os.DirFS(artifactPath)
		err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			name := filepath.Join(artifactPath, path)

			if !strings.HasSuffix(name, ".dbg.json") {
				return nil
			}

			// Remove ".dbg.json"
			target := filepath.Base(name[:len(name)-9])
			if fqn.ContractName != target {
				return nil
			}

			file, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			var debugFile DebugFile
			if err := json.Unmarshal(file, &debugFile); err != nil {
				return err
			}
			relPath := filepath.Join(filepath.Dir(name), debugFile.BuildInfo)
			if err != nil {
				return err
			}
			debugPath, _ := filepath.Abs(relPath)

			buildInfoFile, err := os.ReadFile(debugPath)
			if err != nil {
				return err
			}

			var buildInfo BuildInfo
			if err := json.Unmarshal(buildInfoFile, &buildInfo); err != nil {
				return err
			}

			buildInfos = append(buildInfos, &buildInfo)

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// TODO(tynes): handle multiple contracts with same name when required
	if len(buildInfos) > 1 {
		return nil, fmt.Errorf("Multiple contracts with name %s", name)
	}
	if len(buildInfos) == 0 {
		return nil, fmt.Errorf("Cannot find BuildInfo for %s", name)
	}

	return buildInfos[0], nil
}

// TODO(tynes): handle fully qualified names properly
func (h *Hardhat) GetStorageLayout(name string) (*solc.StorageLayout, error) {
	fqn := ParseFullyQualifiedName(name)

	buildInfo, err := h.GetBuildInfo(name)
	if err != nil {
		return nil, err
	}

	for _, source := range buildInfo.Output.Contracts {
		for name, contract := range source {
			if name == fqn.ContractName {
				return &contract.StorageLayout, nil
			}
		}
	}

	return nil, fmt.Errorf("contract not found for %s", fqn.ContractName)
}
