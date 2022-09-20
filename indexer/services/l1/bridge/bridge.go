package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type DepositsMap map[common.Hash][]db.Deposit
type WithdrawalsMap map[common.Hash][]db.Withdrawal // Finalizations

type Bridge interface {
	Address() common.Address
	GetDepositsByBlockRange(uint64, uint64) (DepositsMap, error)
	GetWithdrawalsByBlockRange(uint64, uint64) (WithdrawalsMap, error)
	String() string
}

type implConfig struct {
	name string
	impl string
	addr string
}

var defaultBridgeCfgs = map[uint64][]*implConfig{
	// Devnet
	900: {
		{"Standard", "StandardBridge", predeploys.DevL1StandardBridge},
		{"ETH", "ETHBridge", predeploys.DevL1StandardBridge},
	},
	// Goerli
	5: {
		{"Standard", "StandardBridge", "0xFf94B6C486350aD92561Ba09bad3a59df764Da92"},
		{"ETH", "ETHBridge", "0xFf94B6C486350aD92561Ba09bad3a59df764Da92"},
	},
}

var customBridgeCfgs = map[uint64][]*implConfig{
	// Mainnet
	1: {
		{"BitBTC", "StandardBridge", "0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128"},
		{"DAI", "StandardBridge", "0x10E6593CDda8c58a1d0f14C5164B376352a55f2F"},
	},
	// Kovan
	42: {
		{"BitBTC", "StandardBridge", "0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746"},
		{"USX", "StandardBridge", "0x40E862341b2416345F02c41Ac70df08525150dC7"},
		{"DAI", "StandardBridge", "0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3"},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractBackend, ctx context.Context) (map[string]Bridge, error) {
	allCfgs := make([]*implConfig, 0)
	allCfgs = append(allCfgs, defaultBridgeCfgs[chainID.Uint64()]...)
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		switch bridge.impl {
		case "StandardBridge":
			l1StandardBridgeAddress := common.HexToAddress(bridge.addr)
			l1StandardBridgeFilter, err := bindings.NewL1StandardBridgeFilterer(l1StandardBridgeAddress, client)
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
			l1StandardBridgeAddress := common.HexToAddress(bridge.addr)
			l1EthBridgeFilter, err := bindings.NewL1StandardBridgeFilterer(l1StandardBridgeAddress, client)
			if err != nil {
				return nil, err
			}

			ethBridge := &EthBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  l1StandardBridgeAddress,
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
