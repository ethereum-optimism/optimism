package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type DumbEngine struct {
	/*Author
	Finalize*/
}

func (ethash *DumbEngine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func main() {
	bc := core.NewBlockChain()
	statedb := &state.StateDB{}
	vmconfig := vm.Config{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, nil)
	fmt.Println("made state processor")

	f, _ := os.Open("../data/block_13247502")
	defer f.Close()
	var header types.Header
	rlpheader := rlp.NewStream(f, 0)
	rlpheader.Decode(&header)
	block := types.NewBlockWithHeader(&header)
	fmt.Println("made block, parent:", header.ParentHash)

	processor.Process(block, statedb, vmconfig)
}
