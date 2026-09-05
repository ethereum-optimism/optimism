// Command gen-flags generates MDX flag-reference tables for OP Stack Go
// services from their urfave/cli flag definitions, so the docs flag catalogs
// are derived from source instead of hand-transcribed from `--help` output.
//
// For each component listed in components below, the tool walks the composed
// flag slice the binary actually registers (including the flag families
// appended from op-service: rpc, log, metrics, pprof, and txmgr) and emits
// a snippet into docs/public-docs/snippets/generated/. The snippet carries a
// do-not-edit header and a provenance line naming the release tag recorded in
// manifest.json.
//
// Run from the monorepo root:
//
//	go run ./docs/public-docs/scripts/gen-flags
//
// Regeneration is keyed to finalized component releases: the snippets are
// generated from the source at the release tag recorded in manifest.json and
// refreshed when a new finalized (non-rc) tag is published — see README.md in
// this directory for the procedure. Pass -check to verify the committed
// snippets match the SHA-256 recorded in manifest.json (hand-edit detection;
// see README.md for why regeneration parity is only checked informationally),
// without rewriting anything.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	batcherflags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	challengerflags "github.com/ethereum-optimism/optimism/op-challenger/flags"
	conductorflags "github.com/ethereum-optimism/optimism/op-conductor/flags"
	nodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	proposerflags "github.com/ethereum-optimism/optimism/op-proposer/flags"
)

// component describes one service whose flags are rendered into a snippet.
type component struct {
	// name is the service name, e.g. "op-batcher". The snippet is written to
	// <snippets-dir>/generated/<name>-flags.mdx and the release tag is looked
	// up under this key in manifest.json.
	name string
	// flags is the composed flag slice the binary registers.
	flags []cli.Flag
	// required lists the flag names the service's CheckRequired enforces.
	// urfave/cli does not expose this (the components keep their required
	// slices unexported), so it is mirrored here; a sanity check below
	// verifies every listed name exists in the flag slice.
	required []string
}

var components = []component{
	{
		name:  "op-batcher",
		flags: batcherflags.Flags,
		// Mirrors requiredFlags in op-batcher/flags/flags.go.
		required: []string{"l1-eth-rpc", "l2-eth-rpc", "rollup-rpc"},
	},
	{
		name:  "op-node",
		flags: nodeflags.Flags,
		// Mirrors requiredFlags in op-node/flags/flags.go. op-node additionally
		// requires exactly one of --network / --rollup.config at startup
		// (opflags.CheckRequiredXor); that pair is kept in the optional table
		// and called out in the reference page's prose instead.
		required: []string{"l1", "l2", "l2.jwt-secret"},
	},
	{
		name:  "op-proposer",
		flags: proposerflags.Flags,
		// Mirrors requiredFlags in op-proposer/flags/flags.go.
		required: []string{"l1-eth-rpc"},
	},
	{
		name:  "op-challenger",
		flags: challengerflags.Flags,
		// Mirrors requiredFlags in op-challenger/flags/flags.go, plus
		// l2-eth-rpc, which CheckRequired enforces unconditionally even though
		// the component keeps it in its optional slice. CheckRequired enforces
		// further flags conditionally (per game type, and one of
		// network / game-factory-address); the unconditional set is listed
		// here and the conditional rules are described in the reference
		// page's prose.
		required: []string{"l1-eth-rpc", "datadir", "l1-beacon", "l2-eth-rpc"},
	},
	{
		name:  "op-conductor",
		flags: conductorflags.Flags,
		// Mirrors requiredFlags in op-conductor/flags/flags.go.
		required: []string{
			"consensus.addr",
			"consensus.port",
			"raft.server.id",
			"raft.storage.dir",
			"node.rpc",
			"execution.rpc",
			"healthcheck.interval",
			"healthcheck.unsafe-interval",
			"healthcheck.min-peer-count",
		},
	},
}

// manifest maps component name -> provenance of its committed snippet: the
// finalized release tag recorded in the provenance line (e.g.
// "op-batcher/v1.16.11") and the SHA-256 of the snippet bytes written at
// regeneration time. The tag is bumped, and the snippet + hash regenerated,
// when a new finalized (non-rc) component release is published.
//
// The hash is what -check verifies: the snippet documents the release tag,
// not the current tree, and the flag-defining sources on develop may
// legitimately move past the tag between releases — so byte-comparing the
// committed snippet against a regeneration from an arbitrary tree is only
// meaningful at (or at parity with) the tag itself. The hash pin makes
// hand-edit detection portable to every commit.
type manifest map[string]*manifestEntry

type manifestEntry struct {
	Tag    string `json:"tag"`
	Sha256 string `json:"sha256,omitempty"`
}

type row struct {
	name        string
	usage       string
	defaultText string
	envVar      string
}

func main() {
	docsDir := flag.String("docs-dir", "docs/public-docs", "path to the docs root (run from the monorepo root)")
	check := flag.Bool("check", false, "verify committed snippets against the sha256 recorded in manifest.json instead of writing; exit nonzero on any mismatch (current-tree parity is reported informationally)")
	only := flag.String("only", "", "comma-separated component names to process (default: all); used when generating a single component's snippet at its release tag")
	flag.Parse()

	selected := make(map[string]bool)
	if *only != "" {
		for _, name := range strings.Split(*only, ",") {
			selected[strings.TrimSpace(name)] = true
		}
		known := make(map[string]bool, len(components))
		for _, c := range components {
			known[c.name] = true
		}
		for name := range selected {
			if !known[name] {
				fatalf("unknown component %q in -only (known: op-batcher, op-node, op-proposer, op-challenger, op-conductor)", name)
			}
		}
	}

	docsRoot, err := filepath.Abs(filepath.Clean(*docsDir))
	if err != nil {
		fatalf("resolving docs dir %s: %v", *docsDir, err)
	}

	manifestPath, err := pathUnder(docsRoot, "scripts", "gen-flags", "manifest.json")
	if err != nil {
		fatalf("%v", err)
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		fatalf("reading %s (run from the monorepo root): %v", manifestPath, err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		fatalf("parsing %s: %v", manifestPath, err)
	}

	outDir, err := pathUnder(docsRoot, "snippets", "generated")
	if err != nil {
		fatalf("%v", err)
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		fatalf("creating %s: %v", outDir, err)
	}

	manifestDirty := false
	for _, c := range components {
		if len(selected) > 0 && !selected[c.name] {
			continue
		}
		entry, ok := m[c.name]
		if !ok || entry.Tag == "" {
			fatalf("no release tag for %q in %s", c.name, manifestPath)
		}
		outPath, err := pathUnder(docsRoot, "snippets", "generated", c.name+"-flags.mdx")
		if err != nil {
			fatalf("%v", err)
		}
		content, err := render(c, entry.Tag)
		if err != nil {
			fatalf("rendering %s: %v", c.name, err)
		}
		if *check {
			committed, err := os.ReadFile(outPath)
			if err != nil {
				fatalf("reading committed snippet %s: %v", outPath, err)
			}
			if entry.Sha256 == "" {
				fatalf("no sha256 for %q in %s; regenerate per README.md", c.name, manifestPath)
			}
			if got := hashOf(committed); got != entry.Sha256 {
				fatalf("%s does not match the sha256 recorded in manifest.json for %s (tag %s): the snippet was hand-edited or the manifest was not updated with it; regenerate per README.md", outPath, c.name, entry.Tag)
			}
			if string(committed) == content {
				fmt.Printf("ok: %s matches manifest sha256 and current-tree output (%s)\n", outPath, entry.Tag)
			} else {
				// Not a failure: the snippet documents the release tag, and
				// the flag-defining sources have moved past it on this tree.
				// The snippet is refreshed when the next finalized release is
				// published.
				fmt.Printf("ok: %s matches manifest sha256 (%s); note: this tree's flag definitions have moved past the tag, snippet stays pinned to the release\n", outPath, entry.Tag)
			}
			continue
		}
		if err := os.WriteFile(outPath, []byte(content), 0o640); err != nil {
			fatalf("writing %s: %v", outPath, err)
		}
		entry.Sha256 = hashOf([]byte(content))
		manifestDirty = true
		fmt.Printf("wrote %s (%d flags)\n", outPath, len(c.flags))
	}

	if manifestDirty {
		out, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			fatalf("encoding %s: %v", manifestPath, err)
		}
		if err := os.WriteFile(manifestPath, append(out, '\n'), 0o640); err != nil {
			fatalf("writing %s: %v", manifestPath, err)
		}
	}
}

func hashOf(b []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

func render(c component, tag string) (string, error) {
	requiredSet := make(map[string]bool, len(c.required))
	for _, name := range c.required {
		requiredSet[name] = true
	}

	var required, optional []row
	seen := make(map[string]bool)
	for _, f := range c.flags {
		if vf, ok := f.(cli.VisibleFlag); ok && !vf.IsVisible() {
			continue
		}
		df, ok := f.(cli.DocGenerationFlag)
		if !ok {
			return "", fmt.Errorf("flag %v does not implement cli.DocGenerationFlag", f.Names())
		}
		name := f.Names()[0]
		if seen[name] {
			return "", fmt.Errorf("duplicate flag name %q", name)
		}
		seen[name] = true
		r := row{
			name:        name,
			usage:       canonicalizeUsage(c.name, name, df.GetUsage()),
			defaultText: df.GetDefaultText(),
		}
		if envVars := df.GetEnvVars(); len(envVars) > 0 {
			r.envVar = envVars[0]
		}
		if requiredSet[name] {
			required = append(required, r)
		} else {
			optional = append(optional, r)
		}
	}

	// Sanity check: every declared required flag must exist in the slice, so
	// a rename in the component surfaces here instead of drifting silently.
	for _, name := range c.required {
		if !seen[name] {
			return "", fmt.Errorf("required flag %q not found in %s flag definitions", name, c.name)
		}
	}

	sort.Slice(required, func(i, j int) bool { return required[i].name < required[j].name })
	sort.Slice(optional, func(i, j int) bool { return optional[i].name < optional[j].name })

	var b strings.Builder
	fmt.Fprintf(&b, `{/*
  GENERATED FILE — DO NOT EDIT.

  Generated by docs/public-docs/scripts/gen-flags from the %[1]s urfave/cli
  flag definitions at the release tag below. Do not hand-edit; regenerate
  per docs/public-docs/scripts/gen-flags/README.md when a new finalized
  %[1]s release is published:

    go run ./docs/public-docs/scripts/gen-flags
*/}

Generated from [`+"`%[2]s`"+`](https://github.com/ethereum-optimism/optimism/releases/tag/%[3]s)
flag definitions. %[4]d flags: %[5]d required, %[6]d optional.

`, c.name, tag, strings.ReplaceAll(tag, "/", "%2F"), len(required)+len(optional), len(required), len(optional))

	b.WriteString("### Required flags\n\n")
	b.WriteString("| Flag | Description | Environment variable |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, r := range required {
		fmt.Fprintf(&b, "| `--%s` | %s | %s |\n", r.name, cell(r.usage), envCell(r.envVar))
	}

	b.WriteString("\n### Optional flags\n\n")
	b.WriteString("| Flag | Description | Default | Environment variable |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, r := range optional {
		fmt.Fprintf(&b, "| `--%s` | %s | %s | %s |\n", r.name, cell(r.usage), defaultCell(r.defaultText), envCell(r.envVar))
	}

	return b.String(), nil
}

// canonicalizeUsage makes usage strings that are unstable at the source
// stable across runs and registry bumps, so regeneration is byte-stable and
// keyed only to component releases.
//
//   - op-batcher builds the --compressor option list from map iteration
//     (op-batcher/compressor.KindKeys via slices.Collect(maps.Keys(...))), so
//     its order changes on every process start — including in
//     `op-batcher --help` itself. Sort that one list here. Remove this once
//     KindKeys is sorted upstream (proposed as a follow-up to the B1 slice).
//   - The --network usage string enumerates chaincfg.AvailableNetworks(),
//     which is read from the superchain-registry bundle
//     (op-core/superchain/superchain-configs.zip) — it changes with every
//     registry submodule bump, independent of any component release, and
//     would otherwise make the snippet stale (and -check red) by
//     construction. Replace the enumerated list with a stable pointer at the
//     registry; `<component> --help` always shows the bundled list.
func canonicalizeUsage(component, flagName, usage string) string {
	if component == "op-batcher" && flagName == "compressor" {
		const marker = "Valid options: "
		if i := strings.Index(usage, marker); i >= 0 {
			head, list := usage[:i+len(marker)], usage[i+len(marker):]
			options := strings.Split(list, ", ")
			sort.Strings(options)
			return head + strings.Join(options, ", ")
		}
	}
	if flagName == "network" {
		const marker = "Available networks: "
		if i := strings.Index(usage, marker); i >= 0 {
			return usage[:i+len(marker)] + fmt.Sprintf(
				"every chain bundled from the [superchain-registry](https://github.com/ethereum-optimism/superchain-registry) at the release; run `%s --help` for the exact list", component)
		}
	}
	return usage
}

// cell escapes a usage string for use inside an MDX table cell: newlines
// become spaces, and characters MDX would treat as JSX or expression syntax
// ({, }, <) or as a column separator (|) are escaped.
func cell(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	replacer := strings.NewReplacer(
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		"<", "\\<",
	)
	return replacer.Replace(s)
}

func defaultCell(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + strings.ReplaceAll(s, "|", "\\|") + "`"
}

func envCell(s string) string {
	if s == "" {
		return "—"
	}
	return "`" + s + "`"
}

// pathUnder joins elems onto root, cleans the result, and verifies the
// resolved path still lies under root. Every path this tool reads or writes
// is built from in-repo constants (component names from the components table,
// manifest.json), not user input, but the check keeps a future bad component
// name or docs-dir value from escaping the docs tree.
func pathUnder(root string, elems ...string) (string, error) {
	p := filepath.Clean(filepath.Join(append([]string{root}, elems...)...))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s escapes docs dir %s", p, root)
	}
	return p, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen-flags: "+format+"\n", args...)
	os.Exit(1)
}
