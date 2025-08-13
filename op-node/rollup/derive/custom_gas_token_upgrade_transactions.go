package derive

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// CustomGasTokenNetworkUpgradeTransactions returns the transactions required to upgrade to use a custom gas token.
// For now, this function returns an empty slice of transactions as requested.
func CustomGasTokenNetworkUpgradeTransactions() ([]hexutil.Bytes, error) {
	// TODO: Implement custom gas token upgrade transactions
	// Deploy controller, liquidity, set implementations, mint ETH to liquidity
	upgradeTxns := make([]hexutil.Bytes, 0)
	return upgradeTxns, nil
}
