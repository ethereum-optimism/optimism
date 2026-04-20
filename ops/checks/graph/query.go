package graph

import "strings"

// CheckPrerequisites returns the transitive set of check IDs that
// must run before checkID, derived from the dataflow graph: for every
// artifact this check consumes, every check that produces the artifact
// is a prerequisite (transitively).
//
// Returns IDs WITHOUT the "check:" prefix, in deterministic
// lexicographic order. Determinism is load-bearing:
// ExecutionItem.Prerequisites feeds the scheduler and CI log output,
// both of which require stable ordering across runs.
//
// checkID must include the "check:" prefix.
func CheckPrerequisites(g *Graph, checkID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	producers := make(map[string]bool)
	queue := []string{checkID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, outEdge := range g.outgoing[cur] {
			if outEdge.Kind != EdgeConsumes {
				continue
			}
			for _, inEdge := range g.incoming[outEdge.To] {
				if inEdge.Kind != EdgeProduces {
					continue
				}
				if inEdge.From == checkID {
					continue
				}
				if producers[inEdge.From] {
					continue
				}
				producers[inEdge.From] = true
				queue = append(queue, inEdge.From)
			}
		}
	}
	out := make([]string, 0, len(producers))
	for id := range producers {
		out = append(out, strings.TrimPrefix(id, "check:"))
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
