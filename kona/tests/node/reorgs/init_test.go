package reorgs

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	node_utils "github.com/op-rs/kona/node/utils"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	l2Config := node_utils.ParseL2NodeConfigFromEnv()

	fmt.Printf("Running e2e reorg tests with Config: %d\n", l2Config)

	presets.DoMain(m, node_utils.WithMixedWithTestSequencer(l2Config))
}
