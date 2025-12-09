package node

import (
	"fmt"
	"testing"

	node_utils "github.com/ethereum-optimism/optimism/kona/tests/node/utils"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	config := node_utils.ParseL2NodeConfigFromEnv()

	fmt.Printf("Running e2e tests with Config: %d\n", config)
	presets.DoMain(m, node_utils.WithMixedOpKona(config))
}
