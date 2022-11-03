package derive

import (
	"github.com/ethereum-optimism/optimism/op-node/testutils/fuzzerutils"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"testing"
)

// FuzzBatchRoundTrip executes a fuzz test similar to TestBatchRoundTrip, which tests that arbitrary BatchData will be
// encoded and decoded without loss of its original values.
func FuzzBatchRoundTrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		// Create our fuzzer wrapper to generate complex values
		typeProvider := fuzz.NewFromGoFuzz(fuzzedData).NilChance(0).MaxDepth(10000).NumElements(0, 0x100).AllowUnexportedFields(true)
		fuzzerutils.AddFuzzerFunctions(typeProvider)

		// Create our batch data from fuzzed data
		var batchData BatchData
		typeProvider.Fuzz(&batchData)

		// Encode our batch data
		enc, err := batchData.MarshalBinary()
		assert.NoError(t, err)

		// Decode our encoded batch data
		var dec BatchData
		err = dec.UnmarshalBinary(enc)
		assert.NoError(t, err)

		// Ensure the round trip encoding of batch data did not result in data loss
		assert.Equal(t, &batchData, &dec, "round trip batch encoding/decoding did not match original values")
	})
}
