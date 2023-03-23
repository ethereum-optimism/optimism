package util

import (
	crand "crypto/rand"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

var testAddr = common.Address{0: 0xff}

func TestStateIteratorWorkers(t *testing.T) {
	_, factory, _ := setupRandTest(t)

	for i := -1; i <= 0; i++ {
		require.Panics(t, func() {
			_ = IterateState(factory, testAddr, func(db *state.StateDB, key, value common.Hash) error {
				return nil
			}, i)
		})
	}
}

func TestStateIteratorNonexistentAccount(t *testing.T) {
	_, factory, _ := setupRandTest(t)

	require.ErrorContains(t, IterateState(factory, common.Address{}, func(db *state.StateDB, key, value common.Hash) error {
		return nil
	}, 1), "account does not exist")
}

func TestStateIteratorRandomOK(t *testing.T) {
	for i := 0; i < 100; i++ {
		hashes, factory, workerCount := setupRandTest(t)

		seenHashes := make(map[common.Hash]bool)
		hashCh := make(chan common.Hash)
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			for hash := range hashCh {
				seenHashes[hash] = true
			}
		}()

		require.NoError(t, IterateState(factory, testAddr, func(db *state.StateDB, key, value common.Hash) error {
			hashCh <- key
			return nil
		}, workerCount))

		close(hashCh)
		<-doneCh

		// Perform a less or equal check here in case of duplicates. The map check below will assert
		// that all of the hashes are accounted for.
		require.LessOrEqual(t, len(seenHashes), len(hashes))

		// Every hash we put into state should have been iterated over.
		for _, hash := range hashes {
			require.Contains(t, seenHashes, hash)
		}
	}
}

func TestStateIteratorRandomError(t *testing.T) {
	for i := 0; i < 100; i++ {
		hashes, factory, workerCount := setupRandTest(t)

		failHash := hashes[rand.Intn(len(hashes))]
		require.ErrorContains(t, IterateState(factory, testAddr, func(db *state.StateDB, key, value common.Hash) error {
			if key == failHash {
				return fmt.Errorf("test error")
			}
			return nil
		}, workerCount), "test error")
	}
}

func TestPartitionKeyspace(t *testing.T) {
	tests := []struct {
		i        int
		count    int
		expected [2]common.Hash
	}{
		{
			i:     0,
			count: 1,
			expected: [2]common.Hash{
				common.HexToHash("0x00"),
				common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			},
		},
		{
			i:     0,
			count: 2,
			expected: [2]common.Hash{
				common.HexToHash("0x00"),
				common.HexToHash("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			},
		},
		{
			i:     1,
			count: 2,
			expected: [2]common.Hash{
				common.HexToHash("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
				common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			},
		},
		{
			i:     0,
			count: 3,
			expected: [2]common.Hash{
				common.HexToHash("0x00"),
				common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
			},
		},
		{
			i:     1,
			count: 3,
			expected: [2]common.Hash{
				common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
				common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			},
		},
		{
			i:     2,
			count: 3,
			expected: [2]common.Hash{
				common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("i %d, count %d", tt.i, tt.count), func(t *testing.T) {
			start, end := PartitionKeyspace(tt.i, tt.count)
			require.Equal(t, tt.expected[0], start)
			require.Equal(t, tt.expected[1], end)
		})
	}

	t.Run("panics on invalid i or count", func(t *testing.T) {
		require.Panics(t, func() {
			PartitionKeyspace(1, 1)
		})
		require.Panics(t, func() {
			PartitionKeyspace(-1, 1)
		})
		require.Panics(t, func() {
			PartitionKeyspace(0, -1)
		})
		require.Panics(t, func() {
			PartitionKeyspace(-1, -1)
		})
	})
}

func setupRandTest(t *testing.T) ([]common.Hash, DBFactory, int) {
	memDB := rawdb.NewMemoryDatabase()
	db, err := state.New(common.Hash{}, state.NewDatabaseWithConfig(memDB, &trie.Config{
		Preimages: true,
		Cache:     1024,
	}), nil)
	require.NoError(t, err)

	hashCount := rand.Intn(100)
	if hashCount == 0 {
		hashCount = 1
	}

	hashes := make([]common.Hash, hashCount)

	db.CreateAccount(testAddr)

	for j := 0; j < hashCount; j++ {
		hashes[j] = randHash(t)
		db.SetState(testAddr, hashes[j], hashes[j])
	}

	root, err := db.Commit(false)
	require.NoError(t, err)

	err = db.Database().TrieDB().Commit(root, true)
	require.NoError(t, err)

	factory := func() (*state.StateDB, error) {
		return state.New(root, state.NewDatabaseWithConfig(memDB, &trie.Config{
			Preimages: true,
			Cache:     1024,
		}), nil)
	}

	workerCount := rand.Intn(64)
	if workerCount == 0 {
		workerCount = 1
	}
	return hashes, factory, workerCount
}

func randHash(t *testing.T) common.Hash {
	var h common.Hash
	_, err := crand.Read(h[:])
	require.NoError(t, err)
	return h
}
