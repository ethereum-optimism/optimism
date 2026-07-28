package chaincfg

import (
	"math/big"
	"testing"

	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestGetRollupConfig tests that the configs sourced from the superchain-registry match
// the configs that were embedded in the op-node manually before the superchain-registry was utilized.
//
// The superchain-registry repository is a work in progress.
// At a later date, it will be proposed to, and must be approved by, Optimism Governance.
// Until that time, the configuration described in the superchain-registry is subject to change.
//
// This test ensures no op-node config-loading behavior changes before
// the superchain-registry is no longer deemed experimental.
func TestGetRollupConfig(t *testing.T) {
	configsByName := map[string]rollup.Config{
		"mainnet":                           mainnetCfg,
		"sepolia":                           sepoliaCfg,
		"sepolia-devnet-2-sepolia-devnet-2": sepoliaDevnet2Cfg,
	}

	for name, expectedCfg := range configsByName {
		t.Run(name, func(t *testing.T) {
			gotCfg, err := GetRollupConfig(name)
			require.NoError(t, err)
			require.Equalf(t, expectedCfg, *gotCfg, "rollup-configs from superchain-registry must match for %v", name)
		})
	}
}

func TestKarstUpgradeGasCompatibilityByNetwork(t *testing.T) {
	// These values preserve each chain's behavior when it activated Karst.
	tests := []struct {
		network string
		want    bool
	}{
		{network: "mode-mainnet", want: false},
		{network: "metal-mainnet", want: false},
		{network: "zora-mainnet", want: false},
		{network: "op-mainnet", want: true},
	}

	for _, test := range tests {
		t.Run(test.network, func(t *testing.T) {
			cfg, err := GetRollupConfig(test.network)
			require.NoError(t, err)
			require.Equal(t, test.want, cfg.KeepKarstUpgradeGas)
		})
	}
}

var defaultOpConfig = &opparams.OptimismConfig{
	EIP1559Elasticity:        6,
	EIP1559Denominator:       50,
	EIP1559DenominatorCanyon: u64Ptr(250),
}

var mainnetCfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x438335a20d98863a4c0c97999eb2481921ccd28553eac6f913af7c12aec04108"),
			Number: 17422590,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0xdbf6a80fef073de06add9b0d14026d6e5a86c85f6d102c36d3d8e9cf89c2afd3"),
			Number: 105235063,
		},
		L2Time: 1686068903,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x6887246668a3b87f54deb3b94ba47a6f63f32985"),
			Overhead:    eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bc")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000a6fe0")),
			GasLimit:    30_000_000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      600,
	SeqWindowSize:          3600,
	ChannelTimeoutBedrock:  300,
	L1ChainID:              big.NewInt(1),
	L2ChainID:              big.NewInt(10),
	BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000010"),
	DepositContractAddress: common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
	L1SystemConfigAddress:  common.HexToAddress("0x229047fed2591dbec1eF1118d64F7aF3dB9EB290"),
	RegolithTime:           u64Ptr(0),
	CanyonTime:             u64Ptr(1704992401),
	DeltaTime:              u64Ptr(1708560000),
	EcotoneTime:            u64Ptr(1710374401),
	FjordTime:              u64Ptr(1720627201),
	GraniteTime:            u64Ptr(1726070401),
	HoloceneTime:           u64Ptr(1736445601),
	IsthmusTime:            u64Ptr(1746806401),
	JovianTime:             u64Ptr(1764691201),
	KarstTime:              u64Ptr(1783526401),
	KeepKarstUpgradeGas:    true,
	ChainOpConfig:          defaultOpConfig,
}

var sepoliaCfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x48f520cf4ddaf34c8336e6e490632ea3cf1e5e93b0b2bc6e917557e31845371b"),
			Number: 4071408,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d"),
			Number: 0,
		},
		L2Time: 1691802540,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x8F23BB38F531600e5d8FDDaAEC41F13FaB46E98c"),
			Overhead:    eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bc")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000a6fe0")),
			GasLimit:    30000000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      600,
	SeqWindowSize:          3600,
	ChannelTimeoutBedrock:  300,
	L1ChainID:              big.NewInt(11155111),
	L2ChainID:              big.NewInt(11155420),
	BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000011155420"),
	DepositContractAddress: common.HexToAddress("0x16fc5058f25648194471939df75cf27a2fdc48bc"),
	L1SystemConfigAddress:  common.HexToAddress("0x034edd2a225f7f429a63e0f1d2084b9e0a93b538"),
	RegolithTime:           u64Ptr(0),
	CanyonTime:             u64Ptr(1699981200),
	DeltaTime:              u64Ptr(1703203200),
	EcotoneTime:            u64Ptr(1708534800),
	FjordTime:              u64Ptr(1716998400),
	GraniteTime:            u64Ptr(1723478400),
	HoloceneTime:           u64Ptr(1732633200),
	PectraBlobScheduleTime: u64Ptr(1742486400),
	IsthmusTime:            u64Ptr(1744905600),
	JovianTime:             u64Ptr(1763568001),
	KarstTime:              u64Ptr(1781712001),
	KeepKarstUpgradeGas:    true,
	ChainOpConfig:          defaultOpConfig,
}

var sepoliaDevnet2Cfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x46fb808baa0c4cf0b71e2f85bff8529011a84944529ccbf257e9c82ca88200ba"),
			Number: 11231178,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x782e7ef559d86bd85ca14b3b51bcd64b2a5abba5bf3017d83d34e666d8dc0a57"),
			Number: 0,
		},
		L2Time: 1783536096,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x62a6Ed699757EFA7C6D7E14e11AFf684E904E91a"),
			Scalar:      eth.Bytes32(common.HexToHash("0x010000000000000000000000000000000000000000000000000c3c9d00000558")),
			GasLimit:    60000000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      600,
	SeqWindowSize:          3600,
	ChannelTimeoutBedrock:  300,
	L1ChainID:              big.NewInt(11155111),
	L2ChainID:              big.NewInt(420130015),
	BatchInboxAddress:      common.HexToAddress("0x00ab2D21A3e869A42603c37731699D1EedF89eB3"),
	DepositContractAddress: common.HexToAddress("0xCe313e6d194260417FF9Fee5C58f487D1da9fce0"),
	L1SystemConfigAddress:  common.HexToAddress("0x5F91Ea5EEA70E505b457A442Dc7A8e5D9641b937"),
	RegolithTime:           u64Ptr(0),
	CanyonTime:             u64Ptr(0),
	DeltaTime:              u64Ptr(0),
	EcotoneTime:            u64Ptr(0),
	FjordTime:              u64Ptr(0),
	GraniteTime:            u64Ptr(0),
	HoloceneTime:           u64Ptr(0),
	IsthmusTime:            u64Ptr(0),
	JovianTime:             u64Ptr(0),
	KarstTime:              u64Ptr(0),
	ChainOpConfig:          defaultOpConfig,
}

func u64Ptr(v uint64) *uint64 {
	return &v
}
