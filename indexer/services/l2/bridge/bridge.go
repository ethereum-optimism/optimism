package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type DepositsMap map[common.Hash][]db.Deposit // Finalizations
type WithdrawalsMap map[common.Hash][]db.Withdrawal

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
	901: {
		{"Standard", "StandardBridge", L2StandardBridgeAddr},
	},
	// Goerli Alpha Testnet
	28528: {
		{"Standard", "StandardBridge", L2StandardBridgeAddr},
	},
}

var customBridgeCfgs = map[uint64][]*implConfig{
	// Mainnet
	10: {
		{"BitBTC", StandardBridgeImpl, "0x158F513096923fF2d3aab2BcF4478536de6725e2"},
		//{"DAI", "DAIBridge", "0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65"},
	},
	// Kovan
	69: {
		{"BitBTC", StandardBridgeImpl, "0x0CFb46528a7002a7D8877a5F7a69b9AaF1A9058e"},
		{"USX", StandardBridgeImpl, "0xB4d37826b14Cd3CB7257A2A5094507d701fe715f"},
		//{"DAI", "	DAIBridge", "0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65"},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractFilterer, ctx context.Context) (map[string]Bridge, error) {
	allCfgs := make([]*implConfig, 0)
	allCfgs = append(allCfgs, defaultBridgeCfgs[chainID.Uint64()]...)
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		switch bridge.impl {
		case "StandardBridge":
			l2StandardBridgeAddress := common.HexToAddress(bridge.addr)
			l2StandardBridgeFilter, err := bindings.NewL2StandardBridgeFilterer(l2StandardBridgeAddress, client)
			if err != nil {
				return nil, err
			}

			standardBridge := &StandardBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  l2StandardBridgeAddress,
				client:   client,
				filterer: l2StandardBridgeFilter,
			}
			bridges[bridge.name] = standardBridge
		default:
			return nil, errors.New("unsupported bridge")
		}
	}
	return bridges, nil
}
