package ether

import (
	"testing"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/stretchr/testify/require"
)

func TestCalcLegacyProxyKey(t *testing.T) {
	expectedOwnerSlot := libcommon.HexToHash("0x3260c767fcfbc5a878cdd765d557c2dc0ec469dd5a59ab1a2625587d230ef95f")
	expectedImplSlot := libcommon.HexToHash("0x77c70ab2411972e3fdfbab35b6ae1519d867baa21725dd08c381964443dcc9aa")
	actualOwnerSlot := CalcLegacyProxyKey("proxyOwner", libcommon.Big0)
	actualImplSlot := CalcLegacyProxyKey("proxyTarget", libcommon.Big0)
	require.Equal(t, expectedOwnerSlot, actualOwnerSlot)
	require.Equal(t, expectedImplSlot, actualImplSlot)
}
