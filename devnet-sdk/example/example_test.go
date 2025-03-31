package example

import (
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"testing"
)

// TODO
// var orchestrator = system2.Orchestrator(nil)
func newSetup(t *testing.T) system2.Setup {

}

// orchestrator instantiation
// preset choice with options
// system is hydrated
// returns DSL view over system
// test interacts with DSL

// TODO: some func to signal that a test could have been started with different preset params

func TestExample(t *testing.T) {
	preset := newSetup(t)
	l2Net := preset.System.L2Network(query.Any)
	seq := l2Net.L2CLNode(query.ActiveSequencer)
	faucet := l2Net.Faucet()
	user := faucet.NewUser()

}
