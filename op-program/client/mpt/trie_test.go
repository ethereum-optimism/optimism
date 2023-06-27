package mpt

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type trieCase struct {
	name     string
	elements []hexutil.Bytes
}

func (tc *trieCase) run(t *testing.T) {
	root, preimages := WriteTrie(tc.elements)
	byHash := make(map[common.Hash][]byte)
	for _, v := range preimages {
		k := crypto.Keccak256Hash(v)
		byHash[k] = v
	}
	results := ReadTrie(root, func(key common.Hash) []byte {
		v, ok := byHash[key]
		if !ok {
			panic(fmt.Errorf("missing key %s", key))
		}
		return v
	})
	require.Equal(t, len(tc.elements), len(results), "expected equal amount of values")
	for i, result := range results {
		// hex encoded for debugging readability
		require.Equal(t, tc.elements[i].String(), result.String(),
			"value %d does not match, expected equal value data", i)
	}
}

func TestListTrieRoundtrip(t *testing.T) {
	testCases := []trieCase{
		{name: "empty list", elements: []hexutil.Bytes{}},
		{name: "nil list", elements: nil},
		{name: "simple", elements: []hexutil.Bytes{[]byte("hello"), []byte("world")}},
	}
	rng := rand.New(rand.NewSource(1234))
	// add some randomized cases
	for i := 0; i < 30; i++ {
		n := rng.Intn(300)
		elems := make([]hexutil.Bytes, n)
		for i := range elems {
			length := 1 + rng.Intn(300) // empty items not allowed
			data := make([]byte, length)
			rng.Read(data[:])
			elems[i] = data
		}
		testCases = append(testCases, trieCase{name: fmt.Sprintf("rand_%d", i), elements: elems})
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.run)
	}
}
