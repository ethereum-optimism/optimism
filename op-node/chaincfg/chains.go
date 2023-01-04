package chaincfg

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var Beta1 = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash("0x59c72db5fec5bf231e61ba59854cff33945ff6652699c55f2431ac2c010610d5"),
			Number: 8046397,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash("0xa89b19033c8b43365e244f425a7e4acb5bae21d1893e1be0eb8cddeb29950d72"),
			Number: 0,
		},
		L2Time: 1669088016,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress("0x793b6822fd651af8c58039847be64cb9ee854bc9"),
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
	BatchInboxAddress:      common.HexToAddress("0xFb3aECf08940785D4fB3Ad87cDC6e1Ceb20e9aac"),
	DepositContractAddress: common.HexToAddress("0xf91795564662DcC9a17de67463ec5BA9C6DC207b"),
	L1SystemConfigAddress:  common.HexToAddress("0x686df068eaa71af78dadc1c427e35600e0fadac5"),
}

var Goerli = rollup.Config{
	Genesis: rollup.Genesis{
		L1: eth.BlockID{
			Hash:   common.HexToHash(""),
			Number: 0,
		},
		L2: eth.BlockID{
			Hash:   common.HexToHash(""),
			Number: 0,
		},
		L2Time: 0,
		SystemConfig: eth.SystemConfig{
			BatcherAddr: common.HexToAddress(""),
			Overhead:    eth.Bytes32(common.HexToHash("")),
			Scalar:      eth.Bytes32(common.HexToHash("")),
			GasLimit:    0,
		},
	},
	BlockTime:              2,
	MaxSequencerDrift:      0,
	SeqWindowSize:          0,
	ChannelTimeout:         0,
	L1ChainID:              big.NewInt(5),
	L2ChainID:              big.NewInt(420),
	BatchInboxAddress:      common.HexToAddress(""),
	DepositContractAddress: common.HexToAddress(""),
	L1SystemConfigAddress:  common.HexToAddress(""),
}

var NetworksByName = map[string]rollup.Config{
	"beta-1": Beta1,
	//"goerli": Goerli,
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
