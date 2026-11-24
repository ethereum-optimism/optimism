package fees

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

const (
	// IsthmusL1AttributesLen is the length of the Isthmus-style L1-attributes deposit calldata.
	IsthmusL1AttributesLen = 176
	// JovianL1AttributesLen is the length of the Jovian-style L1-attributes deposit calldata,
	// which appends the uint16 daFootprintGasScalar to the Isthmus layout.
	JovianL1AttributesLen = 178
)

// jovianL1AttributesSelector is the function selector indicating Jovian style L1 gas attributes.
const jovianL1AttributesSelector uint32 = 0x3db6be2b

// ExtractDAFootprintGasScalar extracts the DA footprint gas scalar from the L1 attributes
// transaction data of a Jovian-enabled block.
func ExtractDAFootprintGasScalar(data []byte) (uint16, error) {
	if len(data) < JovianL1AttributesLen {
		return 0, fmt.Errorf("L1 attributes transaction data too short for DA footprint gas scalar: %d", len(data))
	}
	// Future forks need to be added here
	if binary.BigEndian.Uint32(data[:4]) != jovianL1AttributesSelector {
		return 0, errors.New("L1 attributes transaction data does not have Jovian selector")
	}
	daFootprintGasScalar := binary.BigEndian.Uint16(data[JovianL1AttributesLen-2 : JovianL1AttributesLen])
	return daFootprintGasScalar, nil
}

// CalcDAFootprint calculates the total DA footprint of a block for an OP Stack chain from
// the block's binary-encoded transactions, which start with the L1-attributes deposit. The
// entries must be the canonical encodings of a valid block's transactions: only the first is
// decoded, the others are classified by their type byte and measured as they are.
// Jovian introduces a DA footprint block limit which is stored in the BlobGasUsed header field
// and that is taken into account during base fee updates.
// CalcDAFootprint must not be called for pre-Jovian blocks.
// Deposit and post-exec transactions do not contribute to the footprint.
func CalcDAFootprint(txs []hexutil.Bytes) (uint64, error) {
	for i, tx := range txs {
		if len(tx) == 0 {
			return 0, fmt.Errorf("empty transaction at index %d", i)
		}
	}
	if len(txs) == 0 || !optypes.IsDepositTxBytes(txs[0]) {
		return 0, errors.New("missing deposit transaction")
	}
	l1Attributes, err := optypes.UnmarshalDepositTx(txs[0])
	if err != nil {
		return 0, fmt.Errorf("invalid L1-attributes deposit: %w", err)
	}

	// First Jovian block doesn't set the DA footprint gas scalar yet and
	// it must not have user transactions.
	data := l1Attributes.Data
	if len(data) == IsthmusL1AttributesLen {
		if !optypes.IsDepositTxBytes(txs[len(txs)-1]) {
			// sufficient to check last transaction because deposits precede non-deposit txs
			return 0, errors.New("unexpected non-deposit transactions in Jovian activation block")
		}
		return 0, nil
	} // ExtractDAFootprintGasScalar catches all invalid lengths

	daFootprintGasScalar, err := ExtractDAFootprintGasScalar(data)
	if err != nil {
		return 0, err
	}
	var daFootprint uint64
	for _, tx := range txs {
		if optypes.IsDepositTxBytes(tx) || optypes.IsPostExecTxBytes(tx) {
			continue
		}
		daFootprint += bigs.Uint64Strict(NewRollupCostData(tx).EstimatedDASize()) * uint64(daFootprintGasScalar)
	}
	return daFootprint, nil
}
