package trie

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Variant enum
const (
	// Generate a test case with a valid proof of inclusion for the k/v pair in the trie.
	valid string = "valid"
	// Generate an invalid test case with an extra proof element attached to an otherwise
	// valid proof of inclusion for the passed k/v.
	extraProofElems = "extra_proof_elems"
	// Generate an invalid test case where the proof is malformed.
	corruptedProof = "corrupted_proof"
	// Generate an invalid test case where a random element of the proof has more bytes than the
	// length designates within the RLP list encoding.
	invalidDataRemainder = "invalid_data_remainder"
	// Generate an invalid test case where a long proof element is incorrect for the root.
	invalidLargeInternalHash = "invalid_large_internal_hash"
	// Generate an invalid test case where a small proof element is incorrect for the root.
	invalidInternalNodeHash = "invalid_internal_node_hash"
	// Generate a valid test case with a key that has been given a random prefix
	prefixedValidKey = "prefixed_valid_key"
	// Generate a valid test case with a proof of inclusion for an empty key.
	emptyKey = "empty_key"
	// Generate an invalid test case with a partially correct proof
	partialProof = "partial_proof"
)

// Generate an abi-encoded `trieTestCase` of a specified variant
func FuzzTrie(variant string) {
	if len(variant) < 1 {
		log.Fatal("Must pass a variant to the trie fuzzer!")
	}

	var testCase trieTestCase
	switch variant {
	case valid:
		testCase = genTrieTestCase(false)
	case extraProofElems:
		testCase = genTrieTestCase(false)
		// Duplicate the last element of the proof
		testCase.Proof = append(testCase.Proof, [][]byte{testCase.Proof[len(testCase.Proof)-1]}...)
	case corruptedProof:
		testCase = genTrieTestCase(false)

		// Re-encode a random element within the proof
		idx := randRange(0, int64(len(testCase.Proof)))
		encoded, _ := rlp.EncodeToBytes(testCase.Proof[idx])
		testCase.Proof[idx] = encoded
	case invalidDataRemainder:
		testCase = genTrieTestCase(false)

		// Alter true length of random proof element by appending random bytes
		// Do not update the encoded length
		idx := randRange(0, int64(len(testCase.Proof)))
		bytes := make([]byte, randRange(1, 512))
		_, err := rand.Read(bytes)
		if err != nil {
			log.Fatal("Error generating random bytes for invalidDataRemainder")
		}
		testCase.Proof[idx] = append(testCase.Proof[idx], bytes...)
	case invalidLargeInternalHash:
		testCase = genTrieTestCase(false)

		// Clobber 4 bytes within a list element of a random proof element
		// TODO: Improve this by decoding the proof elem and choosing random
		// bytes to overwrite.
		idx := randRange(1, int64(len(testCase.Proof)))
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatal("Error generating random bytes for invalidLargeInternalHash")
		}
		testCase.Proof[idx] = append(
			testCase.Proof[idx][:20],
			append(
				b,
				testCase.Proof[idx][24:]...,
			)...,
		)
	case invalidInternalNodeHash:
		testCase = genTrieTestCase(false)
		// Assign the last proof element to an encoded list containing a
		// random 29 byte value
		b := make([]byte, 29)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatal("Error generating random bytes for invalidInternalNodeHash")
		}
		e, _ := rlp.EncodeToBytes(b)
		testCase.Proof[len(testCase.Proof)-1] = append([]byte{0xc0 + 30}, e...)
	case prefixedValidKey:
		testCase = genTrieTestCase(false)

		bytes := make([]byte, randRange(1, 16))
		_, err := rand.Read(bytes)
		if err != nil {
			log.Fatal("Error generating random bytes for prefixedValidKey")
		}
		testCase.Key = append(bytes, testCase.Key...)
	case emptyKey:
		testCase = genTrieTestCase(true)
	case partialProof:
		testCase = genTrieTestCase(false)

		// Cut the proof in half
		proofLen := len(testCase.Proof)
		newProof := make([][]byte, proofLen/2)
		for i := 0; i < proofLen/2; i++ {
			newProof[i] = testCase.Proof[i]
		}

		testCase.Proof = newProof
	default:
		log.Fatal("Invalid variant passed to trie fuzzer!")
	}

	// Print encoded test case with no newline so that foundry's FFI can read the output
	fmt.Print(testCase.AbiEncode())
}

// Generate a random test case for Bedrock's MerkleTrie verifier.
func genTrieTestCase(selectEmptyKey bool) trieTestCase {
	// Create an empty merkle trie
	randTrie := trie.NewEmpty(trie.NewDatabase(rawdb.NewMemoryDatabase()))

	// Get a random number of elements to put into the trie
	randN := randRange(2, 1024)
	// Get a random key/value pair to generate a proof of inclusion for
	randSelect := randRange(0, randN)

	// Create a fixed-length key as well as a randomly-sized value
	// We create these out of the loop to reduce mem allocations.
	randKey := make([]byte, 32)
	randValue := make([]byte, randRange(2, 1024))

	// Randomly selected key/value for proof generation
	var key []byte
	var value []byte

	// Add `randN` elements to the trie
	for i := int64(0); i < randN; i++ {
		// Randomize the contents of `randKey` and `randValue`
		_, err := rand.Read(randKey)
		if err != nil {
			log.Fatal("Error generating random key")
		}
		_, err = rand.Read(randValue)
		if err != nil {
			log.Fatal("Error generating random value")
		}

		// Clear the selected key if `selectEmptyKey` is true
		if i == randSelect && selectEmptyKey {
			randKey = make([]byte, 0)
		}

		// Insert the random k/v pair into the trie
		randTrie.MustUpdate(randKey, randValue)

		// If this is our randomly selected k/v pair, store it in `key` & `value`
		if i == randSelect {
			key = randKey
			value = randValue
		}
	}

	// Generate the merkle proof for the randomly selected key.
	// This writes the proof to the passed `memdb`.
	proofDB := memorydb.New()
	if err := randTrie.Prove(key, proofDB); err != nil {
		log.Fatal("Error creating proof for randomly selected key's inclusion in generated trie")
	}

	// Verify the proof
	root := randTrie.Hash()
	val, err := trie.VerifyProof(root, key, proofDB)
	if err != nil {
		log.Fatal("Error verifying proof for randomly selected key's inclusion in generated trie")
	}

	// Create our test case with the data collected
	testCase := trieTestCase{
		Root:  root,
		Key:   key,
		Value: value,
		Proof: [][]byte{val},
	}

	return testCase
}

// Represents a test case for bedrock's `MerkleTrie.sol`
type trieTestCase struct {
	Root  common.Hash
	Key   []byte
	Value []byte
	Proof [][]byte
}

// Tuple type to encode `TrieTestCase`
var (
	trieTestCaseTuple, _ = abi.NewType("tuple", "TrieTestCase", []abi.ArgumentMarshaling{
		{Name: "root", Type: "bytes32"},
		{Name: "key", Type: "bytes"},
		{Name: "value", Type: "bytes"},
		{Name: "proof", Type: "bytes[]"},
	})

	encoder = abi.Arguments{
		{Type: trieTestCaseTuple},
	}
)

// Encodes the trieTestCase as the `trieTestCaseTuple`.
func (t *trieTestCase) AbiEncode() string {
	// Encode the contents of the struct as a tuple
	packed, err := encoder.Pack(&t)
	if err != nil {
		log.Fatalf("Error packing TrieTestCase: %v", err)
	}

	// Remove the pointer and encode the packed bytes as a hex string
	return hexutil.Encode(packed[32:])
}

// Helper that generates a cryptographically secure random 64-bit integer
// between the range [min, max)
func randRange(min int64, max int64) int64 {
	r, err := rand.Int(rand.Reader, new(big.Int).Sub(new(big.Int).SetInt64(max), new(big.Int).SetInt64(min)))
	if err != nil {
		log.Fatal("Failed to generate random number within bounds")
	}

	return (new(big.Int).Add(r, new(big.Int).SetInt64(min))).Int64()
}
