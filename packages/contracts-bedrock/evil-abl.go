package contracts_bedrock

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed snapshots/abi
var abis embed.FS

func LoadFaultDisputeGameABI() (*abi.ABI, error) {
	return loadABI("FaultDisputeGame")
}

func loadABI(name string) (*abi.ABI, error) {
	in, err := abis.Open(filepath.Join("snapshots/abi", name+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load ABI for contract %v: %w", name, err)
	}
	if parsed, err := abi.JSON(in); err != nil {
		return nil, err
	} else {
		return &parsed, nil
	}
}
