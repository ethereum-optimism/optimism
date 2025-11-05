package predeploys

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestGethAddresses(t *testing.T) {
	// We test if the addresses in geth match those in op monorepo, to avoid an import-cycle:
	// we import geth in the monorepo, and do not want to import op monorepo into geth.
	require.Equal(t, L1BlockAddr, types.L1BlockAddr)
}
