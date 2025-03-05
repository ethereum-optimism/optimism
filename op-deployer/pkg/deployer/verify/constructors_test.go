package verify

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

func TestCalculateTypeSlots(t *testing.T) {
	t.Run("nested tuple", func(t *testing.T) {
		constructorArgsJSON := `[
			{
				"name": "_superchainConfig",
				"type": "address",
				"internalType": "contract ISuperchainConfig"
			},
			{
				"name": "_protocolVersions",
				"type": "address",
				"internalType": "contract IProtocolVersions"
			},
			{
				"name": "_superchainProxyAdmin",
				"type": "address",
				"internalType": "contract IProxyAdmin"
			},
			{
				"name": "_l1ContractsRelease",
				"type": "string",
				"internalType": "string"
			},
			{
				"name": "_blueprints",
				"type": "tuple",
				"internalType": "struct OPContractsManager.Blueprints",
				"components": [
					{
						"name": "addressManager",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "proxy",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "proxyAdmin",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "l1ChugSplashProxy",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "resolvedDelegateProxy",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "permissionedDisputeGame1",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "permissionedDisputeGame2",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "permissionlessDisputeGame1",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "permissionlessDisputeGame2",
						"type": "address",
						"internalType": "address"
					}
				]
			},
			{
				"name": "_implementations",
				"type": "tuple",
				"internalType": "struct OPContractsManager.Implementations",
				"components": [
					{
						"name": "superchainConfigImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "protocolVersionsImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "l1ERC721BridgeImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "optimismPortalImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "systemConfigImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "optimismMintableERC20FactoryImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "l1CrossDomainMessengerImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "l1StandardBridgeImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "disputeGameFactoryImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "anchorStateRegistryImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "delayedWETHImpl",
						"type": "address",
						"internalType": "address"
					},
					{
						"name": "mipsImpl",
						"type": "address",
						"internalType": "address"
					}
				]
			},
			{
				"name": "_upgradeController",
				"type": "address",
				"internalType": "address"
			}
		]`

		var args abi.Arguments
		err := json.Unmarshal([]byte(constructorArgsJSON), &args)
		require.NoError(t, err)

		totalSlots := 0
		for _, arg := range args {
			totalSlots += calculateTypeSlots(arg.Type)
		}

		require.Equal(t, 28, totalSlots)
	})
}
