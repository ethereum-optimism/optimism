package preimage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithVerification(t *testing.T) {
	validData := []byte{1, 2, 3, 4, 5, 6}
	keccak256Key := Keccak256Key(Keccak256(validData))
	anError := errors.New("boom")

	tests := []struct {
		name         string
		key          Key
		data         []byte
		err          error
		expectedErr  error
		expectedData []byte
	}{
		{
			name:         "LocalKey NoVerification",
			key:          LocalIndexKey(1),
			data:         []byte{4, 3, 5, 7, 3},
			expectedData: []byte{4, 3, 5, 7, 3},
		},
		{
			name:         "Keccak256 Valid",
			key:          keccak256Key,
			data:         validData,
			expectedData: validData,
		},
		{
			name:        "Keccak256 Error",
			key:         keccak256Key,
			data:        validData,
			err:         anError,
			expectedErr: anError,
		},
		{
			name:        "Keccak256 InvalidData",
			key:         keccak256Key,
			data:        []byte{6, 7, 8},
			expectedErr: ErrIncorrectData,
		},
		{
			name:        "EmptyData",
			key:         keccak256Key,
			data:        []byte{},
			expectedErr: ErrIncorrectData,
		},
		{
			name:        "UnknownKey",
			key:         invalidKey([32]byte{0xaa}),
			data:        []byte{},
			expectedErr: ErrUnsupportedKeyType,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			source := WithVerification(func(key [32]byte) ([]byte, error) {
				return test.data, test.err
			})
			actual, err := source(test.key.PreimageKey())
			require.ErrorIs(t, err, test.expectedErr)
			require.Equal(t, test.expectedData, actual)
		})
	}
}

type invalidKey [32]byte

func (k invalidKey) PreimageKey() (out [32]byte) {
	out = k            // copy the source hash
	out[0] = byte(254) // apply invalid prefix
	return
}
