// Package bundle packs a built forge project into the contracts bundle that op-deployer runs forge on.
package bundle

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Write writes the contracts bundle of the built forge project at absBase to w as a tar
// archive. The bundle leaves out the AST of each artifact: forge parses every artifact of a
// script's build at the start of a run, and the AST is most of the artifacts' size and of forge's
// memory. It also leaves out the Solidity files that are not sources of the build, and the other
// files of the libraries except for license files and forge configs. It keeps the compile cache and
// the build infos as they are, so that forge does not compile. exclude holds more glob patterns of
// files to leave out. A commit that is not empty is written to the COMMIT file.
//
// A bundle is code and data that scripts with a deployer key read, so Write reads only files of the
// project: it takes a link only for a source of the build whose target is another source of the
// build, and skips every other link. A link in the artifacts directory, or in ScriptReadDir, fails
// it.
func Write(w io.Writer, absBase string, exclude []string, commit string) error {
	for _, pattern := range exclude {
		if _, err := path.Match(filepath.ToSlash(pattern), ""); err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
	}
	cache, err := readCompileCache(absBase)
	if err != nil {
		return err
	}

	// The targets of links resolve to paths below the resolved project.
	absBase, err = filepath.EvalSymlinks(absBase)
	if err != nil {
		return fmt.Errorf("resolve the forge project: %w", err)
	}
	root, err := os.OpenRoot(absBase)
	if err != nil {
		return fmt.Errorf("open the forge project: %w", err)
	}
	defer root.Close()

	tw := tar.NewWriter(w)
	// The walk lists the project through root too, so that a directory replaced by a link lists
	// nothing outside it.
	if err := fs.WalkDir(root.FS(), ".", func(slashRel string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := filepath.FromSlash(slashRel)

		// Before the filters: a link that hides artifacts or sources must fail the pack, not drop
		// them.
		if d.Type()&fs.ModeSymlink != 0 {
			if cache.inArtifacts(slashRel) || under(slashRel, ScriptReadDir) {
				return fmt.Errorf("%s is a link, which the artifacts of a bundle may not have", slashRel)
			}
			if cache.hasSourceUnder(slashRel) {
				return fmt.Errorf("%s links to a directory of sources, which a bundle does not follow", slashRel)
			}
		}
		if shouldExclude(rel, d, exclude) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() && cache.inLibrary(slashRel) {
			// Untar creates the directories of the library files that the bundle keeps.
			return nil
		}
		if !d.IsDir() && !cache.keeps(slashRel) {
			return nil
		}
		if slashRel == "." {
			return nil
		}

		var info fs.FileInfo
		var content []byte
		var err error
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			if !cache.isSource(slashRel) {
				return nil
			}
			// forge reads a source through its link, so the bundle needs a copy of its target, which
			// must be a source of the build too.
			target, err := linkedSource(absBase, rel, cache)
			if err != nil {
				return fmt.Errorf("the link of the source %s: %w", slashRel, err)
			}
			if content, info, err = readFile(root, target); err != nil {
				return fmt.Errorf("read the target of the source %s: %w", slashRel, err)
			}
		case d.Type().IsRegular():
			if content, info, err = readFile(root, rel); err != nil {
				return err
			}
			if cache.isArtifact(slashRel) {
				if content, err = stripAST(content); err != nil {
					return fmt.Errorf("strip the AST of %s: %w", slashRel, err)
				}
			}
		case d.IsDir():
			if info, err = d.Info(); err != nil {
				return err
			}
		default:
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = slashRel
		hdr.ModTime = info.ModTime()
		hdr.AccessTime = info.ModTime()
		hdr.Size = int64(len(content))

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = tw.Write(content)
		return err
	}); err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	if commit != "" {
		if err := addCommitFile(tw, commit); err != nil {
			return fmt.Errorf("failed to add COMMIT file: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	return nil
}

// CompileCacheFile is forge's compile cache, relative to the project.
const CompileCacheFile = "cache/solidity-files-cache.json"

// ScriptReadDir is the directory of a bundle that op-deployer lets scripts read, whatever the compile
// cache names as the artifacts directory.
const ScriptReadDir = "forge-artifacts"

// compileCache is the part of forge's compile cache that decides what the bundle keeps. Its paths
// are relative to the project and use slashes.
type compileCache struct {
	Paths struct {
		Artifacts string   `json:"artifacts"`
		Libraries []string `json:"libraries"`
	} `json:"paths"`
	// Files are the sources of the build, keyed by path.
	Files map[string]json.RawMessage `json:"files"`
}

func readCompileCache(absBase string) (*compileCache, error) {
	data, err := os.ReadFile(filepath.Join(absBase, CompileCacheFile))
	if err != nil {
		return nil, fmt.Errorf("read the compile cache; build the contracts first: %w", err)
	}
	var cache compileCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("decode the compile cache: %w", err)
	}
	cache.Paths.Artifacts = path.Clean(cache.Paths.Artifacts)
	for i, lib := range cache.Paths.Libraries {
		cache.Paths.Libraries[i] = path.Clean(lib)
	}
	for _, dir := range append([]string{cache.Paths.Artifacts}, cache.Paths.Libraries...) {
		if dir == "." || !filepath.IsLocal(dir) {
			return nil, fmt.Errorf("the compile cache has the path %q, not a directory in the project", dir)
		}
	}
	return &cache, nil
}

// ArtifactsDir returns the artifacts directory of the forge project at dir, relative to it and with
// slashes, as its compile cache records it. It fails with fs.ErrNotExist when the project has no
// compile cache.
func ArtifactsDir(dir string) (string, error) {
	cache, err := readCompileCache(dir)
	if err != nil {
		return "", err
	}
	return cache.Paths.Artifacts, nil
}

// readFile reads the regular file at name in root, and refuses a link, or a file that changed while
// it read it: os.Root follows a link in the root even with O_NOFOLLOW.
func readFile(root *os.Root, name string) ([]byte, fs.FileInfo, error) {
	// Opening a pipe or a device could block or read without end.
	found, err := root.Lstat(name)
	if err != nil {
		return nil, nil, err
	}
	if !found.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", name)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !os.SameFile(found, info) {
		return nil, nil, fmt.Errorf("%s changed while op-deployer read it", name)
	}
	content, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	return content, info, nil
}

// linkedSource returns the path in the project at absBase of the target of the link at rel, and
// fails unless that target is a kept source of the build.
func linkedSource(absBase string, rel string, cache *compileCache) (string, error) {
	target, err := filepath.EvalSymlinks(filepath.Join(absBase, rel))
	if err != nil {
		return "", err
	}
	targetRel, err := filepath.Rel(absBase, target)
	if err != nil {
		return "", err
	}
	if slashTarget := filepath.ToSlash(targetRel); !filepath.IsLocal(targetRel) || !cache.isSource(slashTarget) || !cache.keeps(slashTarget) {
		return "", fmt.Errorf("its target %s is not a source that the bundle keeps", target)
	}
	return targetRel, nil
}

// under reports whether rel is dir or in it.
func under(rel string, dir string) bool {
	return rel == dir || strings.HasPrefix(rel, dir+"/")
}

// inArtifacts reports whether rel is the artifacts directory or in it.
func (c *compileCache) inArtifacts(rel string) bool {
	return under(rel, c.Paths.Artifacts)
}

// hasSourceUnder reports whether a source of the build is in the directory rel.
func (c *compileCache) hasSourceUnder(rel string) bool {
	for source := range c.Files {
		if strings.HasPrefix(source, rel+"/") {
			return true
		}
	}
	return false
}

// isSource reports whether rel is a source of the build.
func (c *compileCache) isSource(rel string) bool {
	_, ok := c.Files[rel]
	return ok
}

// isArtifact reports whether the file at rel is an artifact.
func (c *compileCache) isArtifact(rel string) bool {
	return strings.HasPrefix(rel, c.Paths.Artifacts+"/") && strings.HasSuffix(rel, ".json")
}

// inLibrary reports whether rel is in a library directory.
func (c *compileCache) inLibrary(rel string) bool {
	for _, lib := range c.Paths.Libraries {
		if strings.HasPrefix(rel, lib+"/") {
			return true
		}
	}
	return false
}

// keeps reports whether the bundle keeps the file at rel. It keeps the sources of the build, which
// include every file that a kept source imports, but no other Solidity file. Of the other files in a
// library, it keeps the license files and the forge configs, which forge reads for the library's
// remappings. It keeps no file named .env* or in a directory named so.
func (c *compileCache) keeps(rel string) bool {
	// A project's environment can hold secrets, and the bundle stays on disk.
	if slices.ContainsFunc(strings.Split(rel, "/"), func(name string) bool { return strings.HasPrefix(name, ".env") }) {
		return false
	}
	if c.isSource(rel) {
		return true
	}
	if strings.HasSuffix(rel, ".sol") {
		return false
	}
	return !c.inLibrary(rel) || isLicenseFile(rel) || isForgeConfig(rel)
}

func isLicenseFile(rel string) bool {
	name := strings.ToUpper(path.Base(rel))
	for _, prefix := range []string{"LICENSE", "LICENCE", "COPYING", "NOTICE"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isForgeConfig(rel string) bool {
	name := path.Base(rel)
	return name == "foundry.toml" || name == "remappings.txt"
}

// stripAST returns the artifact without its "ast" field. Neither forge scripts nor the Go script
// host read the AST, and forge's compile cache does not depend on it.
func stripAST(artifact []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(artifact, &fields); err != nil {
		return nil, err
	}
	if _, ok := fields["ast"]; !ok {
		return artifact, nil
	}
	delete(fields, "ast")
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fields); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func shouldExclude(rel string, d os.DirEntry, exclude []string) bool {
	if rel == "." {
		return false
	}

	rel = filepath.ToSlash(rel)

	for _, pattern := range exclude {
		pattern = filepath.ToSlash(pattern)
		if matchPattern(pattern, rel) {
			return true
		}
	}

	if strings.HasPrefix(rel, "book/") || rel == "book" {
		return true
	}
	if strings.HasPrefix(rel, "snapshots/") || rel == "snapshots" {
		return true
	}

	if !d.IsDir() {
		if strings.HasSuffix(d.Name(), ".t.sol") {
			return true
		}
	}

	return false
}

// matchPattern reports whether rel or one of its parent directories matches pattern, which
// Write has validated.
func matchPattern(pattern, rel string) bool {
	if matched, _ := path.Match(pattern, rel); matched {
		return true
	}

	if !strings.HasSuffix(pattern, "/") {
		pattern = pattern + "/"
	}
	matched, _ := path.Match(pattern+"*", rel+"/")
	return matched
}

func addCommitFile(tw *tar.Writer, commit string) error {
	hdr := &tar.Header{
		Name:       "COMMIT",
		Size:       int64(len(commit)),
		Mode:       0644,
		ModTime:    time.Now(),
		AccessTime: time.Now(),
		ChangeTime: time.Now(),
		Typeflag:   tar.TypeReg,
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if _, err := tw.Write([]byte(commit)); err != nil {
		return err
	}

	return nil
}
