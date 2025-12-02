package derive

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils/fuzzerutils"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
)

// FuzzBatchRoundTrip executes a fuzz test similar to TestBatchRoundTrip, which tests that arbitrary BatchData will be
// encoded and decoded without loss of its original values.
// Does not test the span batch type because the fuzzer is not aware of the structure of a span batch.
func FuzzBatchRoundTrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, fuzzedData []byte) {
		// Create our fuzzer wrapper to generate complex values
		typeProvider := fuzz.NewFromGoFuzz(fuzzedData).NilChance(0).MaxDepth(10000).NumElements(0, 0x100).AllowUnexportedFields(true)
		fuzzerutils.AddFuzzerFunctions(typeProvider)

		var singularBatch SingularBatch
		typeProvider.Fuzz(&singularBatch)

		// Create our batch data from fuzzed data
		var batchData BatchData
		// force batchdata to only contain singular batch
		batchData.inner = &singularBatch

		// Encode our batch data
		enc, err := batchData.MarshalBinary()
		require.NoError(t, err)

		// Decode our encoded batch data
		var dec BatchData
		err = dec.UnmarshalBinary(enc)
		require.NoError(t, err)

		// Ensure the round trip encoding of batch data did not result in data loss
		require.Equal(t, &batchData, &dec, "round trip batch encoding/decoding did not match original values")
	})
}
