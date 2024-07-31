package setup

import (
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/setup/script"
)

type Chain interface {
	Genesis() *core.Genesis
}

type Labels map[common.Address]string

func (l Labels) AddLabels(other Labels) {
	for k, v := range other {
		l[k] = v
	}
}

func (l Labels) LabelDeployments(dep map[string]common.Address) {
	for k, v := range dep {
		l[v] = k
	}
}

type chain struct {
	w *world

	log log.Logger

	chainID *uint256.Int

	// L1 specific properties
	l1 *l1Props

	// L2 specific properties
	l2 *l2Props

	state script.State

	labels Labels

	genesis *core.Genesis
}

// TODO more read funcs. Assert deployment is complete before allowing access to the data.
func (ch *chain) Genesis() *core.Genesis {
	ch.w.req.NotNil(ch.genesis, "must have genesis")
	return ch.genesis
}
