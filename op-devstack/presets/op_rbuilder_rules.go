package presets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func WithOPRBuilderRules(ruleContent string, refreshInterval uint64) stack.CommonOption {
	return stack.MakeCommon(
		sysgo.WithGlobalOPRBuilderNodeOption(sysgo.OPRBuilderNodeOptionFn(
			func(p devtest.P, id stack.ComponentID, cfg *sysgo.OPRBuilderNodeConfig) {
				cfg.RulesEnabled = true
				// Create a fixed directory for rules config
				rulesDir := filepath.Join(os.TempDir(), "rules")
				if err := os.MkdirAll(rulesDir, 0755); err != nil {
					p.Errorf("Failed to create rules dir: %v", err)
				}
				// Write rules
				rulesPath := filepath.Join(rulesDir, "ruleset.yaml")
				if err := os.WriteFile(rulesPath, []byte(ruleContent), 0644); err != nil {
					p.Errorf("Failed to create rules dir: %v", err)
				}
				// Write rule config pointing to rules file
				rulesConfigContent := fmt.Sprintf(`
file:
  - path: %s
    name: "Test Rules"
    enabled: true

refresh_interval: %d
`, rulesPath, refreshInterval)
				rulesConfigPath := filepath.Join(rulesDir, "rules_config.yaml")
				if err := os.WriteFile(rulesConfigPath, []byte(rulesConfigContent), 0644); err != nil {
					p.Errorf("Failed to write registry file: %v", err)
				}
				cfg.RulesConfigPath = rulesConfigPath
			})))
}
