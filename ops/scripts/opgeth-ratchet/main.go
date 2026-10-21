// Command opgeth-ratchet type-checks the monorepo against upstream go-ethereum and compares the
// fork-API uses it finds against a committed allowlist, so that no new dependency on op-geth
// lands unnoticed and the cutover to upstream go-ethereum is rehearsed on every change.
//
// It type-checks in two passes. The first is against the upstream version op-geth is based on,
// and fails when a use is not on the allowlist, or when an allowlist entry is no longer used.
// The second is against the upstream release in latest-upstream.txt, and fails on any
// error in an upstream-clean package: one whose imports reach no package with a fork-API use.
// Modules outside the monorepo that import these packages build them against current upstream.
//
// With -update it removes the entries that are no longer used; it never adds entries.
package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	allowlistFile = "ops/scripts/opgeth-ratchet/allowlist.txt"
	latestFile    = "ops/scripts/opgeth-ratchet/latest-upstream.txt"
	decouplingDoc = "docs/ai/opgeth-decoupling.md"
)

func main() {
	update := flag.Bool("update", false, "remove allowlist entries that are no longer used")
	flag.Parse()
	// Cancelling instead of dying on a signal lets run remove its temporary module directory. A
	// second signal kills it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	context.AfterFunc(ctx, stop)
	ok, err := run(ctx, *update)
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
	if !ok {
		os.Exit(1)
	}
}

func run(ctx context.Context, update bool) (bool, error) {
	root, err := opservice.FindMonorepoRoot(".")
	if err != nil {
		return false, err
	}
	allowPath := filepath.Join(root, allowlistFile)
	allow, err := readAllowlist(allowPath)
	if err != nil {
		return false, err
	}
	base, err := baseVersion(root)
	if err != nil {
		return false, err
	}
	latest, err := latestVersion(latestUpstreamPin, base)
	if err != nil {
		return false, err
	}

	tmp, err := os.MkdirTemp("", "opgeth-ratchet")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	// cgo_test is the one build tag in the tree that hides geth uses from a default build.
	const tags = "-tags=cgo_test"
	patterns := []string{"./..."}

	modPath, baseMod, err := upstreamModFile(ctx, root, filepath.Join(tmp, "base"), base)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(os.Stderr, "Type-checking %s against go-ethereum %s...\n", modPath, base)
	uses, err := collectUses(ctx, root, modPath, goEthereumPath, []string{"-modfile=" + baseMod, tags}, patterns)
	if err != nil {
		return false, err
	}
	ok, err := checkAllowlist(allow, allowPath, uses, base, update)
	if err != nil {
		return false, err
	}
	if latest == base {
		// The first pass already type-checked every package against this version.
		return ok, nil
	}

	_, latestMod, err := upstreamModFile(ctx, root, filepath.Join(tmp, "latest"), latest)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(os.Stderr, "Type-checking upstream-clean packages against go-ethereum %s...\n", latest)
	drift, err := collectDrift(ctx, root, modPath, uses, []string{"-modfile=" + latestMod, tags}, patterns)
	if err != nil {
		return false, err
	}
	if len(drift) > 0 {
		fmt.Printf(`These errors against go-ethereum %s are in packages that use no op-geth-only
API, directly or through their imports, or in their dependencies:

  %s

Modules outside the monorepo build these packages against current upstream
go-ethereum, so they must compile against both op-geth and the release in
%s. There is no allowlist for this: write code that compiles
against both, or move what can't behind a seam (see %s).
`, latest, strings.Join(drift, "\n  "), latestFile, decouplingDoc)
	}
	return ok && len(drift) == 0, nil
}

// checkAllowlist compares uses with the allowlist and reports the differences. With update, it
// removes the entries no longer used from the allowlist file instead of failing on them.
func checkAllowlist(allow *Allowlist, allowPath string, uses map[Use]bool, version string, update bool) (bool, error) {
	unlisted, stale := allow.diff(uses)
	if update && len(stale) > 0 {
		if err := allow.writeWithout(allowPath, stale); err != nil {
			return false, err
		}
		fmt.Printf("Removed from %s, no longer used:\n\n  %s\n", allowlistFile, joinUses(stale, "\n  "))
		stale = nil
	}
	if len(unlisted) > 0 {
		fmt.Printf(`These packages use go-ethereum APIs that only op-geth provides:

  %s

The monorepo is moving to upstream go-ethereum (%s). Please avoid the
dependency: %s describes the replacements (op-core/*, the
op-service seams, op-reth) and where each kind of use is headed.

If there is no way around it yet, add the lines below to %s,
replacing %s with the issue tracking its removal and why the use can't be
avoided for now:

%s
`, joinUses(unlisted, "\n  "), version, decouplingDoc, allowlistFile, reasonPlaceholder, entryLines(unlisted))
	}
	if len(stale) > 0 {
		fmt.Printf(`These allowlist entries are no longer used:

  %s

Run "just opgeth-ratchet-update" to remove them from %s.
`, joinUses(stale, "\n  "), allowlistFile)
	}
	return len(unlisted) == 0 && len(stale) == 0, nil
}

func sortUses(uses []Use) {
	slices.SortFunc(uses, func(a, b Use) int {
		return cmp.Or(cmp.Compare(a.Package, b.Package), cmp.Compare(a.Need, b.Need))
	})
}

func joinUses(uses []Use, sep string) string {
	s := make([]string, len(uses))
	for i, u := range uses {
		s[i] = u.String()
	}
	return strings.Join(s, sep)
}

func entryLines(uses []Use) string {
	var b strings.Builder
	for _, u := range uses {
		b.WriteString(entryLine(u, reasonPlaceholder))
		b.WriteByte('\n')
	}
	return b.String()
}
