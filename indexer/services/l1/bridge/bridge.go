package bridge

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// DepositsMap is a collection of deposit objects keyed
// on block hashes.
type DepositsMap map[common.Hash][]db.Deposit

// WithdrawalsMap is a collection of withdrawal objects keyed
// on block hashes.
type InitiatedWithdrawalMap map[common.Hash][]db.Withdrawal

// ProvenWithdrawalsMap is a collection of proven withdrawal
// objects keyed on block hashses
type ProvenWithdrawalsMap map[common.Hash][]db.ProvenWithdrawal

// FinalizedWithdrawalsMap is a collection of finalized withdrawal
// objects keyed on block hashes.
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
		switch bridge.impl {
		case "StandardBridge":
			l1SB, err := bindings.NewL1StandardBridge(bridge.addr, client)
			if err != nil {
				return nil, err
			}
			standardBridge := &StandardBridge{
				name:     bridge.name,
				address:  bridge.addr,
				contract: l1SB,
			}
			bridges[bridge.name] = standardBridge
		case "ETHBridge":
			l1SB, err := bindings.NewL1StandardBridge(bridge.addr, client)
			if err != nil {
				return nil, err
			}
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

func StateCommitmentChainScanner(client bind.ContractFilterer, addrs services.AddressManager) (*legacy_bindings.StateCommitmentChainFilterer, error) {
	sccAddr, _ := addrs.StateCommitmentChain()
	filter, err := legacy_bindings.NewStateCommitmentChainFilterer(sccAddr, client)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
