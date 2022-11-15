package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/trie"
)

// Represents a test case for bedrock's `MerkleTrie.sol`
type TrieTestCase struct {
	Root  *[32]byte
	Key   []byte
	Value []byte
	Proof [][]byte
}

// Tuple type to encode `TrieTestCase`
var (
	trieTestCase, _ = abi.NewType("tuple", "TrieTestCase", []abi.ArgumentMarshaling{
		{Name: "root", Type: "bytes32"},
		{Name: "key", Type: "bytes"},
		{Name: "value", Type: "bytes"},
		{Name: "proof", Type: "bytes[]"},
	})

	encoder = abi.Arguments{
		{Type: trieTestCase},
	}
)

// Encodes the TrieTestCase as the `trieTestCase` tuple.
func (t *TrieTestCase) AbiEncode() string {
	// Encode the contents of the struct as a tuple
	packed, err := encoder.Pack(&t)
	if err != nil {
		log.Fatalf("Error packing TrieTestCase: %v", err)
	}

	// Remove the pointer and encode the packed bytes as a hex string
	return hexutil.Encode(packed[32:])
}

func main() {
	// Create an empty merkle trie
	memdb := memorydb.New()
	randTrie := trie.NewEmpty(trie.NewDatabase(memdb))

	// Seed the random number generator with the current unix timestamp
	rand.Seed(time.Now().UnixNano())

	// Get a random number of elements to put into the trie
	randN := randRange(2, 1024)
	// Get a random key/value pair to generate a proof of inclusion for
	randSelect := randRange(0, randN)

	// Create a fixed-length key as well as a randomly-sized value
	// We create these out of the loop to reduce mem allocations.
	randKey := make([]byte, 32)
	randValue := make([]byte, randRange(1, 256))

	// Randomly selected key/value for proof generation
	var key []byte
	var value []byte

	// Add `randN` elements to the trie
	for i := 0; i < randN; i++ {
		// Randomize the contents of `randKey` and `randValue`
		rand.Read(randKey)
		rand.Read(randValue)

		// Insert the random k/v pair into the trie
		if err := randTrie.TryUpdate(randKey, randValue); err != nil {
			log.Fatal("Error adding key-value pair to trie")
		}

		// If this is our randomly selected k/v pair, store it in `key` & `value`
		if i == randSelect {
			key = randKey
			value = randValue
		}
	}

	// Generate proof for `key`'s inclusion in our trie
	proofDB := memorydb.New()
	if err := randTrie.Prove(key, 0, proofDB); err != nil {
		log.Fatal("Error creating proof for randomly selected key's inclusion in generated trie")
	}
	_, err := trie.VerifyProof(randTrie.Hash(), key, proofDB)
	if err != nil {
		log.Println("Failed to verify generated proof!")
	}

	// Pull the proof out of the `proofDB`
	// TODO: This is not the correct way to do this it seems (?)
	// I believe it's due to the `NewIterator` function returning a sorted
	// collection
	proof := make([][]byte, 0)
	proof_iter := proofDB.NewIterator(make([]byte, 0), make([]byte, 0))
	for proof_iter.Next() {
		proof = append(proof, [][]byte{proof_iter.Value()}...)
	}

	// Create our test case with the data collected
	testCase := TrieTestCase{
		Root:  (*[32]byte)(randTrie.Hash().Bytes()),
		Key:   key,
		Value: value,
		Proof: proof,
	}

	// Print encoded test case with no newline so that foundry's FFI can read the output
	fmt.Print(testCase.AbiEncode())
}

// Helper that generates a random positive integer between the range [min, max]
func randRange(min int, max int) int {
	return rand.Intn(max-min) + min
}
