package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/bindings/legacy/scc"
	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type DepositsMap map[common.Hash][]db.Deposit
type InitiatedWithdrawalMap map[common.Hash][]db.Withdrawal
type FinalizedWithdrawalsMap map[common.Hash][]db.FinalizedWithdrawal

type Bridge interface {
	Address() common.Address
	GetDepositsByBlockRange(context.Context, uint64, uint64) (DepositsMap, error)
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
		{"wstETH", "StandardBridge", common.HexToAddress("0x76943C0D61395d8F2edF9060e1533529cAe05dE6")},
	},
	// Kovan
	42: {
		{"BitBTC", "StandardBridge", common.HexToAddress("0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746")},
		{"USX", "StandardBridge", common.HexToAddress("0x40E862341b2416345F02c41Ac70df08525150dC7")},
		{"DAI", "StandardBridge", common.HexToAddress("0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3")},
		{"wstETH", "StandardBridge", common.HexToAddress("0x65321bf24210b81500230dCEce14Faa70a9f50a7")},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractBackend, addrs services.AddressManager) (map[string]Bridge, error) {
	l1SBAddr, _ := addrs.L1StandardBridge()
	allCfgs := []*implConfig{
		{"Standard", "StandardBridge", l1SBAddr},
		{"ETH", "ETHBridge", l1SBAddr},
	}
	allCfgs = append(allCfgs, customBridgeCfgs[chainID.Uint64()]...)

	bridges := make(map[string]Bridge)
	for _, bridge := range allCfgs {
		l1SB, err := bindings.NewL1StandardBridge(bridge.addr, client)
		if err != nil {
			return nil, err
		}

		switch bridge.impl {
		case "StandardBridge":
			standardBridge := &StandardBridge{
				name:     bridge.name,
				address:  bridge.addr,
				contract: l1SB,
			}
			bridges[bridge.name] = standardBridge
		case "ETHBridge":
			ethBridge := &EthBridge{
				name:     bridge.name,
				address:  bridge.addr,
				contract: l1SB,
			}
			bridges[bridge.name] = ethBridge
		default:
			return nil, errors.New("unsupported bridge")
		}
	}
	return bridges, nil
}

func StateCommitmentChainScanner(client bind.ContractFilterer, addrs services.AddressManager) (*scc.StateCommitmentChainFilterer, error) {
	sccAddr, _ := addrs.StateCommitmentChain()
	filter, err := scc.NewStateCommitmentChainFilterer(sccAddr, client)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
