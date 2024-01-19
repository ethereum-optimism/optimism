package keccak

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type PreimageVerifier struct {
	log log.Logger
}

func NewPreimageVerifier(logger log.Logger) *PreimageVerifier {
	return &PreimageVerifier{
		log: logger,
	}
}

func (v *PreimageVerifier) Verify(ctx context.Context, oracle types.LargePreimageOracle, preimage types.LargePreimageMetaData) {
	// No verification currently performed.
}
