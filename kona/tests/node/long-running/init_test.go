package node

import (
	"flag"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	node_utils "github.com/op-rs/kona/node/utils"
)

var (
	num_threads           = flag.Int("num-threads", 10, "number of threads to use for the test")
	percentageNewAccounts = flag.Int("percentage-new-accounts", 20, "percentage of new accounts to produce transactions for")
	fundAmount            = flag.Int("fund-amount", 10, "eth amount to fund each new account with")
	initNumAccounts       = flag.Int("init-num-accounts", 10, "initial number of accounts to fund")
)

// TestMain creates the test-setups against the shared backend
func TestMain(m *testing.M) {
	flag.Parse()

	presets.DoMain(m, node_utils.WithMixedOpKona(node_utils.L2NodeConfig{
		OpSequencerNodesWithGeth:   0,
		OpSequencerNodesWithReth:   0,
		KonaSequencerNodesWithGeth: 1,
		KonaSequencerNodesWithReth: 0,
		OpNodesWithGeth:            1,
		OpNodesWithReth:            1,
		KonaNodesWithGeth:          1,
		KonaNodesWithReth:          1,
	}))
}
