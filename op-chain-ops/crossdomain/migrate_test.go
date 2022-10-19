package crossdomain_test

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestMigrateWithdrawal(t *testing.T) {
	withdrawals := make([]*crossdomain.LegacyWithdrawal, 0)

	for _, receipt := range receipts {
		msg, err := findCrossDomainMessage(receipt)
		require.Nil(t, err)
		withdrawal, err := msg.ToWithdrawal()
		require.Nil(t, err)
		legacyWithdrawal, ok := withdrawal.(*crossdomain.LegacyWithdrawal)
		require.True(t, ok)
		withdrawals = append(withdrawals, legacyWithdrawal)
	}

	l1CrossDomainMessenger := common.HexToAddress("0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1")
	l1StandardBridge := common.HexToAddress("0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1")

	for i, legacy := range withdrawals {
		t.Run(fmt.Sprintf("test%d", i), func(t *testing.T) {
			withdrawal, err := crossdomain.MigrateWithdrawal(legacy, &l1CrossDomainMessenger, &l1StandardBridge)
			require.Nil(t, err)
			require.NotNil(t, withdrawal)

			require.Equal(t, legacy.Nonce.Uint64(), withdrawal.Nonce.Uint64())
			require.Equal(t, *withdrawal.Sender, predeploys.L2CrossDomainMessengerAddr)
			require.Equal(t, *withdrawal.Target, l1CrossDomainMessenger)
		})
	}
}
