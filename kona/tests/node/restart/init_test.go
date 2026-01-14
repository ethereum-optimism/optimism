package node_restart

import (
	"fmt"
	"testing"

	node_utils "github.com/ethereum-optimism/optimism/kona/tests/node/utils"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	// Currently, the restart tests only support kona nodes. The op node based configs are not supported (because of req-resp sync incompatibility).
	config := node_utils.L2NodeConfig{
		KonaSequencerNodesWithGeth: 1,
		KonaNodesWithGeth:          1,
	}

	fmt.Printf("Running restart e2e tests with Config: %d\n", config)
	presets.DoMain(m, node_utils.WithMixedOpKona(config))
}
