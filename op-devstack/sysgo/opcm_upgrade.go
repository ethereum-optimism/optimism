package sysgo

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	w3eth "github.com/lmittmann/w3/module/eth"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
	var lastUsedOPCM common.Address
	require.NoError(w3Client.Call(w3eth.CallFunc(systemConfig, lastUsedOPCMFn).Returns(&lastUsedOPCM)), "failed to read the last used OPCM")
	require.NotEqual(common.Address{}, lastUsedOPCM, "SystemConfig %s has no last used OPCM", systemConfig)
	var respected uint32
	portal := l2.RollupConfig().DepositContractAddress
	require.NoError(w3Client.Call(w3eth.CallFunc(portal, respectedGameTypeFn).Returns(&respected)), "failed to read the respected game type")

	keys := stackKeys{l2.Keys()}
	l1PAOKey := l1ProxyAdminOwnerKey(t, keys, l2.L1().ChainID())
	executeOPCMUpgrade(t, ethclient.NewClient(rpcClient), l1PAOKey, lastUsedOPCM, opcm.UpgradeInputV2{
		SystemConfig:       systemConfig,
		DisputeGameConfigs: disputeGameConfigsForRuntime(t, keys, l2.ChainID(), enabledGameTypes),
		ExtraInstructions: []opcm.ExtraInstruction{
			{Key: "overrides.cfg.startingRespectedGameType", Data: encodeStartingRespectedGameType(t, respected)},
		},
	})
}

// stackKeys adapts a network's keys, which can't fail, to devkeys.Keys.
type stackKeys struct{ stack.Keys }

func (k stackKeys) Secret(key devkeys.Key) (*ecdsa.PrivateKey, error) { return k.Keys.Secret(key), nil }

func (k stackKeys) Address(key devkeys.Key) (common.Address, error) { return k.Keys.Address(key), nil }

// l1ProxyAdminOwnerKey returns the key of the L1 proxy-admin owner.
func l1ProxyAdminOwnerKey(t devtest.T, keys devkeys.Keys, l1ChainID eth.ChainID) *ecdsa.PrivateKey {
	role := devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole)
	key, err := keys.Secret(role)
	t.Require().NoError(err, "failed to resolve L1 proxy-admin owner key")
	return key
}

// executeOPCMUpgrade calls OPContractsManagerV2.upgrade(input) on L1 as a SetCode delegatecall from
// l1PAOKey, the L1 proxy-admin owner that the upgrade requires as caller.
func executeOPCMUpgrade(
	t devtest.T,
	client *ethclient.Client,
	l1PAOKey *ecdsa.PrivateKey,
	opcmAddr common.Address,
	input opcm.UpgradeInputV2,
) {
	calldata, err := input.UpgradeCalldata()
	t.Require().NoError(err, "failed to encode the OPCM upgrade")

	t.Log("Executing opcm.upgrade via SetCode delegatecall")
	t.Require().NoError(transactions.DelegateCallWithSetCode(t.Ctx(), client, l1PAOKey, opcmAddr, calldata))
}
