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
	"github.com/ethereum/go-ethereum/trie"
)

func main() {
	bc := core.NewBlockChain()
	statedb := &state.StateDB{}
	vmconfig := vm.Config{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, bc.Engine())
	fmt.Println("made state processor")

	// read header
	var header types.Header
	{
		f, _ := os.Open("data/block_13247502")
		defer f.Close()
		rlpheader := rlp.NewStream(f, 0)
		rlpheader.Decode(&header)
	}

	// read txs
	var txs []*types.Transaction
	{
		f, _ := os.Open("data/txs_13247502")
		defer f.Close()
		rlpheader := rlp.NewStream(f, 0)
		rlpheader.Decode(&txs)
	}
	fmt.Println("read", len(txs), "transactions")

	var uncles []*types.Header
	var receipts []*types.Receipt
	block := types.NewBlock(&header, txs, uncles, receipts, trie.NewStackTrie(nil))
	fmt.Println("made block, parent:", header.ParentHash)

	// if this is correct, the trie is working
	if header.TxHash != block.Header().TxHash {
		panic("wrong transactions for block")
	}

	_, _, _, err := processor.Process(block, statedb, vmconfig)
	fmt.Println(err)
	fmt.Println("process done")
}
