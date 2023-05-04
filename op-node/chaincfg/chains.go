package chaincfg

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var Goerli = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x6ffc1bf3754c01f6bb9fe057c1578b87a8571ce2e9be5ca14bace6eccfd336c7"),
			Number: 8300214,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x0f783549ea4313b784eadd9b8e8a69913b368b7366363ea814d7707ac505175f"),
			Number: 4061224,
		},
		L2Time: 1673550516,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x7431310e026B69BFC676C0013E12A1A11411EEc9"),
			Overhead:    eth.Bytes32(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000834")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000f4240")),
			GasLimit:    25_000_000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      600,
	SeqWindowSize:          3600,
	ChannelTimeout:         300,
	L1ChainID:              big.NewInt(5),
	L2ChainID:              big.NewInt(420),
	BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000420"),
	DepositContractAddress: common.HexToAddress("0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383"),
	L1SystemConfigAddress:  common.HexToAddress("0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60"),
	RegolithTime:           u64Ptr(1679079600),
}

var Mainnet = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0xfaa0ca49136c2ed86b6d5d0cf357bc099323d64534eff4501d760186230b0cb9"),
			Number: 16780230,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x8ced54d739f12aec842ee41f50a6e38ef0759690edec99b9ea0f5b5e948d777c"),
			Number: 79149098,
		},
		L2Time: 1683226991,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x70997970c51812dc3a010c7d01b50e0d17dc79c8"),
			Overhead:    eth.Bytes32(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000834")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000f4240")),
			GasLimit:    25_000_000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      600,
	SeqWindowSize:          3600,
	ChannelTimeout:         300,
	L1ChainID:              big.NewInt(1),
	L2ChainID:              big.NewInt(10),
	BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000010"),
	DepositContractAddress: common.HexToAddress("0xb468647b04bf657c9ee2de65252037d781eabafd"),
	L1SystemConfigAddress:  common.HexToAddress("0x408f924baec71cc3968614cb2c58e155a35e6890"),
	RegolithTime:           u64Ptr(0),
}

var NetworksByName = map[string]rollup.Config{
	"goerli":  Goerli,
	"mainnet": Mainnet,
}

var L2ChainIDToNetworkName = func() map[string]string {
	out := make(map[string]string)
	for name, netCfg := range NetworksByName {
		out[netCfg.L2ChainID.String()] = name
	}
	return out
}()

func AvailableNetworks() []string {
	var networks []string
	for name := range NetworksByName {
		networks = append(networks, name)
	}
	return networks
}

func GetRollupConfig(name string) (rollup.Config, error) {
	network, ok := NetworksByName[name]
	if !ok {
		return rollup.Config{}, fmt.Errorf("invalid network %s", name)
	}

	return network, nil
}

func u64Ptr(v uint64) *uint64 {
	return &v
}
