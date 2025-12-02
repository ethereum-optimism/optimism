package crossdomain_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// FuzzEncodeDecodeWithdrawal will fuzz encoding and decoding of a Withdrawal
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

// TestWithdrawalHashing will test the correct computation of Withdrawal hashes
// and the storage slot that the withdrawal hash is stored in. Test vectors
// generated with forge
func TestWithdrawalHashing(t *testing.T) {
	type expect struct {
		Hash common.Hash
		Slot common.Hash
	}

	cases := []struct {
		Withdrawal *crossdomain.Withdrawal
		Expect     expect
	}{
		{
			Withdrawal: crossdomain.NewWithdrawal(
				big.NewInt(0),
				ptr(common.HexToAddress("0xaa179e0640054db6ba4fe9b291dd3b248f4b4960")),
				ptr(common.HexToAddress("0x9b2b72e299e04f00fc5b386972d8951bb870d65e")),
				big.NewInt(1),
				decimalStringToBig("124808255574871339965699013847079823271"),
				hexutil.MustDecode("0x2e1d8f26c6611c04d9f8ea352444b9d366f76c19897c851f5ce9a4d650cf2355f92da68491af279f78110a31c6cb26db09b20b3b1307ff99be0bc410d8bf6994b0e87ced86b747773597dfd1da84268508e34a46a087088ed9276738ffe39e7a1264"),
			),
			Expect: expect{
				Hash: common.HexToHash("0xbddee6e1e89962069cb559abae8342ea3490f9488509c22c482c4ba73988165c"),
				Slot: common.HexToHash("0x26bea3ec4f60cfc1152358454086b7f6a3b669d84a0ec088b2e316ff88c2a892"),
			},
		},
		{
			Withdrawal: crossdomain.NewWithdrawal(
				big.NewInt(0),
				ptr(common.HexToAddress("0x00000000000000000000000000000000000011bc")),
				ptr(common.HexToAddress("0x00000000000000000000000000000000000033eb")),
				big.NewInt(26),
				decimalStringToBig("22338"),
				hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000004"),
			),
			Expect: expect{
				Hash: common.HexToHash("0x65768976d27ba8a7f91c5b267b97d29830103171863c0ba24f3234ef07d0f8e3"),
				Slot: common.HexToHash("0xd73bc49fa8e52d7717fb65cbec7ff0e30bf4e2fbbd38924d1b2efa1f96381517"),
			},
		},
		{
			Withdrawal: crossdomain.NewWithdrawal(
				big.NewInt(0),
				ptr(common.HexToAddress("0x4b0ca57cb88a41771d2cc24ac9fd50afeaa3eedd")),
				ptr(common.HexToAddress("0x8a5e8410b2c3e1036c49ff8acae1e659e2508200")),
				big.NewInt(3),
				decimalStringToBig("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				hexutil.MustDecode("0xce6b96a23be7a1ac1de74f3202dfc4cedaef69502204c0d92f7b352a837a"),
			),
			Expect: expect{
				Hash: common.HexToHash("0x4ba164b689ac62c27c68f41b5f3c4731eb2c25c2d39e4aadcc413d150764624f"),
				Slot: common.HexToHash("0xf055f7cec6a95c9bfc93fc2dc0262d2323a7d4e74af5ee608f0fe2acc83fa1ef"),
			},
		},
	}

	for i, test := range cases {
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			hash, err := test.Withdrawal.Hash()
			require.Nil(t, err)
			require.Equal(t, hash, test.Expect.Hash)

			slot, err := test.Withdrawal.StorageSlot()
			require.Nil(t, err)
			require.Equal(t, slot, test.Expect.Slot)
		})
	}
}

func decimalStringToBig(n string) *big.Int {
	ret, ok := new(big.Int).SetString(n, 10)
	if !ok {
		panic("")
	}
	return ret
}

func ptr(i common.Address) *common.Address {
	return &i
}
