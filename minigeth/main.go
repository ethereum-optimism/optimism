package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	bc := &core.BlockChain{}
	//engine := ethash.NewFullFaker()
	statedb := &state.StateDB{}
	vmconfig := vm.Config{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, nil)
	fmt.Println("made state processor")

	f, _ := os.Open("../data/block_block_13247502")
	defer f.Close()

	block := &types.Block{}
	block.DecodeRLP(rlp.NewStream(f, 0))
	fmt.Println("read block RLP")

	processor.Process(block, statedb, vmconfig)
}
