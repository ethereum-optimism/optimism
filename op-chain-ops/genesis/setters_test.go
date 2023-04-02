package genesis

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

func TestWipePredeployStorage(t *testing.T) {
	rawDB := rawdb.NewMemoryDatabase()
	rawStateDB := state.NewDatabaseWithConfig(rawDB, &trie.Config{
		Preimages: true,
		Cache:     1024,
	})
	stateDB, err := state.New(common.Hash{}, rawStateDB, nil)
	require.NoError(t, err)

	storeVal := common.Hash{31: 0xff}

	for _, addr := range predeploys.Predeploys {
		a := *addr
		stateDB.SetState(a, storeVal, storeVal)
		stateDB.SetBalance(a, big.NewInt(99))
		stateDB.SetNonce(a, 99)
	}

	root, err := stateDB.Commit(false)
	require.NoError(t, err)

	err = stateDB.Database().TrieDB().Commit(root, true)
	require.NoError(t, err)

	require.NoError(t, WipePredeployStorage(stateDB))

	for _, addr := range predeploys.Predeploys {
		a := *addr
		if FrozenStoragePredeploys[a] {
			require.Equal(t, storeVal, stateDB.GetState(a, storeVal))
		} else {
			require.Equal(t, common.Hash{}, stateDB.GetState(a, storeVal))
		}
		require.Equal(t, big.NewInt(99), stateDB.GetBalance(a))
		require.Equal(t, uint64(99), stateDB.GetNonce(a))
	}
}
