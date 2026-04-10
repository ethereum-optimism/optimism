package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/components"
)

const optimismModuleLine = "module github.com/ethereum-optimism/optimism"

func defaultMonorepoPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return filepath.Clean(wd)
	}
	return filepath.Clean(abs)
}

func ValidateMonorepoRoot(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("oprm must be run from the optimism monorepo root")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve monorepo path %q: %w", path, err)
	}
	goModPath := filepath.Join(abs, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("oprm must be run from the optimism monorepo root; could not read %s: %w", goModPath, err)
	}
	if !strings.Contains(string(data), optimismModuleLine) {
		return fmt.Errorf("oprm must be run from the optimism monorepo root; %s does not declare %q", goModPath, optimismModuleLine)
	}
	rootMarkers := []string{"justfile", "op-node"}
	for _, marker := range rootMarkers {
		markerPath := filepath.Join(abs, marker)
		if _, err := os.Stat(markerPath); err != nil {
			return fmt.Errorf("oprm must be run from the optimism monorepo root; missing %s: %w", markerPath, err)
		}
	}
	return nil
}

func (a *App) MonorepoPath() string {
	if a != nil && a.Config != nil && strings.TrimSpace(a.Config.MonorepoPath) != "" {
		return filepath.Clean(a.Config.MonorepoPath)
	}
	return defaultMonorepoPath()
}

func (a *App) OpGethPath() string {
	checkout := ""
	if a != nil && a.Config != nil {
		checkout = strings.TrimSpace(a.Config.OpGeth.CheckoutPath)
	}
	if checkout == "" {
		checkout = "../op-geth"
	}
	if filepath.IsAbs(checkout) {
		return filepath.Clean(checkout)
	}
	base := a.MonorepoPath()
	if base == "" {
		return filepath.Clean(checkout)
	}
	return filepath.Clean(filepath.Join(base, checkout))
}

func (a *App) checkoutPath(componentID string) (string, error) {
	spec, err := a.componentSpec(componentID)
	if err != nil {
		return "", err
	}
	return a.checkoutPathForSpec(spec), nil
}

func (a *App) checkoutPathForSpec(spec components.ComponentSpec) string {
	if spec.Kind == components.KindExternalGo {
		return a.OpGethPath()
	}
	return a.MonorepoPath()
}
