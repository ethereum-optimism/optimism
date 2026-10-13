package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
)

const (
	goEthereumPath = "github.com/ethereum/go-ethereum"
	opGethPath     = "github.com/ethereum-optimism/op-geth"
)

var retryDelay = 5 * time.Second

// opGethVersion matches op-geth versions, whose minor version encodes the upstream geth version
// it is based on: v1.101702.4 is based on geth v1.17.2.
var opGethVersion = regexp.MustCompile(`^v(\d+)\.10(\d{2})(\d{2})\.\d+(-rc\.\d+)?$`)

// upstreamVersion returns the upstream go-ethereum version an op-geth version is based on.
func upstreamVersion(opGeth string) (string, error) {
	m := opGethVersion.FindStringSubmatch(opGeth)
	if m == nil {
		return "", fmt.Errorf("unexpected op-geth version %q", opGeth)
	}
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return fmt.Sprintf("v%s.%d.%d", m[1], minor, patch), nil
}

// upstreamModFile derives from go.mod a module file without the op-geth replace, requiring the
// upstream go-ethereum version op-geth is based on instead. It writes the file and its go.sum
// to dir, resolves upstream's dependencies into them, and returns the module path, the upstream
// version and the file's path.
// The repository's go.mod and go.sum are left untouched.
func upstreamModFile(root, dir string) (modPath, version, path string, err error) {
	goMod := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goMod)
	if err != nil {
		return "", "", "", err
	}
	f, err := modfile.Parse(goMod, content, nil)
	if err != nil {
		return "", "", "", err
	}
	var opGeth, replacedVersion string
	for _, r := range f.Replace {
		if r.Old.Path == goEthereumPath && r.New.Path == opGethPath {
			opGeth, replacedVersion = r.New.Version, r.Old.Version
		}
	}
	if opGeth == "" {
		return "", "", "", fmt.Errorf("no replace of %s by %s in %s", goEthereumPath, opGethPath, goMod)
	}
	if version, err = upstreamVersion(opGeth); err != nil {
		return "", "", "", err
	}
	if err := f.DropReplace(goEthereumPath, replacedVersion); err != nil {
		return "", "", "", err
	}
	if err := f.AddRequire(goEthereumPath, version); err != nil {
		return "", "", "", err
	}
	out, err := f.Format()
	if err != nil {
		return "", "", "", err
	}

	path = filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", "", "", err
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		return "", "", "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
		return "", "", "", err
	}
	if err := resolve(root, path, goEthereumPath+"@"+version); err != nil {
		return "", "", "", err
	}
	// go mod tidy may upgrade go-ethereum while looking for packages only op-geth provides.
	resolved, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	rf, err := modfile.Parse(path, resolved, nil)
	if err != nil {
		return "", "", "", err
	}
	for _, r := range rf.Require {
		if r.Mod.Path == goEthereumPath && r.Mod.Version != version {
			return "", "", "", fmt.Errorf("resolving moved %s from %s to %s", goEthereumPath, version, r.Mod.Version)
		}
	}
	return f.Module.Mod.Path, version, path, nil
}

// resolve adds upstream, a go-ethereum module query, to the module file at path and completes
// its build list and go.sum. Upstream needs modules the repository's go.sum doesn't cover, so it
// retries to ride out transient module proxy errors.
func resolve(root, path, upstream string) error {
	steps := [][]string{
		// Fails if go-ethereum can't be downloaded, which go mod tidy -e would only warn about,
		// dropping go-ethereum from the build list.
		{"go", "get", "-modfile=" + path, upstream},
		// -e tolerates imports of packages only op-geth provides. Other modules it fails to add
		// leave dependencies failing to load, which collectUses reports.
		{"go", "mod", "tidy", "-e", "-modfile=" + path},
	}
	var err error
	for attempt := range 3 {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * retryDelay)
		}
		for _, step := range steps {
			cmd := exec.Command(step[0], step[1:]...)
			cmd.Dir = root
			var stderr bytes.Buffer
			cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
			if err = cmd.Run(); err != nil {
				err = fmt.Errorf("%s: %w: %s", strings.Join(step, " "), err, bytes.TrimSpace(stderr.Bytes()))
				break
			}
		}
		if err == nil {
			return nil
		}
	}
	return err
}
