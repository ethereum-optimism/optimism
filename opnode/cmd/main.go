package main

import (
	"github.com/ethereum-optimism/optimistic-specs/opnode/node"
	"github.com/protolambda/ask"
)

type MainCmd struct {
}

func (c *MainCmd) Help() string {
	return "Run Optimism rollup node."
}

func (c *MainCmd) Cmd(route string) (cmd interface{}, err error) {
	switch route {
	case "run":
		return &node.OpNodeCmd{}, nil
	default:
		return nil, ask.UnrecognizedErr
	}
}

// TODO: we can support additional utils etc.
func (c *MainCmd) Routes() []string {
	return []string{"run"}
}

func main() {
	ask.Run(new(MainCmd))
}
