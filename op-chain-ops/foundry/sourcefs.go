package foundry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/op-chain-ops/srcmap"
)

// SourceMapFS wraps an FS to provide source-maps.
// This FS relies on the following file path assumptions:
// - `/artifacts/build-info/X.json` (build-info path is read from the below file): build files, of foundry incremental builds.
// - `/cache/solidity-files-cache.json`: a JSON file enumerating all files, and when the build last changed.
// - `/` a root dir, relative to where the source files are located (as per the compilationTarget metadata in an artifact).
type SourceMapFS struct {
	fs fs.FS

	// optionally, the source-map FS can utilize the build-data of a specific compiler-profile.
	// If left empty, assume there is a single compiler profile in the solidity-files-cache, and use that.
	// The profile can be changed with SetProfile. Forge uses "default" as default profile name.
	profile string
}

// NewSourceMapFS creates a new SourceMapFS.
// The source-map FS loads identifiers for srcmap.ParseSourceMap
// and provides a util to retrieve a source-map for an Artifact.
// The solidity source-files are lazy-loaded when using the produced sourcemap.
func NewSourceMapFS(fs fs.FS) *SourceMapFS {
	return &SourceMapFS{fs: fs}
}

// SetCompilerProfile changes the compiler-profile that is looked
// for when reversing build-info of artifacts.
func (s *SourceMapFS) SetCompilerProfile(profile string) {
	s.profile = profile
}

// ForgeBuild represents the JSON content of a forge-build entry in the `artifacts/build-info` output.
type ForgeBuild struct {
	ID             string                     `json:"id"`                // ID of the build itself
	SourceIDToPath map[srcmap.SourceID]string `json:"source_id_to_path"` // srcmap ID to source filepath
}

func (s *SourceMapFS) readBuild(buildInfoPath string, id string) (*ForgeBuild, error) {
	buildPath := path.Join(buildInfoPath, id+".json")
	f, err := s.fs.Open(buildPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open build: %w", err)
	}
	defer f.Close()
	var build ForgeBuild
	if err := json.NewDecoder(f).Decode(&build); err != nil {
		return nil, fmt.Errorf("failed to read build: %w", err)
	}
	return &build, nil
}

// ForgeBuildEntry represents a JSON entry that links the build job of a contract source file.
type ForgeBuildEntry struct {
	Path    string `json:"path"`
	BuildID string `json:"build_id"`
}

// ForgeBuildInfo represents a JSON entry that enumerates the latest builds per contract per compiler version.
type ForgeBuildInfo struct {
	// contract name -> solidity version -> profile -> build entry
	Artifacts map[string]map[string]map[string]ForgeBuildEntry `json:"artifacts"`
}

// ForgeBuildCache rep
type ForgeBuildCache struct {
	Paths struct {
		BuildInfos string `json:"build_infos"`
	} `json:"paths"`
	Files map[string]ForgeBuildInfo `json:"files"`
}

func (s *SourceMapFS) readBuildCache() (*ForgeBuildCache, error) {
	cachePath := path.Join("cache", "solidity-files-cache.json")
	f, err := s.fs.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open build cache: %w", err)
	}
	defer f.Close()
	var buildCache ForgeBuildCache
	if err := json.NewDecoder(f).Decode(&buildCache); err != nil {
		return nil, fmt.Errorf("failed to read build cache: %w", err)
	}
	return &buildCache, nil
}

// ReadSourceIDs reads the source-identifier to source file-path mapping that is needed to translate a source-map
// of the given contract, the given compiler version, and within the given source file path.
func (s *SourceMapFS) ReadSourceIDs(path string, contract string, compilerVersion string, profile string) (map[srcmap.SourceID]string, error) {
	buildCache, err := s.readBuildCache()
	if err != nil {
		return nil, err
	}
	artifactBuilds, ok := buildCache.Files[path]
	if !ok {
		return nil, fmt.Errorf("no known builds for path %q", path)
	}
	byCompilerVersion, ok := artifactBuilds.Artifacts[contract]
	if !ok {
		return nil, fmt.Errorf("contract not found in artifact: %q", contract)
	}
	var byProfile map[string]ForgeBuildEntry
	if compilerVersion != "" {
		byProfile, ok = byCompilerVersion[compilerVersion]
		if !ok {
			return nil, fmt.Errorf("no known build for compiler version: %q", compilerVersion)
		}
	} else {
		if len(byCompilerVersion) == 0 {
			return nil, errors.New("no known build, unspecified compiler version")
		}
		if len(byCompilerVersion) > 1 {
			return nil, fmt.Errorf("no compiler version specified, and more than one option: %s", strings.Join(maps.Keys(byCompilerVersion), ", "))
		}
		// select the only remaining entry
		for _, v := range byCompilerVersion {
			byProfile = v
		}
	}
	var buildEntry ForgeBuildEntry
	if profile != "" {
		buildEntry, ok = byProfile[profile]
		if !ok {
			return nil, fmt.Errorf("no known build for profile: %q", profile)
		}
	} else {
		if len(byProfile) == 0 {
			return nil, errors.New("no known build, unspecified profile")
		}
		if len(byProfile) > 1 {
			return nil, fmt.Errorf("no profile specified, and more than one option: %s", strings.Join(maps.Keys(byProfile), ", "))
		}
		// select the only remaining entry
		for _, v := range byProfile {
			buildEntry = v
		}
	}
	build, err := s.readBuild(filepath.ToSlash(buildCache.Paths.BuildInfos), buildEntry.BuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to read build %q of contract %q: %w", buildEntry.BuildID, contract, err)
	}
	return build.SourceIDToPath, nil
}

// SourceMap retrieves a source-map for a given contract of a foundry Artifact.
func (s *SourceMapFS) SourceMap(artifact *Artifact, contract string) (*srcmap.SourceMap, error) {
	srcPath := ""
	for path, name := range artifact.Metadata.Settings.CompilationTarget {
		if name == contract {
			srcPath = path
			break
		}
	}
	if srcPath == "" {
		return nil, fmt.Errorf("no known source path for contract %s in artifact", contract)
	}
	// The commit suffix is ignored, the core semver part is what is used in the resolution of builds.
	basicCompilerVersion := strings.SplitN(artifact.Metadata.Compiler.Version, "+", 2)[0]
	// Unfortunately, the "metadata" of an artifact does not store which compiler-profile it used.
	// It's only part of the artifact name, which we don't have here.
	// E.g. `Arithmetic.0.8.15.dispute.json` for "dispute" profile,
	// and `Arithmetic.0.8.15.json` for the default profile.
	// We allow the user to specify the profile to use here, with SourceMapFS.SetCompilerProfile.
	profile := s.profile
	ids, err := s.ReadSourceIDs(srcPath, contract, basicCompilerVersion, profile)
	if err != nil {
		return nil, fmt.Errorf("failed to read source IDs of %q: %w", srcPath, err)
	}
	return srcmap.ParseSourceMap(s.fs, ids, artifact.DeployedBytecode.Object, artifact.DeployedBytecode.SourceMap)
}
