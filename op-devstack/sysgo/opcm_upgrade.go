package sysgo

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	w3eth "github.com/lmittmann/w3/module/eth"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

var (
	lastUsedOPCMFn      = w3.MustNewFunc("lastUsedOPCM()", "address")
	respectedGameTypeFn = w3.MustNewFunc("respectedGameType()", "uint32")
)

// RerunOPCMUpgrade re-runs OPCMv2.upgrade on a chain after setup, with the OPCM that last upgraded it, so a
// test can create state before an upgrade and check it after. The upgrade re-initializes the chain's
// contracts and keeps its respected game type. OPCMv2 requires a config for every game type, so game types
// not in enabledGameTypes are disabled, as with WithGameTypeAdded.
func RerunOPCMUpgrade(t devtest.T, l1EL stack.L1ELNode, l2 stack.L2Network, enabledGameTypes ...gameTypes.GameType) {
	require := t.Require()
	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	defer rpcClient.Close()
	w3Client := w3.NewClient(rpcClient)

	systemConfig := l2.Deployment().SystemConfigProxyAddr()
	var opcm common.Address
	require.NoError(w3Client.Call(w3eth.CallFunc(systemConfig, lastUsedOPCMFn).Returns(&opcm)), "failed to read the last used OPCM")
	require.NotEqual(common.Address{}, opcm, "SystemConfig %s has no last used OPCM", systemConfig)
	var respected uint32
	portal := l2.RollupConfig().DepositContractAddress
	require.NoError(w3Client.Call(w3eth.CallFunc(portal, respectedGameTypeFn).Returns(&respected)), "failed to read the respected game type")

	keys := stackKeys{l2.Keys()}
	l1PAO, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l2.L1().ChainID())
	artifactsFS, err := artifacts.Download(t.Ctx(), LocalArtifacts(t), ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	executeOPCMUpgrade(t, rpcClient, ethclient.NewClient(rpcClient), l1PAOKey, artifactsFS, embedded.UpgradeOPChainInput{
		Prank: l1PAO,
		Opcm:  opcm,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       systemConfig,
			DisputeGameConfigs: disputeGameConfigsForRuntime(t, keys, l2.ChainID(), enabledGameTypes),
			ExtraInstructions: []embedded.ExtraInstruction{
				{Key: "overrides.cfg.startingRespectedGameType", Data: encodeStartingRespectedGameType(t, respected)},
			},
		},
	})
}

// stackKeys adapts a network's keys, which can't fail, to devkeys.Keys.
type stackKeys struct{ stack.Keys }

func (k stackKeys) Secret(key devkeys.Key) (*ecdsa.PrivateKey, error) { return k.Keys.Secret(key), nil }

func (k stackKeys) Address(key devkeys.Key) (common.Address, error) { return k.Keys.Address(key), nil }

// resolveL1ProxyAdminOwner returns the L1 proxy-admin owner address and key.
func resolveL1ProxyAdminOwner(t devtest.T, keys devkeys.Keys, l1ChainID eth.ChainID) (common.Address, *ecdsa.PrivateKey) {
	require := t.Require()
	role := devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole)
	addr, err := keys.Address(role)
	require.NoError(err, "failed to resolve L1 proxy-admin owner address")
	key, err := keys.Secret(role)
	require.NoError(err, "failed to resolve L1 proxy-admin owner key")
	return addr, key
}

// executeOPCMUpgrade simulates input against a forked UpgradeOPChain.s.sol
// script host and submits the resulting calldata on L1 as a SetCode
// delegatecall from l1PAOKey.
func executeOPCMUpgrade(
	t devtest.T,
	rpcClient *rpc.Client,
	client *ethclient.Client,
	l1PAOKey *ecdsa.PrivateKey,
	artifactsFS foundry.StatDirFs,
	input embedded.UpgradeOPChainInput,
) {
	require := t.Require()
	bcaster := new(broadcaster.CalldataBroadcaster)
	host, err := env.DefaultForkedScriptHost(
		t.Ctx(), bcaster, t.Logger(), common.Address{'D'}, artifactsFS, rpcClient,
	)
	require.NoError(err, "failed to create script host")
	require.NoError(embedded.Upgrade(host, input), "failed to run UpgradeOPChain.s.sol")

	calldata, err := bcaster.Dump()
	require.NoError(err, "failed to dump calldata")
	require.Len(calldata, 1, "calldata must contain one entry")

	t.Log("Executing opcm.upgrade via SetCode delegatecall")
	delegateCallWithSetCode(t, l1PAOKey, client, input.Opcm, calldata[0].Data)
}
