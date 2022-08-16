package derive

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMarshalBinary(t *testing.T) {
	var a big.Int
	a.Exp(big.NewInt(2), big.NewInt(256), nil) // 2**256
	fee := &a

	var hash []byte

	l1 := L1BlockInfo{
		Number:         0,
		Time:           0,
		BaseFee:        fee,
		BlockHash:      common.BytesToHash(hash),
		SequenceNumber: 0,
	}

	_, err := l1.MarshalBinary()
	require.NoError(t, err)
}

func TestMarshalDepositLogEvent(t *testing.T) {
	var a big.Int
	a.Exp(big.NewInt(2), big.NewInt(256), nil) // 2**256
	big_val := &a

	rng := rand.New(rand.NewSource(1234))
	source := UserDepositSource{
		L1BlockHash: testutils.RandomHash(rng),
		LogIndex:    uint64(rng.Intn(10000)),
	}
	depInput := testutils.GenerateDeposit(source.SourceHash(), rng)
	depInput.Value = big_val
	depInput.Mint = big_val
	MarshalDepositLogEvent(MockDepositContractAddr, depInput)
}
