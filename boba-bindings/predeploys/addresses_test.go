package predeploys

import (
	"math/big"
	"testing"

	"github.com/bobanetwork/boba/boba-bindings/bindings"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/stretchr/testify/require"
)

var (
	expectedL1BaseFeeSlot = common.BigToHash(big.NewInt(1))
	expectedOverheadSlot  = common.BigToHash(big.NewInt(5))
	expectedScalarSlot    = common.BigToHash(big.NewInt(6))

	expectedL1BlockAddr = common.HexToAddress("0x4200000000000000000000000000000000000015")
)

func TestGethAddresses(t *testing.T) {
	// We test if the addresses in geth match those in op-bindings, to avoid an import-cycle:
	// we import geth in the monorepo, and do not want to import op-bindings into geth.
	require.Equal(t, L1BlockAddr, expectedL1BlockAddr)
}

// TestL1BlockSlots ensures that the storage layout of the L1Block
// contract matches the hardcoded values in `op-geth`.
func TestL1BlockSlots(t *testing.T) {
	layout, err := bindings.GetStorageLayout("L1Block")
	require.NoError(t, err)

	var l1BaseFeeSlot, overHeadSlot, scalarSlot common.Hash
	for _, entry := range layout.Storage {
		switch entry.Label {
		case "l1FeeOverhead":
			overHeadSlot = common.BigToHash(big.NewInt(int64(entry.Slot)))
		case "l1FeeScalar":
			scalarSlot = common.BigToHash(big.NewInt(int64(entry.Slot)))
		case "basefee":
			l1BaseFeeSlot = common.BigToHash(big.NewInt(int64(entry.Slot)))
		}
	}

	require.Equal(t, expectedOverheadSlot, overHeadSlot)
	require.Equal(t, expectedScalarSlot, scalarSlot)
	require.Equal(t, expectedL1BaseFeeSlot, l1BaseFeeSlot)
}
