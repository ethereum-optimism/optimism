package sysgo

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLiveOPCMDeployment(t *testing.T) {
	for _, test := range []struct {
		name     string
		gameType gameTypes.GameType
	}{
		{"permissioned", gameTypes.SuperPermissionedGameType},
		{"super-root", gameTypes.SuperCannonKonaGameType},
		{"output-root", gameTypes.CannonKonaGameType},
	} {
		t.Run(test.name, func(t *testing.T) {
			permissionless := test.gameType != gameTypes.SuperPermissionedGameType
			p := devtest.SerialT(t)
			cfg := PresetConfig{}
			if permissionless {
				cfg.DeployerOptions = []DeployerOption{WithJovianAtGenesis}
				cfg.AddedGameTypes = []gameTypes.GameType{test.gameType}
			} else {
				cfg.DeployerOptions = []DeployerOption{WithEcotoneAtGenesis}
			}
			runtime := NewMinimalNoFaultProofsRuntimeWithConfig(p, cfg)
			l2 := runtime.L2Network
			genesis := runtime.L1Network.genesis
			require.NotEmpty(t, genesis.Alloc[l2.opcmImpl].Code, "OPCM must exist in L1 genesis")
			require.NotEmpty(t, genesis.Alloc[l2.mipsImpl].Code, "common implementations must exist in L1 genesis")
			addresses := []common.Address{
				l2.deployment.SystemConfigProxyAddr(),
				l2.rollupCfg.DepositContractAddress,
				l2.deployment.DisputeGameFactoryProxyAddr(),
			}
			for _, address := range addresses {
				require.Empty(t, genesis.Alloc[address].Code, "chain contract %s must not exist in L1 genesis", address)
			}

			rpcClient, err := rpc.DialContext(t.Context(), runtime.L1EL.UserRPC())
			require.NoError(t, err)
			defer rpcClient.Close()
			client := ethclient.NewClient(rpcClient)
			for _, address := range addresses {
				code, err := client.CodeAt(t.Context(), address, nil)
				require.NoError(t, err)
				require.NotEmpty(t, code, "live deployment must create %s", address)
			}
			head, err := client.BlockNumber(t.Context())
			require.NoError(t, err)
			deployments := 0
			var deploymentBlock *big.Int
			for number := uint64(1); number <= head; number++ {
				block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(number))
				require.NoError(t, err)
				for _, tx := range block.Transactions() {
					if tx.To() == nil || *tx.To() != l2.opcmImpl {
						continue
					}
					receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
					require.NoError(t, err)
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
					deploymentBlock = receipt.BlockNumber
					deployments++
				}
			}
			require.Equal(t, 1, deployments, "each chain must require one live OPCM deployment")
			start, err := client.HeaderByNumber(t.Context(), new(big.Int).SetUint64(l2.rollupCfg.Genesis.L1.Number))
			require.NoError(t, err)
			require.Equal(t, l2.rollupCfg.Genesis.L1.Hash, start.Hash(), "L1 start must be canonical")
			callAtDeployment := func(address common.Address, method *w3.Func, outputs ...any) {
				data, err := method.EncodeArgs()
				require.NoError(t, err)
				result, err := client.CallContract(t.Context(), ethereum.CallMsg{To: &address, Data: data}, deploymentBlock)
				require.NoError(t, err)
				require.NoError(t, method.DecodeReturns(result, outputs...))
			}
			var registry common.Address
			callAtDeployment(l2.rollupCfg.DepositContractAddress, w3.MustNewFunc("anchorStateRegistry()", "address"), &registry)
			var root common.Hash
			var sequence *big.Int
			callAtDeployment(registry, w3.MustNewFunc("getAnchorRoot()", "bytes32,uint256"), &root, &sequence)
			if permissionless {
				header := l2.genesis.ToBlock().Header()
				require.NotNil(t, header.WithdrawalsHash)
				output := eth.OutputRoot(&eth.OutputV0{
					StateRoot:                eth.Bytes32(header.Root),
					MessagePasserStorageRoot: eth.Bytes32(*header.WithdrawalsHash),
					BlockHash:                header.Hash(),
				})
				if test.gameType == gameTypes.CannonKonaGameType {
					require.Equal(t, common.Hash(output), root)
					require.Zero(t, bigs.Uint64Strict(sequence))
				} else {
					expected := eth.SuperRoot(eth.NewSuperV1(header.Time, eth.ChainIDAndOutput{ChainID: l2.ChainID(), Output: output}))
					require.Equal(t, common.Hash(expected), root)
					require.Equal(t, header.Time, bigs.Uint64Strict(sequence))
				}
			} else {
				require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, root, "permissioned chains may retain the placeholder")
			}
			l2Client, err := ethclient.DialContext(t.Context(), runtime.L2EL.UserRPC())
			require.NoError(t, err)
			defer l2Client.Close()
			require.Eventually(t, func() bool {
				number, err := l2Client.BlockNumber(t.Context())
				return err == nil && number > 0
			}, 30*time.Second, 100*time.Millisecond, "L2 must progress after deployment")
			rollupRPC, err := rpc.DialContext(t.Context(), runtime.L2CL.UserRPC())
			require.NoError(t, err)
			defer rollupRPC.Close()
			require.Eventually(t, func() bool {
				var status eth.SyncStatus
				err := rollupRPC.CallContext(t.Context(), &status, "optimism_syncStatus")
				return err == nil && status.SafeL2.Number > 0
			}, 90*time.Second, 100*time.Millisecond, "batch submission and derivation must advance the safe head")
		})
	}
}

func TestLiveOPCMMultichainDeployment(t *testing.T) {
	for _, delayedInterop := range []bool{false, true} {
		name := "super-root"
		if delayedInterop {
			name = "pre-Lagoon-allocations"
		}
		t.Run(name, func(t *testing.T) {
			p := devtest.SerialT(t)
			keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
			require.NoError(t, err)
			jwtPath, _ := writeJWTSecret(p)
			var l1EL *L1Geth
			opts := []DeployerOption{func(p devtest.T, keys devkeys.Keys, builder intentbuilder.Builder) {
				builder.WithGlobalOverride("respectedGameType", uint32(gameTypes.SuperCannonKonaGameType))
				builder.WithGlobalOverride("faultGameAbsolutePrestate", PrestateForGameType(p, gameTypes.SuperCannonKonaGameType))
				for _, chain := range builder.L2s() {
					owner, err := keys.Address(devkeys.DeployerRole.Key(big.NewInt(0)))
					require.NoError(t, err)
					chain.WithL1ProxyAdminOwner(owner)
					chain.WithAdditionalDisputeGames([]state.AdditionalDisputeGame{{
						ChainProofParams: state.ChainProofParams{
							DisputeGameType:                         255,
							DisputeAbsolutePrestate:                 standard.DisputeAbsolutePrestate,
							DisputeMaxGameDepth:                     50,
							DisputeSplitDepth:                       14,
							DisputeClockExtension:                   1,
							DisputeMaxClockDuration:                 10,
							DangerouslyAllowCustomDisputeParameters: true,
						},
						VMType: state.VMTypeAlphabet,
					}})
				}
			}}
			if !delayedInterop {
				opts = append(opts, WithJovianAtGenesis)
			}
			wb, l1, l2s := buildMultiL2RuntimeWorld(p, keys, delayedInterop, 30, "",
				[]runtimeChainSpec{{Name: "l2a", ID: DefaultL2AID}, {Name: "l2b", ID: DefaultL2BID}}, nil,
				func(l1 *L1Network) (*L1Geth, *L1CLNode) {
					var l1CL *L1CLNode
					l1EL, l1CL = startInProcessL1(p, l1, jwtPath)
					return l1EL, l1CL
				},
				opts...)
			l2a, l2b := l2s[0], l2s[1]
			chainOutputs := make([]eth.ChainIDAndOutput, 0, 2)
			for _, l2 := range []*L2Network{l2a, l2b} {
				require.Empty(t, l1.genesis.Alloc[l2.deployment.SystemConfigProxyAddr()].Code)
				header := l2.genesis.ToBlock().Header()
				require.NotNil(t, header.WithdrawalsHash)
				chainOutputs = append(chainOutputs, eth.ChainIDAndOutput{
					ChainID: l2.ChainID(),
					Output: eth.OutputRoot(&eth.OutputV0{
						StateRoot:                eth.Bytes32(header.Root),
						MessagePasserStorageRoot: eth.Bytes32(*header.WithdrawalsHash),
						BlockHash:                header.Hash(),
					}),
				})
			}
			expected := common.Hash(eth.SuperRoot(eth.NewSuperV1(l2a.genesis.Timestamp, chainOutputs...)))
			client, err := ethclient.DialContext(t.Context(), l1EL.UserRPC())
			require.NoError(t, err)
			defer client.Close()
			for _, chain := range wb.output.Chains {
				code, err := client.CodeAt(t.Context(), chain.SystemConfigProxy, nil)
				require.NoError(t, err)
				require.NotEmpty(t, code)
				method := w3.MustNewFunc("getAnchorRoot()", "bytes32,uint256")
				data, err := method.EncodeArgs()
				require.NoError(t, err)
				result, err := client.CallContract(t.Context(), ethereum.CallMsg{To: &chain.AnchorStateRegistryProxy, Data: data}, nil)
				require.NoError(t, err)
				var root common.Hash
				var sequence *big.Int
				require.NoError(t, method.DecodeReturns(result, &root, &sequence))
				require.Equal(t, expected, root)
				require.Equal(t, l2a.genesis.Timestamp, bigs.Uint64Strict(sequence))
				gameMethod := w3.MustNewFunc("gameImpls(uint32)", "address")
				data, err = gameMethod.EncodeArgs(uint32(255))
				require.NoError(t, err)
				result, err = client.CallContract(t.Context(), ethereum.CallMsg{To: &chain.DisputeGameFactoryProxy, Data: data}, nil)
				require.NoError(t, err)
				var game common.Address
				require.NoError(t, gameMethod.DecodeReturns(result, &game))
				require.NotEqual(t, common.Address{}, game, "configured additional games must be available")
			}
			head, err := client.BlockNumber(t.Context())
			require.NoError(t, err)
			deployments := 0
			for number := uint64(1); number <= head; number++ {
				block, err := client.BlockByNumber(t.Context(), new(big.Int).SetUint64(number))
				require.NoError(t, err)
				for _, tx := range block.Transactions() {
					if tx.To() == nil || *tx.To() != l2a.opcmImpl {
						continue
					}
					receipt, err := client.TransactionReceipt(t.Context(), tx.Hash())
					require.NoError(t, err)
					require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
					deployments++
				}
			}
			require.Equal(t, 2, deployments, "each chain must have one successful OPCM deployment")
		})
	}
}
