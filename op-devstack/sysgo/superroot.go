package sysgo

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/delegatecallproxy"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/errutil"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	w3eth "github.com/lmittmann/w3/module/eth"
)

// V2 structs for OPCM >= 7.0.0 (using IOPContractsManagerMigrator interface)
type DisputeGameConfigV2 struct {
	Enabled  bool
	InitBond *big.Int
	GameType uint32
	GameArgs []byte
}

type MigrateInputV2 struct {
	ChainSystemConfigs        []common.Address
	DisputeGameConfigs        []DisputeGameConfigV2
	StartingAnchorRoot        bindings.Proposal
	StartingRespectedGameType uint32
}

func deployDelegateCallProxy(t devtest.CommonT, transactOpts *bind.TransactOpts, client *ethclient.Client, owner common.Address) (common.Address, *delegatecallproxy.Delegatecallproxy) {
	deployAddress, tx, proxyContract, err := delegatecallproxy.DeployDelegatecallproxy(transactOpts, client, owner)
	t.Require().NoError(err, "DelegateCallProxy deployment failed")
	// Make sure the transaction actually got included rather than just being sent
	_, err = wait.ForReceiptOK(t.Ctx(), client, tx.Hash())
	t.Require().NoError(err, "DelegateCallProxy deployment tx was not included successfully")
	return deployAddress, proxyContract
}

func awaitSuperrootTime(t devtest.T, cls ...L2CLNode) uint64 {
	t.Require().NotEmpty(cls, "at least one L2 CL is required")

	var superrootTime uint64
	for _, l2CL := range cls {
		rollupClient, err := dial.DialRollupClientWithTimeout(t.Ctx(), t.Logger(), l2CL.UserRPC())
		t.Require().NoError(err)
		defer rollupClient.Close()

		ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
		err = wait.For(ctx, time.Second, func() (bool, error) {
			status, err := rollupClient.SyncStatus(ctx)
			if err != nil {
				return false, err
			}
			if status == nil || status.SafeL2.Number == 0 {
				return false, nil
			}
			superrootTime = status.SafeL2.Time
			return true, nil
		})
		cancel()
		t.Require().NoError(err, "waiting for chain safe head to advance failed")
	}
	return superrootTime
}

func getSupervisorSuperRoot(t devtest.T, supervisor Supervisor, timestamp uint64) eth.Bytes32 {
	client, err := dial.DialSupervisorClientWithTimeout(t.Ctx(), t.Logger(), supervisor.UserRPC())
	t.Require().NoError(err)

	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	err = wait.For(ctx, time.Second, func() (bool, error) {
		status, err := client.SyncStatus(ctx)
		if err != nil {
			return false, err
		}
		return timestamp < status.MinSyncedL1.Time, nil
	})
	cancel()
	t.Require().NoError(err, "waiting for supervisor to sync failed")

	super, err := client.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(timestamp))
	t.Require().NoError(err, "super root at timestamp failed")
	return super.SuperRoot
}

func getSupernodeSuperRoot(t devtest.T, supernode *SuperNode, timestamp uint64) eth.Bytes32 {
	client, err := dial.DialSuperNodeClientWithTimeout(t.Ctx(), t.Logger(), supernode.UserRPC())
	t.Require().NoError(err)

	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	err = wait.For(ctx, time.Second, func() (bool, error) {
		resp, err := client.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			t.Logf("DEBUG: Failed to get super root at timestamp %d: err: %v", timestamp, err)
			return false, err
		}
		return resp.Data != nil, nil
	})
	cancel()
	t.Require().NoError(err, "waiting for supernode superroot to be ready failed")

	resp, err := client.SuperRootAtTimestamp(t.Ctx(), timestamp)
	t.Require().NoError(err, "super root at timestamp failed")
	t.Require().NotNil(resp.Data, "super root data must be present")
	return resp.Data.SuperRoot
}

func migrateSuperRoots(
	t devtest.T,
	keys devkeys.Keys,
	migration *interopMigrationState,
	l1ChainID eth.ChainID,
	l1EL L1ELNode,
	superRoot eth.Bytes32,
	superrootTime uint64,
	primaryL2 eth.ChainID,
) common.Address {
	require := t.Require()
	require.NotNil(migration, "interop migration state is required")
	require.NotEmpty(migration.opcmImpl, "must have an OPCM implementation")
	require.NotEmpty(migration.superchainConfigAddr, "must have a superchain deployment")
	require.NotEmpty(migration.l2Deployments, "must have L2 deployments for interop migration")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	client := ethclient.NewClient(rpcClient)
	w3Client := w3.NewClient(rpcClient)

	l1pao, err := keys.Address(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "must have L1 proxy admin owner private key")

	superchainProxyAdmin := getProxyAdmin(t, w3Client, migration.superchainConfigAddr)
	require.NotEmpty(superchainProxyAdmin, "superchain proxy admin address is empty")

	useV2 := isOPCMV2(t, w3Client, migration.opcmImpl)
	absoluteCannonPrestate := getInteropCannonAbsolutePrestate(t)
	absoluteCannonKonaPrestate := getInteropCannonKonaAbsolutePrestate(t)

	permissionedChainOps := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(permissionedChainOps(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")
	challenger, err := keys.Address(permissionedChainOps(devkeys.ChallengerRole))
	require.NoError(err, "must have configured challenger")

	var opChainConfigs []bindings.OPContractsManagerOpChainConfig
	var l2ChainIDs []eth.ChainID
	for l2ChainID, l2Deployment := range migration.l2Deployments {
		l2ChainIDs = append(l2ChainIDs, l2ChainID)
		opChainConfigs = append(opChainConfigs, bindings.OPContractsManagerOpChainConfig{
			SystemConfigProxy:  l2Deployment.SystemConfigProxyAddr(),
			CannonPrestate:     absoluteCannonPrestate,
			CannonKonaPrestate: absoluteCannonKonaPrestate,
		})
	}

	opcmABI, err := bindings.OPContractsManagerMetaData.GetAbi()
	require.NoError(err, "invalid OPCM ABI")
	contract := batching.NewBoundContract(opcmABI, migration.opcmImpl)

	var migrateCallData []byte
	if useV2 {
		var chainSystemConfigs []common.Address
		for _, cfg := range opChainConfigs {
			chainSystemConfigs = append(chainSystemConfigs, cfg.SystemConfigProxy)
		}
		migrateInputV2 := MigrateInputV2{
			ChainSystemConfigs: chainSystemConfigs,
			DisputeGameConfigs: []DisputeGameConfigV2{
				{
					Enabled:  true,
					InitBond: big.NewInt(0),
					GameType: superCannonGameType,
					GameArgs: absoluteCannonPrestate[:],
				},
			},
			StartingAnchorRoot: bindings.Proposal{
				Root:             common.Hash(superRoot),
				L2SequenceNumber: big.NewInt(int64(superrootTime)),
			},
			StartingRespectedGameType: superCannonGameType,
		}
		migrateCall := contract.Call("migrate", migrateInputV2)
		migrateCallData, err = migrateCall.Pack()
		require.NoError(err)
	} else {
		migrateInputV1 := bindings.OPContractsManagerInteropMigratorMigrateInput{
			UsePermissionlessGame: true,
			StartingAnchorRoot: bindings.Proposal{
				Root:             common.Hash(superRoot),
				L2SequenceNumber: big.NewInt(int64(superrootTime)),
			},
			GameParameters: bindings.OPContractsManagerInteropMigratorGameParameters{
				Proposer:         proposer,
				Challenger:       challenger,
				MaxGameDepth:     big.NewInt(73),
				SplitDepth:       big.NewInt(30),
				InitBond:         big.NewInt(0),
				ClockExtension:   10800,
				MaxClockDuration: 302400,
			},
			OpChainConfigs: opChainConfigs,
		}
		migrateCall := contract.Call("migrate", migrateInputV1)
		migrateCallData, err = migrateCall.Pack()
		require.NoError(err)
	}

	l1PAOKey, err := keys.Secret(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "must have configured L1 proxy admin owner")
	transactOpts, err := bind.NewKeyedTransactorWithChainID(l1PAOKey, l1ChainID.ToBig())
	require.NoError(err, "must have transact opts")
	transactOpts.Context = t.Ctx()

	t.Log("Deploying delegate call proxy contract")
	delegateCallProxy, proxyContract := deployDelegateCallProxy(t, transactOpts, client, l1pao)
	oldSuperchainProxyAdminOwner := getOwner(t, w3Client, superchainProxyAdmin)
	transferOwnership(t, l1PAOKey, client, superchainProxyAdmin, delegateCallProxy)

	oldDisputeGameFactories := make(map[eth.ChainID]common.Address)
	for i, opChainConfig := range opChainConfigs {
		var portal common.Address
		require.NoError(w3Client.Call(w3eth.CallFunc(opChainConfig.SystemConfigProxy, optimismPortalFn).Returns(&portal)))
		portalProxyAdmin := getProxyAdmin(t, w3Client, portal)
		transferOwnership(t, l1PAOKey, client, portalProxyAdmin, delegateCallProxy)

		dgf := getDisputeGameFactory(t, w3Client, portal)
		transferOwnership(t, l1PAOKey, client, dgf, delegateCallProxy)
		oldDisputeGameFactories[l2ChainIDs[i]] = dgf
	}

	t.Log("Executing delegate call")
	migrateTx, err := proxyContract.ExecuteDelegateCall(transactOpts, migration.opcmImpl, migrateCallData)
	require.NoErrorf(err, "migrate delegatecall failed: %v", errutil.TryAddRevertReason(err))
	_, err = wait.ForReceiptOK(t.Ctx(), client, migrateTx.Hash())
	require.NoError(err)

	var sharedDGF common.Address
	for _, l2Deployment := range migration.l2Deployments {
		portal := getOptimismPortal(t, w3Client, l2Deployment.SystemConfigProxyAddr())
		addr := getDisputeGameFactory(t, w3Client, portal)
		if sharedDGF == (common.Address{}) {
			sharedDGF = addr
		} else {
			require.Equal(sharedDGF, addr, "dispute game factory address is not the same for all deployments")
		}
	}
	require.NotEmpty(getSuperGameImpl(t, w3Client, sharedDGF))

	resetOwnershipAfterMigration(
		t,
		keys,
		l1ChainID.ToBig(),
		l1PAOKey,
		w3Client,
		client,
		delegateCallProxy,
		opChainConfigs,
	)
	resetOldDisputeGameFactoriesAfterMigration(
		t,
		keys,
		l1ChainID.ToBig(),
		l1PAOKey,
		client,
		delegateCallProxy,
		oldDisputeGameFactories,
	)
	transferOwnershipForDelegateCallProxy(t, l1ChainID.ToBig(), l1PAOKey, client, delegateCallProxy, superchainProxyAdmin, oldSuperchainProxyAdminOwner)

	superchainProxyAdminOwner := getOwner(t, w3Client, superchainProxyAdmin)
	require.Equal(oldSuperchainProxyAdminOwner, superchainProxyAdminOwner, "superchain proxy admin owner is not the L1PAO")

	for chainID, l2Deployment := range migration.l2Deployments {
		l2Deployment.disputeGameFactoryProxy = sharedDGF
		migration.l2Deployments[chainID] = l2Deployment
	}
	t.Log("Interop migration complete")
	return sharedDGF
}

func getInteropCannonAbsolutePrestate(t devtest.CommonT) common.Hash {
	return getAbsolutePrestate(t, "op-program/bin/prestate-proof-interop.json")
}

func getInteropCannonKonaAbsolutePrestate(t devtest.CommonT) common.Hash {
	return getAbsolutePrestate(t, "rust/kona/prestate-artifacts-cannon-interop/prestate-proof.json")
}

func getCannonKonaAbsolutePrestate(t devtest.CommonT) common.Hash {
	return getAbsolutePrestate(t, "rust/kona/prestate-artifacts-cannon/prestate-proof.json")
}

func getAbsolutePrestate(t devtest.CommonT, prestatePath string) common.Hash {
	root, err := findMonorepoRoot(prestatePath)
	t.Require().NoError(err)
	p := path.Join(root, prestatePath)
	file, err := os.Open(p)
	t.Require().NoError(err)
	decoder := json.NewDecoder(file)
	var prestate map[string]interface{}
	err = decoder.Decode(&prestate)
	t.Require().NoError(err)
	t.Require().NotEmpty(prestate, "prestate is empty")
	return common.HexToHash(prestate["pre"].(string))
}

const (
	superCannonGameType       = 4
	superPermissionedGameType = 5
)

var (
	optimismPortalFn      = w3.MustNewFunc("optimismPortal()", "address")
	disputeGameFactoryFn  = w3.MustNewFunc("disputeGameFactory()", "address")
	gameImplsFn           = w3.MustNewFunc("gameImpls(uint32)", "address")
	gameArgsFn            = w3.MustNewFunc("gameArgs(uint32)", "bytes")
	ownerFn               = w3.MustNewFunc("owner()", "address")
	proxyAdminFn          = w3.MustNewFunc("proxyAdmin()", "address")
	adminFn               = w3.MustNewFunc("admin()", "address")
	proxyAdminOwnerFn     = w3.MustNewFunc("proxyAdminOwner()", "address")
	ethLockboxFn          = w3.MustNewFunc("ethLockbox()", "address")
	anchorStateRegistryFn = w3.MustNewFunc("anchorStateRegistry()", "address")
	transferOwnershipFn   = w3.MustNewFunc("transferOwnership(address)", "")
	versionFn             = w3.MustNewFunc("version()", "string")
)

// isOPCMV2 is a helper function that checks the OPCM version and returns true if it is at least 7.0.0
func isOPCMV2(t devtest.CommonT, client *w3.Client, opcmAddr common.Address) bool {
	var version string
	err := client.Call(w3eth.CallFunc(opcmAddr, versionFn).Returns(&version))
	t.Require().NoError(err, "failed to get OPCM version")

	isVersionAtLeast, err := deployer.IsVersionAtLeast(version, 7, 0, 0)
	t.Require().NoError(err, "failed to check OPCM version")
	return isVersionAtLeast
}

func getOptimismPortal(t devtest.CommonT, client *w3.Client, systemConfigProxy common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(systemConfigProxy, optimismPortalFn).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

func getDisputeGameFactory(t devtest.CommonT, client *w3.Client, portal common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(portal, disputeGameFactoryFn).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

func getSuperGameImpl(t devtest.CommonT, client *w3.Client, dgf common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(dgf, gameImplsFn, uint32(superCannonGameType)).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

func getOwner(t devtest.CommonT, client *w3.Client, addr common.Address) common.Address {
	var owner common.Address
	err := client.Call(w3eth.CallFunc(addr, ownerFn).Returns(&owner))
	t.Require().NoError(err)
	return owner
}

func getAdmin(t devtest.CommonT, client *w3.Client, addr common.Address) common.Address {
	var admin common.Address
	err := client.Call(w3eth.CallFunc(addr, adminFn).Returns(&admin))
	t.Require().NoError(err)
	return admin
}

func getProxyAdminOwner(t devtest.CommonT, client *w3.Client, addr common.Address) common.Address {
	var proxyAdminOwner common.Address
	err := client.Call(w3eth.CallFunc(addr, proxyAdminOwnerFn).Returns(&proxyAdminOwner))
	t.Require().NoError(err)
	return proxyAdminOwner
}

func getProxyAdmin(t devtest.CommonT, client *w3.Client, addr common.Address) common.Address {
	var proxyAdmin common.Address
	err := client.Call(w3eth.CallFunc(addr, proxyAdminFn).Returns(&proxyAdmin))
	t.Require().NoError(err)
	return proxyAdmin
}

func transferOwnership(t devtest.CommonT, privateKey *ecdsa.PrivateKey, client *ethclient.Client, l1ProxyAdmin common.Address, newOwner common.Address) {
	data, err := transferOwnershipFn.EncodeArgs(newOwner)
	t.Require().NoError(err)

	candidate := txmgr.TxCandidate{
		To:       &l1ProxyAdmin,
		TxData:   data,
		GasLimit: 1_000_000,
	}
	_, receipt, err := transactions.SendTx(t.Ctx(), client, candidate, privateKey)
	t.Require().NoErrorf(err, "transferOwnership failed: %v", errutil.TryAddRevertReason(err))
	t.Require().Equal(receipt.Status, types.ReceiptStatusSuccessful, "transferOwnership failed")
}

func transferOwnershipForDelegateCallProxy(
	t devtest.CommonT,
	transactChainID *big.Int,
	privateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	delegateCallProxy common.Address,
	proxyAdminOwned common.Address,
	newOwner common.Address,
) {
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, transactChainID)
	t.Require().NoError(err, "must have transact opts")
	transactOpts.Context = t.Ctx()

	abi, err := delegatecallproxy.DelegatecallproxyMetaData.GetAbi()
	t.Require().NoError(err, "failed to get abi")
	contract := batching.NewBoundContract(abi, delegateCallProxy)
	call := contract.Call("transferOwnership", proxyAdminOwned, newOwner)
	data, err := call.Pack()
	t.Require().NoError(err)

	candidate := txmgr.TxCandidate{
		To:       &delegateCallProxy,
		TxData:   data,
		GasLimit: 1_000_000,
	}
	_, receipt, err := transactions.SendTx(t.Ctx(), client, candidate, privateKey)
	t.Require().NoErrorf(err, "transferOwnership failed: %v", errutil.TryAddRevertReason(err))
	t.Require().Equal(receipt.Status, types.ReceiptStatusSuccessful, "transferOwnership failed")
}

func resetOwnershipAfterMigration(
	t devtest.CommonT,
	keys devkeys.Keys,
	l1ChainID *big.Int,
	ownerPrivateKey *ecdsa.PrivateKey,
	w3Client *w3.Client,
	client *ethclient.Client,
	delegateCallProxy common.Address,
	opChainConfigs []bindings.OPContractsManagerOpChainConfig,
) {
	l1PAO, err := keys.Address(devkeys.ChainOperatorKeys(l1ChainID)(devkeys.L1ProxyAdminOwnerRole))
	t.Require().NoError(err, "must have L1 proxy admin owner private key")

	portal0 := getOptimismPortal(t, w3Client, opChainConfigs[0].SystemConfigProxy)
	sharedDGF := getDisputeGameFactory(t, w3Client, portal0)
	transferOwnershipForDelegateCallProxy(
		t,
		l1ChainID,
		ownerPrivateKey,
		client,
		delegateCallProxy,
		sharedDGF,
		l1PAO,
	)

	var sharedEthLockboxProxy common.Address
	err = w3Client.Call(w3eth.CallFunc(portal0, ethLockboxFn).Returns(&sharedEthLockboxProxy))
	t.Require().NoError(err)
	proxyAdmin := getAdmin(t, w3Client, sharedEthLockboxProxy)
	transferOwnershipForDelegateCallProxy(
		t,
		l1ChainID,
		ownerPrivateKey,
		client,
		delegateCallProxy,
		proxyAdmin,
		l1PAO,
	)

	for _, cfg := range opChainConfigs {
		portal := getOptimismPortal(t, w3Client, cfg.SystemConfigProxy)
		portalProxyAdmin := getProxyAdmin(t, w3Client, portal)
		if getOwner(t, w3Client, portalProxyAdmin) == delegateCallProxy {
			transferOwnershipForDelegateCallProxy(
				t,
				l1ChainID,
				ownerPrivateKey,
				client,
				delegateCallProxy,
				portalProxyAdmin,
				l1PAO,
			)
		}
	}

	var sharedAnchorStateRegistryProxy common.Address
	err = w3Client.Call(w3eth.CallFunc(portal0, anchorStateRegistryFn).Returns(&sharedAnchorStateRegistryProxy))
	t.Require().NoError(err)
	asrAAdminOwner := getProxyAdminOwner(t, w3Client, sharedAnchorStateRegistryProxy)
	t.Require().Equal(l1PAO, asrAAdminOwner, "sharedAnchorStateRegistryProxy proxy admin owner is not the L1PAO")

	gameTypes := []uint32{superPermissionedGameType, superCannonGameType}
	for _, gameType := range gameTypes {
		var gameArgsBytes []byte
		err = w3Client.Call(w3eth.CallFunc(sharedDGF, gameArgsFn, gameType).Returns(&gameArgsBytes))
		t.Require().NoError(err)
		gameArgs, err := gameargs.Parse(gameArgsBytes)
		t.Require().NoErrorf(err, "invalid game args for gameType %d", gameType)
		wethAdminOwner := getProxyAdminOwner(t, w3Client, gameArgs.Weth)
		t.Require().Equal(l1PAO, wethAdminOwner, "wethProxy proxy admin owner is not the L1PAO")
	}
}

func resetOldDisputeGameFactoriesAfterMigration(
	t devtest.CommonT,
	keys devkeys.Keys,
	l1ChainID *big.Int,
	ownerPrivateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	delegateCallProxy common.Address,
	oldDisputeGameFactories map[eth.ChainID]common.Address,
) {
	for l2ChainID, oldDGF := range oldDisputeGameFactories {
		chainOpsForL2 := devkeys.ChainOperatorKeys(l2ChainID.ToBig())
		l1PAOForL2, err := keys.Address(chainOpsForL2(devkeys.L1ProxyAdminOwnerRole))
		t.Require().NoError(err, "must have configured L1 proxy admin owner private key")
		transferOwnershipForDelegateCallProxy(
			t,
			l1ChainID,
			ownerPrivateKey,
			client,
			delegateCallProxy,
			oldDGF,
			l1PAOForL2,
		)
	}
}
