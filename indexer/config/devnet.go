package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	op_service "github.com/ethereum-optimism/optimism/op-service"
)

var DevnetPresetId = 901

func DevnetPreset() (*Preset, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	root, err := op_service.FindMonorepoRoot(cwd)
	if err != nil {
		return nil, err
	}

	devnetFilepath := filepath.Join(root, ".devnet", "addresses.json")
	if _, err := os.Stat(devnetFilepath); errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	content, err := os.ReadFile(devnetFilepath)
	if err != nil {
		return nil, err
	}

	var l1Contracts L1Contracts
	if err := json.Unmarshal(content, &l1Contracts); err != nil {
		return nil, err
	}

	return &Preset{
		Name:        "Local Devnet",
		ChainConfig: ChainConfig{Preset: DevnetPresetId, L1Contracts: l1Contracts},
	}, nil
}
