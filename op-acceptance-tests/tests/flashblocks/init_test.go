package flashblocks

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum/go-ethereum/common"
)

// TestMain creates the test-setups against the shared backend.
// If FLASHBLOCKS_RULES_TEST is set, it enables rule-based ordering for tests.
func TestMain(m *testing.M) {
	// Check if rules testing is enabled
	if os.Getenv("FLASHBLOCKS_RULES_TEST") != "" {
		runWithRules(m)
	} else {
		// Default: run without rules (original behavior)
		presets.DoMain(m, presets.WithSingleChainSystemWithFlashblocks())
	}
}

func runWithRules(m *testing.M) {
	// Create a fixed directory for rules config (reused across runs)
	rulesDir := filepath.Join(os.TempDir(), "flashblocks-test-rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create rules dir: %v\n", err)
		os.Exit(1)
	}

	// Write rules file
	rulesPath := filepath.Join(rulesDir, "test_rules.yaml")
	if err := os.WriteFile(rulesPath, []byte(TestRulesYAML), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write rules file: %v\n", err)
		os.Exit(1)
	}

	// Write registry config pointing to rules file
	registryConfig := fmt.Sprintf(`file:
  - path: %s
    name: "Test Rules"
    enabled: true

refresh_interval: 5
`, rulesPath)

	registryPath := filepath.Join(rulesDir, "test_registry.yaml")
	if err := os.WriteFile(registryPath, []byte(registryConfig), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write registry file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Rules config created at: %s\n", registryPath)

	// Set up orchestrator with rules enabled
	rulesConfig := sysgo.RulesConfig{
		Enabled:    true,
		ConfigPath: registryPath,
	}

	presets.DoMain(m, presets.WithSingleChainSystemWithFlashblocksAndRules(rulesConfig))
}

// BoostedRecipient is the well-known address that receives boosted transactions in tests.
// Transactions sent TO this address will be prioritized by the block builder when rules are enabled.
var BoostedRecipient = common.HexToAddress("0x1111111111111111111111111111111111111111")

// HighPriorityRecipient receives transactions with the highest boost (weight: 5000)
var HighPriorityRecipient = common.HexToAddress("0x2222222222222222222222222222222222222222")

// MediumPriorityRecipient receives transactions with medium boost (weight: 2000)
var MediumPriorityRecipient = common.HexToAddress("0x3333333333333333333333333333333333333333")

// LowPriorityRecipient receives transactions with low boost (weight: 500)
var LowPriorityRecipient = common.HexToAddress("0x4444444444444444444444444444444444444444")

// TestRulesYAML is the rules configuration used for rule ordering tests.
// It defines multiple boost levels to test priority ordering:
// - High priority (weight 5000): transactions TO 0x2222...
// - Medium priority (weight 2000): transactions TO 0x3333...
// - Low priority (weight 500): transactions TO 0x4444...
// - Legacy boost (weight 1000): transactions TO 0x1111... (BoostedRecipient)
const TestRulesYAML = `version: 1

aliases:
  high_priority_recipients:
    - "0x2222222222222222222222222222222222222222"
  medium_priority_recipients:
    - "0x3333333333333333333333333333333333333333"
  low_priority_recipients:
    - "0x4444444444444444444444444444444444444444"
  boosted_recipients:
    - "0x1111111111111111111111111111111111111111"

rules:
  boost:
    - name: "High Priority Boost"
      description: "Highest priority transactions"
      type: to
      aliases:
        - "high_priority_recipients"
      weight: 5000
    - name: "Medium Priority Boost"
      description: "Medium priority transactions"
      type: to
      aliases:
        - "medium_priority_recipients"
      weight: 2000
    - name: "Low Priority Boost"
      description: "Low priority transactions"
      type: to
      aliases:
        - "low_priority_recipients"
      weight: 500
    - name: "Legacy Boosted Recipient"
      description: "Boost transactions to test recipient address"
      type: to
      aliases:
        - "boosted_recipients"
      weight: 1000
`
