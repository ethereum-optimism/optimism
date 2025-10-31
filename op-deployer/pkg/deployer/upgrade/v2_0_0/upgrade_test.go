package v2_0_0

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestUpgradeOPChainInput_OpChainConfigs(t *testing.T) {
	input := &UpgradeOPChainInput{
		Prank: common.Address{0xaa},
		Opcm:  common.Address{0xbb},
		EncodedChainConfigs: []OPChainConfig{
			{
				SystemConfigProxy: common.Address{0x01},
				ProxyAdmin:        common.Address{0x02},
				AbsolutePrestate:  common.Hash{0x03},
			},
			{
				SystemConfigProxy: common.Address{0x04},
				ProxyAdmin:        common.Address{0x05},
				AbsolutePrestate:  common.Hash{0x06},
			},
		},
	}
	data, err := input.OpChainConfigs()
	require.NoError(t, err)
	require.Equal(
		t,
		"0000000000000000000000000000000000000000000000000000000000000020"+
			"0000000000000000000000000000000000000000000000000000000000000002"+
			"0000000000000000000000000100000000000000000000000000000000000000"+
			"0000000000000000000000000200000000000000000000000000000000000000"+
			"0300000000000000000000000000000000000000000000000000000000000000"+
			"0000000000000000000000000400000000000000000000000000000000000000"+
			"0000000000000000000000000500000000000000000000000000000000000000"+
			"0600000000000000000000000000000000000000000000000000000000000000",
		hex.EncodeToString(data),
	)
}
