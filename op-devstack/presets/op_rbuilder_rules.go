package presets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func WithOPRBuilderRules(ruleContent string, refreshInterval uint64) Option {
	return WithOPRBuilderOption(sysgo.OPRBuilderNodeOptionFn(
		func(p devtest.CommonT, _ sysgo.ComponentTarget, cfg *sysgo.OPRBuilderNodeConfig) {
			cfg.RulesEnabled = true
			rulesDir := filepath.Join(os.TempDir(), "rules")
			if err := os.MkdirAll(rulesDir, 0o755); err != nil {
				p.Errorf("failed to create rules dir: %v", err)
			}
			rulesPath := filepath.Join(rulesDir, "ruleset.yaml")
			if err := os.WriteFile(rulesPath, []byte(ruleContent), 0o644); err != nil {
				p.Errorf("failed to write rules file: %v", err)
			}
			rulesConfigContent := fmt.Sprintf(`
file:
  - path: %s
    name: "Test Rules"
    enabled: true

refresh_interval: %d
`, rulesPath, refreshInterval)
			rulesConfigPath := filepath.Join(rulesDir, "rules_config.yaml")
			if err := os.WriteFile(rulesConfigPath, []byte(rulesConfigContent), 0o644); err != nil {
				p.Errorf("failed to write rules config file: %v", err)
			}
			cfg.RulesConfigPath = rulesConfigPath
		}))
}
