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
	var l1BasefeeScalarSlot, l1BlobBasefeeScalarSlot, blobBasefeeSlot common.Hash // new in Ecotone
	var l1BasefeeScalarOffset, l1BlobBasefeeScalarOffset uint                     // new in Ecotone
	for _, entry := range layout.Storage {
		switch entry.Label {
		case "l1FeeOverhead":
			overHeadSlot = uintToHash(entry.Slot)
		case "l1FeeScalar":
			scalarSlot = uintToHash(entry.Slot)
		case "basefee":
			l1BaseFeeSlot = uintToHash(entry.Slot)
		case "blobBasefee":
			blobBasefeeSlot = uintToHash(entry.Slot)
		case "basefeeScalar":
			l1BasefeeScalarSlot = uintToHash(entry.Slot)
			l1BasefeeScalarOffset = entry.Offset
		case "blobBasefeeScalar":
			l1BlobBasefeeScalarSlot = uintToHash(entry.Slot)
			l1BlobBasefeeScalarOffset = entry.Offset
		}
	}

	require.Equal(t, types.OverheadSlot, overHeadSlot)
	require.Equal(t, types.ScalarSlot, scalarSlot)
	require.Equal(t, types.L1BasefeeSlot, l1BaseFeeSlot)
	// new in Ecotone
	require.Equal(t, types.L1BlobBasefeeSlot, blobBasefeeSlot)
	require.Equal(t, types.L1FeeScalarsSlot, l1BasefeeScalarSlot)
	require.Equal(t, types.L1FeeScalarsSlot, l1BlobBasefeeScalarSlot)
	require.Equal(t, uint(types.BasefeeScalarSlotOffset), l1BasefeeScalarOffset)
	require.Equal(t, uint(types.BlobBasefeeScalarSlotOffset), l1BlobBasefeeScalarOffset)
}
