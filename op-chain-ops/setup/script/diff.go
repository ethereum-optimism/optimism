package script

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// CallDiff encodes the Call to a script on top of a pre-state, and the produced diff,
// that resulted in the post-state with the given checksum.
type CallDiff struct {
	// Call is the script-invocation that resulted in the diff.
	Call *Call `json:"call"`

	// Diff to apply to the pre-state.
	Diff *foundry.ForgeAllocsDiff `json:"allocsDiff"`

	// Labels added during this call (foundry vm.label cheatcode)
	LabelsDiff map[common.Address]string `json:"labelsDiff"`

	// Deployments are named contract deployments that were added during this call.
	DeploymentsDiff map[string]common.Address `json:"deploymentsDiff"`

	// PostChecksum is used to check if the application of the diff on the pre-state results in the correct post-state.
	PostChecksum common.Hash `json:"postChecksum"`
}

func WriteCallDiff(callDiff *CallDiff, destPath string) error {
	f, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %q to write diff to: %w", destPath, err)
	}
	defer f.Close()
	// Indent our cached diff files nicely, they are small anyway, good to make them readable.
	enc := json.NewEncoder(f)
	enc.SetIndent("  ", "  ")
	if err := enc.Encode(callDiff); err != nil {
		return fmt.Errorf("failed to write diff to file %q: %w", destPath, err)
	}
	return nil
}

func LoadCallDiff(srcPath string) (*CallDiff, error) {
	f, err := os.OpenFile(srcPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q to load diff from: %w", srcPath, err)
	}
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields() // be strict with cache loading
	var diff CallDiff
	if err := dec.Decode(&diff); err != nil {
		return nil, fmt.Errorf("failed to load diff from file %q: %w", srcPath, err)
	}
	return &diff, nil
}
