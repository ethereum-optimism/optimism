package snapshots

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed abi
var abis embed.FS

func LoadDisputeGameFactoryABI() (*abi.ABI, error) {
	return loadABI("DisputeGameFactory")
}
func LoadFaultDisputeGameABI() (*abi.ABI, error) {
	return loadABI("FaultDisputeGame")
}
func LoadPreimageOracleABI() (*abi.ABI, error) {
	return loadABI("PreimageOracle")
}
func LoadMIPSABI() (*abi.ABI, error) {
	return loadABI("MIPS")
}
func LoadDelayedWETHABI() (*abi.ABI, error) {
	return loadABI("DelayedWETH")
}

func loadABI(name string) (*abi.ABI, error) {
	in, err := abis.Open(filepath.Join("abi", name+".json"))
	if err != nil {
		return nil, fmt.Errorf("failed to load ABI for contract %v: %w", name, err)
	}
	defer in.Close()
	if parsed, err := abi.JSON(in); err != nil {
		return nil, err
	} else {
		return &parsed, nil
	}
}
