package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
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
	require.Equal(t, standard.DefaultGenesisTimeOffsetSeconds, cfg.GenesisTimeOffset,
		"genesis time offset must default when the flag is not passed")
}

func TestNewPrepareConfig_GenesisTimeOffsetOverride(t *testing.T) {
	cliCtx := newPrepareCtx(t, testPrivKey)
	require.NoError(t, cliCtx.Set(GenesisTimeOffsetFlagName, "900"))
	cfg := newPrepareConfig(cliCtx, log.NewLogger(log.DiscardHandler()))
	require.EqualValues(t, 900, cfg.GenesisTimeOffset)
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
	require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, dci.StartingAnchorRoot)
	require.Equal(t, dci.DisputeAbsolutePrestate, dci.CannonAbsolutePrestate)
}

func TestMakePredictionInput_Permissionless(t *testing.T) {
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	chain := &state.ChainIntent{
		ID:              common.HexToHash("0x0a"),
		DeployOverrides: map[string]any{"respectedGameType": embedded.GameTypeCannonKona},
	}

	dci, err := makePredictionInput(&state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
	}, st, chain)
	require.NoError(t, err)
	require.Equal(t, uint32(embedded.GameTypeCannonKona), dci.DisputeGameType)

	// The dry run executes DeployOPChain.checkInput and the OPCM's config validation, which
	// for permissionless deploys reject a zero or permissioned-placeholder anchor root and a
	// Cannon prestate that is unset or equal to the Kona prestate.
	require.NotEqual(t, common.Hash{}, dci.StartingAnchorRoot)
	require.NotEqual(t, opcm.DefaultStartingAnchorRoot.Root, dci.StartingAnchorRoot)
	require.NotEqual(t, common.Hash{}, dci.CannonAbsolutePrestate)
	require.NotEqual(t, dci.DisputeAbsolutePrestate, dci.CannonAbsolutePrestate)
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

func TestCheckReservedOverrides(t *testing.T) {
	chainID := common.HexToHash("0x0a")
	newIntent := func() *state.Intent {
		return &state.Intent{Chains: []*state.ChainIntent{{ID: chainID}}}
	}

	t.Run("an intent without reserved keys passes", func(t *testing.T) {
		intent := newIntent()
		intent.GlobalDeployOverrides = map[string]any{"l2GenesisFjordTimeOffset": "0x1"}
		intent.Chains[0].DeployOverrides = map[string]any{"respectedGameType": 0}
		require.NoError(t, checkReservedOverrides(intent, &state.State{}))
	})

	t.Run("a reserved key in the global overrides is rejected", func(t *testing.T) {
		intent := newIntent()
		intent.GlobalDeployOverrides = map[string]any{"l1StartingBlockTag": "0x1234"}
		err := checkReservedOverrides(intent, &state.State{})
		require.ErrorContains(t, err, `globalDeployOverrides key "l1StartingBlockTag" is reserved`)
		require.ErrorContains(t, err, "l1StartBlockHash")
		require.ErrorContains(t, err, GenesisTimeOffsetFlagName)
	})

	t.Run("a reserved key in a chain's overrides is rejected", func(t *testing.T) {
		intent := newIntent()
		intent.Chains[0].DeployOverrides = map[string]any{"l2GenesisBlockTimestamp": hexutil.Uint64(1)}
		err := checkReservedOverrides(intent, &state.State{})
		require.ErrorContains(t, err, chainID.Hex())
		require.ErrorContains(t, err, `deployOverrides key "l2GenesisBlockTimestamp" is reserved`)
	})

	t.Run("reserved keys matchs are case insensitive", func(t *testing.T) {
		intent := newIntent()
		intent.Chains[0].DeployOverrides = map[string]any{"L2GenesisBlockTimestamp": hexutil.Uint64(1)}
		err := checkReservedOverrides(intent, &state.State{})
		require.ErrorContains(t, err, `"L2GenesisBlockTimestamp" is reserved`)
	})

	t.Run("an already deployed chain's overrides are ignored", func(t *testing.T) {
		intent := newIntent()
		intent.Chains[0].DeployOverrides = map[string]any{"l2GenesisBlockTimestamp": hexutil.Uint64(1)}
		st := &state.State{}
		st.SetChainContracts(chainID, addresses.OpChainContracts{}, true)
		require.NoError(t, checkReservedOverrides(intent, st))
	})
}

// TestPredictionDryRun_Permissionless exercises the prediction dry-run end to end for a
// permissionless chain: it deploys a superchain + OPCM onto anvil, then runs
// the DeployOPChain script against a fork with the prediction input.
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

	salt := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	deployIntent := &state.Intent{
		GlobalDeployOverrides: make(map[string]any),
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.Address{'S'},
			SuperchainGuardian:        common.Address{'G'},
			Challenger:                common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"),
		},
	}
	deployState := &state.State{Version: 1, Create2Salt: salt}

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

	require.NoError(t, pipeline.DeploySuperchain(pEnv, deployIntent, deployState))
	require.NoError(t, pipeline.DeployImplementations(pEnv, deployIntent, deployState))

	opcmAddr := deployState.ImplementationsDeployment.OpcmV2Impl
	superchainConfigProxy := deployState.SuperchainDeployment.SuperchainConfigProxy
	require.NotEqual(t, common.Address{}, opcmAddr)
	require.NotEqual(t, common.Address{}, superchainConfigProxy)

	chain := &state.ChainIntent{
		ID:              common.HexToHash("0x0300"),
		DeployOverrides: map[string]any{"respectedGameType": embedded.GameTypeCannonKona},
	}
	predictIntent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfigProxy,
		Chains:                []*state.ChainIntent{chain},
	}
	predictState := &state.State{Version: 1, Create2Salt: salt}

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

	// A different placeholder anchor root must produce identical predicted addresses
	outDifferentRoot := runPrediction(func(dci *opcm.DeployOPChainInput) {
		dci.StartingAnchorRoot = common.Hash{0xaa}
	})
	require.Equal(t, out, outDifferentRoot)
}

func TestPredictChains_SkipsDeployed(t *testing.T) {
	deployedID := common.HexToHash("0x0a")
	freshID := common.HexToHash("0x0b")

	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")

	intent := &state.Intent{
		OPCMAddress:           &opcmAddr,
		SuperchainConfigProxy: &superchainConfig,
		GlobalDeployOverrides: make(map[string]any),
		Chains: []*state.ChainIntent{
			{ID: deployedID},
			{ID: freshID},
		},
	}

	var deployedContracts addresses.OpChainContracts
	deployedContracts.SystemConfigProxy = common.HexToAddress("0xdead")
	st := &state.State{Create2Salt: common.HexToHash("0x03")}
	st.SetChainContracts(deployedID, deployedContracts, true)

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

	anchor := &state.L1BlockRefJSON{Hash: common.HexToHash("0xa11c"), Number: 100, Time: 5000}
	selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
		return anchor, nil
	}
	const genesisTimeOffset = 600

	// Without an override the anchor is the safe block itself.
	require.NoError(t, predictChains(testlog.Logger(t, slog.LevelInfo), intent, st, run, selectAnchor, anchor, genesisTimeOffset))

	require.Equal(t, []common.Hash{freshID}, ran)

	deployed, err := st.Chain(deployedID)
	require.NoError(t, err)
	require.NotNil(t, deployed.Deployed)
	require.True(t, *deployed.Deployed)
	require.Equal(t, deployedContracts.SystemConfigProxy, deployed.SystemConfigProxy)

	fresh, err := st.Chain(freshID)
	require.NoError(t, err)
	require.NotNil(t, fresh.Deployed)
	require.False(t, *fresh.Deployed)
	require.Equal(t, common.HexToAddress("0xbeef"), fresh.SystemConfigProxy)
	require.Equal(t, anchor, fresh.StartBlock, "fresh chain must have its anchor block pinned")
	require.NotNil(t, fresh.GenesisTime, "fresh chain must have its genesis time committed")
	require.EqualValues(t, uint64(anchor.Time)+genesisTimeOffset, *fresh.GenesisTime)

	// The already-deployed chain is skipped, so its anchor is never resolved or pinned.
	require.Nil(t, deployed.StartBlock)
	require.Nil(t, deployed.GenesisTime)
}

func TestPredictChains_ReusesPinnedAnchor(t *testing.T) {
	chainID := common.HexToHash("0x0b")
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")

	newIntent := func() *state.Intent {
		return &state.Intent{
			OPCMAddress:           &opcmAddr,
			SuperchainConfigProxy: &superchainConfig,
			GlobalDeployOverrides: make(map[string]any),
			Chains:                []*state.ChainIntent{{ID: chainID}},
		}
	}

	// Simulate a successful dry-run
	run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
		var out opcm.DeployOPChainOutput
		out.SystemConfigProxy = common.HexToAddress("0xbeef")
		return out, nil
	}
	pinnedAnchor := &state.L1BlockRefJSON{Hash: common.HexToHash("0xa11c"), Number: 100, Time: 5000}
	pinnedGenesisTime := hexutil.Uint64(5600)
	// The current safe head on re-runs: newer than the pinned anchor, but still
	// below the committed genesis time so the pin is not yet stale.
	currentSafe := &state.L1BlockRefJSON{Hash: common.HexToHash("0x5afe"), Number: 105, Time: 5100}
	newPinnedState := func() *state.State {
		st := &state.State{Create2Salt: common.HexToHash("0x03")}
		st.PinChainAnchor(chainID, pinnedAnchor, pinnedGenesisTime)
		return st
	}
	lgr := testlog.Logger(t, slog.LevelInfo)

	t.Run("a re-run reuses the pinned commitment after revalidating it", func(t *testing.T) {
		st := newPinnedState()

		// The re-run sees a newer safe block and a different offset.
		// Revalidation must query the already pinned hash in the state.
		var revalidated []*common.Hash
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			revalidated = append(revalidated, overrideHash)
			if overrideHash != nil {
				return pinnedAnchor, nil
			}
			return currentSafe, nil
		}

		require.NoError(t, predictChains(lgr, newIntent(), st, run, selectAnchor, currentSafe, 9999))

		require.Len(t, revalidated, 1)
		require.NotNil(t, revalidated[0])
		require.Equal(t, pinnedAnchor.Hash, *revalidated[0], "revalidation must target the pinned hash")

		got, err := st.Chain(chainID)
		require.NoError(t, err)
		require.Equal(t, pinnedAnchor, got.StartBlock, "pinned anchor must be reused")
		require.Equal(t, pinnedGenesisTime, *got.GenesisTime, "pinned genesis time must not be recomputed")
		require.Equal(t, common.HexToAddress("0xbeef"), got.SystemConfigProxy, "prediction still runs on a re-run")
	})

	t.Run("an override matching the pinned anchor is accepted", func(t *testing.T) {
		st := newPinnedState()
		intent := newIntent()
		override := pinnedAnchor.Hash
		intent.Chains[0].L1StartBlockHash = &override

		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			return pinnedAnchor, nil
		}
		require.NoError(t, predictChains(lgr, intent, st, run, selectAnchor, currentSafe, 0))
	})

	t.Run("an override conflicting with the pinned anchor errors", func(t *testing.T) {
		st := newPinnedState()
		intent := newIntent()
		override := common.HexToHash("0xother")
		intent.Chains[0].L1StartBlockHash = &override

		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			t.Fatal("selectAnchor must not be called on an override conflict")
			return nil, nil
		}
		err := predictChains(lgr, intent, st, run, selectAnchor, currentSafe, 0)
		require.ErrorContains(t, err, "conflicts with the anchor block pinned by a previous run")
	})

	t.Run("a pinned anchor that fails revalidation errors", func(t *testing.T) {
		st := newPinnedState()
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			return nil, errors.New("not canonical")
		}
		err := predictChains(lgr, newIntent(), st, run, selectAnchor, currentSafe, 0)
		require.ErrorContains(t, err, "no longer valid")
	})

	t.Run("a state with only a start block is re-pinned like a fresh chain", func(t *testing.T) {
		st := &state.State{Create2Salt: common.HexToHash("0x03")}
		st.SetChainContracts(chainID, addresses.OpChainContracts{}, false)
		chainState, err := st.Chain(chainID)
		require.NoError(t, err)
		chainState.StartBlock = pinnedAnchor // no genesis time

		freshAnchor := &state.L1BlockRefJSON{Hash: common.HexToHash("0xffff"), Number: 200, Time: 9000}
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			require.Nil(t, overrideHash, "a half-pinned chain must be selected fresh, not revalidated")
			return freshAnchor, nil
		}

		require.NoError(t, predictChains(lgr, newIntent(), st, run, selectAnchor, freshAnchor, 600))

		got, err := st.Chain(chainID)
		require.NoError(t, err)
		require.Equal(t, freshAnchor, got.StartBlock, "half-pinned anchor must be replaced")
		require.EqualValues(t, 9600, *got.GenesisTime)
	})
}

func TestPredictChains_StaleGenesisTime(t *testing.T) {
	chainID := common.HexToHash("0x0b")
	opcmAddr := common.HexToAddress("0xaaaa000000000000000000000000000000000001")
	superchainConfig := common.HexToAddress("0xbbbb000000000000000000000000000000000002")

	newIntent := func() *state.Intent {
		return &state.Intent{
			OPCMAddress:           &opcmAddr,
			SuperchainConfigProxy: &superchainConfig,
			GlobalDeployOverrides: make(map[string]any),
			Chains:                []*state.ChainIntent{{ID: chainID}},
		}
	}
	run := func(in opcm.DeployOPChainInput) (opcm.DeployOPChainOutput, error) {
		t.Fatal("prediction must not run for a stale genesis time")
		return opcm.DeployOPChainOutput{}, nil
	}
	pinnedAnchor := &state.L1BlockRefJSON{Hash: common.HexToHash("0xa11c"), Number: 100, Time: 5000}
	pinnedGenesisTime := hexutil.Uint64(5600)
	lgr := testlog.Logger(t, slog.LevelInfo)

	t.Run("a reused pin whose genesis time has elapsed errors", func(t *testing.T) {
		st := &state.State{Create2Salt: common.HexToHash("0x03")}
		st.PinChainAnchor(chainID, pinnedAnchor, pinnedGenesisTime)

		// The re-run happens long after the pin. The safe head has passed the
		// committed genesis time, so the deployment can no longer land before it.
		lateSafe := &state.L1BlockRefJSON{Hash: common.HexToHash("0x5afe"), Number: 500, Time: 9000}
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			return pinnedAnchor, nil
		}

		err := predictChains(lgr, newIntent(), st, run, selectAnchor, lateSafe, 600)
		require.ErrorContains(t, err, "the deployment window has elapsed")
		require.ErrorContains(t, err, "clear the chain's state to re-pin")
	})

	t.Run("a genesis time equal to the safe head timestamp errors", func(t *testing.T) {
		st := &state.State{Create2Salt: common.HexToHash("0x03")}
		st.PinChainAnchor(chainID, pinnedAnchor, pinnedGenesisTime)

		boundarySafe := &state.L1BlockRefJSON{Hash: common.HexToHash("0x5afe"), Number: 500, Time: pinnedGenesisTime}
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			return pinnedAnchor, nil
		}

		err := predictChains(lgr, newIntent(), st, run, selectAnchor, boundarySafe, 600)
		require.ErrorContains(t, err, "the deployment window has elapsed")
	})

	t.Run("a fresh pin from an old anchor override errors", func(t *testing.T) {
		st := &state.State{Create2Salt: common.HexToHash("0x03")}
		intent := newIntent()
		oldAnchor := &state.L1BlockRefJSON{Hash: common.HexToHash("0x01d"), Number: 10, Time: 1000}
		intent.Chains[0].L1StartBlockHash = &oldAnchor.Hash

		safe := &state.L1BlockRefJSON{Hash: common.HexToHash("0x5afe"), Number: 500, Time: 9000}
		selectAnchor := func(overrideHash *common.Hash) (*state.L1BlockRefJSON, error) {
			return oldAnchor, nil
		}

		// The override anchor is valid but so old that
		// anchor time + offset is already in the past.
		err := predictChains(lgr, intent, st, run, selectAnchor, safe, 600)
		require.ErrorContains(t, err, "the deployment window has elapsed")
	})
}
