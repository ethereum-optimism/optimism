package fees

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

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

// CalcDAFootprint calculates the total DA footprint of a block for an OP Stack chain.
// Jovian introduces a DA footprint block limit which is stored in the BlobGasUsed header field
// and that is taken into account during base fee updates.
// CalcDAFootprint must not be called for pre-Jovian blocks.
func CalcDAFootprint(txs []*types.Transaction) (uint64, error) {
	if len(txs) == 0 || txs[0].Type() != types.DepositTxType {
		return 0, errors.New("missing deposit transaction")
	}

	// First Jovian block doesn't set the DA footprint gas scalar yet and
	// it must not have user transactions.
	data := txs[0].Data()
	if len(data) == IsthmusL1AttributesLen {
		if txs[len(txs)-1].Type() != types.DepositTxType {
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
		if tx.Type() == types.DepositTxType {
			continue
		}
		daFootprint += bigs.Uint64Strict(TxRollupCostData(tx).EstimatedDASize()) * uint64(daFootprintGasScalar)
	}
	return daFootprint, nil
}
