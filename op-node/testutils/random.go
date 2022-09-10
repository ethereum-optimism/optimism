package testutils

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func RandomHash(rng *rand.Rand) (out common.Hash) {
	rng.Read(out[:])
	return
}

func RandomAddress(rng *rand.Rand) (out common.Address) {
	rng.Read(out[:])
	return
}

func RandomETH(rng *rand.Rand, max int64) *big.Int {
	x := big.NewInt(rng.Int63n(max))
	x = new(big.Int).Mul(x, big.NewInt(1e18))
	return x
}

func RandomKey() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}
	return key
}

func RandomData(rng *rand.Rand, size int) []byte {
	out := make([]byte, size)
	rng.Read(out)
	return out
}

func RandomBlockID(rng *rand.Rand) eth.BlockID {
	return eth.BlockID{
		Hash:   RandomHash(rng),
		Number: rng.Uint64() & ((1 << 50) - 1), // be json friendly
	}
}

func RandomBlockRef(rng *rand.Rand) eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       RandomHash(rng),
		Number:     rng.Uint64(),
		ParentHash: RandomHash(rng),
		Time:       rng.Uint64(),
	}
}

func NextRandomRef(rng *rand.Rand, parent eth.L1BlockRef) eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       RandomHash(rng),
		Number:     parent.Number + 1,
		ParentHash: parent.Hash,
		Time:       parent.Time + uint64(rng.Intn(100)),
	}
}

func RandomL2BlockRef(rng *rand.Rand) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:           RandomHash(rng),
		Number:         rng.Uint64(),
		ParentHash:     RandomHash(rng),
		Time:           rng.Uint64(),
		L1Origin:       RandomBlockID(rng),
		SequenceNumber: rng.Uint64(),
	}
}

func NextRandomL2Ref(rng *rand.Rand, l2BlockTime uint64, parent eth.L2BlockRef, origin eth.BlockID) eth.L2BlockRef {
	seq := parent.SequenceNumber + 1
	if parent.L1Origin != origin {
		seq = 0
	}
	return eth.L2BlockRef{
		Hash:           RandomHash(rng),
		Number:         parent.Number + 1,
		ParentHash:     parent.Hash,
		Time:           parent.Time + l2BlockTime,
		L1Origin:       eth.BlockID{},
		SequenceNumber: seq,
	}
}
