// compute-genesis-hashes computes the genesis block hash and state root for both
// L1 and L2 genesis files using go-ethereum's Genesis.ToBlock() method.
//
// This ensures the hash computation matches exactly what op-program/op-geth use.
//
// Usage: go run compute-genesis-hashes.go <l1-genesis.json> <l2-genesis.json> <output.json>
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/core"
)

type GenesisHashes struct {
	L1 GenesisBlockInfo `json:"l1"`
	L2 GenesisBlockInfo `json:"l2"`
}

type GenesisBlockInfo struct {
	Hash      string `json:"hash"`
	StateRoot string `json:"stateRoot"`
}

func computeGenesisInfo(genesisPath string) (GenesisBlockInfo, error) {
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return GenesisBlockInfo{}, fmt.Errorf("failed to read %s: %w", genesisPath, err)
	}

	var genesis core.Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		return GenesisBlockInfo{}, fmt.Errorf("failed to parse %s: %w", genesisPath, err)
	}

	block := genesis.ToBlock()
	return GenesisBlockInfo{
		Hash:      block.Hash().Hex(),
		StateRoot: block.Root().Hex(),
	}, nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <l1-genesis.json> <l2-genesis.json> <output.json>\n", os.Args[0])
		os.Exit(1)
	}

	l1Info, err := computeGenesisInfo(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "L1 genesis error: %v\n", err)
		os.Exit(1)
	}

	l2Info, err := computeGenesisInfo(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "L2 genesis error: %v\n", err)
		os.Exit(1)
	}

	result := GenesisHashes{L1: l1Info, L2: l2Info}
	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal output: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(os.Args[3], out, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "L1: hash=%s stateRoot=%s\n", l1Info.Hash, l1Info.StateRoot)
	fmt.Fprintf(os.Stderr, "L2: hash=%s stateRoot=%s\n", l2Info.Hash, l2Info.StateRoot)
}
