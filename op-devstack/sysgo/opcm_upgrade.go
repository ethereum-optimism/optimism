package sysgo

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

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
