package sysgo

import (
	"fmt"
	"math/big"
	"net/url"
	"path"
	"runtime"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/manage"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func WithCannonGameTypeAdded(l1ELID stack.L1ELNodeID, l2ChainID eth.ChainID) stack.Option[*Orchestrator] {
	return stack.FnOption[*Orchestrator]{
		FinallyFn: func(o *Orchestrator) {
			// TODO(#17867): Rebuild the op-program prestate using the newly minted L2 chain configs before using it.
			absolutePrestate := getAbsolutePrestate(o.P(), "op-program/bin/prestate-proof-mt64.json")
			addGameType(o, absolutePrestate, 0 /* CANNON */, l1ELID, l2ChainID)
		},
	}
}

func addGameType(o *Orchestrator, absolutePrestate common.Hash, gameType uint32, l1ELID stack.L1ELNodeID, l2ChainID eth.ChainID) {
	t := o.P()
	require := t.Require()
	require.NotNil(o.wb, "must have a world builder")
	l1ChainID := l1ELID.ChainID()

	opcmAddr := o.wb.output.ImplementationsDeployment.OpcmImpl

	l1EL, ok := o.l1ELs.Get(l1ELID)
	require.True(ok, "l1El must exist")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	client := ethclient.NewClient(rpcClient)

	l1PAO, err := o.keys.Address(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner address")

	cfg := manage.AddGameTypeConfig{
		L1RPCUrl:                l1EL.UserRPC(),
		Logger:                  t.Logger(),
		ArtifactsLocator:        LocalArtifacts(t),
		CacheDir:                t.TempDir(),
		L1ProxyAdminOwner:       l1PAO,
		OPCMImpl:                opcmAddr,
		SystemConfigProxy:       o.wb.outL2Deployment[l2ChainID].SystemConfigProxyAddr(),
		OPChainProxyAdmin:       o.wb.outL2Deployment[l2ChainID].ProxyAdminAddr(),
		DelayedWETHProxy:        o.wb.outL2Deployment[l2ChainID].PermissionlessDelayedWETHProxyAddr(),
		DisputeGameType:         gameType,
		DisputeAbsolutePrestate: absolutePrestate,
		DisputeMaxGameDepth:     big.NewInt(73),
		DisputeSplitDepth:       big.NewInt(30),
		DisputeClockExtension:   10800,
		DisputeMaxClockDuration: 302400,
		InitialBond:             eth.GWei(80_000_000).ToBig(), // 0.08 ETH
		VM:                      o.wb.output.ImplementationsDeployment.MipsImpl,
		Permissionless:          true,
		SaltMixer:               fmt.Sprintf("devstack-%s-%s", l2ChainID, absolutePrestate.Hex()),
	}

	_, addGameTypeCalldata, err := manage.AddGameType(t.Ctx(), cfg)
	require.NoError(err, "failed to create add game type calldata")
	require.Len(addGameTypeCalldata, 1, "calldata must contain one entry")

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	l1PAOKey, err := o.keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get l1 proxy admin owner key")
	transactOpts, err := bind.NewKeyedTransactorWithChainID(l1PAOKey, l1ChainID.ToBig())
	require.NoError(err, "must have transact opts")
	transactOpts.Context = t.Ctx()

	t.Log("Deploying delegate call proxy contract")
	delegateCallProxy, proxyContract := deployDelegateCallProxy(t, transactOpts, client, l1PAO)
	// transfer ownership to the proxy so that we can delegatecall the opcm
	transferOwnership(t, l1PAOKey, client, cfg.OPChainProxyAdmin, delegateCallProxy)
	dgf := o.wb.outL2Deployment[l2ChainID].DisputeGameFactoryProxyAddr()
	transferOwnership(t, l1PAOKey, client, dgf, delegateCallProxy)

	t.Log("sending opcm.addGameType transaction")
	tx, err := proxyContract.ExecuteDelegateCall(transactOpts, opcmAddr, addGameTypeCalldata[0].Data)
	require.NoError(err, "failed to send add game type tx")
	_, err = wait.ForReceiptOK(t.Ctx(), client, tx.Hash())
	require.NoError(err, "failed to wait for add game type receipt")

	// reset ProxyAdmin ownership transfers
	transferOwnershipForDelegateCallProxy(t, l1ChainID.ToBig(), l1PAOKey, client, delegateCallProxy, cfg.OPChainProxyAdmin, l1PAO)
	transferOwnershipForDelegateCallProxy(t, l1ChainID.ToBig(), l1PAOKey, client, delegateCallProxy, dgf, l1PAO)
}

func LocalArtifacts(t devtest.P) *artifacts.Locator {
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
