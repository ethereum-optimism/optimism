package foundry

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

type StatDirFs interface {
	fs.StatFS
	fs.ReadDirFS
}

func OpenArtifactsDir(dirPath string) *ArtifactsFS {
	dir := os.DirFS(dirPath)
	if d, ok := dir.(StatDirFs); !ok {
		panic("Go DirFS guarantees changed")
	} else {
		return &ArtifactsFS{FS: d}
	}
}

// ArtifactsFS wraps a filesystem (read-only access) of a forge-artifacts bundle.
// The root contains directories for every artifact,
// each containing one or more entries (one per solidity compiler version) for a solidity contract.
// See OpenArtifactsDir for reading from a local directory.
// Alternative FS systems, like a tarball, may be used too.
type ArtifactsFS struct {
	FS StatDirFs
}

// ListArtifacts lists the artifacts. Each artifact matches a source-file name.
// This name includes the extension, e.g. ".sol"
// (no other artifact-types are supported at this time).
func (af *ArtifactsFS) ListArtifacts() ([]string, error) {
	entries, err := af.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to list artifacts: %w", err)
	}
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		// Some artifacts may be nested in directories not suffixed with ".sol"
		// Nested artifacts, and non-solidity artifacts, are not supported.
		if name := d.Name(); strings.HasSuffix(name, ".sol") {
			out = append(out, d.Name())
		}
	}
	return out, nil
}

// ListContracts lists the contracts of the named artifact, including the file extension.
// E.g. "Owned.sol" might list "Owned.0.8.15", "Owned.0.8.25", and "Owned".
func (af *ArtifactsFS) ListContracts(name string) ([]string, error) {
	f, err := af.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact %q: %w", name, err)
	}
	defer f.Close()
	dirFile, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, fmt.Errorf("no dir for artifact %q, but got %T", name, f)
	}
	entries, err := dirFile.ReadDir(0)
	if err != nil {
		return nil, fmt.Errorf("failed to list artifact contents of %q: %w", name, err)
	}
	out := make([]string, 0, len(entries))
	for _, d := range entries {
		if name := d.Name(); strings.HasSuffix(name, ".json") {
			out = append(out, strings.TrimSuffix(name, ".json"))
		}
	}
	return out, nil
}

// ReadArtifact reads a specific JSON contract artifact from the FS.
// The contract name may be suffixed by a solidity compiler version, e.g. "Owned.0.8.25".
// The contract name does not include ".json", this is a detail internal to the artifacts.
// The name of the artifact is the source-file name, this must include the suffix such as ".sol".
func (af *ArtifactsFS) ReadArtifact(name string, contract string) (*Artifact, error) {
	artifactPath := path.Join(name, contract+".json")
	f, err := af.FS.Open(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open artifact %q: %w", artifactPath, err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var out Artifact
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode artifact %q: %w", name, err)
	}
	return &out, nil
}
