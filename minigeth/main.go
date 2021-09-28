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

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// init secp256k1BytePoints
	crypto.S256()

	// non mips
	if len(os.Args) > 1 {
		pkw := oracle.PreimageKeyValueWriter{}
		pkwtrie := trie.NewStackTrie(pkw)

		blockNumber, _ := strconv.Atoi(os.Args[1])
		oracle.PrefetchBlock(big.NewInt(int64(blockNumber)), true, nil)
		oracle.PrefetchBlock(big.NewInt(int64(blockNumber)+1), false, pkwtrie)
		hash, err := pkwtrie.Commit()
		check(err)
		fmt.Println("commited transactions", hash, err)
	}

	// read start block header
	var parent types.Header
	check(rlp.DecodeBytes(oracle.Preimage(oracle.Input(0)), &parent))

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
	newheader.Time = oracle.Input(5).Big().Uint64()

	bc := core.NewBlockChain(&parent)
	database := state.NewDatabase(parent)
	statedb, _ := state.New(parent.Root, database, nil)
	vmconfig := vm.Config{}
	processor := core.NewStateProcessor(params.MainnetChainConfig, bc, bc.Engine())
	fmt.Println("processing state:", parent.Number, "->", newheader.Number)

	newheader.Difficulty = bc.Engine().CalcDifficulty(bc, newheader.Time, &parent)

	// read txs
	//traverseStackTrie(newheader.TxHash)

	//fmt.Println(fn)
	//fmt.Println(txTrieRoot)
	var txs []*types.Transaction

	triedb := trie.NewDatabase(parent)
	tt, _ := trie.New(newheader.TxHash, &triedb)
	tni := tt.NodeIterator([]byte{})
	for tni.Next(true) {
		//fmt.Println(tni.Hash(), tni.Leaf(), tni.Path(), tni.Error())
		if tni.Leaf() {
			tx := types.Transaction{}
			var rlpKey uint64
			check(rlp.DecodeBytes(tni.LeafKey(), &rlpKey))
			check(tx.UnmarshalBinary(tni.LeafBlob()))
			// TODO: resize an array in go?
			for uint64(len(txs)) <= rlpKey {
				txs = append(txs, nil)
			}
			txs[rlpKey] = &tx
		}
	}
	fmt.Println("read", len(txs), "transactions")
	// TODO: OMG the transaction ordering isn't fixed

	var uncles []*types.Header
	check(rlp.DecodeBytes(oracle.Preimage(newheader.UncleHash), &uncles))

	var receipts []*types.Receipt
	block := types.NewBlock(&newheader, txs, uncles, receipts, trie.NewStackTrie(nil))
	fmt.Println("made block, parent:", newheader.ParentHash)

	// if this is correct, the trie is working
	// TODO: it's the previous block now
	if newheader.TxHash != block.Header().TxHash {
		panic("wrong transactions for block")
	}
	if newheader.UncleHash != block.Header().UncleHash {
		panic("wrong uncles for block " + newheader.UncleHash.String() + " " + block.Header().UncleHash.String())
	}

	_, _, _, err := processor.Process(block, statedb, vmconfig)
	if err != nil {
		log.Fatal(err)
	}
	newroot := statedb.IntermediateRoot(bc.Config().IsEIP158(newheader.Number))

	fmt.Println("process done with hash", parent.Root, "->", newroot)
	oracle.Output(newroot)
}
