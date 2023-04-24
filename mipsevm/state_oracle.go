package mipsevm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

type StateOracle interface {
	Get(key [32]byte) (a, b [32]byte)
	Remember(a, b [32]byte) [32]byte
}

type StateCache struct {
	data    map[[32]byte][2][32]byte
	reverse map[[2][32]byte][32]byte
}

func NewStateCache() *StateCache {
	return &StateCache{
		data:    make(map[[32]byte][2][32]byte),
		reverse: make(map[[2][32]byte][32]byte),
	}
}

func (s *StateCache) Get(key [32]byte) (a, b [32]byte) {
	ab, ok := s.data[key]
	if !ok {
		panic(fmt.Errorf("missing key %x", key))
	}
	return ab[0], ab[1]
}

func (s *StateCache) Remember(left [32]byte, right [32]byte) [32]byte {
	value := [2][32]byte{left, right}
	if key, ok := s.reverse[value]; ok {
		return key
	}
	key := crypto.Keccak256Hash(left[:], right[:])
	s.data[key] = value
	s.reverse[value] = key
	return key
}
