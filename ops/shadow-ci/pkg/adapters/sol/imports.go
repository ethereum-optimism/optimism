package sol

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var importRegex = regexp.MustCompile(`import\s+(?:{[^}]+}\s+from\s+)?["']([^"']+)["']`)

// Remappings maps import prefixes to filesystem paths.
type Remappings map[string]string

// ParseRemappings reads a remappings.txt file into a Remappings map.
func ParseRemappings(path string) (Remappings, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Remappings{}, nil
		}
		return nil, err
	}
	defer f.Close()

	m := make(Remappings)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: context:prefix=target or prefix=target
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := parts[0]
		// Strip context if present (e.g., "ds-test/:forge-std/=lib/forge-std/src/").
		if idx := strings.LastIndex(prefix, ":"); idx >= 0 {
			prefix = prefix[idx+1:]
		}
		m[prefix] = parts[1]
	}
	return m, scanner.Err()
}

// ResolveImport resolves an import path using remappings.
func (r Remappings) ResolveImport(importPath string) string {
	for prefix, target := range r {
		if strings.HasPrefix(importPath, prefix) {
			return target + importPath[len(prefix):]
		}
	}
	return importPath
}

// ParseImports extracts Solidity import paths from a .sol file.
func ParseImports(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := importRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			imports = append(imports, matches[1])
		}
	}
	return imports, scanner.Err()
}

// CollectSolFiles walks directories and returns all .sol file paths relative to root.
func CollectSolFiles(root string, dirs []string) ([]string, error) {
	var files []string
	for _, dir := range dirs {
		absDir := filepath.Join(root, dir)
		err := filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".sol") {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return relErr
				}
				files = append(files, rel)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	return files, nil
}
