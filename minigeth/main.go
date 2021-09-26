package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/oracle"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func main() {
	// init secp256k1BytePoints
	crypto.S256()

	// before this isn't run on chain (confirm this isn't cached)
	os.Stderr.WriteString("********* on chain starts here *********\n")

	blockNumber, _ := strconv.Atoi(os.Args[1])

	// non mips
	oracle.PrefetchBlock(big.NewInt(int64(blockNumber)), true, trie.NewStackTrie(nil))
	oracle.PrefetchBlock(big.NewInt(int64(blockNumber)+1), false, trie.NewStackTrie(nil))

	// read start block header
	var parent types.Header
	rlperr := rlp.DecodeBytes(oracle.Preimage(oracle.Input(0)), &parent)
	if rlperr != nil {
		log.Fatal(rlperr)
	}

	// read header
	var newheader types.Header
	// from parent
	newheader.ParentHash = parent.Hash()
	newheader.Number = big.NewInt(0).Add(parent.Number, big.NewInt(1))
	newheader.BaseFee = misc.CalcBaseFee(params.MainnetChainConfig, &parent)

	// from input oracle
	newheader.TxHash = oracle.Input(1)
	newheader.Coinbase = common.BigToAddress(oracle.Input(2).Big())
	newheader.UncleHash = oracle.Input(3)
	newheader.GasLimit = oracle.Input(4).Big().Uint64()

	bc := core.NewBlockChain()
	database := state.NewDatabase(parent)
	statedb, _ := state.New(parent.Root, database, nil)
	vmconfig := vm.Config{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, bc.Engine())
	fmt.Println("made state processor")

	// read txs
	var txs []*types.Transaction
	{
		f, _ := os.Open(fmt.Sprintf("data/txs_%d", blockNumber+1))
		rlpheader := rlp.NewStream(f, 0)
		rlpheader.Decode(&txs)
		f.Close()
	}
	fmt.Println("read", len(txs), "transactions")

	var uncles []*types.Header
	var receipts []*types.Receipt
	block := types.NewBlock(&newheader, txs, uncles, receipts, trie.NewStackTrie(nil))
	fmt.Println("made block, parent:", newheader.ParentHash)

	// if this is correct, the trie is working
	// TODO: it's the previous block now
	if newheader.TxHash != block.Header().TxHash {
		panic("wrong transactions for block")
	}
	if newheader.UncleHash != block.Header().UncleHash {
		panic("wrong uncles for block")
	}

	_, _, _, err := processor.Process(block, statedb, vmconfig)
	if err != nil {
		log.Fatal(err)
	}
	newroot := statedb.IntermediateRoot(bc.Config().IsEIP158(newheader.Number))

	fmt.Println("process done with hash", parent.Root, "->", newroot)
	oracle.Output(newroot)
}
