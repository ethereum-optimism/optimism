package ether

import (
	"encoding/json"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/core"
)

// ReadGenesisFromFile reads a genesis object from a file.
func ReadGenesisFromFile(path string) (*core.Genesis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadGenesis(f)
}

// ReadGenesis reads a genesis object from an io.Reader.
func ReadGenesis(r io.Reader) (*core.Genesis, error) {
	genesis := new(core.Genesis)
	if err := json.NewDecoder(r).Decode(genesis); err != nil {
		return nil, err
	}
	return genesis, nil
}
