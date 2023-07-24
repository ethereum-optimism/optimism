package test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault"
)

func NewAlphabetClaimBuilder(t *testing.T, maxDepth int) *ClaimBuilder {
	alphabetProvider := &alphabetWithProofProvider{fault.NewAlphabetProvider("abcdefghijklmnopqrstuvwxyz", uint64(maxDepth))}
	return NewClaimBuilder(t, maxDepth, alphabetProvider)
}

type alphabetWithProofProvider struct {
	*fault.AlphabetProvider
}

func (a *alphabetWithProofProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetProvider.GetPreimage(i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}
