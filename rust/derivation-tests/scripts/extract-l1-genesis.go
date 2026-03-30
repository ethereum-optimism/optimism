// extract-l1-genesis reconstructs the full L1 genesis JSON from an op-deployer state.json.
//
// The op-deployer state.json stores:
//   - L1StateDump: gzipped L1 account allocations
//   - L1DevGenesis: template without allocs (tagged json:"-" so not serialized to JSON,
//     but the StateHash field is available)
//   - AppliedIntent: contains L1 chain ID and L1DevGenesisParams
//
// This tool recreates what SealL1DevGenesis does: builds a genesis.Genesis from the
// template + allocs, computes the state root via ToBlock(), and outputs the full genesis JSON.
//
// Usage: go run extract-l1-genesis.go <state.json> <output.json>
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <state.json> <output.json>\n", os.Args[0])
		os.Exit(1)
	}

	stateFile := os.Args[1]
	outputFile := os.Args[2]

	// Read state.json
	stateBytes, err := os.ReadFile(stateFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read state file: %v\n", err)
		os.Exit(1)
	}

	var st state.State
	if err := json.Unmarshal(stateBytes, &st); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse state file: %v\n", err)
		os.Exit(1)
	}

	if st.L1StateDump == nil {
		fmt.Fprintf(os.Stderr, "State file has no L1StateDump\n")
		os.Exit(1)
	}

	// Access the decompressed L1 allocs directly via the GzipData wrapper
	forgeAllocs := st.L1StateDump.Data

	// Get L1 genesis parameters from intent
	intent := st.AppliedIntent
	l1DevParams := intent.L1DevGenesisParams
	if l1DevParams == nil {
		l1DevParams = &state.L1DevGenesisParams{}
	}

	bp := &l1DevParams.BlockParams
	timestamp := bp.Timestamp
	// When timestamp is 0, NewL1GenesisMinimal uses time.Now() internally,
	// matching what op-deployer does during apply.
	excessBlobGas := bp.ExcessBlobGas

	// Reconstruct the genesis template (same logic as SealL1DevGenesis)
	genesisTemplate, err := genesis.NewL1GenesisMinimal(&genesis.DevL1DeployConfigMinimal{
		DevL1DeployConfig: genesis.DevL1DeployConfig{
			L1GenesisBlockTimestamp:     hexutil.Uint64(timestamp),
			L1GenesisBlockGasLimit:      hexutil.Uint64(bp.GasLimit),
			L1GenesisBlockExcessBlobGas: (*hexutil.Uint64)(&excessBlobGas),
		},
		L1ChainID:          eth.ChainIDFromUInt64(intent.L1ChainID),
		L1PragueTimeOffset: l1DevParams.PragueTimeOffset,
		L1OsakaTimeOffset:  l1DevParams.OsakaTimeOffset,
		L1BPO1TimeOffset:   l1DevParams.BPO1TimeOffset,
		BlobScheduleConfig: l1DevParams.BlobSchedule,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create L1 genesis template: %v\n", err)
		os.Exit(1)
	}

	// Set the allocs
	genesisTemplate.Alloc = forgeAllocs.Accounts

	// Output the full genesis JSON
	out, err := json.MarshalIndent(genesisTemplate, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal genesis: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "L1 genesis written to %s (%d alloc entries)\n", outputFile, len(genesisTemplate.Alloc))
}
