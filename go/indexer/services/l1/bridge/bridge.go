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

type DepositsMap map[common.Hash][]db.Deposit

type Bridge interface {
	Address() common.Address
	GetDepositsByBlockRange(uint64, uint64) (DepositsMap, error)
	String() string
}

type implConfig struct {
	name string
	impl string
	addr common.Address
}

var customBridgeCfgs = map[uint64][]*implConfig{
	// Mainnet
	1: {
		{"BitBTC", "StandardBridge", common.HexToAddress("0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128")},
		{"DAI", "StandardBridge", common.HexToAddress("0x10E6593CDda8c58a1d0f14C5164B376352a55f2F")},
	},
	// Kovan
	42: {
		{"BitBTC", "StandardBridge", common.HexToAddress("0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746")},
		{"USX", "StandardBridge", common.HexToAddress("0x40E862341b2416345F02c41Ac70df08525150dC7")},
		{"DAI", "StandardBridge", common.HexToAddress("0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3")},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractBackend, addrs *Addresses, ctx context.Context) (map[string]Bridge, error) {
	allCfgs := []*implConfig{
		{"Standard", "StandardBridge", addrs.L1StandardBridge()},
		{"ETH", "ETHBridge", addrs.L1StandardBridge()},
	}
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		switch bridge.impl {
		case "StandardBridge":
			l1StandardBridgeFilter, err := l1bridge.NewL1StandardBridgeFilterer(bridge.addr, client)
			if err != nil {
				return nil, err
			}

			standardBridge := &StandardBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  bridge.addr,
				client:   client,
				filterer: l1StandardBridgeFilter,
			}
			bridges[bridge.name] = standardBridge
		case "ETHBridge":
			l1EthBridgeFilter, err := l1bridge.NewL1StandardBridgeFilterer(bridge.addr, client)
			if err != nil {
				return nil, err
			}

			ethBridge := &EthBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  bridge.addr,
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

func StateCommitmentChainScanner(client bind.ContractFilterer, addrs *Addresses) (*scc.StateCommitmentChainFilterer, error) {
	filter, err := scc.NewStateCommitmentChainFilterer(addrs.StateCommitmentChain(), client)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
