package main

import (
	"bytes"
	"context"
	_ "embed"
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
	"golang.org/x/mod/semver"
)

const (
	goEthereumPath = "github.com/ethereum/go-ethereum"
	opGethPath     = "github.com/ethereum-optimism/op-geth"
)

// latestUpstreamPin is the upstream go-ethereum release that upstream-clean packages must also
// type-check against. Bump it with upstream releases.
//
//go:embed latest-upstream.txt
var latestUpstreamPin string

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

// latestVersion validates pin, the content of latest-upstream.txt, as a go-ethereum release no
// older than base and returns it.
func latestVersion(pin, base string) (string, error) {
	v := strings.TrimSpace(pin)
	if semver.Canonical(v) != v || semver.Prerelease(v) != "" {
		return "", fmt.Errorf("latest-upstream.txt: want a go-ethereum release version, got %q", v)
	}
	if semver.Compare(v, base) < 0 {
		return "", fmt.Errorf("latest-upstream.txt: %s is older than op-geth's upstream base %s", v, base)
	}
	return v, nil
}

func parseGoMod(root string) (*modfile.File, error) {
	goMod := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goMod)
	if err != nil {
		return nil, err
	}
	return modfile.Parse(goMod, content, nil)
}

// opGethReplace returns the op-geth version that f replaces go-ethereum with, and the
// go-ethereum version the replace applies to.
func opGethReplace(f *modfile.File) (opGeth, replaced string, err error) {
	for _, r := range f.Replace {
		if r.Old.Path == goEthereumPath && r.New.Path == opGethPath {
			return r.New.Version, r.Old.Version, nil
		}
	}
	return "", "", fmt.Errorf("no replace of %s by %s in %s", goEthereumPath, opGethPath, f.Syntax.Name)
}

// baseVersion returns the upstream go-ethereum version that the op-geth in go.mod is based on.
func baseVersion(root string) (string, error) {
	f, err := parseGoMod(root)
	if err != nil {
		return "", err
	}
	opGeth, _, err := opGethReplace(f)
	if err != nil {
		return "", err
	}
	return upstreamVersion(opGeth)
}

// upstreamModFile derives from go.mod a module file without the op-geth replace, requiring
// upstream go-ethereum at version instead. It writes the file and its go.sum to dir, resolves
// upstream's dependencies into them, and returns the module path and the file's path.
// The repository's go.mod and go.sum are left untouched.
func upstreamModFile(ctx context.Context, root, dir, version string) (modPath, path string, err error) {
	f, err := parseGoMod(root)
	if err != nil {
		return "", "", err
	}
	_, replaced, err := opGethReplace(f)
	if err != nil {
		return "", "", err
	}
	if err := f.DropReplace(goEthereumPath, replaced); err != nil {
		return "", "", err
	}
	if err := f.AddRequire(goEthereumPath, version); err != nil {
		return "", "", err
	}
	out, err := f.Format()
	if err != nil {
		return "", "", err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	path = filepath.Join(dir, "go.mod")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", "", err
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
		return "", "", err
	}
	if err := resolve(ctx, root, path, goEthereumPath+"@"+version); err != nil {
		return "", "", err
	}
	// go mod tidy may upgrade go-ethereum while looking for packages only op-geth provides.
	resolved, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	rf, err := modfile.Parse(path, resolved, nil)
	if err != nil {
		return "", "", err
	}
	for _, r := range rf.Require {
		if r.Mod.Path == goEthereumPath && r.Mod.Version != version {
			return "", "", fmt.Errorf("resolving moved %s from %s to %s", goEthereumPath, version, r.Mod.Version)
		}
	}
	return f.Module.Mod.Path, path, nil
}

// resolve adds upstream, a go-ethereum module query, to the module file at path and completes
// its build list and go.sum. Upstream needs modules the repository's go.sum doesn't cover, so it
// retries to ride out transient module proxy errors.
func resolve(ctx context.Context, root, path, upstream string) error {
	run := func(args ...string) error {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = root
		var stderr bytes.Buffer
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(stderr.Bytes()))
		}
		return nil
	}
	var err error
	for attempt := range 3 {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * retryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		// Fails if go-ethereum can't be downloaded, which go mod tidy -e would only warn about,
		// dropping go-ethereum from the build list.
		if err = run("go", "get", "-modfile="+path, upstream); err != nil {
			continue
		}
		// -e tolerates imports of packages only op-geth provides. Other modules it fails to add
		// leave dependencies failing to load, which collectUses reports.
		if err = run("go", "mod", "tidy", "-e", "-modfile="+path); err == nil {
			return nil
		}
	}
	return err
}
