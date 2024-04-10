package state_test

import (
	crand "crypto/rand"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestAddBalance(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	db := state.NewMemoryStateDB(nil)

	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)
		value := uint256.NewInt(uint64(rng.Intn(1000)))

		db.CreateAccount(addr)
		db.AddBalance(addr, value)

		account := db.GetAccount(addr)
		require.NotNil(t, account)
		require.Equal(t, uint256.MustFromBig(account.Balance), value)
	}
}

func TestCode(t *testing.T) {
	t.Parallel()

	db := state.NewMemoryStateDB(nil)

	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		addr := crypto.PubkeyToAddress(key.PublicKey)

		db.CreateAccount(addr)

		pre := db.GetCode(addr)
		require.Nil(t, pre)

		code := make([]byte, rand.Intn(1024))
		_, err := crand.Read(code)
		require.NoError(t, err)

		db.SetCode(addr, code)

		post := db.GetCode(addr)
		if len(code) == 0 {
			require.Nil(t, post)
		} else {
			require.Equal(t, post, code)
		}

		size := db.GetCodeSize(addr)
		require.Equal(t, size, len(code))

		codeHash := db.GetCodeHash(addr)
		require.Equal(t, codeHash, common.BytesToHash(crypto.Keccak256(code)))
	}
}

func BigEqual(a, b *big.Int) bool {
	if a == nil || b == nil {
		return a == b
	} else {
		return a.Cmp(b) == 0
	}
}
