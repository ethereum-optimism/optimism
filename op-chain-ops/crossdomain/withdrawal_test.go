package crossdomain_test

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func FuzzEncodeDecodeWithdrawal(f *testing.F) {
	f.Fuzz(func(t *testing.T, _nonce, _sender, _target, _value, _gasLimit, data []byte) {
		nonce := new(big.Int).SetBytes(_nonce)
		sender := common.BytesToAddress(_sender)
		target := common.BytesToAddress(_target)
		value := new(big.Int).SetBytes(_value)
		gasLimit := new(big.Int).SetBytes(_gasLimit)

		withdrawal := crossdomain.NewWithdrawal(
			nonce,
			&sender,
			&target,
			value,
			gasLimit,
			data,
		)

		encoded, err := withdrawal.Encode()
		require.Nil(t, err)

		var w crossdomain.Withdrawal
		err = w.Decode(encoded)
		require.Nil(t, err)

		require.Equal(t, withdrawal.Nonce.Uint64(), w.Nonce.Uint64())
		require.Equal(t, withdrawal.Sender, w.Sender)
		require.Equal(t, withdrawal.Target, w.Target)
		require.Equal(t, withdrawal.Value.Uint64(), w.Value.Uint64())
		require.Equal(t, withdrawal.GasLimit.Uint64(), w.GasLimit.Uint64())
		require.Equal(t, withdrawal.Data, w.Data)
	})
}
