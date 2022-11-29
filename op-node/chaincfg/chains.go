package chaincfg

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
)

var Beta1 = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x87ba22412f6d081a28ca0d8aafcf630d13a0d2fc16a7345eb3a0d5cd329f935e"),
			Number: 7996739,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0x76aac93f04b04b051c0232dc96cdfb2ebd150ce726aa9f776a4713c2ac524dc8"),
			Number: 0,
		},
		L2Time: 1669088016,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0xc02551cde892e9716363b8e99d655298909e1a84"),
			Overhead:    eth.Bytes32(common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000834")),
			Scalar:      eth.Bytes32(common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000f4240")),
			GasLimit:    30000000,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      3600,
	SeqWindowSize:          120,
	ChannelTimeout:         30,
	L1ChainID:              big.NewInt(5),
	L2ChainID:              big.NewInt(902),
	P2PSequencerAddress:    common.HexToAddress("0x1491418a70b592f8ad0e4279bb700f496d3b9abb"),
	BatchInboxAddress:      common.HexToAddress("0x880fb147c4e76adeed5b90f11172abf234111dee"),
	DepositContractAddress: common.HexToAddress("0xa581ca3353db73115c4625ffc7adf5db379434a8"),
	L1SystemConfigAddress:  common.HexToAddress("0x2a4daa073b98a092ee235badfed23b54f1d416c9"),
}

var NetworksByName = map[string]rollup.Config{
	"beta-1": Beta1,
}

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
