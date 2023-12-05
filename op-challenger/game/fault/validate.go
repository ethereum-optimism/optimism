package fault

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type HashLoader = func(ctx context.Context) (common.Hash, error)

// ValidateGenesisOutputRoot validates the genesis output root of the provider.
func ValidateGenesisOutputRoot(ctx context.Context, provider types.PrestateProvider, loader HashLoader) error {
	providerGenesisOutputRoot, err := provider.GenesisOutputRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's genesis output root: %w", err)
	}
	onchainGenesisOutputRoot, err := loader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain genesis output root: %w", err)
	}
	if !bytes.Equal(providerGenesisOutputRoot[:], onchainGenesisOutputRoot[:]) {
		return fmt.Errorf("provider's genesis output root does not match onchain genesis output root: Provider: %s | Chain %s", providerGenesisOutputRoot.Hex(), onchainGenesisOutputRoot.Hex())
	}
	return nil
}

// ValidateAbsolutePrestate validates the absolute prestate of the fault game.
func ValidateAbsolutePrestate(ctx context.Context, provider types.PrestateProvider, loader HashLoader) error {
	providerPrestateHash, err := provider.AbsolutePreStateCommitment(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's absolute prestate: %w", err)
	}
	onchainPrestate, err := loader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain absolute prestate: %w", err)
	}
	if !bytes.Equal(providerPrestateHash[:], onchainPrestate[:]) {
		return fmt.Errorf("provider's absolute prestate does not match onchain absolute prestate: Provider: %s | Chain %s", providerPrestateHash.Hex(), onchainPrestate.Hex())
	}
	return nil
}
