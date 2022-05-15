package db_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const (
	testMaxDepth uint64 = 10
)

var (
	testGenesisHash = common.HexToHash("0xabcd")

	genesisEntry = eth.BlockID{
		Number: 0,
		Hash:   testGenesisHash,
	}
)

func TestOpenJSONFileDatabaseNoFile(t *testing.T) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()

	err = os.Remove(filename)
	require.Nil(t, err)

	hdb, err := db.OpenJSONFileDatabase(filename, testMaxDepth, testGenesisHash)
	require.Nil(t, err)
	require.NotNil(t, hdb)

	err = hdb.Close()
	require.Nil(t, err)
}

func TestOpenJSONFileDatabaseEmptyFile(t *testing.T) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()
	defer os.Remove(filename)

	hdb, err := db.OpenJSONFileDatabase(filename, testMaxDepth, testGenesisHash)
	require.Nil(t, err)
	require.NotNil(t, hdb)

	err = hdb.Close()
	require.Nil(t, err)
}

func TestOpenJSONFileDatabase(t *testing.T) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()
	defer os.Remove(filename)

	hdb, err := db.OpenJSONFileDatabase(filename, testMaxDepth, testGenesisHash)
	require.Nil(t, err)
	require.NotNil(t, hdb)

	err = hdb.Close()
	require.Nil(t, err)
}

func makeDB(t *testing.T) (*db.JSONFileDatabase, func()) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()
	hdb, err := db.OpenJSONFileDatabase(filename, testMaxDepth, testGenesisHash)
	require.Nil(t, err)
	require.NotNil(t, hdb)

	cleanup := func() {
		_ = hdb.Close()
		_ = os.Remove(filename)
	}

	return hdb, cleanup
}

func TestLoadHistoryEmpty(t *testing.T) {
	hdb, cleanup := makeDB(t)
	defer cleanup()

	history, err := hdb.LoadHistory()
	require.Nil(t, err)
	require.NotNil(t, history)
	require.Equal(t, int(1), len(history.BlockIDs))

	expHistory := &db.History{
		BlockIDs: []eth.BlockID{genesisEntry},
	}
	require.Equal(t, expHistory, history)
}

func TestAppendEntry(t *testing.T) {
	hdb, cleanup := makeDB(t)
	defer cleanup()

	genExpHistory := func(n uint64) *db.History {
		var history db.History
		history.AppendEntry(genesisEntry, testMaxDepth)
		for i := uint64(0); i < n+1; i++ {
			history.AppendEntry(eth.BlockID{
				Number: i,
				Hash:   common.Hash{byte(i)},
			}, testMaxDepth)
		}
		return &history
	}

	for i := uint64(0); i < 2*testMaxDepth; i++ {
		err := hdb.AppendEntry(eth.BlockID{
			Number: i,
			Hash:   common.Hash{byte(i)},
		})
		require.Nil(t, err)

		history, err := hdb.LoadHistory()
		require.Nil(t, err)

		expHistory := genExpHistory(i)
		require.Equal(t, expHistory, history)
		require.LessOrEqual(t, uint64(len(history.BlockIDs)), testMaxDepth+1)
	}
}
