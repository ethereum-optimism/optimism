// Command opgeth-ratchet type-checks the monorepo against upstream go-ethereum and compares the
// fork-API uses it finds against a committed allowlist, so that no new dependency on op-geth
// lands unnoticed and the cutover to upstream go-ethereum is rehearsed on every change.
//
// It fails when a use is not on the allowlist, or when an allowlist entry is no longer used.
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

	tmp, err := os.MkdirTemp("", "opgeth-ratchet")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmp)
	modPath, version, modFile, err := upstreamModFile(ctx, root, tmp)
	if err != nil {
		return false, err
	}
	fmt.Fprintf(os.Stderr, "Type-checking %s against go-ethereum %s...\n", modPath, version)
	// cgo_test is the one build tag in the tree that hides geth uses from a default build.
	flags := []string{"-modfile=" + modFile, "-tags=cgo_test"}
	uses, err := collectUses(ctx, root, modPath, goEthereumPath, flags, []string{"./..."})
	if err != nil {
		return false, err
	}

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
