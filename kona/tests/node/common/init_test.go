package node

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	node_utils "github.com/op-rs/kona/node/utils"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	config := node_utils.ParseL2NodeConfigFromEnv()

	fmt.Printf("Running e2e tests with Config: %d\n", config)
	presets.DoMain(m, node_utils.WithMixedOpKona(config))
}
