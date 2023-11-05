package fault

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
)

type PrestateLoader interface {
	GetAbsolutePrestateHash(ctx context.Context) (common.Hash, error)
}

func newSingleTracePrestateValidator(trace types.TraceProvider) absolutePrestateValidator {
	return func(ctx context.Context, gameContract *contracts.FaultDisputeGameContract) error {
		return ValidateAbsolutePrestate(ctx, trace, gameContract)
	}
}

func noopPrestateValidator(_ context.Context, _ *contracts.FaultDisputeGameContract) error {
	return nil
}

// ValidateAbsolutePrestate validates the absolute prestate of the fault game.
func ValidateAbsolutePrestate(ctx context.Context, trace types.TraceProvider, loader PrestateLoader) error {
	providerPrestateHash, err := trace.AbsolutePreStateCommitment(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the trace provider's absolute prestate: %w", err)
	}
	onchainPrestate, err := loader.GetAbsolutePrestateHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the onchain absolute prestate: %w", err)
	}
	if !bytes.Equal(providerPrestateHash[:], onchainPrestate[:]) {
		return fmt.Errorf("trace provider's absolute prestate does not match onchain absolute prestate: Provider: %s | Chain %s", providerPrestateHash.Hex(), onchainPrestate.Hex())
	}
	return nil
}
