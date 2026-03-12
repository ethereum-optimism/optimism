package rules

import (
	"os"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common"
)

const RULES_TEST_ENABLE_ENV = "OP_RBUILDER_RULES_TEST"

func rulesEnabled() bool {
	return os.Getenv(RULES_TEST_ENABLE_ENV) == "1"
}

func newRulesSystem(t devtest.T) *presets.SingleChainWithFlashblocks {
	return presets.NewSingleChainWithFlashblocks(t, presets.WithOPRBuilderRules(TestRulesYAML, TestRefreshInterval))
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

const TestRefreshInterval = 5

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
