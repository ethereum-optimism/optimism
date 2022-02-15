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

type Bridge interface {
	Address() common.Address
	GetWithdrawalsByBlockRange(uint64, uint64) (map[common.Hash][]db.Withdrawal, error)
	String() string
}

var CONTRACT_ADDRESSES = map[uint64]struct {
	OVM_L2ToL1MessagePasser,
	OVM_DeployerWhitelist,
	L2CrossDomainMessenger,
	OVM_GasPriceOracle,
	L2StandardBridge,
	OVM_SequencerFeeVault,
	L2StandardTokenFactory,
	OVM_L1BlockNumber,
	OVM_ETH,
	WETH9 string
}{
	// Mainnet
	10: {
		"0x4200000000000000000000000000000000000000",
		"0x4200000000000000000000000000000000000002",
		"0x4200000000000000000000000000000000000007",
		"0x420000000000000000000000000000000000000F",
		"0x4200000000000000000000000000000000000010",
		"0x4200000000000000000000000000000000000011",
		"0x4200000000000000000000000000000000000012",
		"0x4200000000000000000000000000000000000013",
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"0x4200000000000000000000000000000000000006",
	},
	// Kovan
	69: {
		"0x4200000000000000000000000000000000000000",
		"0x4200000000000000000000000000000000000002",
		"0x4200000000000000000000000000000000000007",
		"0x420000000000000000000000000000000000000F",
		"0x4200000000000000000000000000000000000010",
		"0x4200000000000000000000000000000000000011",
		"0x4200000000000000000000000000000000000012",
		"0x4200000000000000000000000000000000000013",
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"0x4200000000000000000000000000000000000006",
	},
	// Goerli
	690: {
		"0x4200000000000000000000000000000000000000",
		"0x4200000000000000000000000000000000000002",
		"0x4200000000000000000000000000000000000007",
		"0x420000000000000000000000000000000000000F",
		"0x4200000000000000000000000000000000000010",
		"0x4200000000000000000000000000000000000011",
		"0x4200000000000000000000000000000000000012",
		"0x4200000000000000000000000000000000000013",
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"0x4200000000000000000000000000000000000006",
	},
	// Hardhat local
	420: {
		"0x4200000000000000000000000000000000000000",
		"0x4200000000000000000000000000000000000002",
		"0x4200000000000000000000000000000000000007",
		"0x420000000000000000000000000000000000000F",
		"0x4200000000000000000000000000000000000010",
		"0x4200000000000000000000000000000000000011",
		"0x4200000000000000000000000000000000000012",
		"0x4200000000000000000000000000000000000013",
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"0x4200000000000000000000000000000000000006",
	},
}

var BRIDGE_ADDRESSES = map[uint64][]struct{ name, impl, addr string }{
	// Mainnet
	10: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[10].L2StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[10].L2StandardBridge},
		{"BitBTC", "StandardBridge", "0xaBA2c5F108F7E820C049D5Af70B16ac266c8f128"},
		//{"DAI", "DAIBridge", "0x10E6593CDda8c58a1d0f14C5164B376352a55f2F"},
	},
	// Kovan
	69: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[69].L2StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[69].L2StandardBridge},
		{"BitBTC", "StandardBridge", "0x0b651A42F32069d62d5ECf4f2a7e5Bd3E9438746"},
		{"USX", "StandardBridge", "0x40E862341b2416345F02c41Ac70df08525150dC7"},
		//{"DAI", "	DAIBridge", "0xb415e822C4983ecD6B1c1596e8a5f976cf6CD9e3"},
	},
	// Goerli
	690: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[690].L2StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[690].L2StandardBridge},
	},
	// Hardhat local
	420: {
		{"Standard", "StandardBridge", CONTRACT_ADDRESSES[420].L2StandardBridge},
		{"ETH", "ETHBridge", CONTRACT_ADDRESSES[420].L2StandardBridge},
	},
}

func BridgesByChainID(chainID *big.Int, client bind.ContractFilterer, ctx context.Context) (map[string]Bridge, error) {
	bridges := make(map[string]Bridge)
	for _, bridge := range BRIDGE_ADDRESSES[chainID.Uint64()] {
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
		case "ETHBridge":
			l2EthBridgeAddress := common.HexToAddress(bridge.addr)
			l2EthBridgeFilter, err := l2bridge.NewL2StandardBridgeFilterer(l2EthBridgeAddress, client)
			if err != nil {
				return nil, err
			}

			ethBridge := &EthBridge{
				name:     bridge.name,
				ctx:      ctx,
				address:  l2EthBridgeAddress,
				client:   client,
				filterer: l2EthBridgeFilter,
			}
			bridges[bridge.name] = ethBridge
		default:
			return nil, errors.New("unsupported bridge")
		}
	}
	return bridges, nil
}
