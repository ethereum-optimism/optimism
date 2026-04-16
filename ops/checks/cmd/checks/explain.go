package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/diff"
	"github.com/ethereum-optimism/optimism/ops/checks/freshness"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
	"github.com/ethereum-optimism/optimism/ops/checks/selector"
)

func cmdExplain(args []string) error {
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	policyPath := fs.String("policy", "", "optional policy override YAML")
	fs.Parse(args)

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: checks explain [--graph FILE] [--catalog FILE] FILE_PATH")
	}

	filePath := fs.Arg(0)

	g, err := graph.Load(*graphPath)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}
	cat, err := catalog.Load(*catalogPath)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}
	pol, err := loadPolicy(*policyPath)
	if err != nil {
		return err
	}
	fresh := freshness.New(findRepoRoot(), pol)

	// Raw graph view — direct edges and reachable checks.
	nodeIDs, unknown := diff.FilesToNodeIDs(g, []string{filePath})
	if len(unknown) > 0 {
		fmt.Printf("File %q did not match any graph node.\n", filePath)
		return nil
	}

	fmt.Printf("File: %s\n", filePath)
	for _, nodeID := range nodeIDs {
		fmt.Printf("Node: %s\n\n", nodeID)

		edges := g.EdgesFrom(nodeID)
		if len(edges) > 0 {
			fmt.Println("Direct outgoing edges:")
			for _, e := range edges {
				fr := fresh.Assess(e)
				freshPart := ""
				if fr < 1.0 {
					freshPart = fmt.Sprintf(", freshness=%.2f", fr)
				}
				fmt.Printf("  → %s (%s/%s, strength=%.2f, confidence=%.2f%s)\n",
					e.To, e.Kind, e.Source, e.Strength, e.Confidence, freshPart)
			}
			fmt.Println()
		}
	}

	// Candidate-level view: run Phase 1 against a synthetic file-level
	// diff and print the resulting candidates with full provenance. This
	// is what the selector would actually reason about.
	diffs := []diff.FileDiff{{Path: filePath}}
	candidates := selector.Resolve(g, diffs, cat, pol, fresh)
	if len(candidates) == 0 {
		fmt.Println("No candidates produced for this file.")
		return nil
	}

	fmt.Printf("Candidates (%d):\n", len(candidates))
	for _, c := range candidates {
		scope := c.Scope
		if scope == "" {
			scope = "(unscoped)"
		}
		profile := c.Profile
		if profile == "" {
			profile = "(default)"
		}
		fmt.Printf("  %s  scope=%s  profile=%s  signal=%.3f\n",
			c.CheckID, scope, profile, c.Signal)
		for _, p := range c.Provenance {
			raw := formatRaw(p.Raw)
			fmt.Printf("    ← %s/%s  contribution=%.3f  %s\n",
				p.Source, p.EdgeKind, p.Contribution, raw)
		}
	}

	return nil
}

func formatRaw(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}
	var parts []string
	for k, v := range raw {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
