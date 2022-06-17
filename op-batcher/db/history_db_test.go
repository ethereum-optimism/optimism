package db_test

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
	"github.com/stretchr/testify/require"
)

func TestOpenJSONFileDatabaseNoFile(t *testing.T) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()

	err = os.Remove(filename)
	require.Nil(t, err)

	hdb, err := db.OpenJSONFileDatabase(filename)
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

	hdb, err := db.OpenJSONFileDatabase(filename)
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

	hdb, err := db.OpenJSONFileDatabase(filename)
	require.Nil(t, err)
	require.NotNil(t, hdb)

	err = hdb.Close()
	require.Nil(t, err)
}

func makeDB(t *testing.T) (*db.JSONFileDatabase, func()) {
	file, err := ioutil.TempFile("", "history_db.*.json")
	require.Nil(t, err)

	filename := file.Name()
	hdb, err := db.OpenJSONFileDatabase(filename)
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
	require.Equal(t, int(0), len(history.Channels))

	expHistory := &db.History{
		Channels: make(map[derive.ChannelID]uint64),
	}
	require.Equal(t, expHistory, history)
}

func TestUpdate(t *testing.T) {
	hdb, cleanup := makeDB(t)
	defer cleanup()

	rng := rand.New(rand.NewSource(1234))

	// mock some random channel updates in a time range
	genUpdate := func(n uint64, minTime uint64, maxTime uint64) map[derive.ChannelID]uint64 {
		out := make(map[derive.ChannelID]uint64)
		for i := uint64(0); i < n; i++ {
			var id derive.ChannelID
			rng.Read(id.Data[:])
			id.Time = minTime + uint64(rng.Intn(int(maxTime-minTime)))
			out[id] = uint64(rng.Intn(1000))
		}
		return out
	}

	history, err := hdb.LoadHistory()
	require.Nil(t, err)

	first := genUpdate(20, 1000, 2000)
	// first update: be generous with a large timeout, merge in full update
	history.Update(first, 10000, 2000)
	require.Equal(t, history.Channels, first)
	require.Equal(t, len(history.Channels), 20)

	// now try to add something completely new
	second := genUpdate(10, 1500, 2400)
	history.Update(second, 10000, 2000)
	require.Equal(t, len(history.Channels), 20+10)

	// now time out some older channels, while adding a few new ones that are too old
	third := genUpdate(15, 800, 1500)
	history.Update(third, 1000, 2500)
	// check if second is not pruned
	for id := range second {
		require.Contains(t, history.Channels, id)
	}
	// check if third is fully pruned
	for id := range third {
		require.NotContains(t, history.Channels, id)
	}

	// try store history back in the db
	require.NoError(t, hdb.Update(history.Channels, 0, 0))

	// time out everything
	history.Update(nil, 0, 2400)
	require.Len(t, history.Channels, 0)
}
