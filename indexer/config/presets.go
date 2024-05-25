package config

import (
	"github.com/ethereum/go-ethereum/common"
)

type Preset struct {
	Name        string
	ChainConfig ChainConfig
}

// In the future, presets can just be onchain config and fetched on initialization

// Mapping of L2 chain ids to their preset chain configurations
var Presets = map[int]Preset{
	10: {
		Name: "Optimism",
		ChainConfig: ChainConfig{
			Preset: 10,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0xdE1FCfB0851916CA5101820A69b13a4E276bd81F"),
				SystemConfigProxy:           common.HexToAddress("0x229047fed2591dbec1eF1118d64F7aF3dB9EB290"),
				OptimismPortalProxy:         common.HexToAddress("0xbEb5Fc579115071764c7423A4f12eDde41f106Ed"),
				L2OutputOracleProxy:         common.HexToAddress("0xdfe97868233d1aa22e815a266982f2cf17685a27"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1"),
				L1StandardBridgeProxy:       common.HexToAddress("0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x5a7749f83b81B301cAb5f48EB8516B986DAef23D"),

				// pre-bedrock
				LegacyCanonicalTransactionChain: common.HexToAddress("0x5e4e65926ba27467555eb562121fac00d24e9dd2"),
				LegacyStateCommitmentChain:      common.HexToAddress("0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19"),
			},
			L1StartingHeight:        13596466,
			L1BedrockStartingHeight: 17422590,
			L2BedrockStartingHeight: 105235063,
		},
	},
	420: {
		Name: "Optimism Goerli",
		ChainConfig: ChainConfig{
			Preset: 420,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0xa6f73589243a6A7a9023b1Fa0651b1d89c177111"),
				SystemConfigProxy:           common.HexToAddress("0xAe851f927Ee40dE99aaBb7461C00f9622ab91d60"),
				OptimismPortalProxy:         common.HexToAddress("0x5b47E1A08Ea6d985D6649300584e6722Ec4B1383"),
				L2OutputOracleProxy:         common.HexToAddress("0xE6Dfba0953616Bacab0c9A8ecb3a9BBa77FC15c0"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x5086d1eEF304eb5284A0f6720f79403b4e9bE294"),
				L1StandardBridgeProxy:       common.HexToAddress("0x636Af16bf2f682dD3109e60102b8E1A089FedAa8"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x8DD330DdE8D9898d43b4dc840Da27A07dF91b3c9"),

				// pre-bedrock
				LegacyCanonicalTransactionChain: common.HexToAddress("0x607F755149cFEB3a14E1Dc3A4E2450Cde7dfb04D"),
				LegacyStateCommitmentChain:      common.HexToAddress("0x9c945aC97Baf48cB784AbBB61399beB71aF7A378"),
			},
			L1StartingHeight:        7017096,
			L1BedrockStartingHeight: 8300214,
			L2BedrockStartingHeight: 4061224,
		},
	},
	11155420: {
		Name: "Optimism Sepolia",
		ChainConfig: ChainConfig{
			Preset: 11155420,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x9bFE9c5609311DF1c011c47642253B78a4f33F4B"),
				SystemConfigProxy:           common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538"),
				OptimismPortalProxy:         common.HexToAddress("0x16Fc5058F25648194471939df75CF27A2fdC48BC"),
				L2OutputOracleProxy:         common.HexToAddress("0x90E9c4f8a994a250F6aEfd61CAFb4F2e895D458F"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x58Cc85b8D04EA49cC6DBd3CbFFd00B4B8D6cb3ef"),
				L1StandardBridgeProxy:       common.HexToAddress("0xFBb0621E0B23b5478B630BD55a5f21f67730B0F1"),
				L1ERC721BridgeProxy:         common.HexToAddress("0xd83e03D576d23C9AEab8cC44Fa98d058D2176D1f"),
			},
			L1StartingHeight: 4071408,
		},
	},
	8453: {
		Name: "Base",
		ChainConfig: ChainConfig{
			Preset: 8453,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x8EfB6B5c4767B09Dc9AA6Af4eAA89F749522BaE2"),
				SystemConfigProxy:           common.HexToAddress("0x73a79Fab69143498Ed3712e519A88a918e1f4072"),
				OptimismPortalProxy:         common.HexToAddress("0x49048044D57e1C92A77f79988d21Fa8fAF74E97e"),
				L2OutputOracleProxy:         common.HexToAddress("0x56315b90c40730925ec5485cf004d835058518A0"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x866E82a600A1414e583f7F13623F1aC5d58b0Afa"),
				L1StandardBridgeProxy:       common.HexToAddress("0x3154Cf16ccdb4C6d922629664174b904d80F2C35"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x608d94945A64503E642E6370Ec598e519a2C1E53"),
			},
			L1StartingHeight: 17481768,
		},
	},
	84531: {
		Name: "Base Goerli",
		ChainConfig: ChainConfig{
			Preset: 84531,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x4Cf6b56b14c6CFcB72A75611080514F94624c54e"),
				SystemConfigProxy:           common.HexToAddress("0xb15eea247eCE011C68a614e4a77AD648ff495bc1"),
				OptimismPortalProxy:         common.HexToAddress("0xe93c8cD0D409341205A592f8c4Ac1A5fe5585cfA"),
				L2OutputOracleProxy:         common.HexToAddress("0x2A35891ff30313CcFa6CE88dcf3858bb075A2298"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x8e5693140eA606bcEB98761d9beB1BC87383706D"),
				L1StandardBridgeProxy:       common.HexToAddress("0xfA6D8Ee5BE770F84FC001D098C4bD604Fe01284a"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x5E0c967457347D5175bF82E8CCCC6480FCD7e568"),
			},
			L1StartingHeight: 8410981,
		},
	},
	84532: {
		Name: "Base Sepolia",
		ChainConfig: ChainConfig{
			Preset: 84532,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x709c2B8ef4A9feFc629A8a2C1AF424Dc5BD6ad1B"),
				SystemConfigProxy:           common.HexToAddress("0xf272670eb55e895584501d564AfEB048bEd26194"),
				OptimismPortalProxy:         common.HexToAddress("0x49f53e41452C74589E85cA1677426Ba426459e85"),
				L2OutputOracleProxy:         common.HexToAddress("0x84457ca9D0163FbC4bbfe4Dfbb20ba46e48DF254"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0xC34855F4De64F1840e5686e64278da901e261f20"),
				L1StandardBridgeProxy:       common.HexToAddress("0xfd0Bf71F60660E2f608ed56e1659C450eB113120"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x21eFD066e581FA55Ef105170Cc04d74386a09190"),
			},
			L1StartingHeight: 4370868,
		},
	},
	7777777: {
		Name: "Zora",
		ChainConfig: ChainConfig{
			Preset: 7777777,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0xEF8115F2733fb2033a7c756402Fc1deaa56550Ef"),
				SystemConfigProxy:           common.HexToAddress("0xA3cAB0126d5F504B071b81a3e8A2BBBF17930d86"),
				OptimismPortalProxy:         common.HexToAddress("0x1a0ad011913A150f69f6A19DF447A0CfD9551054"),
				L2OutputOracleProxy:         common.HexToAddress("0x9E6204F750cD866b299594e2aC9eA824E2e5f95c"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0xdC40a14d9abd6F410226f1E6de71aE03441ca506"),
				L1StandardBridgeProxy:       common.HexToAddress("0x3e2Ea9B92B7E48A52296fD261dc26fd995284631"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x83A4521A3573Ca87f3a971B169C5A0E1d34481c3"),
			},
			L1StartingHeight: 17473923,
		},
	},
	999: {
		Name: "Zora Goerli",
		ChainConfig: ChainConfig{
			Preset: 999,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x54f4676203dEDA6C08E0D40557A119c602bFA246"),
				SystemConfigProxy:           common.HexToAddress("0xF66C9A5E4fE1A8a9bc44a4aF80505a4C3620Ee64"),
				OptimismPortalProxy:         common.HexToAddress("0xDb9F51790365e7dc196e7D072728df39Be958ACe"),
				L2OutputOracleProxy:         common.HexToAddress("0xdD292C9eEd00f6A32Ff5245d0BCd7f2a15f24e00"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0xD87342e16352D33170557A7dA1e5fB966a60FafC"),
				L1StandardBridgeProxy:       common.HexToAddress("0x7CC09AC2452D6555d5e0C213Ab9E2d44eFbFc956"),
				L1ERC721BridgeProxy:         common.HexToAddress("0x57C1C6b596ce90C0e010c358DD4Aa052404bB70F"),
			},
			L1StartingHeight: 8942381,
		},
	},
	424: {
		Name: "PGN",
		ChainConfig: ChainConfig{
			Preset: 424,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x09d5DbA52F0ee2C4A5E94FD5C802bD74Ca9cAD3e"),
				SystemConfigProxy:           common.HexToAddress("0x7Df716EAD1d83a2BF35B416B7BC84bd0700357C9"),
				OptimismPortalProxy:         common.HexToAddress("0xb26Fd985c5959bBB382BAFdD0b879E149e48116c"),
				L2OutputOracleProxy:         common.HexToAddress("0xA38d0c4E6319F9045F20318BA5f04CDe94208608"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x97BAf688E5d0465E149d1d5B497Ca99392a6760e"),
				L1StandardBridgeProxy:       common.HexToAddress("0xD0204B9527C1bA7bD765Fa5CCD9355d38338272b"),
				L1ERC721BridgeProxy:         common.HexToAddress("0xaFF0F8aaB6Cc9108D34b3B8423C76d2AF434d115"),
			},
			L1StartingHeight: 17672702,
		},
	},
	58008: {
		Name: "PGN Sepolia",
		ChainConfig: ChainConfig{
			Preset: 58008,
			L1Contracts: L1Contracts{
				AddressManager:              common.HexToAddress("0x0Ad91488288BBe60ff38258785568A6D1EB3B983"),
				SystemConfigProxy:           common.HexToAddress("0x4BCCC52151f0ad7C62D45Ce0aA77d9d8ffCE534e"),
				OptimismPortalProxy:         common.HexToAddress("0xF04BdD5353Bb0EFF6CA60CfcC78594278eBfE179"),
				L2OutputOracleProxy:         common.HexToAddress("0xD5bAc3152ffC25318F848B3DD5dA6C85171BaEEe"),
				L1CrossDomainMessengerProxy: common.HexToAddress("0x97f3558Ce48FE71B8CeFA5497708A49531D5A8E1"),
				L1StandardBridgeProxy:       common.HexToAddress("0xFaE6abCAF30D23e233AC7faF747F2fC3a5a6Bfa3"),
				L1ERC721BridgeProxy:         common.HexToAddress("0xBA8397B6f255618D5985d0fB427D8c0496F3a5FA"),
			},
			L1StartingHeight: 17672702,
		},
	},
}
