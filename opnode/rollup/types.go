package rollup

import "github.com/ethereum-optimism/optimistic-specs/opnode/eth"

type Genesis struct {
	L1 eth.BlockID
	L2 eth.BlockID
}
