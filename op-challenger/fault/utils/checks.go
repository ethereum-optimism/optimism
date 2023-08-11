package utils

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/chain"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"

	"github.com/ethereum/go-ethereum/crypto"
)

// ValidateAbsolutePrestate validates the absolute prestate of the fault game.
func ValidateAbsolutePrestate(ctx context.Context, trace types.TraceProvider, loader chain.Loader) error {
	providerPrestate, err := trace.AbsolutePreState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's absolute prestate: %w", err)
	}
	providerPrestateHash := crypto.Keccak256(providerPrestate)
	onchainPrestate, err := loader.FetchAbsolutePrestateHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain absolute prestate: %w", err)
	}
	if !bytes.Equal(providerPrestateHash, onchainPrestate) {
		return fmt.Errorf("trace provider's absolute prestate does not match onchain absolute prestate")
	}
	return nil
}
