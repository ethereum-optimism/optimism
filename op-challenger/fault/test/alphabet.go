package test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/alphabet"
)

func NewFullAlphabetProvider(t *testing.T, maxDepth int) *alphabetWithProofProvider {
	return &alphabetWithProofProvider{alphabet.NewAlphabetProvider("abcdefghijklmnopqrstuvwxyz", uint64(maxDepth))}
}

func NewAlphabetClaimBuilder(t *testing.T, maxDepth int) *ClaimBuilder {
	return NewClaimBuilder(t, maxDepth, NewFullAlphabetProvider(t, maxDepth))
}

type alphabetWithProofProvider struct {
	*alphabet.AlphabetProvider
}

func (a *alphabetWithProofProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetProvider.GetPreimage(i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}
