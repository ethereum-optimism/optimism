package base

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
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

var portalABI *abi.ABI

func init() {
	if parsed, err := abi.JSON(bytes.NewReader([]byte(portalABIString))); err != nil {
		panic(fmt.Sprintf("failed to parse portal abi: %s", err))
	} else {
		portalABI = &parsed
	}
}

func TestInteropReadiness(t *testing.T) {
	systest.SystemTest(t, interopReadinessTestScenario())
}

func interopReadinessTestScenario() systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		logger := testlog.Logger(t, log.LevelInfo)
		logger.Info("Started test")

		l1Client, err := sys.L1().Nodes()[0].GethClient()
		require.NoError(t, err)
		l1Caller := batching.NewMultiCaller(l1Client.Client(), batching.DefaultBatchSize)

		checkAbsolutePrestate(t, sys, l1Client)
		checkL1PAO(t, sys, l1Caller)
		checkSuperchainConfig(t, sys, l1Caller)
		checkPermissionless(t, sys, l1Caller)
	}
}

func checkAbsolutePrestate(t systest.T, sys system.System, l1Client *ethclient.Client) {
	var prestate *[32]byte
	for _, chain := range sys.L2s() {
		p := getPrestate(t, l1Client, chain)
		if prestate == nil {
			prestate = &p
		} else {
			require.Equal(t, *prestate, p)
		}
	}
	require.NotNil(t, prestate)
}

func checkL1PAO(t systest.T, sys system.System, l1Caller *batching.MultiCaller) {
	var l1PAO common.Address
	for _, chain := range sys.L2s() {
		owner := getL1PAO(t, l1Caller, chain)
		if l1PAO == (common.Address{}) {
			l1PAO = owner
		} else {
			require.Equal(t, l1PAO, owner)
		}
	}
	require.NotNil(t, l1PAO)
}

func checkSuperchainConfig(t systest.T, sys system.System, l1Caller *batching.MultiCaller) {
	var superchainConfig common.Address
	for _, chain := range sys.L2s() {
		address := getSuperchainConfigFromPortal(t, l1Caller, chain)
		if superchainConfig == (common.Address{}) {
			superchainConfig = address
		} else {
			require.Equal(t, superchainConfig, address)
		}
	}
	require.NotNil(t, superchainConfig)
}

func checkPermissionless(t systest.T, sys system.System, l1Caller *batching.MultiCaller) {
	for _, chain := range sys.L2s() {
		gameType := getRespectedGameType(t, l1Caller, chain)
		require.Equal(t, uint32(0), gameType, "chain is not permissionless")
	}
}

func getL1PAO(t systest.T, l1Caller *batching.MultiCaller, l2Chain system.L2Chain) common.Address {
	portalAddress, ok := l2Chain.L1Addresses()["OptimismPortalProxy"]
	require.True(t, ok, "OptimismPortalProxy not found")
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("proxyAdminOwner"))
	require.NoError(t, err)
	return results.GetAddress(0)
}

func getSuperchainConfigFromPortal(t systest.T, l1Caller *batching.MultiCaller, l2Chain system.L2Chain) common.Address {
	portalAddress, ok := l2Chain.L1Addresses()["OptimismPortalProxy"]
	require.True(t, ok, "OptimismPortalProxy not found")
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("superchainConfig"))
	require.NoError(t, err)
	return results.GetAddress(0)
}

func getPrestate(t systest.T, l1Client *ethclient.Client, l2Chain system.L2Chain) [32]byte {
	dgf, ok := l2Chain.L1Addresses()["DisputeGameFactoryProxy"]
	require.True(t, ok, "DisputeGameFactoryProxy not found")
	dgfContract, err := bindings.NewDisputeGameFactory(dgf, l1Client)
	require.NoError(t, err)

	gameImpl, err := dgfContract.GameImpls(nil, 0)
	require.NoError(t, err)
	fdgContract, err := bindings.NewFaultDisputeGame(gameImpl, l1Client)
	require.NoError(t, err)

	prestate, err := fdgContract.AbsolutePrestate(nil)
	require.NoError(t, err)
	return prestate
}

func getRespectedGameType(t systest.T, l1Caller *batching.MultiCaller, l2Chain system.L2Chain) uint32 {
	portalAddress, ok := l2Chain.L1Addresses()["OptimismPortalProxy"]
	require.True(t, ok, "OptimismPortalProxy not found")
	contract := batching.NewBoundContract(portalABI, portalAddress)
	results, err := l1Caller.SingleCall(context.Background(), rpcblock.Latest, contract.Call("respectedGameType"))
	require.NoError(t, err)
	return results.GetUint32(0)
}
