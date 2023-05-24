package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/common"
)

type WithdrawalsMap map[common.Hash][]db.Withdrawal

type Bridge interface {
	Address() common.Address
	GetWithdrawalsByBlockRange(context.Context, uint64, uint64) (WithdrawalsMap, error)
	String() string
}

type implConfig struct {
	name string
	impl string
	addr common.Address
}

var defaultBridgeCfgs = []*implConfig{
	{"Standard", "StandardBridge", predeploys.L2StandardBridgeAddr},
}

var customBridgeCfgs = map[uint64][]*implConfig{
	// Mainnet
	10: {
		{"BitBTC", StandardBridgeImpl, common.HexToAddress("0x158F513096923fF2d3aab2BcF4478536de6725e2")},
		//{"DAI", "DAIBridge", "0x467194771dAe2967Aef3ECbEDD3Bf9a310C76C65"},
		{"wstETH", StandardBridgeImpl, common.HexToAddress("0x8E01013243a96601a86eb3153F0d9Fa4fbFb6957")},
	},
}

func BridgesByChainID(chainID *big.Int, client *ethclient.Client, isBedrock bool) (map[string]Bridge, error) {
	allCfgs := make([]*implConfig, 0)
	allCfgs = append(allCfgs, defaultBridgeCfgs...)
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	var l2L1MP *bindings.L2ToL1MessagePasser
	var err error
	if isBedrock {
		l2L1MP, err = bindings.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, client)
		if err != nil {
			return nil, err
		}
	}

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		switch bridge.impl {
		case "StandardBridge":
			l2SB, err := bindings.NewL2StandardBridge(bridge.addr, client)
			if err != nil {
				return nil, err
			}
			bridges[bridge.name] = &StandardBridge{
				name:      bridge.name,
				address:   bridge.addr,
				client:    client,
				l2SB:      l2SB,
				l2L1MP:    l2L1MP,
				isBedrock: isBedrock,
			}
		default:
			return nil, errors.New("unsupported bridge")
		}
	}
	return bridges, nil
}
