package fault

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type PrestateLoader = func(ctx context.Context) (common.Hash, error)

type Validator interface {
	Validate(ctx context.Context) error
}

var _ Validator = (*PrestateValidator)(nil)

type PrestateValidator struct {
	load     PrestateLoader
	provider types.PrestateProvider
}

func NewPrestateValidator(loader PrestateLoader, provider types.PrestateProvider) *PrestateValidator {
	return &PrestateValidator{
		load:     loader,
		provider: provider,
	}
}

func (v *PrestateValidator) Validate(ctx context.Context) error {
	prestateHash, err := v.load(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prestate hash from loader: %w", err)
	}
	prestateCommitment, err := v.provider.AbsolutePreStateCommitment(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch provider's prestate hash: %w", err)
	}
	if !bytes.Equal(prestateCommitment[:], prestateHash[:]) {
		return fmt.Errorf("provider's absolute prestate does not match contract's absolute prestate: Provider: %s | Contract: %s", prestateCommitment.Hex(), prestateHash.Hex())
	}
	return nil
}
