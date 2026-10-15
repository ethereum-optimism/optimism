package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	sep               = " | "
	reasonPlaceholder = "<reason>"
)

// Allowlist is the parsed allowlist file: the fork-API uses that may remain until the cutover.
type Allowlist struct {
	lines   []string
	entries map[Use]int // index into lines
}

func readAllowlist(path string) (*Allowlist, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	a := &Allowlist{entries: make(map[Use]int)}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		a.lines = append(a.lines, line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// The need may contain "|" itself (type unions), the package and the reason can't.
		pkg, rest, ok1 := strings.Cut(trimmed, "|")
		i := strings.LastIndex(rest, "|")
		if !ok1 || i < 0 {
			return nil, fmt.Errorf("%s:%d: want \"<package>%s<need>%s<reason>\"", path, n, sep, sep)
		}
		u := Use{Package: strings.TrimSpace(pkg), Need: strings.TrimSpace(rest[:i])}
		reason := strings.TrimSpace(rest[i+1:])
		if reason == "" || reason == reasonPlaceholder {
			return nil, fmt.Errorf("%s:%d: entry for %s needs a reason", path, n, u)
		}
		if _, dup := a.entries[u]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate entry %s", path, n, u)
		}
		a.entries[u] = len(a.lines) - 1
	}
	return a, sc.Err()
}

// diff returns the uses missing from the allowlist and the entries no longer used.
func (a *Allowlist) diff(uses map[Use]bool) (unlisted, stale []Use) {
	for u := range uses {
		if _, ok := a.entries[u]; !ok {
			unlisted = append(unlisted, u)
		}
	}
	for u := range a.entries {
		if !uses[u] {
			stale = append(stale, u)
		}
	}
	sortUses(unlisted)
	sortUses(stale)
	return unlisted, stale
}

// writeWithout rewrites the allowlist to path without the given entries, keeping everything else
// as it is.
func (a *Allowlist) writeWithout(path string, remove []Use) error {
	drop := make(map[int]bool, len(remove))
	for _, u := range remove {
		drop[a.entries[u]] = true
	}
	var b strings.Builder
	for i, line := range a.lines {
		if !drop[i] {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func entryLine(u Use, reason string) string {
	return u.Package + sep + u.Need + sep + reason
}
