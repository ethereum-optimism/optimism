package rollup

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func u64ptr(n uint64) *uint64 {
	return &n
}

var testConfig = Config{
	Genesis: Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108"),
			Number: 17422590,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3"),
			Number: 105235063,
		},
		L2Time: 0,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x6887246668a3b87f54deb3b94ba47a6f63f32985"),
			Overhead:    eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bc")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000a6fe0")),
			GasLimit:    30_000_000,
		},
	},
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(1),
	L2ChainID:               big.NewInt(10),
	RegolithTime:            u64ptr(10),
	CanyonTime:              u64ptr(20),
	DeltaTime:               u64ptr(30),
	EcotoneTime:             u64ptr(40),
	FjordTime:               u64ptr(50),
	InteropTime:             nil,
	BatchInboxAddress:       common.HexToAddress("0xff00000000000000000000000000000000000010"),
	DepositContractAddress:  common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
	L1SystemConfigAddress:   common.HexToAddress("0x229047fed2591dbec1eF1118d64F7aF3dB9EB290"),
	ProtocolVersionsAddress: common.HexToAddress("0x8062AbC286f5e7D9428a0Ccb9AbD71e50d93b935"),
	UsePlasma:               false,
}

func TestCanyonForkActivation(t *testing.T) {
	c := NewChainSpec(&testConfig)
	tests := []struct {
		name     string
		blockNum uint64
		isCanyon bool
	}{
		{"Genesis", 0, false},
		{"CanyonTimeMinusOne", 19, false},
		{"CanyonTime", 20, true},
		{"CanyonTimePlusOne", 21, true},
		{"DeltaTime", 30, true},
		{"EcotoneTime", 40, true},
		{"FjordTime", 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.IsCanyon(tt.blockNum)
			require.Equal(t, tt.isCanyon, result, "Block number %d should be Canyon", tt.blockNum)
		})
	}
}

func TestMaxChannelBankSize(t *testing.T) {
	c := NewChainSpec(&testConfig)
	tests := []struct {
		name        string
		blockNum    uint64
		expected    uint64
		description string
	}{
		{"Genesis", 0, uint64(maxChannelBankSizeBedrock), "Before Fjord activation, should use Bedrock size"},
		{"FjordTimeMinusOne", 49, uint64(maxChannelBankSizeBedrock), "Just before Fjord, should still use Bedrock size"},
		{"FjordTime", 50, uint64(maxChannelBankSizeFjord), "At Fjord activation, should switch to Fjord size"},
		{"FjordTimePlusOne", 51, uint64(maxChannelBankSizeFjord), "After Fjord activation, should use Fjord size"},
		{"NextForkTime", 60, uint64(maxChannelBankSizeFjord), "Well after Fjord, should continue to use Fjord size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.MaxChannelBankSize(tt.blockNum)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestMaxRLPBytesPerChannel(t *testing.T) {
	c := NewChainSpec(&testConfig)
	tests := []struct {
		name        string
		blockNum    uint64
		expected    uint64
		description string
	}{
		{"Genesis", 0, uint64(maxRLPBytesPerChannelBedrock), "Before Fjord activation, should use Bedrock RLP bytes limit"},
		{"FjordTimeMinusOne", 49, uint64(maxRLPBytesPerChannelBedrock), "Just before Fjord, should still use Bedrock RLP bytes limit"},
		{"FjordTime", 50, uint64(maxRLPBytesPerChannelFjord), "At Fjord activation, should switch to Fjord RLP bytes limit"},
		{"FjordTimePlusOne", 51, uint64(maxRLPBytesPerChannelFjord), "After Fjord activation, should use Fjord RLP bytes limit"},
		{"NextForkTime", 60, uint64(maxRLPBytesPerChannelFjord), "Well after Fjord, should continue to use Fjord RLP bytes limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.MaxRLPBytesPerChannel(tt.blockNum)
			require.Equal(t, tt.expected, result, tt.description)
		})
	}
}
