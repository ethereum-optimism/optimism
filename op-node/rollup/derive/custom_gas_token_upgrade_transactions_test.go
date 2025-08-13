package derive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomGasTokenNetworkTransactions(t *testing.T) {
	upgradeTxns, err := CustomGasTokenNetworkUpgradeTransactions()
	require.NoError(t, err)
	// For now, the function returns an empty slice as requested
	require.Len(t, upgradeTxns, 0)
}
