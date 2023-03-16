package crossdomain_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

var big25Million = big.NewInt(25_000_000)

func TestMigrateWithdrawal(t *testing.T) {
	withdrawals := make([]*crossdomain.LegacyWithdrawal, 0)

	for _, receipt := range receipts {
		msg, err := findCrossDomainMessage(receipt)
		require.Nil(t, err)
		legacyWithdrawal := toWithdrawal(t, predeploys.L2CrossDomainMessengerAddr, msg)
		withdrawals = append(withdrawals, legacyWithdrawal)
	}

	l1CrossDomainMessenger := common.HexToAddress("0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1")
	for i, legacy := range withdrawals {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			withdrawal, err := crossdomain.MigrateWithdrawal(legacy, &l1CrossDomainMessenger)
			require.Nil(t, err)
			require.NotNil(t, withdrawal)

			require.Equal(t, legacy.XDomainNonce.Uint64(), withdrawal.Nonce.Uint64())
			require.Equal(t, *withdrawal.Sender, predeploys.L2CrossDomainMessengerAddr)
			require.Equal(t, *withdrawal.Target, l1CrossDomainMessenger)
			// Always equal to or lower than the cap
			require.True(t, withdrawal.GasLimit.Cmp(big25Million) <= 0)
		})
	}
}

// TestMigrateWithdrawalGasLimitMax computes the migrated withdrawal
// gas limit with a very large amount of data. The max value for a migrated
// withdrawal's gas limit is 25 million.
func TestMigrateWithdrawalGasLimitMax(t *testing.T) {
	size := 300_000_000 / 16
	data := make([]byte, size)
	for _, i := range data {
		data[i] = 0xff
	}

	result := crossdomain.MigrateWithdrawalGasLimit(data)
	require.Equal(t, result, big25Million.Uint64())
}

// TestMigrateWithdrawalGasLimit tests an assortment of zero and non zero
// bytes when computing the migrated withdrawal's gas limit.
func TestMigrateWithdrawalGasLimit(t *testing.T) {
	tests := []struct {
		input  []byte
		output uint64
	}{
		{
			input:  []byte{},
			output: 200_000,
		},
		{
			input:  []byte{0xff},
			output: 200_000 + 16,
		},
		{
			input:  []byte{0xff, 0x00},
			output: 200_000 + 16 + 4,
		},
		{
			input:  []byte{0x00},
			output: 200_000 + 4,
		},
		{
			input:  []byte{0x00, 0x00, 0x00},
			output: 200_000 + 4 + 4 + 4,
		},
	}

	for _, test := range tests {
		result := crossdomain.MigrateWithdrawalGasLimit(test.input)
		require.Equal(t, test.output, result)
	}
}
