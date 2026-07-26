package broadcaster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPadGasLimitAppliesProxyAdminFloor(t *testing.T) {
	data := []byte{0x96, 0x23, 0x60, 0x9d}

	require.Equal(t, uint64(ProxyAdminCallGasFloor), padGasLimit(data, 1, false, 30_000_000))
}

func TestPadGasLimitClampsProxyAdminFloorToBlockLimit(t *testing.T) {
	data := []byte{0xf2, 0xfd, 0xe3, 0x8b}

	require.Equal(t, uint64(100_000), padGasLimit(data, 1, false, 100_000))
}

func TestPadGasLimitDoesNotApplyProxyAdminFloorToCreates(t *testing.T) {
	data := []byte{0x96, 0x23, 0x60, 0x9d}

	require.Less(t, padGasLimit(data, 1, true, 30_000_000), uint64(ProxyAdminCallGasFloor))
}
