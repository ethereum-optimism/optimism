package preimage

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithVerification(t *testing.T) {
	validData := []byte{1, 2, 3, 4, 5, 6}
	keccak256Key := Keccak256Key(Keccak256(validData))
	sha256Key := Sha256Key(sha256.Sum256(validData))
	anError := errors.New("boom")

	validKeys := []Key{keccak256Key, sha256Key}

	type testData struct {
		name         string
		key          Key
		data         []byte
		err          error
		expectedErr  error
		expectedData []byte
	}
	tests := []testData{
		{
			name:         "LocalKey NoVerification",
			key:          LocalIndexKey(1),
			data:         []byte{4, 3, 5, 7, 3},
			expectedData: []byte{4, 3, 5, 7, 3},
		},
		{
			name:         "BlobKey NoVerification",
			key:          BlobKey([32]byte{1, 2, 3, 4}),
			data:         []byte{4, 3, 5, 7, 3},
			expectedData: []byte{4, 3, 5, 7, 3},
		},
		{
			name:         "KZGPointEvaluationKey NoVerification",
			key:          PrecompileKey([32]byte{1, 2, 3, 4}),
			data:         []byte{4, 3, 5, 7, 3},
			expectedData: []byte{4, 3, 5, 7, 3},
		},
		{
			name:        "UnknownKey",
			key:         invalidKey([32]byte{0xaa}),
			data:        []byte{},
			expectedErr: ErrUnsupportedKeyType,
		},
	}

	for _, key := range validKeys {
		name := reflect.TypeOf(key).Name()
		tests = append(tests,
			testData{
				name:         fmt.Sprintf("%v-Valid", name),
				key:          key,
				data:         validData,
				expectedData: validData,
			},
			testData{
				name:        fmt.Sprintf("%v-Error", name),
				key:         key,
				data:        validData,
				err:         anError,
				expectedErr: anError,
			},
			testData{
				name:        fmt.Sprintf("%v-InvalidData", name),
				key:         key,
				data:        []byte{6, 7, 8},
				expectedErr: ErrIncorrectData,
			},
			testData{
				name:        fmt.Sprintf("%v-EmptyData", name),
				key:         key,
				data:        []byte{},
				expectedErr: ErrIncorrectData,
			})
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
