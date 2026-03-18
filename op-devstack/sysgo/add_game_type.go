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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/manage"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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

func addGameTypeForRuntime(
	t devtest.T,
	keys devkeys.Keys,
	absolutePrestate common.Hash,
	gameType gameTypes.GameType,
	l1ChainID eth.ChainID,
	l1ELRPC string,
	l2Net *L2Network,
) {
	require := t.Require()
	require.NotNil(l2Net, "l2 network must exist")
	require.NotNil(l2Net.deployment, "l2 deployment must exist")
	require.NotEqual(common.Address{}, l2Net.opcmImpl, "missing OPCM implementation address")
	require.NotEqual(common.Address{}, l2Net.mipsImpl, "missing MIPS implementation address")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1ELRPC)
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)

	l1PAO, err := keys.Address(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner address")

	cfg := manage.AddGameTypeConfig{
		L1RPCUrl:                l1ELRPC,
		Logger:                  t.Logger(),
		ArtifactsLocator:        LocalArtifacts(t),
		CacheDir:                t.TempDir(),
		L1ProxyAdminOwner:       l1PAO,
		OPCMImpl:                l2Net.opcmImpl,
		SystemConfigProxy:       l2Net.deployment.SystemConfigProxyAddr(),
		DelayedWETHProxy:        l2Net.deployment.PermissionlessDelayedWETHProxyAddr(),
		DisputeGameType:         uint32(gameType),
		DisputeAbsolutePrestate: absolutePrestate,
		DisputeMaxGameDepth:     big.NewInt(73),
		DisputeSplitDepth:       big.NewInt(30),
		DisputeClockExtension:   10800,
		DisputeMaxClockDuration: 302400,
		InitialBond:             eth.GWei(80_000_000).ToBig(), // 0.08 ETH
		VM:                      l2Net.mipsImpl,
		Permissionless:          true,
		SaltMixer:               fmt.Sprintf("devstack-%s-%s", l2Net.ChainID(), absolutePrestate.Hex()),
	}

	opChainProxyAdmin := l2Net.deployment.ProxyAdminAddr()

	_, addGameTypeCalldata, err := manage.AddGameType(t.Ctx(), cfg)
	require.NoError(err, "failed to create add game type calldata")
	require.Len(addGameTypeCalldata, 1, "calldata must contain one entry")

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	l1PAOKey, err := keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner key")
	transactOpts, err := bind.NewKeyedTransactorWithChainID(l1PAOKey, l1ChainID.ToBig())
	require.NoError(err, "must have transact opts")
	transactOpts.Context = t.Ctx()

	t.Log("Deploying delegate call proxy contract")
	delegateCallProxy, proxyContract := deployDelegateCallProxy(t, transactOpts, client, l1PAO)
	// transfer ownership to the proxy so that we can delegatecall the opcm
	transferOwnership(t, l1PAOKey, client, opChainProxyAdmin, delegateCallProxy)
	dgf := l2Net.deployment.DisputeGameFactoryProxyAddr()
	transferOwnership(t, l1PAOKey, client, dgf, delegateCallProxy)

	t.Log("sending opcm.addGameType transaction")
	tx, err := proxyContract.ExecuteDelegateCall(transactOpts, l2Net.opcmImpl, addGameTypeCalldata[0].Data)
	require.NoError(err, "failed to send add game type tx")
	_, err = wait.ForReceiptOK(t.Ctx(), client, tx.Hash())
	require.NoError(err, "failed to wait for add game type receipt")

	// reset ProxyAdmin ownership transfers
	transferOwnershipForDelegateCallProxy(t, l1ChainID.ToBig(), l1PAOKey, client, delegateCallProxy, opChainProxyAdmin, l1PAO)
	transferOwnershipForDelegateCallProxy(t, l1ChainID.ToBig(), l1PAOKey, client, delegateCallProxy, dgf, l1PAO)
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
