package keccak

import (
	"context"

	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
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

func (v *PreimageVerifier) Verify(ctx context.Context, oracle keccakTypes.LargePreimageOracle, preimage keccakTypes.LargePreimageMetaData) {
	// No verification currently performed.
}
