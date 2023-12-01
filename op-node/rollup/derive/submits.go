package derive

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/hashicorp/go-multierror"
)

// UserSubmits transforms the L2 block-height and L1 receipts into the transaction inputs for a full L2 block
func UserSubmits(receipts []*types.Receipt, depositContractAddr common.Address) ([]*types.SubmitTx, error) {
	var out []*types.SubmitTx
	var result error
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == depositContractAddr && len(log.Topics) > 0 && log.Topics[0] == SubmitEventABIHash {
				dep, err := UnmarshalSubmitsLogEvent(log)
				if err != nil {
					result = multierror.Append(result, fmt.Errorf("malformatted L1 deposit log in receipt %d, log %d: %w", i, j, err))
				} else {
					out = append(out, dep)
				}
			}
		}
	}
	return out, result
}

func DeriveSubmits(receipts []*types.Receipt, depositContractAddr common.Address) ([]hexutil.Bytes, error) {
	var result error
	userSubmits, err := UserSubmits(receipts, depositContractAddr)
	if err != nil {
		result = multierror.Append(result, err)
	}
	encodedTxs := make([]hexutil.Bytes, 0, len(userSubmits))
	for i, tx := range userSubmits {
		opaqueTx, err := types.NewTx(tx).MarshalBinary()
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to encode user tx %d", i))
		} else {
			encodedTxs = append(encodedTxs, opaqueTx)
		}
	}
	return encodedTxs, result
}
