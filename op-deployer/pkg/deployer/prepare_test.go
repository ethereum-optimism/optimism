package deployer

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	opdenv "github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const (
	testL1RPCUrl = "http://localhost:8545"
	testPrivKey  = "0000000000000000000000000000000000000000000000000000000000000001"
)

// newPrepareCtx builds a CLI context with the prepare flags applied and the
// given private key + a test L1 RPC URL set.
func newPrepareCtx(t *testing.T, privKey string) *cli.Context {
	t.Helper()

	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-prepare", flag.ContinueOnError)
	for _, f := range PrepareFlags {
		require.NoError(t, f.Apply(flagSet))
	}
	require.NoError(t, flagSet.Set(PrivateKeyFlagName, privKey))
	require.NoError(t, flagSet.Set(L1RPCURLFlagName, testL1RPCUrl))

	return cli.NewContext(app, flagSet, nil)
}

func TestNewPrepareConfig_FlagsPassed(t *testing.T) {
	cfg := newPrepareConfig(newPrepareCtx(t, testPrivKey), log.NewLogger(log.DiscardHandler()))
	require.Equal(t, testPrivKey, cfg.PrivateKey)
	require.Equal(t, testL1RPCUrl, cfg.L1RPCUrl)
}

func TestMakePredictionInput(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	salt := common.HexToHash("0xcccc000000000000000000000000000000000000000000000000000000000003")
	chainID := common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000000a")

	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
	}
	st := &state.State{Create2Salt: salt}
	chain := &state.ChainIntent{ID: chainID} // GasLimit unset -> defaulted

	dci, err := makePredictionInput(intent, st, chain)
	require.NoError(t, err)

	// Committed values are passed through verbatim so the prediction matches the
	// eventual broadcast.
	require.Equal(t, opcmAddr, dci.Opcm)
	require.Equal(t, superchainConfig, dci.SuperchainConfig)
	require.Equal(t, salt.String(), dci.SaltMixer)
	require.Equal(t, chainID.Big(), dci.L2ChainId)
	require.Equal(t, standard.GasLimit, dci.GasLimit)

	// Roles are non-zero placeholders (they don't affect the predicted addresses,
	// but DeployOPChain.checkInput requires them set).
	for _, role := range []common.Address{
		dci.OpChainProxyAdminOwner, dci.SystemConfigOwner, dci.Batcher,
		dci.UnsafeBlockSigner, dci.Proposer, dci.Challenger,
	} {
		require.Equal(t, standard.PlaceholderAddress, role)
		require.NotEqual(t, common.Address{}, role)
	}

	// Permissioned predictions mirror the eventual broadcast values.
	require.Equal(t, standard.DisputeGameType, dci.DisputeGameType)
	require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, dci.StartingAnchorRoot.Root)
	require.Equal(t, common.Big0, dci.StartingAnchorRoot.L2SequenceNumber)
	require.Equal(t, dci.DisputeAbsolutePrestate, dci.CannonAbsolutePrestate)
}

func TestMakePredictionInput_GameTypeInputs(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
	}

	tests := []struct {
		name                       string
		gameType                   embedded.GameType
		usesPredictionAnchor       bool
		usesFixedCannonPlaceholder bool
	}{
		{
			name:                       "CANNON_KONA",
			gameType:                   embedded.GameTypeCannonKona,
			usesPredictionAnchor:       true,
			usesFixedCannonPlaceholder: true,
		},
		{
			name:                 "SUPER_CANNON_KONA",
			gameType:             embedded.GameTypeSuperCannonKona,
			usesPredictionAnchor: true,
		},
		{
			name:     "PERMISSIONED_CANNON",
			gameType: embedded.GameTypePermissionedCannon,
		},
		{
			name:     "SUPER_PERMISSIONED",
			gameType: embedded.GameTypeSuperPermissioned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := &state.ChainIntent{
				ID:              common.HexToHash("0x0a"),
				DeployOverrides: map[string]any{"respectedGameType": tt.gameType},
			}

			dci, err := makePredictionInput(intent, st, chain)
			require.NoError(t, err)
			require.Equal(t, uint32(tt.gameType), dci.DisputeGameType)
			require.Equal(t, common.Big0, dci.StartingAnchorRoot.L2SequenceNumber)

			if tt.usesPredictionAnchor {
				require.Equal(t, predictionStartingAnchorRoot, dci.StartingAnchorRoot.Root)
			} else {
				require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, dci.StartingAnchorRoot.Root)
			}

			if tt.usesFixedCannonPlaceholder {
				require.Equal(t, opcm.PermissionedGamePrestatePlaceholder, dci.CannonAbsolutePrestate)
				require.NotEqual(t, dci.DisputeAbsolutePrestate, dci.CannonAbsolutePrestate)
			} else {
				require.Equal(t, dci.DisputeAbsolutePrestate, dci.CannonAbsolutePrestate)
			}
		})
	}
}

func TestMakePredictionInput_MissingRequiredAddresses(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	chain := &state.ChainIntent{ID: common.HexToHash("0x0a")}

	_, err := makePredictionInput(&state.Intent{SuperchainConfigProxy: &superchainConfig}, st, chain)
	require.ErrorContains(t, err, "opcmAddress must be set")

	_, err = makePredictionInput(&state.Intent{OPCMAddress: &opcmAddr}, st, chain)
	require.ErrorContains(t, err, "superchainConfigProxy must be set")
}

func TestValidateL1ChainID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "eth_chainId", req.Method)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "0x384", // 900
		}))
	}))
	defer srv.Close()

	l1RPC, err := rpc.Dial(srv.URL)
	require.NoError(t, err)
	defer l1RPC.Close()

	ctx := context.Background()
	require.NoError(t, validateL1ChainID(ctx, l1RPC, &state.Intent{L1ChainID: 900}))

	err = validateL1ChainID(ctx, l1RPC, &state.Intent{L1ChainID: 901})
	require.ErrorContains(t, err, "l1 chain ID mismatch: got 900, expected 901")
}

func TestResolveSuperchainConfigProxy(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superCfg := common.HexToAddress("0xcccc000000000000000000000000000000000003")

	// Stub JSON-RPC endpoint
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     any               `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "eth_call", req.Method)

		var msg struct {
			To    *common.Address `json:"to"`
			Data  hexutil.Bytes   `json:"data"`
			Input hexutil.Bytes   `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		require.Equal(t, &opcmAddr, msg.To)
		calldata := msg.Data
		if len(calldata) == 0 {
			calldata = msg.Input
		}
		require.Equal(t, crypto.Keccak256([]byte("superchainConfig()"))[:4], []byte(calldata))

		calls++
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  common.BytesToHash(superCfg.Bytes()).Hex(),
		}))
	}))
	defer srv.Close()

	l1RPC, err := rpc.Dial(srv.URL)
	require.NoError(t, err)
	defer l1RPC.Close()

	ctx := context.Background()

	// Unset -> resolved from the OPCM in memory only.
	intent := &state.Intent{OPCMAddress: &opcmAddr}
	require.NoError(t, resolveSuperchainConfigProxy(ctx, l1RPC, intent, opcmAddr))
	require.NotNil(t, intent.SuperchainConfigProxy)
	require.Equal(t, superCfg, *intent.SuperchainConfigProxy)
	require.Equal(t, 1, calls)

	// Already set -> left untouched without an RPC call.
	pinned := common.HexToAddress("0xdddd000000000000000000000000000000000004")
	intent = &state.Intent{OPCMAddress: &opcmAddr, SuperchainConfigProxy: &pinned}
	require.NoError(t, resolveSuperchainConfigProxy(ctx, l1RPC, intent, opcmAddr))
	require.Equal(t, pinned, *intent.SuperchainConfigProxy)
	require.Equal(t, 1, calls)
}

func TestPrepareConfigCheck(t *testing.T) {
	valid := PrepareConfig{
		Workdir:    "/tmp",
		Logger:     log.NewLogger(log.DiscardHandler()),
		PrivateKey: testPrivKey,
		L1RPCUrl:   testL1RPCUrl,
	}
	require.NoError(t, valid.Check())

	missingKey := valid
	missingKey.PrivateKey = ""
	require.ErrorContains(t, missingKey.Check(), "private key must be specified")

	invalidKey := valid
	invalidKey.PrivateKey = "not-a-valid-key"
	require.ErrorContains(t, invalidKey.Check(), "failed to parse private key")

	missingL1RPC := valid
	missingL1RPC.L1RPCUrl = ""
	require.ErrorContains(t, missingL1RPC.Check(), "l1 RPC URL must be specified")
}

// TestPredictionDryRun_Permissionless exercises the prediction dry-run end to end for both
// permissionless game types: it deploys OPCMs in output-root and super-root modes onto anvil,
// then runs the DeployOPChain script against a fork with the prediction input.
func TestPredictionDryRun_Permissionless(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	anvil, err := devnet.NewAnvil(lgr)
	require.NoError(t, err)
	require.NoError(t, anvil.Start())
	t.Cleanup(func() {
		require.NoError(t, anvil.Stop())
	})

	l1RPCUrl := anvil.RPCUrl()
	privateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	l1RPC, err := rpc.Dial(l1RPCUrl)
	require.NoError(t, err)
	l1Client := ethclient.NewClient(l1RPC)

	host, err := opdenv.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
		script.WithForkHook(func(cfg *script.ForkConfig) (forking.ForkSource, error) {
			src, err := forking.RPCSourceByNumber(cfg.URLOrAlias, l1RPC, *cfg.BlockNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to create RPC fork source: %w", err)
			}
			return forking.Cache(src), nil
		}),
	)
	require.NoError(t, err)

	latest, err := l1Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	_, err = host.CreateSelectFork(
		script.ForkWithURLOrAlias("main"),
		script.ForkWithBlockNumberU256(latest.Number),
	)
	require.NoError(t, err)

	superchainIntent := &state.Intent{
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.Address{'S'},
			SuperchainGuardian:        common.Address{'G'},
			Challenger:                common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"),
		},
	}
	superchainState := &state.State{Version: 1}

	pEnv := &pipeline.Env{
		Logger:       lgr,
		Scripts:      &opcm.Scripts{},
		ForgeClient:  forgeClient,
		UseForge:     true,
		Context:      ctx,
		Broadcaster:  broadcaster.NoopBroadcaster(),
		StateWriter:  pipeline.NoopStateWriter(),
		L1ScriptHost: host,
		L1RPCUrl:     l1RPCUrl,
		PrivateKey:   privateKey,
	}

	require.NoError(t, pipeline.DeploySuperchain(pEnv, superchainIntent, superchainState))

	tests := []struct {
		name             string
		gameType         embedded.GameType
		devFeatureBitmap common.Hash
		chainID          common.Hash
		salt             common.Hash
	}{
		{
			name:     "CANNON_KONA",
			gameType: embedded.GameTypeCannonKona,
			chainID:  common.HexToHash("0x0300"),
			salt:     common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		},
		{
			name:             "SUPER_CANNON_KONA",
			gameType:         embedded.GameTypeSuperCannonKona,
			devFeatureBitmap: devfeatures.SuperRootGamesMigrationFlag,
			chainID:          common.HexToHash("0x0301"),
			salt:             common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901235"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployIntent := &state.Intent{
				GlobalDeployOverrides: map[string]any{"devFeatureBitmap": tt.devFeatureBitmap},
			}
			deployState := &state.State{
				Version:              1,
				Create2Salt:          tt.salt,
				SuperchainDeployment: superchainState.SuperchainDeployment,
				SuperchainRoles:      superchainState.SuperchainRoles,
			}
			require.NoError(t, pipeline.DeployImplementations(pEnv, deployIntent, deployState))

			opcmAddr := deployState.ImplementationsDeployment.OpcmV2Impl
			superchainConfigProxy := deployState.SuperchainDeployment.SuperchainConfigProxy
			require.NotEqual(t, common.Address{}, opcmAddr)
			require.NotEqual(t, common.Address{}, superchainConfigProxy)

			chain := &state.ChainIntent{
				ID:              tt.chainID,
				DeployOverrides: map[string]any{"respectedGameType": tt.gameType},
			}
			predictIntent := &state.Intent{
				OPCMAddress:           &opcmAddr,
				SuperchainConfigProxy: &superchainConfigProxy,
				Chains:                []*state.ChainIntent{chain},
			}
			predictState := &state.State{Version: 1, Create2Salt: tt.salt}

			// runPrediction mirrors Prepare: a fresh fork of the live L1 with a no-op
			// broadcaster, running the DeployOPChain script with the prediction input.
			runPrediction := func(mutate func(*opcm.DeployOPChainInput)) opcm.DeployOPChainOutput {
				predictHost, err := opdenv.DefaultForkedScriptHost(
					ctx,
					broadcaster.NoopBroadcaster(),
					lgr,
					common.Address{'D'},
					afacts,
					l1RPC,
				)
				require.NoError(t, err)

				deployScript, err := opcm.NewDeployOPChainScript(predictHost)
				require.NoError(t, err)

				dci, err := makePredictionInput(predictIntent, predictState, chain)
				require.NoError(t, err)
				if mutate != nil {
					mutate(&dci)
				}

				out, err := deployScript.Run(dci)
				require.NoError(t, err)
				return out
			}

			out := runPrediction(nil)
			require.NotEqual(t, common.Address{}, out.SystemConfigProxy)
			require.NotEqual(t, common.Address{}, out.OptimismPortalProxy)
			require.NotEqual(t, common.Address{}, out.DisputeGameFactoryProxy)
			require.NotEqual(t, common.Address{}, out.AnchorStateRegistryProxy)
			// The permissionless game must be registered alongside the permissioned fallback.
			require.NotEqual(t, common.Address{}, out.FaultDisputeGame)
			require.NotEqual(t, common.Address{}, out.PermissionedDisputeGame)

			// A different placeholder anchor root must produce identical predicted addresses.
			outDifferentRoot := runPrediction(func(dci *opcm.DeployOPChainInput) {
				dci.StartingAnchorRoot.Root = common.Hash{0xaa}
			})
			require.Equal(t, out, outDifferentRoot)
		})
	}
}

func TestPredictChains_ClearsOnlyRepredictedPrestates(t *testing.T) {
	deployedID := common.HexToHash("0x0a")
	freshID := common.HexToHash("0x0b")
	deployedPrestate := common.HexToHash("0xdead")
	freshPrestate := common.HexToHash("0xbeef")

	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")

	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
		GlobalDeployOverrides: make(map[string]any),
		Chains: []*state.ChainIntent{
			{ID: deployedID},
			{
				ID: freshID,
				DeployOverrides: map[string]any{
					"respectedGameType": embedded.GameTypeCannonKona,
				},
			},
		},
	}

	var deployedContracts addresses.OpChainContracts
	deployedContracts.SystemConfigProxy = common.HexToAddress("0xdead")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	st.SetChainContracts(deployedID, deployedContracts, true)
	st.SetChainContracts(freshID, addresses.OpChainContracts{}, false)
	deployed, err := st.Chain(deployedID)
	require.NoError(t, err)
	deployed.Prestate = deployedPrestate
	fresh, err := st.Chain(freshID)
	require.NoError(t, err)
	fresh.Prestate = freshPrestate

	var ran []common.Hash
	run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
		ran = append(ran, common.BigToHash(in.L2ChainId))
		return opcm.DeployOPChainOutput{
			OpChainProxyAdmin:                  common.Address{0x01},
			AddressManager:                     common.Address{0x02},
			L1ERC721BridgeProxy:                common.Address{0x03},
			OptimismMintableERC20FactoryProxy:  common.Address{0x04},
			L1StandardBridgeProxy:              common.Address{0x05},
			L1CrossDomainMessengerProxy:        common.Address{0x06},
			OptimismPortalProxy:                common.Address{0x07},
			EthLockboxProxy:                    common.Address{0x08},
			DisputeGameFactoryProxy:            common.Address{0x09},
			AnchorStateRegistryProxy:           common.Address{0x0a},
			FaultDisputeGame:                   common.Address{0x0b},
			PermissionedDisputeGame:            common.Address{0x0c},
			DelayedWETHPermissionedGameProxy:   common.Address{0x0d},
			DelayedWETHPermissionlessGameProxy: common.Address{0x0e},
			SystemConfigProxy:                  common.HexToAddress("0xbeef"),
		}, nil
	}

	require.NoError(t, predictChains(testlog.Logger(t, slog.LevelInfo), intent, st, run))

	require.Equal(t, []common.Hash{freshID}, ran)

	require.NotNil(t, deployed.Deployed)
	require.True(t, *deployed.Deployed)
	require.Equal(t, deployedContracts.SystemConfigProxy, deployed.SystemConfigProxy)
	require.Equal(t, deployedPrestate, deployed.Prestate)

	require.NotNil(t, fresh.Deployed)
	require.False(t, *fresh.Deployed)
	require.Equal(t, common.HexToAddress("0xbeef"), fresh.SystemConfigProxy)
	require.Zero(t, fresh.Prestate)
}

func TestPrepareChainsBuildsInteropDepSetBeforePrediction(t *testing.T) {
	firstID := common.HexToHash("0x0a")
	secondID := common.HexToHash("0x0b")
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
		GlobalDeployOverrides: make(map[string]any),
		Chains: []*state.ChainIntent{
			{ID: firstID},
			{ID: secondID},
		},
	}
	st := &state.State{Create2Salt: common.HexToHash("0x03")}

	var predictions int
	run := func(opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
		predictions++
		require.NotNil(t, st.InteropDepSet)
		require.Len(t, st.InteropDepSet.Chains(), 2)
		return emptyDeployOPChainOutput(), nil
	}

	require.NoError(t, prepareChains(testlog.Logger(t, slog.LevelInfo), intent, st, run))
	require.Equal(t, 2, predictions)
}

func TestPrepareChainsPredictionFailureOnlyMutatesSuccessfullyPredictedChainsInMemory(t *testing.T) {
	firstID := common.HexToHash("0x0a")
	secondID := common.HexToHash("0x0b")
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
		GlobalDeployOverrides: make(map[string]any),
		Chains: []*state.ChainIntent{
			{ID: firstID},
			{ID: secondID},
		},
	}
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	st.SetChainContracts(firstID, addresses.OpChainContracts{}, false)
	st.SetChainContracts(secondID, addresses.OpChainContracts{}, false)
	for _, chainID := range []common.Hash{firstID, secondID} {
		chain, err := st.Chain(chainID)
		require.NoError(t, err)
		chain.Prestate = common.HexToHash("0x11")
	}

	var predictions int
	run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
		predictions++
		if predictions == 2 {
			return opcm.DeployOPChainOutput{}, fmt.Errorf("prediction sentinel")
		}
		return emptyDeployOPChainOutput(), nil
	}

	err := prepareChains(testlog.Logger(t, slog.LevelInfo), intent, st, run)
	require.ErrorContains(t, err, "prediction sentinel")

	first, err := st.Chain(firstID)
	require.NoError(t, err)
	require.Zero(t, first.Prestate)
	second, err := st.Chain(secondID)
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x11"), second.Prestate)
}

func TestPredictChainsPrestateReminders(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")

	tests := []struct {
		name            string
		gameType        embedded.GameType
		reminderMessage string
	}{
		{
			name:            "CANNON_KONA",
			gameType:        embedded.GameTypeCannonKona,
			reminderMessage: "selected prestate must be committed",
		},
		{
			name:            "SUPER_CANNON_KONA",
			gameType:        embedded.GameTypeSuperCannonKona,
			reminderMessage: "selected prestate must be committed",
		},
		{
			name:     "PERMISSIONED_CANNON",
			gameType: embedded.GameTypePermissionedCannon,
		},
		{
			name:     "SUPER_PERMISSIONED",
			gameType: embedded.GameTypeSuperPermissioned,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainID := common.HexToHash(fmt.Sprintf("0x%x", i+1))
			intent := &state.Intent{
				OPCMAddress:           &opcmAddr,
				SuperchainConfigProxy: &superchainConfig,
				GlobalDeployOverrides: make(map[string]any),
				Chains: []*state.ChainIntent{{
					ID: chainID,
					DeployOverrides: map[string]any{
						"respectedGameType": tt.gameType,
					},
				}},
			}
			st := &state.State{Create2Salt: common.HexToHash("0x03")}
			st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)
			chain, err := st.Chain(chainID)
			require.NoError(t, err)
			chain.Prestate = common.HexToHash("0x11")
			lgr, logs := testlog.CaptureLogger(t, slog.LevelInfo)

			run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
				require.Equal(t, uint32(tt.gameType), in.DisputeGameType)
				return emptyDeployOPChainOutput(), nil
			}

			require.NoError(t, predictChains(lgr, intent, st, run))
			require.Zero(t, chain.Prestate)

			chainFilter := testlog.NewAttributesFilter("chain", chainID.Hex())
			if tt.reminderMessage == "" {
				require.Nil(t, logs.FindLog(
					testlog.NewMessageContainsFilter("run op-deployer prestate before continue"),
					chainFilter,
				))
				return
			}
			logs.RequireMessageContainedOnce(t, tt.reminderMessage, chainFilter)
		})
	}
}

func emptyDeployOPChainOutput() opcm.DeployOPChainOutput {
	return opcm.DeployOPChainOutput{
		OpChainProxyAdmin:                  common.Address{},
		AddressManager:                     common.Address{},
		L1ERC721BridgeProxy:                common.Address{},
		SystemConfigProxy:                  common.Address{},
		OptimismMintableERC20FactoryProxy:  common.Address{},
		L1StandardBridgeProxy:              common.Address{},
		L1CrossDomainMessengerProxy:        common.Address{},
		OptimismPortalProxy:                common.Address{},
		EthLockboxProxy:                    common.Address{},
		DisputeGameFactoryProxy:            common.Address{},
		AnchorStateRegistryProxy:           common.Address{},
		FaultDisputeGame:                   common.Address{},
		PermissionedDisputeGame:            common.Address{},
		DelayedWETHPermissionedGameProxy:   common.Address{},
		DelayedWETHPermissionlessGameProxy: common.Address{},
	}
}
