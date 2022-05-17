package db

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
)

type History struct {
	BlockIDs []eth.BlockID `json:"block_ids"`
}

func (h *History) LatestID() eth.BlockID {
	return h.BlockIDs[len(h.BlockIDs)-1]
}

func (h *History) AppendEntry(blockID eth.BlockID, maxEntries uint64) {
	for _, id := range h.BlockIDs {
		if id.Hash == blockID.Hash {
			return
		}
	}

	h.BlockIDs = append(h.BlockIDs, blockID)
	if uint64(len(h.BlockIDs)) > maxEntries {
		h.BlockIDs = h.BlockIDs[len(h.BlockIDs)-int(maxEntries):]
	}
}

func (h *History) Ancestors() []common.Hash {
	var sortedBlockIDs = make([]eth.BlockID, 0, len(h.BlockIDs))
	sortedBlockIDs = append(sortedBlockIDs, h.BlockIDs...)

	// Keep block ids sorted in ascending order to minimize the number of swaps.
	// Use stable sort so that newest are prioritized over older ones.
	sort.SliceStable(sortedBlockIDs, func(i, j int) bool {
		return sortedBlockIDs[i].Number < sortedBlockIDs[j].Number
	})

	var ancestors = make([]common.Hash, 0, len(h.BlockIDs))
	for i := len(h.BlockIDs) - 1; i >= 0; i-- {
		ancestors = append(ancestors, h.BlockIDs[i].Hash)
	}

	return ancestors
}

type HistoryDatabase interface {
	LoadHistory() (*History, error)
	AppendEntry(eth.BlockID) error
	Close() error
}

type JSONFileDatabase struct {
	filename    string
	maxEntries  uint64
	genesisHash common.Hash
}

func OpenJSONFileDatabase(
	filename string,
	maxEntries uint64,
	genesisHash common.Hash,
) (*JSONFileDatabase, error) {

	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		file, err := os.Create(filename)
		if err != nil {
			return nil, err
		}
		err = file.Close()
		if err != nil {
			return nil, err
		}
	}

	return &JSONFileDatabase{
		filename:    filename,
		maxEntries:  maxEntries,
		genesisHash: genesisHash,
	}, nil
}

func (d *JSONFileDatabase) LoadHistory() (*History, error) {
	fileContents, err := os.ReadFile(d.filename)
	if err != nil {
		return nil, err
	}

	if len(fileContents) == 0 {
		return &History{
			BlockIDs: []eth.BlockID{
				{
					Number: 0,
					Hash:   d.genesisHash,
				},
			},
		}, nil
	}

	var history History
	err = json.Unmarshal(fileContents, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func (d *JSONFileDatabase) AppendEntry(blockID eth.BlockID) error {
	history, err := d.LoadHistory()
	if err != nil {
		return err
	}

	history.AppendEntry(blockID, d.maxEntries)

	newFileContents, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(d.filename, newFileContents, 0644)
}

func (d *JSONFileDatabase) Close() error {
	return nil
}
