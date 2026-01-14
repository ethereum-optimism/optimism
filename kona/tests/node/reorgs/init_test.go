package reorgs

import (
	"fmt"
	"testing"

	node_utils "github.com/ethereum-optimism/optimism/kona/tests/node/utils"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	l2Config := node_utils.ParseL2NodeConfigFromEnv()

	fmt.Printf("Running e2e reorg tests with Config: %d\n", l2Config)

	presets.DoMain(m, node_utils.WithMixedWithTestSequencer(l2Config))
}
