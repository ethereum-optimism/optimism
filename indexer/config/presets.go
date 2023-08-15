package config

import (
	"github.com/ethereum/go-ethereum/common"
)

// in future presets can just be onchain config and fetched on initialization

// Mapping of l2 chain ids to their preset chain configurations
var presetL1Contracts = map[int]L1Contracts{
	// OP Mainnet
	10: {
		OptimismPortal:         common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
		L2OutputOracle:         common.HexToAddress("0xdfe97868233d1aa22e815a266982f2cf17685a27"),
		L1CrossDomainMessenger: common.HexToAddress("0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1"),
		L1StandardBridge:       common.HexToAddress("0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1"),
		L1ERC721Bridge:         common.HexToAddress("0x5a7749f83b81B301cAb5f48EB8516B986DAef23D"),
	},
	// OP Goerli
	420: {
		OptimismPortal:         common.HexToAddress("0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383"),
		L2OutputOracle:         common.HexToAddress("0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0"),
		L1CrossDomainMessenger: common.HexToAddress("0x5086d1eEF304eb5284A0f6720f79403b4e9bE294"),
		L1StandardBridge:       common.HexToAddress("0x636Af16bf2f682dD3109e60102b8E1A089FedAa8"),
		L1ERC721Bridge:         common.HexToAddress("0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9"),
	},
	// Base Mainnet
	8453: {
		OptimismPortal:         common.HexToAddress("0x49048044D57e1C92A77f79988d21Fa8fAF74E97e"),
		L2OutputOracle:         common.HexToAddress("0x56315b90c40730925ec5485cf004d835058518A0"),
		L1CrossDomainMessenger: common.HexToAddress("0x866E82a600A1414e583f7F13623F1aC5d58b0Afa"),
		L1StandardBridge:       common.HexToAddress("0x3154Cf16ccdb4C6d922629664174b904d80F2C35"),
		// FIXME update this to the correct address
		L1ERC721Bridge: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	},
	// Base Goerli
	84531: {
		OptimismPortal:         common.HexToAddress("0xe93c8cD0D409341205A592f8c4Ac1A5fe5585cfA"),
		L2OutputOracle:         common.HexToAddress("0x2A35891ff30313CcFa6CE88dcf3858bb075A2298"),
		L1CrossDomainMessenger: common.HexToAddress("0x8e5693140eA606bcEB98761d9beB1BC87383706D"),
		L1StandardBridge:       common.HexToAddress("0xfA6D8Ee5BE770F84FC001D098C4bD604Fe01284a"),
		// FIXME update this to the correct address
		L1ERC721Bridge: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	},
	// Zora mainnet
	7777777: {
		OptimismPortal:         common.HexToAddress("0x1a0ad011913A150f69f6A19DF447A0CfD9551054"),
		L2OutputOracle:         common.HexToAddress("0x9E6204F750cD866b299594e2aC9eA824E2e5f95c"),
		L1CrossDomainMessenger: common.HexToAddress("0xdC40a14d9abd6F410226f1E6de71aE03441ca506"),
		L1StandardBridge:       common.HexToAddress("0x3e2Ea9B92B7E48A52296fD261dc26fd995284631"),
		// FIXME update this to the correct address
		L1ERC721Bridge: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	},
	// Zora goerli
	999: {
		OptimismPortal:         common.HexToAddress("0xDb9F51790365e7dc196e7D072728df39Be958ACe"),
		L2OutputOracle:         common.HexToAddress("0xdD292C9eEd00f6A32Ff5245d0BCd7f2a15f24e00"),
		L1CrossDomainMessenger: common.HexToAddress("0xD87342e16352D33170557A7dA1e5fB966a60FafC"),
		L1StandardBridge:       common.HexToAddress("0x7CC09AC2452D6555d5e0C213Ab9E2d44eFbFc956"),
		// FIXME update this to the correct address
		L1ERC721Bridge: common.HexToAddress("0x0000000000000000000000000000000000000000"),
	},
}
