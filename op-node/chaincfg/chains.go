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

var NetworksByName = map[string]rollup.Config{
	"goerli": Goerli,
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
