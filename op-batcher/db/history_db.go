package db

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type History struct {
	Channels map[derive.ChannelID]uint64 `json:"channels"`
}

func (h *History) Update(add map[derive.ChannelID]uint64, timeout uint64, l1Time uint64) {
	// merge the two maps
	for id, frameNr := range add {
		if prev, ok := h.Channels[id]; ok && prev > frameNr {
			continue // don't roll back channels
		}
		h.Channels[id] = frameNr
	}
	// prune everything that is timed out
	for id := range h.Channels {
		if id.Time+timeout < l1Time {
			delete(h.Channels, id) // removal of the map during iteration is safe in Go
		}
	}
}

type HistoryDatabase interface {
	LoadHistory() (*History, error)
	Update(add map[derive.ChannelID]uint64, timeout uint64, l1Time uint64) error
	Close() error
}

type JSONFileDatabase struct {
	filename string
}

func OpenJSONFileDatabase(
	filename string,
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
		filename: filename,
	}, nil
}

func (d *JSONFileDatabase) LoadHistory() (*History, error) {
	fileContents, err := os.ReadFile(d.filename)
	if err != nil {
		return nil, err
	}

	if len(fileContents) == 0 {
		return &History{
			Channels: make(map[derive.ChannelID]uint64),
		}, nil
	}

	var history History
	err = json.Unmarshal(fileContents, &history)
	if err != nil {
		return nil, err
	}

	return &history, nil
}

func (d *JSONFileDatabase) Update(add map[derive.ChannelID]uint64, timeout uint64, l1Time uint64) error {
	history, err := d.LoadHistory()
	if err != nil {
		return err
	}

	history.Update(add, timeout, l1Time)

	newFileContents, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(d.filename, newFileContents, 0644)
}

func (d *JSONFileDatabase) Close() error {
	return nil
}
