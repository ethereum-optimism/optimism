package sysgo

import (
	"fmt"
	"math/big"
	"net/url"
	"path"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func setRespectedGameTypeForRuntime(
	t devtest.T,
	keys devkeys.Keys,
	gameType gameTypes.GameType,
	l1ChainID eth.ChainID,
	l1ELRPC string,
	l2Net *L2Network,
) {
	require := t.Require()
	require.NotNil(l2Net, "l2 network must exist")
	require.NotNil(l2Net.rollupCfg, "l2 rollup config must exist")

	portalAddr := l2Net.rollupCfg.DepositContractAddress

	rpcClient, err := rpc.DialContext(t.Ctx(), l1ELRPC)
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)

	guardianKey, err := keys.Secret(devkeys.SuperchainOperatorKeys(l1ChainID.ToBig())(devkeys.SuperchainConfigGuardianKey))
	require.NoError(err, "failed to get guardian key")

	transactOpts, err := bind.NewKeyedTransactorWithChainID(guardianKey, l1ChainID.ToBig())
	require.NoError(err, "must have transact opts")
	transactOpts.Context = t.Ctx()

	portalBindings := bindings.NewBindings[bindings.OptimismPortal2](bindings.WithTo(portalAddr), bindings.WithTest(t))
	f := portalBindings.AnchorStateRegistry()
	calldata, err := f.EncodeInput()
	require.NoError(err, "failed to encode anchorStateRegistry() calldata")
	result, err := client.CallContract(t.Ctx(), ethereum.CallMsg{
		To:   &portalAddr,
		Data: calldata,
	}, nil)
	require.NoError(err, "failed to read anchor state registry address from portal")
	asrAddr, err := f.DecodeOutput(result)
	require.NoError(err, "failed to decode anchor state registry address from portal")

	txOpts := txplan.Combine(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(guardianKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithEstimator(client, true),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(client, 5, retry.Exponential()))

	asrBindings := bindings.NewBindings[bindings.AnchorStateRegistry](bindings.WithTo(asrAddr), bindings.WithTest(t))
	rcpt, err := contractio.Write(asrBindings.SetRespectedGameType(uint32(gameType)), t.Ctx(), txOpts)
	require.NoError(err, "failed to set respected game type")
	require.Equal(rcpt.Status, gethTypes.ReceiptStatusSuccessful, "set respected game type tx did not execute correctly")
}

// addGameTypesForRuntime uses OPCMv2.upgrade to configure dispute game types.
// Game types in enabledGameTypes are enabled; the rest are disabled.
func addGameTypesForRuntime(
	t devtest.T,
	keys devkeys.Keys,
	enabledGameTypes []gameTypes.GameType,
	l1ChainID eth.ChainID,
	l1ELRPC string,
	l2Net *L2Network,
) {
	require := t.Require()
	require.NotNil(l2Net, "l2 network must exist")
	require.NotNil(l2Net.deployment, "l2 deployment must exist")
	require.NotEqual(common.Address{}, l2Net.opcmImpl, "missing OPCM implementation address")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1ELRPC)
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)

	l1PAO, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l1ChainID)

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	proposer, err := keys.Address(chainOps(devkeys.ProposerRole))
	require.NoError(err, "failed to get proposer address")
	challenger, err := keys.Address(chainOps(devkeys.ChallengerRole))
	require.NoError(err, "failed to get challenger address")

	enabled := make(map[gameTypes.GameType]bool)
	for _, gt := range enabledGameTypes {
		enabled[gt] = true
	}
	initBond := eth.GWei(80_000_000).ToBig() // 0.08 ETH

	cannonPrestate := PrestateForGameType(t, gameTypes.CannonGameType)
	cannonKonaPrestate := PrestateForGameType(t, gameTypes.CannonKonaGameType)

	var zkDisputeGameConfig *embedded.ZKDisputeGameConfig
	if enabled[gameTypes.ZKDisputeGameType] {
		zkDisputeGameConfig = ZKDisputeGameConfigForRuntime(t)
	}

	// OPCMv2 requires all 6 game configs in order:
	// CANNON, PERMISSIONED_CANNON, CANNON_KONA, SUPER_PERMISSIONED_CANNON, SUPER_CANNON_KONA, ZK_DISPUTE_GAME.
	configs := []embedded.DisputeGameConfig{
		{
			Enabled:  enabled[gameTypes.CannonGameType],
			InitBond: initBond,
			GameType: embedded.GameTypeCannon,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{
				AbsolutePrestate: cannonPrestate,
			},
		},
		{
			Enabled:  true, // Permissioned cannon is always enabled.
			InitBond: initBond,
			GameType: embedded.GameTypePermissionedCannon,
			PermissionedDisputeGameConfig: &embedded.PermissionedDisputeGameConfig{
				AbsolutePrestate: cannonPrestate,
				Proposer:         proposer,
				Challenger:       challenger,
			},
		},
		{
			Enabled:  enabled[gameTypes.CannonKonaGameType],
			InitBond: initBond,
			GameType: embedded.GameTypeCannonKona,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{
				AbsolutePrestate: cannonKonaPrestate,
			},
		},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeSuperPermCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeSuperCannonKona},
		{
			Enabled:             enabled[gameTypes.ZKDisputeGameType],
			InitBond:            initBond,
			GameType:            embedded.GameTypeZKDisputeGame,
			ZKDisputeGameConfig: zkDisputeGameConfig,
		},
	}
	// Zero out init bond for disabled games.
	for i := range configs {
		if !configs[i].Enabled {
			configs[i].InitBond = new(big.Int)
		}
	}

	artifactsFS, err := artifacts.Download(t.Ctx(), LocalArtifacts(t), ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	executeOPCMUpgrade(t, rpcClient, client, l1PAOKey, artifactsFS, embedded.UpgradeOPChainInput{
		Prank: l1PAO,
		Opcm:  l2Net.opcmImpl,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       l2Net.deployment.SystemConfigProxyAddr(),
			DisputeGameConfigs: configs,
			ExtraInstructions: []embedded.ExtraInstruction{
				{Key: "PermittedProxyDeployment", Data: []byte("DelayedWETH")},
			},
		},
	})
}

// ZKDisputeGameConfigForRuntime returns a ZKDisputeGameConfig for use in devstack/test environments.
// The verifier is set to address(0) as a placeholder; real deployments must supply a valid verifier.
func ZKDisputeGameConfigForRuntime(t devtest.CommonT) *embedded.ZKDisputeGameConfig {
	return &embedded.ZKDisputeGameConfig{
		AbsolutePrestate:     common.Hash{},    // placeholder for devstack
		Verifier:             common.Address{}, // address(0) — external verifier not yet wired
		MaxChallengeDuration: 604800,           // 7 days
		MaxProveDuration:     259200,           // 3 days
		ChallengerBond:       eth.GWei(80_000_000).ToBig(),
	}
}

func PrestateForGameType(t devtest.CommonT, gameType gameTypes.GameType) common.Hash {
	switch gameType {
	case gameTypes.CannonGameType:
		return getAbsolutePrestate(t, "op-program/bin/prestate-proof-mt64.json")
	case gameTypes.CannonKonaGameType:
		return getCannonKonaAbsolutePrestate(t)
	default:
		t.Require().Fail("no prestate available for game type", gameType)
		return common.Hash{}
	}
}

func LocalArtifacts(t devtest.T) *artifacts.Locator {
	require := t.Require()
	_, testFilename, _, ok := runtime.Caller(0)
	require.Truef(ok, "failed to get test filename")
	monorepoDir, err := op_service.FindMonorepoRoot(testFilename)
	require.NoError(err, "failed to find monorepo root")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")
	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(err, "failed to parse artifacts dir url")
	loc := &artifacts.Locator{
		URL: artifactsURL,
	}
	return loc
}
