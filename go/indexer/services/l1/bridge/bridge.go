package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/bindings/scc"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type Bridge interface {
	Address() common.Address
	GetDepositsByBlockRange(uint64, uint64) (map[common.Hash][]db.Deposit, error)
	String() string
}

var CONTRACT_ADDRESSES = map[uint64]struct {
	AddressManager,
	L1CrossDomainMessenger,
	L1StandardBridge,
	StateCommitmentChain,
	CanonicalTransactionChain,
	BondManager string
}{
	// Mainnet
	1: {
		"0xdE1FCfB0851916CA5101820A69b13a4E276bd81F",
		"0x25ace71c97B33Cc4729CF772ae268934F7ab5fA1",
		"0x99C9fc46f92E8a1c0deC1b1747d010903E884bE1",
		"0xBe5dAb4A2e9cd0F27300dB4aB94BeE3A233AEB19",
		"0x5E4e65926BA27467555EB562121fac00D24E9dD2",
		"0xcd626E1328b41fCF24737F137BcD4CE0c32bc8d1",
	},
	// Kovan
	42: {
		"0x100Dd3b414Df5BbA2B542864fF94aF8024aFdf3a",
		"0x4361d0F75A0186C05f971c566dC6bEa5957483fD",
		"0x22F24361D548e5FaAfb36d1437839f080363982B",
		"0xD7754711773489F31A0602635f3F167826ce53C5",
		"0xf7B88A133202d41Fe5E2Ab22e6309a1A4D50AF74",
		"0xc5a603d273E28185c18Ba4d26A0024B2d2F42740",
	},
	// Goerli
	5: {
		"0x2F7E3cAC91b5148d336BbffB224B4dC79F09f01D",
		"0xEcC89b9EDD804850C4F343A278Be902be11AaF42",
		"0x73298186A143a54c20ae98EEE5a025bD5979De02",
		"0x1afcA918eff169eE20fF8AB6Be75f3E872eE1C1A",
		"0x2ebA8c4EfDB39A8Cd8f9eD65c50ec079f7CEBD81",
		"0xE5AE60bD6F8DEe4D0c2BC9268e23B92F1cacC58F",
	},
	// Hardhat local
	31337: {
		"0x5FbDB2315678afecb367f032d93F642f64180aa3",
		"0x8A791620dd6260079BF849Dc5567aDC3F2FdC318",
		"0x610178dA211FEF7D417bC0e6FeD39F05609AD788",
		"0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
		"0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9",
		"0x5FC8d32690cc91D4c39d9d3abcBD16989F875707",
	},
}

var BRIDGE_ADDRESSES = map[uint64][]struct{ name, impl, addr string }{
	// Mainnet
	1: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[1].L1StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[1].L1StandardBridge},
		{"BitBTC", "StandardBridge", "0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128"},
		{"DAI", "StandardBridge", "0x10E6593CDda8c58a1d0f14C5164B376352a55f2F"},
	},
	// Kovan
	42: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[42].L1StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[42].L1StandardBridge},
		{"BitBTC", "StandardBridge", "0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746"},
		{"USX", "StandardBridge", "0x40E862341b2416345F02c41Ac70df08525150dC7"},
		{"DAI", "StandardBridge", "0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3"},
	},
	// Goerli
	5: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[5].L1StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[5].L1StandardBridge},
	},
	// Hardhat local
	31337: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[31337].L1StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[31337].L1StandardBridge},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractFilterer, ctx context.Context) (map[string]Bridge, error) {
	bridges := make(map[string]Bridge)
	for _, bridge := range BRIDGE_ADDRESSES[chainID.Uint64()] {
		switch bridge.impl {
		case "StandardBridge":
			l1StandardBridgeAddress := common.HexToAddress(bridge.addr)
			l1StandardBridgeFilter, err := l1bridge.NewL1StandardBridgeFilterer(l1StandardBridgeAddress, client)
			if err != nil {
				return nil, err
			}

			standardBridge := &StandardBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  l1StandardBridgeAddress,
				client:   client,
				filterer: l1StandardBridgeFilter,
			}
			bridges[bridge.name] = standardBridge
		case "ETHBridge":
			l1EthBridgeAddress := common.HexToAddress(bridge.addr)
			l1EthBridgeFilter, err := l1bridge.NewL1StandardBridgeFilterer(l1EthBridgeAddress, client)
			if err != nil {
				return nil, err
			}

			ethBridge := &EthBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  l1EthBridgeAddress,
				client:   client,
				filterer: l1EthBridgeFilter,
			}
			bridges[bridge.name] = ethBridge
		default:
			return nil, errors.New("unsupported bridge")
		}
	}
	return bridges, nil
}

func StateCommitmentChainScanner(chainID *big.Int, client bind.ContractFilterer, ctx context.Context) (*scc.StateCommitmentChainFilterer, error) {
	addr := common.HexToAddress(CONTRACT_ADDRESSES[chainID.Uint64()].StateCommitmentChain)
	filter, err := scc.NewStateCommitmentChainFilterer(addr, client)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
