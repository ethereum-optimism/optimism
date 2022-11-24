package genesis

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CreateReceipts will create the set of bedrock genesis receipts given
// a list of legacy withdrawals.
func CreateReceipts(
	hdr *types.Header,
	withdrawals []*crossdomain.LegacyWithdrawal,
	l1CrossDomainMessenger, l1StandardBridge *common.Address,
) ([]*types.Receipt, error) {
	receipts := make([]*types.Receipt, 0)

	for _, withdrawal := range withdrawals {
		wd, err := crossdomain.MigrateWithdrawal(withdrawal, l1CrossDomainMessenger, l1StandardBridge)
		if err != nil {
			return nil, err
		}

		receipt, err := wd.Receipt(hdr)
		if err != nil {
			return nil, err
		}

		receipts = append(receipts, receipt)
	}

	return receipts, nil
}
