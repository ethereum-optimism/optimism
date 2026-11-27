package parityreport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Load reads every record in the *.jsonl files of dir.
func Load(dir string) ([]Record, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var records []Record
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(f)
		for line := 1; scanner.Scan(); line++ {
			var r Record
			if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
				f.Close()
				return nil, fmt.Errorf("%s:%d: %w", path, line, err)
			}
			records = append(records, r)
		}
		err = scanner.Err()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return records, nil
}

var layerOrder = map[string]int{LayerVectors: 0, LayerProof: 1, LayerNode: 2}

// Markdown renders records as a report: case counts per layer, each distinct difference with the
// cases that show it, and every case where the clients disagree.
func Markdown(records []Record) string {
	records = append([]Record(nil), records...)
	sort.SliceStable(records, func(i, j int) bool {
		a, b := records[i], records[j]
		if a.Layer != b.Layer {
			return layerRank(a.Layer) < layerRank(b.Layer)
		}
		if a.Group != b.Group {
			return a.Group < b.Group
		}
		if a.Case != b.Case {
			return a.Case < b.Case
		}
		return a.Forks < b.Forks
	})

	var sb strings.Builder
	sb.WriteString("# Derivation parity: kona vs op-node\n\n")
	writeSummary(&sb, records)
	writeDifferences(&sb, records)
	return sb.String()
}

func layerRank(layer string) int {
	if rank, ok := layerOrder[layer]; ok {
		return rank
	}
	return len(layerOrder)
}

// writeSummary counts cases per layer and group. records must be sorted by layer and group.
func writeSummary(sb *strings.Builder, records []Record) {
	type counts struct {
		layer, group   string
		cases, diverge int
	}
	var groups []*counts
	var total counts
	for _, r := range records {
		if len(groups) == 0 || groups[len(groups)-1].layer != r.Layer || groups[len(groups)-1].group != r.Group {
			groups = append(groups, &counts{layer: r.Layer, group: r.Group})
		}
		c := groups[len(groups)-1]
		c.cases++
		total.cases++
		if !r.Agree {
			c.diverge++
			total.diverge++
		}
	}
	sb.WriteString("| Layer | Group | Cases | Agree | Diverge |\n|---|---|---:|---:|---:|\n")
	for _, c := range groups {
		fmt.Fprintf(sb, "| %s | %s | %d | %d | %d |\n", c.layer, escapeCell(c.group), c.cases, c.cases-c.diverge, c.diverge)
	}
	fmt.Fprintf(sb, "| **Total** | | **%d** | **%d** | **%d** |\n\n", total.cases, total.cases-total.diverge, total.diverge)
}

func writeDifferences(sb *strings.Builder, records []Record) {
	var differences []string
	casesByDifference := map[string][]Record{}
	for _, r := range records {
		if r.Agree {
			continue
		}
		if _, ok := casesByDifference[r.Divergence]; !ok {
			differences = append(differences, r.Divergence)
		}
		casesByDifference[r.Divergence] = append(casesByDifference[r.Divergence], r)
	}
	if len(differences) == 0 {
		sb.WriteString("kona and op-node agree on every case.\n")
		return
	}

	fmt.Fprintf(sb, "## %d differences\n\n", len(differences))
	sb.WriteString("| # | Difference | Cases | Layers |\n|---:|---|---:|---|\n")
	for i, difference := range differences {
		cases := casesByDifference[difference]
		var layers []string
		seen := map[string]bool{}
		for _, r := range cases {
			if !seen[r.Layer] {
				seen[r.Layer] = true
				layers = append(layers, r.Layer)
			}
		}
		fmt.Fprintf(sb, "| %d | %s | %d | %s |\n", i+1, escapeCell(difference), len(cases), strings.Join(layers, ", "))
	}

	sb.WriteString("\n## Diverging cases\n\n| # | Layer | Group | Case | Forks |\n|---:|---|---|---|---|\n")
	for i, difference := range differences {
		for _, r := range casesByDifference[difference] {
			fmt.Fprintf(sb, "| %d | %s | %s | %s | %s |\n", i+1, r.Layer, escapeCell(r.Group), escapeCell(r.Case), escapeCell(r.Forks))
		}
	}
}

func escapeCell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
