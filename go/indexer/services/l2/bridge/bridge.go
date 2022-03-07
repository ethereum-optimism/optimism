package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"

	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
	"github.com/ethereum-optimism/optimism/l2geth/common"
)

type WithdrawalsMap map[common.Hash][]db.Withdrawal

type Bridge interface {
	Address() common.Address
	GetWithdrawalsByBlockRange(uint64, uint64) (WithdrawalsMap, error)
	String() string
}

type implConfig struct {
	name string
	impl string
	addr string
}

var defaultBridgeCfgs = []*implConfig{
	{"Standard", "StandardBridge", L2StandardBridgeAddr},
}

var customBridgeCfgs = map[uint64][]*implConfig{
	// Mainnet
	10: {
		{"BitBTC", StandardBridgeImpl, "0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128"},
		//{"DAI", "DAIBridge", "0x10E6593CDda8c58a1d0f14C5164B376352a55f2F"},
	},
	// Kovan
	69: {
		{"BitBTC", StandardBridgeImpl, "0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746"},
		{"USX", StandardBridgeImpl, "0x40E862341b2416345F02c41Ac70df08525150dC7"},
		//{"DAI", "	DAIBridge", "0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3"},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractFilterer, ctx context.Context) (map[string]Bridge, error) {
	allCfgs := make([]*implConfig, 0)
	allCfgs = append(allCfgs, defaultBridgeCfgs...)
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		switch bridge.impl {
		case "StandardBridge":
			l2StandardBridgeAddress := common.HexToAddress(bridge.addr)
			l2StandardBridgeFilter, err := l2bridge.NewL2StandardBridgeFilterer(l2StandardBridgeAddress, client)
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
