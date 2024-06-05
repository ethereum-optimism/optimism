package chaincfg

import (
	"math/big"
	"testing"

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
		"mainnet":                       mainnetCfg,
		"sepolia":                       sepoliaCfg,
		"oplabs-devnet-0-sepolia-dev-0": sepoliaDev0Cfg,
		"boba-sepolia":                  bobaSepoliaCfg,
		"boba-mainnet":                  bobaMainnetCfg,
	}

	for name, expectedCfg := range configsByName {
		gotCfg, err := GetRollupConfig(name)
		require.NoError(t, err)

		require.Equalf(t, expectedCfg, *gotCfg, "rollup-configs from superchain-registry must match for %v", name)
	}
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
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(1),
	L2ChainID:               big.NewInt(10),
	BatchInboxAddress:       common.HexToAddress("0xff00000000000000000000000000000000000010"),
	DepositContractAddress:  common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
	L1SystemConfigAddress:   common.HexToAddress("0x229047fed2591dbec1eF1118d64F7aF3dB9EB290"),
	RegolithTime:            u64Ptr(0),
	CanyonTime:              u64Ptr(1704992401),
	DeltaTime:               u64Ptr(1708560000),
	EcotoneTime:             u64Ptr(1710374401),
	FjordTime:               u64Ptr(1720627201),
	ProtocolVersionsAddress: common.HexToAddress("0x8062AbC286f5e7D9428a0Ccb9AbD71e50d93b935"),
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
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(11155111),
	L2ChainID:               big.NewInt(11155420),
	BatchInboxAddress:       common.HexToAddress("0xff00000000000000000000000000000011155420"),
	DepositContractAddress:  common.HexToAddress("0x16fc5058f25648194471939df75cf27a2fdc48bc"),
	L1SystemConfigAddress:   common.HexToAddress("0x034edd2a225f7f429a63e0f1d2084b9e0a93b538"),
	RegolithTime:            u64Ptr(0),
	CanyonTime:              u64Ptr(1699981200),
	DeltaTime:               u64Ptr(1703203200),
	EcotoneTime:             u64Ptr(1708534800),
	FjordTime:               u64Ptr(1716998400),
	ProtocolVersionsAddress: common.HexToAddress("0x79ADD5713B383DAa0a138d3C4780C7A1804a8090"),
}

var sepoliaDev0Cfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x5639be97000fec7131a880b19b664cae43f975c773f628a08a9bb658c2a68df0"),
			Number: 5173577,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x027ae1f4f9a441f9c8a01828f3b6d05803a0f524c07e09263264a38b755f804b"),
			Number: 0,
		},
		L2Time: 1706484048,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x19cc7073150d9f5888f09e0e9016d2a39667df14"),
			Overhead:    eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bc")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000a6fe0")),
			GasLimit:    30000000,
		},
	},
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(11155111),
	L2ChainID:               big.NewInt(11155421),
	BatchInboxAddress:       common.HexToAddress("0xff00000000000000000000000000000011155421"),
	DepositContractAddress:  common.HexToAddress("0x76114bd29dFcC7a9892240D317E6c7C2A281Ffc6"),
	L1SystemConfigAddress:   common.HexToAddress("0xa6b72407e2dc9EBF84b839B69A24C88929cf20F7"),
	RegolithTime:            u64Ptr(0),
	CanyonTime:              u64Ptr(0),
	DeltaTime:               u64Ptr(0),
	EcotoneTime:             u64Ptr(1706634000),
	FjordTime:               u64Ptr(1715961600),
	ProtocolVersionsAddress: common.HexToAddress("0x252CbE9517F731C618961D890D534183822dcC8d"),
}

var bobaSepoliaCfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x632d8caedbfd573e09c1b49134bd5147147e0904e0f04eba15c662be0258f517"),
			Number: 5109513,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x097654c4c932c97808933b42179388f7bbcefaed3bd93fdf69157e19f1deea0e"),
			Number: 511,
		},
		L2Time: 1705600788,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0xf598b6388eC06945021699F0bbb23dfCFc5edbE8"),
			Overhead:    eth.Bytes32(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000834")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000f4240")),
			GasLimit:    30000000,
		},
	},
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(11155111),
	L2ChainID:               big.NewInt(28882),
	BatchInboxAddress:       common.HexToAddress("0xfff0000000000000000000000000000000028882"),
	DepositContractAddress:  common.HexToAddress("0xB079E6FA9B3eb072fEbf7F746044834eab308dB6"),
	L1SystemConfigAddress:   common.HexToAddress("0xfdc9bce032cef55a71b4fde9b9a2198ad1551965"),
	RegolithTime:            u64Ptr(1705600788),
	CanyonTime:              u64Ptr(1705600788),
	DeltaTime:               u64Ptr(1709078400),
	EcotoneTime:             u64Ptr(1709078400),
	FjordTime:               nil,
	ProtocolVersionsAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

var bobaMainnetCfg = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x945d6244d259e63892abf93e5e6dd3388b79e25ae5ec0502e290a0d0163aa5cf"),
			Number: 19670718,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x0a555516317be2719d9befcbcca5f5516b6b7ce0f05b759f5a166b697a8a0fbd"),
			Number: 1149019,
		},
		L2Time: 1713302879,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0xe1b64045351b0b6e9821f19b39f81bc4711d2230"),
			Overhead:    eth.Bytes32(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000834")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000f4240")),
			GasLimit:    30000000,
		},
	},
	BlockTime:               2,
	MaxSequencerDrift:       600,
	SeqWindowSize:           3600,
	ChannelTimeout:          300,
	L1ChainID:               big.NewInt(1),
	L2ChainID:               big.NewInt(288),
	BatchInboxAddress:       common.HexToAddress("0xfff0000000000000000000000000000000000288"),
	DepositContractAddress:  common.HexToAddress("0x7b02d13904d8e6e0f0efaf756ab14cb0ff21ee7e"),
	L1SystemConfigAddress:   common.HexToAddress("0x158fd5715f16ac1f2dc959a299b383aaaf9b59eb"),
	RegolithTime:            u64Ptr(1713302879),
	CanyonTime:              u64Ptr(1713302879),
	DeltaTime:               u64Ptr(1713302879),
	EcotoneTime:             u64Ptr(1713302880),
	FjordTime:               nil,
	ProtocolVersionsAddress: common.HexToAddress("0x0000000000000000000000000000000000000000"),
}

func u64Ptr(v uint64) *uint64 {
	return &v
}
