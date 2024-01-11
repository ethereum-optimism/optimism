package predeploys

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGethAddresses(t *testing.T) {
	// We test if the addresses in geth match those in op-bindings, to avoid an import-cycle:
	// we import geth in the monorepo, and do not want to import op-bindings into geth.
	require.Equal(t, L1BlockAddr, types.L1BlockAddr)
}

func uintToHash(v uint) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(uint64(v)))
}

// TestL1BlockSlots ensures that the storage layout of the L1Block
// contract matches the hardcoded values in `op-geth`.
func TestL1BlockSlots(t *testing.T) {
	layout, err := bindings.GetStorageLayout("L1Block")
	require.NoError(t, err)

	var l1BaseFeeSlot, overHeadSlot, scalarSlot common.Hash
	var l1BaseFeeScalarSlot, l1BlobBaseFeeScalarSlot, blobBaseFeeSlot common.Hash // new in Ecotone
	var l1BaseFeeScalarOffset, l1BlobBaseFeeScalarOffset uint                     // new in Ecotone
	for _, entry := range layout.Storage {
		switch entry.Label {
		case "l1FeeOverhead":
			overHeadSlot = uintToHash(entry.Slot)
		case "l1FeeScalar":
			scalarSlot = uintToHash(entry.Slot)
		case "basefee":
			l1BaseFeeSlot = uintToHash(entry.Slot)
		case "blobBaseFee":
			blobBaseFeeSlot = uintToHash(entry.Slot)
		case "baseFeeScalar":
			l1BaseFeeScalarSlot = uintToHash(entry.Slot)
			l1BaseFeeScalarOffset = entry.Offset
		case "blobBaseFeeScalar":
			l1BlobBaseFeeScalarSlot = uintToHash(entry.Slot)
			l1BlobBaseFeeScalarOffset = entry.Offset
		}
	}

	require.Equal(t, types.OverheadSlot, overHeadSlot)
	require.Equal(t, types.ScalarSlot, scalarSlot)
	require.Equal(t, types.L1BaseFeeSlot, l1BaseFeeSlot)
	// new in Ecotone
	require.Equal(t, types.L1BlobBaseFeeSlot, blobBaseFeeSlot)
	require.Equal(t, types.L1FeeScalarsSlot, l1BaseFeeScalarSlot)
	require.Equal(t, types.L1FeeScalarsSlot, l1BlobBaseFeeScalarSlot)
	require.Equal(t, uint(types.BaseFeeScalarSlotOffset), l1BaseFeeScalarOffset)
	require.Equal(t, uint(types.BlobBaseFeeScalarSlotOffset), l1BlobBaseFeeScalarOffset)
}
