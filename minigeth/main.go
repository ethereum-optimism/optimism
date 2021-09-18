package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	bc := &core.BlockChain{}
	//engine := ethash.NewFullFaker()
	statedb := &state.StateDB{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, nil)
	fmt.Println("made state processor")
	processor.Process()
}
