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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
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
// The V2 upgrade requires exactly 3 game configs (CANNON, PERMISSIONED_CANNON, CANNON_KONA)
// in that order. Game types in enabledGameTypes are enabled; the rest are disabled.
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

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())

	l1PAO, err := keys.Address(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner address")

	proposer, err := keys.Address(chainOps(devkeys.ProposerRole))
	require.NoError(err, "failed to get proposer address")

	challenger, err := keys.Address(chainOps(devkeys.ChallengerRole))
	require.NoError(err, "failed to get challenger address")

	// Build enabled set for quick lookup.
	enabled := make(map[gameTypes.GameType]bool)
	for _, gt := range enabledGameTypes {
		enabled[gt] = true
	}

	initBond := eth.GWei(80_000_000).ToBig() // 0.08 ETH

	// OPCMv2 requires all 6 game configs in order:
	// CANNON, PERMISSIONED_CANNON, CANNON_KONA, SUPER_CANNON, SUPER_PERMISSIONED_CANNON, SUPER_CANNON_KONA.
	cannonPrestate := PrestateForGameType(t, gameTypes.CannonGameType)
	cannonKonaPrestate := PrestateForGameType(t, gameTypes.CannonKonaGameType)

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
		{
			Enabled:  false,
			InitBond: new(big.Int),
			GameType: embedded.GameTypeSuperCannon,
		},
		{
			Enabled:  false,
			InitBond: new(big.Int),
			GameType: embedded.GameTypeSuperPermCannon,
		},
		{
			Enabled:  false,
			InitBond: new(big.Int),
			GameType: embedded.GameTypeSuperCannonKona,
		},
	}

	// Zero out init bond for disabled games.
	for i := range configs {
		if !configs[i].Enabled {
			configs[i].InitBond = new(big.Int)
		}
	}

	upgradeInput := embedded.UpgradeOPChainInput{
		Prank: l1PAO,
		Opcm:  l2Net.opcmImpl,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       l2Net.deployment.SystemConfigProxyAddr(),
			DisputeGameConfigs: configs,
			ExtraInstructions: []embedded.ExtraInstruction{
				{
					Key:  "PermittedProxyDeployment",
					Data: []byte("DelayedWETH"),
				},
			},
		},
	}

	// Run UpgradeOPChain.s.sol via a forked script host to produce calldata.
	loc := LocalArtifacts(t)
	artifactsFS, err := artifacts.Download(t.Ctx(), loc, ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	bcaster := new(broadcaster.CalldataBroadcaster)
	host, err := env.DefaultForkedScriptHost(
		t.Ctx(),
		bcaster,
		t.Logger(),
		common.Address{'D'},
		artifactsFS,
		rpcClient,
	)
	require.NoError(err, "failed to create script host")

	err = embedded.Upgrade(host, upgradeInput)
	require.NoError(err, "failed to run upgrade script for add game types")

	calldata, err := bcaster.Dump()
	require.NoError(err, "failed to dump calldata")
	require.Len(calldata, 1, "calldata must contain one entry")

	l1PAOKey, err := keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner key")

	t.Log("Executing opcmV2.upgrade via SetCode delegatecall")
	delegateCallWithSetCode(t, l1PAOKey, client, l2Net.opcmImpl, calldata[0].Data)
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
