package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/modfile"
)

const (
	goEthereumPath = "github.com/ethereum/go-ethereum"
	opGethPath     = "github.com/ethereum-optimism/op-geth"

	// cutoverVersion is the upstream go-ethereum release that replaces op-geth in go.mod at the
	// cutover. Type-checking against it, not against the release op-geth is based on, makes the
	// allowlist list every API the cutover has to fix, including APIs upstream changed since.
	cutoverVersion = "v1.17.10"
)

var retryDelay = 5 * time.Second

// upstreamModFile derives from go.mod a module file without the op-geth replace, requiring
// upstream go-ethereum at version instead. It writes the file and its go.sum to dir, resolves
// upstream's dependencies into them, and returns the module path and the file's path.
// The repository's go.mod and go.sum are left untouched.
func upstreamModFile(ctx context.Context, root, dir, version string) (modPath, path string, err error) {
	goMod := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(goMod)
	if err != nil {
		return "", "", err
	}
	f, err := modfile.Parse(goMod, content, nil)
	if err != nil {
		return "", "", err
	}
	var opGeth *modfile.Replace
	for _, r := range f.Replace {
		if r.Old.Path == goEthereumPath && r.New.Path == opGethPath {
			opGeth = r
		}
	}
	if opGeth == nil {
		return "", "", fmt.Errorf("no replace of %s by %s in %s", goEthereumPath, opGethPath, goMod)
	}
	if err := f.DropReplace(goEthereumPath, opGeth.Old.Version); err != nil {
		return "", "", err
	}
	if err := f.AddRequire(goEthereumPath, version); err != nil {
		return "", "", err
	}
	out, err := f.Format()
	if err != nil {
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
