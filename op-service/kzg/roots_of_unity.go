package kzg

import (
	"math/big"
	"math/bits"

	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
)

var RootsOfUnity *[4096]fr.Element

// generateRootsOfUnity generates the 4096th bit-reversed roots of unity used in EIP-4844 as predefined evaluation points.
// To compute the field element at index i in a blob, the blob polynomial is evaluated at the ith root of unity.
// Based on go-kzg-4844: https://github.com/crate-crypto/go-kzg-4844/blob/8bcf6163d3987313a3194595cf1f33fd45d7301a/internal/kzg/domain.go#L44-L98
// Also, see the consensus specs:
//   - compute_roots_of_unity: https://github.com/ethereum/consensus-specs/blob/bf09edef17e2900258f7e37631e9452941c26e86/specs/deneb/polynomial-commitments.md#compute_roots_of_unity
//   - bit-reversal permutation: https://github.com/ethereum/consensus-specs/blob/bf09edef17e2900258f7e37631e9452941c26e86/specs/deneb/polynomial-commitments.md#bit-reversal-permutation
func generateRootsOfUnity() *[4096]fr.Element {
	rootsOfUnity := new([4096]fr.Element)

	const maxOrderRoot uint64 = 32
	var rootOfUnity fr.Element
	_, err := rootOfUnity.SetString("10238227357739495823651030575849232062558860180284477541189508159991286009131")
	if err != nil {
		panic("failed to initialize root of unity")
	}
	// Find generator subgroup of order x.
	// This can be constructed by powering a generator of the largest 2-adic subgroup of order 2^32 by an exponent
	// of (2^32)/x, provided x is <= 2^32.
	logx := uint64(bits.TrailingZeros64(4096))
	expo := uint64(1 << (maxOrderRoot - logx))

	var generator fr.Element
	generator.Exp(rootOfUnity, big.NewInt(int64(expo))) // Domain.Generator has order x now.
	// Compute all relevant roots of unity, i.e. the multiplicative subgroup of size x.
	current := fr.One()
	for i := uint64(0); i < 4096; i++ {
		rootsOfUnity[i] = current
		current.Mul(&current, &generator)
	}
	shiftCorrection := uint64(64 - bits.TrailingZeros64(4096))

	for i := uint64(0); i < 4096; i++ {
		// Find index irev, such that i and irev get swapped
		irev := bits.Reverse64(i) >> shiftCorrection
		if irev > i {
			rootsOfUnity[i], rootsOfUnity[irev] = rootsOfUnity[irev], rootsOfUnity[i]
		}
	}

	return rootsOfUnity
}

func init() {
	RootsOfUnity = generateRootsOfUnity()
}
