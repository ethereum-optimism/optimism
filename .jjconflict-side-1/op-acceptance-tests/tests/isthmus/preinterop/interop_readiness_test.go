package preinterop

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var portalABIString = `
[
	{
		"inputs": [],
		"name": "proxyAdminOwner",
		"outputs": [{"name": "", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "superchainConfig",
		"outputs": [{"name": "", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	},
	{
    	"inputs": [],
		"name": "respectedGameType",
		"outputs": [
		{
			"name": "",
			"type": "uint32"
		}
		],
		"stateMutability": "view",
		"type": "function"
	}
]
`

var disputeGameFactoryABIString = `
[
	{
		"inputs": [{"name": "gameType", "type": "uint32"}],
		"name": "gameImpls",
		"outputs": [{"name": "", "type": "address"}],
		"stateMutability": "view",
		"type": "function"
	}
]
`

var faultDisputeGameABIString = `
[
	{
		"inputs": [],
		"name": "absolutePrestate",
		"outputs": [{"name": "", "type": "bytes32"}],
		"stateMutability": "view",
		"type": "function"
	}
]
`

var portalABI *abi.ABI
var disputeGameFactoryABI *abi.ABI
var faultDisputeGameABI *abi.ABI

func init() {
	if parsed, err := abi.JSON(bytes.NewReader([]byte(portalABIString))); err != nil {
		panic(fmt.Sprintf("failed to parse portal abi: %s", err))
	} else {
		portalABI = &parsed
	}

	if parsed, err := abi.JSON(bytes.NewReader([]byte(disputeGameFactoryABIString))); err != nil {
		panic(fmt.Sprintf("failed to parse dispute game factory abi: %s", err))
	} else {
		disputeGameFactoryABI = &parsed
	}

	if parsed, err := abi.JSON(bytes.NewReader([]byte(faultDisputeGameABIString))); err != nil {
		panic(fmt.Sprintf("failed to parse fault dispute game abi: %s", err))
	} else {
		faultDisputeGameABI = &parsed
	}
}

func TestInteropReadiness(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSimpleInterop(t)

	t.Logger().Info("Started test")

	l1EL := sys.L1EL
	l1Client := l1EL.EthClient()
	l1Caller := l1Client.NewMultiCaller(batching.DefaultBatchSize)

	checkAbsolutePrestate(t, sys, l1Caller)
	checkL1PAO(t, sys, l1Caller)
	checkSuperchainConfig(t, sys, l1Caller)
	checkPermissionless(t, sys, l1Caller)
}

func checkAbsolutePrestate(t devtest.T, sys *presets.SimpleInterop, l1Caller *batching.MultiCaller) {
	var prestate *[32]byte
	chains := []*dsl.L2Network{sys.L2ChainA, sys.L2ChainB}
	for _, chain := range chains {
		p := getPrestate(t, l1Caller, chain)
		if prestate == nil {
			prestate = &p
		} else {
			t.Require().Equal(*prestate, p)
		}
	}
	t.Require().NotNil(prestate)
}

func checkL1PAO(t devtest.T, sys *presets.SimpleInterop, l1Caller *batching.MultiCaller) {
	var l1PAO common.Address
	chains := []*dsl.L2Network{sys.L2ChainA, sys.L2ChainB}
	for _, chain := range chains {
		owner := getL1PAO(t, l1Caller, chain)
		if l1PAO == (common.Address{}) {
			l1PAO = owner
		} else {
			t.Require().Equal(l1PAO, owner)
		}
	}
	t.Require().NotEqual(common.Address{}, l1PAO)
}

func checkSuperchainConfig(t devtest.T, sys *presets.SimpleInterop, l1Caller *batching.MultiCaller) {
	var superchainConfig common.Address
	chains := []*dsl.L2Network{sys.L2ChainA, sys.L2ChainB}
	for _, chain := range chains {
		address := getSuperchainConfigFromPortal(t, l1Caller, chain)
		if superchainConfig == (common.Address{}) {
			superchainConfig = address
		} else {
			t.Require().Equal(superchainConfig, address)
		}
	}
	t.Require().NotEqual(common.Address{}, superchainConfig)
}

func checkPermissionless(t devtest.T, sys *presets.SimpleInterop, l1Caller *batching.MultiCaller) {
	chains := []*dsl.L2Network{sys.L2ChainA, sys.L2ChainB}
	for _, chain := range chains {
		gameType := getRespectedGameType(t, l1Caller, chain)
		t.Require().Equal(uint32(0), gameType, "chain is not permissionless")
	}
}

func getL1PAO(t devtest.T, l1Caller *batching.MultiCaller, l2Chain *dsl.L2Network) common.Address {
	portalAddress := l2Chain.DepositContractAddr()
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("proxyAdminOwner"))
	t.Require().NoError(err)
	return results.GetAddress(0)
}

func getSuperchainConfigFromPortal(t devtest.T, l1Caller *batching.MultiCaller, l2Chain *dsl.L2Network) common.Address {
	portalAddress := l2Chain.DepositContractAddr()
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("superchainConfig"))
	t.Require().NoError(err)
	return results.GetAddress(0)
}

func getPrestate(t devtest.T, l1Caller *batching.MultiCaller, l2Chain *dsl.L2Network) [32]byte {
	dgf := l2Chain.DisputeGameFactoryProxyAddr()
	dgfContract := batching.NewBoundContract(disputeGameFactoryABI, dgf)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, dgfContract.Call("gameImpls", uint32(0)))
	t.Require().NoError(err)
	gameImpl := results.GetAddress(0)

	fdgContract := batching.NewBoundContract(faultDisputeGameABI, gameImpl)
	prestateResults, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, fdgContract.Call("absolutePrestate"))
	t.Require().NoError(err)
	return prestateResults.GetHash(0)
}

func getRespectedGameType(t devtest.T, l1Caller *batching.MultiCaller, l2Chain *dsl.L2Network) uint32 {
	portalAddress := l2Chain.DepositContractAddr()
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("respectedGameType"))
	t.Require().NoError(err)
	return results.GetUint32(0)
}
