package preimage

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrIncorrectData      = errors.New("incorrect data")
	ErrUnsupportedKeyType = errors.New("unsupported key type")
)

// WithVerification wraps the supplied source to verify that the returned data is a valid pre-image for the key.
func WithVerification(source PreimageGetter) PreimageGetter {
	return func(key [32]byte) ([]byte, error) {
		data, err := source(key)
		if err != nil {
			return nil, err
		}

		switch KeyType(key[0]) {
		case LocalKeyType:
			return data, nil
		case Keccak256KeyType:
			hash := Keccak256(data)
			if !slices.Equal(hash[1:], key[1:]) {
				return nil, fmt.Errorf("%w for key %v, hash: %v data: %x", ErrIncorrectData, key, hash, data)
			}
			return data, nil
		default:
			return nil, fmt.Errorf("%w: %v", ErrUnsupportedKeyType, key[0])
		}
	}
}
