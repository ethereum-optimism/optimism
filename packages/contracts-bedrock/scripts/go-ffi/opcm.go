package main

import (
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/opcmregistry"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// GetOPCMs returns ABI-encoded OPCM data for a given chain ID.
// Called via FFI from Solidity tests.
// Usage: go-ffi opcm <chainID>
// Returns: ABI-encoded array of (address opcmAddress, string version, bool isV1)
func GetOPCMs() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go-ffi opcm <chainID>\n")
		os.Exit(1)
	}

	var chainID uint64
	if _, err := fmt.Sscanf(os.Args[2], "%d", &chainID); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid chain ID: %s\n", os.Args[2])
		os.Exit(1)
	}

	opcms, err := opcmregistry.GetOPCMsForChain(chainID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get OPCMs: %v\n", err)
		os.Exit(1)
	}

	// Encode as ABI-packed array of structs
	// struct OPCMInfo { address addr; string version; bool isV1; }
	type OPCMInfoEncoded struct {
		Addr    common.Address
		Version string
		IsV1    bool
	}

	encoded := make([]OPCMInfoEncoded, len(opcms))
	for i, opcm := range opcms {
		encoded[i] = OPCMInfoEncoded{
			Addr:    opcm.Address,
			Version: opcm.Version,
			IsV1:    opcm.IsV1,
		}
	}

	// Define the ABI type for the array of structs
	opcmInfoType, _ := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "addr", Type: "address"},
		{Name: "version", Type: "string"},
		{Name: "isV1", Type: "bool"},
	})

	args := abi.Arguments{{Type: opcmInfoType}}
	result, err := args.Pack(encoded)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ABI encode: %v\n", err)
		os.Exit(1)
	}

	// Output as hex for Solidity to decode
	fmt.Printf("0x%x", result)
}
