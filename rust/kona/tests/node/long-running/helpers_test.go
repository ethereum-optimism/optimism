package node

import (
	"flag"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

var (
	num_threads           = flag.Int("num-threads", 10, "number of threads to use for the test")
	percentageNewAccounts = flag.Int("percentage-new-accounts", 20, "percentage of new accounts to produce transactions for")
	fundAmount            = flag.Int("fund-amount", 10, "eth amount to fund each new account with")
	initNumAccounts       = flag.Int("init-num-accounts", 10, "initial number of accounts to fund")
)

func newLongRunningPreset(t devtest.T) *node_utils.MixedOpKonaPreset {
	return node_utils.NewMixedOpKonaForConfig(t, node_utils.L2NodeConfig{
		OpSequencerNodesWithGeth:   0,
		OpSequencerNodesWithReth:   0,
		KonaSequencerNodesWithGeth: 1,
		KonaSequencerNodesWithReth: 0,
		OpNodesWithGeth:            1,
		OpNodesWithReth:            1,
		KonaNodesWithGeth:          1,
		KonaNodesWithReth:          1,
	})
}
