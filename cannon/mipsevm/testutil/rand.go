package testutil

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func RandHash(r *rand.Rand) common.Hash {
	var bytes [32]byte
	_, err := r.Read(bytes[:])
	if err != nil {
		panic(err)
	}
	return bytes
}

func RandHint(r *rand.Rand) []byte {
	count := r.Intn(10)

	bytes := make([]byte, count)
	_, err := r.Read(bytes[:])
	if err != nil {
		panic(err)
	}
	return bytes
}

func RandRegisters(r *rand.Rand) *[32]uint32 {
	registers := new([32]uint32)
	for i := 0; i < 32; i++ {
		registers[i] = r.Uint32()
	}
	return registers
}

func RandomBytes(t require.TestingT, seed int64, length uint32) []byte {
	r := rand.New(rand.NewSource(seed))
	randBytes := make([]byte, length)
	if _, err := r.Read(randBytes); err != nil {
		require.NoError(t, err)
	}
	return randBytes
}

func RandPC(r *rand.Rand) uint32 {
	return AlignPC(r.Uint32())
}

func RandStep(r *rand.Rand) uint64 {
	return BoundStep(r.Uint64())
}
