package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestForeignBytesRefused feeds the decoder objects it will genuinely be offered by mistake.
//
// This guards a LIVE path: the claim-follower decodes the calldata of every transaction addressed
// to the registry, and anyone can address a transaction there. So the decoder is routinely handed
// bytes that are not a claim, and the only acceptable outcome is refusal — never a partial decode
// that yields eight plausible-looking hashes read out of a completely different object.
//
// The two stored cases are REAL bytes of this project's retired proof-batch wire (see
// testdata/harvested/index.json for provenance), which is the most plausible near-miss an operator
// could produce: length-prefixed framings with a magic and no relationship to an ABI struct.
func TestForeignBytesRefused(t *testing.T) {
	for _, name := range []string{"proof-batch-v3.bin", "proof-batch-v4.bin"} {
		t.Run(name, func(t *testing.T) {
			data := readFile(t, "testdata/harvested/"+name)
			require.Greater(t, len(data), EncodedSizeEmptyProof, "the case is only interesting if it is long enough to try")
			_, err := DecodeMode(data, ModeProven)
			require.Error(t, err)
			_, err = Decode(data)
			require.Error(t, err)
		})
	}
	// Nothing about this decoder is magic-based any more, so the general case matters more than the
	// specific one: arbitrary bytes of the right length must not decode either.
	junk := make([]byte, EncodedSizeEmptyProof)
	for i := range junk {
		junk[i] = 0xcd
	}
	_, err := DecodeMode(junk, ModeProven)
	require.Error(t, err)
}
