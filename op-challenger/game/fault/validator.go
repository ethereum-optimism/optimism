package fault

import (
	"bytes"
	"context"
	"fmt"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

type PrestateLoader = func(ctx context.Context) (common.Hash, error)

type Validator interface {
	Validate(ctx context.Context) error
}

var _ Validator = (*PrestateValidator)(nil)

type PrestateValidator struct {
	valueName string
	load      PrestateLoader
	provider  types.PrestateProvider
}

func NewPrestateValidator(valueName string, contractProvider PrestateLoader, localProvider types.PrestateProvider) *PrestateValidator {
	return &PrestateValidator{
		valueName: valueName,
		load:      contractProvider,
		provider:  localProvider,
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
		return fmt.Errorf("%v %w: Provider: %s | Contract: %s",
			v.valueName, gameTypes.ErrInvalidPrestate, prestateCommitment.Hex(), prestateHash.Hex())
	}
	return nil
}
