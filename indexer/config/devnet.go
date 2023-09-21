package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

var (
	filePath           = "../.devnet/addresses.json"
	DEVNET_L2_CHAIN_ID = 901
)

func GetDevnetPreset() (*Preset, error) {
	if _, err := os.Stat(filePath); errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var l1Contracts L1Contracts
	if err := json.Unmarshal(content, &l1Contracts); err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	return &Preset{
		Name: "devnet",
		ChainConfig: ChainConfig{
			Preset:      DEVNET_L2_CHAIN_ID,
			L1Contracts: l1Contracts,
		},
	}, nil
}
