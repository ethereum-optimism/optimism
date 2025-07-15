package sysgo

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"path"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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

func WithSuperRoots(l1ChainID eth.ChainID, l1ELID stack.L1ELNodeID, l2CLID stack.L2CLNodeID, supervisorID stack.SupervisorID, primaryL2 eth.ChainID) stack.Option[*Orchestrator] {
	return stack.FnOption[*Orchestrator]{
		FinallyFn: func(o *Orchestrator) {
			t := o.P()
			require := t.Require()
			require.NotNil(o.wb, "must have a world builder")
			require.NotEmpty(o.wb.output.ImplementationsDeployment.OpcmImpl, "must have an OPCM implementation")

			l1EL, ok := o.l1ELs.Get(l1ELID)
			require.True(ok, "must have L1 EL node")
			rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.userRPC)
			require.NoError(err)
			client := ethclient.NewClient(rpcClient)
			w3Client := w3.NewClient(rpcClient)

			l2CL, ok := o.l2CLs.Get(l2CLID)
			require.True(ok, "must have L2 CL node")
			rollupClientProvider, err := dial.NewStaticL2RollupProvider(t.Ctx(), t.Logger(), l2CL.UserRPC())
			require.NoError(err)
			rollupClient, err := rollupClientProvider.RollupClient(t.Ctx())
			require.NoError(err)
			require.NoError(wait.ForSafeBlock(t.Ctx(), rollupClient, 1))
			header, err := client.HeaderByNumber(t.Ctx(), big.NewInt(int64(rpc.SafeBlockNumber)))
			require.NoError(err)
			superRoot := getSuperRoot(t, o, header.Time, supervisorID)

			l1pao, err := o.keys.Address(devkeys.ChainOperatorKeys(l1ChainID.ToBig())(devkeys.L1ProxyAdminOwnerRole))
			require.NoError(err, "must have L1 proxy admin owner private key")

			superchainConfigAddr := o.wb.outSuperchainDeployment.SuperchainConfigAddr()
			superchainProxyAdmin := getProxyAdmin(t, w3Client, superchainConfigAddr)
			require.NotEmpty(superchainProxyAdmin, "superchain proxy admin address is empty")

			absolutePrestate := getInteropAbsolutePrestate(t)
			var opChainConfigs []bindings.OPContractsManagerOpChainConfig
			var l2ChainIDs []eth.ChainID
			for l2ChainID, l2Deployment := range o.wb.outL2Deployment {
				l2ChainIDs = append(l2ChainIDs, l2ChainID)
				opChainConfigs = append(opChainConfigs, bindings.OPContractsManagerOpChainConfig{
					SystemConfigProxy: l2Deployment.SystemConfigProxyAddr(),
					ProxyAdmin:        superchainProxyAdmin,
					AbsolutePrestate:  absolutePrestate,
				})
			}

			// Use primaryL2 to determine which challenger / proposer roles to promote to the shared permissioned fdg
			permissionedChainOps := devkeys.ChainOperatorKeys(primaryL2.ToBig())
			proposer, err := o.keys.Address(permissionedChainOps(devkeys.ProposerRole))
			o.P().Require().NoError(err, "must have configured proposer")
			challenger, err := o.keys.Address(permissionedChainOps(devkeys.ChallengerRole))
			o.P().Require().NoError(err, "must have configured challenger")

			opcmABI, err := bindings.OPContractsManagerMetaData.GetAbi()
			o.P().Require().NoError(err, "invalid OPCM ABI")
			opcmAddr := o.wb.output.ImplementationsDeployment.OpcmImpl
			contract := batching.NewBoundContract(opcmABI, opcmAddr)
			migrateInput := bindings.OPContractsManagerInteropMigratorMigrateInput{
				UsePermissionlessGame: true,
				StartingAnchorRoot: bindings.Proposal{
					Root:             common.Hash(superRoot),
					L2SequenceNumber: big.NewInt(int64(header.Time)),
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
			migrateCall := contract.Call("migrate", migrateInput)
			migrateCallData, err := migrateCall.Pack()
			require.NoError(err)

			chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
			l1PAOKey, err := o.keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
			require.NoError(err, "must have configured L1 proxy admin owner private key")
			transactOpts, err := bind.NewKeyedTransactorWithChainID(l1PAOKey, l1ChainID.ToBig())
			require.NoError(err, "must have transact opts")
			transactOpts.Context = t.Ctx()

			t.Log("Deploying delegate call proxy contract")
			// The DelegateCallProxy is used to simulate a GnosisSafe proxy that satisfies the delegatecall requirement of the OPCM.
			delegateCallProxy, proxyContract := deployDelegateCallProxy(t, transactOpts, client, l1pao)
			oldSuperchainProxyAdminOwner := getOwner(t, w3Client, superchainProxyAdmin)
			transferOwnership(t, l1PAOKey, client, superchainProxyAdmin, delegateCallProxy)

			oldDisputeGameFactories := make(map[eth.ChainID]common.Address)
			for i, opChainConfig := range opChainConfigs {
				var portal common.Address
				require.NoError(
					w3Client.Call(
						w3eth.CallFunc(opChainConfig.SystemConfigProxy, optimismPortalFn).Returns(&portal),
					))
				portalProxyAdmin := getProxyAdmin(t, w3Client, portal)
				transferOwnership(t, l1PAOKey, client, portalProxyAdmin, delegateCallProxy)

				dgf := getDisputeGameFactory(t, w3Client, portal)
				transferOwnership(t, l1PAOKey, client, dgf, delegateCallProxy)
				oldDisputeGameFactories[l2ChainIDs[i]] = dgf
			}

			t.Log("Executing delegate call")
			migrateTx, err := proxyContract.ExecuteDelegateCall(transactOpts, opcmAddr, migrateCallData)
			require.NoErrorf(err, "migrate delegatecall failed: %v", errutil.TryAddRevertReason(err))
			_, err = wait.ForReceiptOK(t.Ctx(), client, migrateTx.Hash())
			require.NoError(err)

			var sharedDGF common.Address
			{
				for _, l2Deployment := range o.wb.outL2Deployment {
					portal := getOptimismPortal(t, w3Client, l2Deployment.SystemConfigProxyAddr())
					addr := getDisputeGameFactory(t, w3Client, portal)
					if sharedDGF == (common.Address{}) {
						sharedDGF = addr
					} else {
						require.Equal(sharedDGF, addr, "dispute game factory address is not the same for all deployments")
					}
				}
				require.NotEmpty(getSuperGameImpl(t, w3Client, sharedDGF))
				o.wb.outInteropMigration = &InteropMigration{
					DisputeGameFactory: sharedDGF,
				}
			}

			// reset ownership transfers
			resetOwnershipAfterMigration(t,
				o,
				l1ChainID.ToBig(),
				l1PAOKey,
				w3Client,
				client,
				delegateCallProxy,
				opChainConfigs,
			)

			resetOldDisputeGameFactories(t,
				o,
				l1ChainID.ToBig(),
				l1PAOKey,
				client,
				delegateCallProxy,
				oldDisputeGameFactories,
			)
			superchainProxyAdminOwner := getOwner(t, w3Client, superchainProxyAdmin)
			t.Require().Equal(oldSuperchainProxyAdminOwner, superchainProxyAdminOwner, "superchain proxy admin owner is not the L1PAO")

			for _, l2Deployment := range o.wb.outL2Deployment {
				l2Deployment.disputeGameFactoryProxy = sharedDGF
			}
			t.Log("Interop migration complete")
		},
	}
}

func deployDelegateCallProxy(t devtest.CommonT, transactOpts *bind.TransactOpts, client *ethclient.Client, owner common.Address) (common.Address, *delegatecallproxy.Delegatecallproxy) {
	deployAddress, _, proxyContract, err := delegatecallproxy.DeployDelegatecallproxy(transactOpts, client, owner)
	t.Require().NoError(err, "DelegateCallProxy deployment failed")
	return deployAddress, proxyContract
}

func getSuperRoot(t devtest.CommonT, o *Orchestrator, timestamp uint64, supervisorID stack.SupervisorID) eth.Bytes32 {
	supervisor, ok := o.supervisors.Get(supervisorID)
	t.Require().True(ok, "must have supervisor")

	client, err := dial.DialSupervisorClientWithTimeout(t.Ctx(), t.Logger(), supervisor.userRPC)
	t.Require().NoError(err)
	super, err := client.SuperRootAtTimestamp(t.Ctx(), hexutil.Uint64(timestamp))
	t.Require().NoError(err, "super root at timestamp failed")
	return super.SuperRoot
}

func getInteropAbsolutePrestate(t devtest.CommonT) common.Hash {
	root, err := findMonorepoRoot("op-program/bin/prestate-proof-interop.json")
	t.Require().NoError(err)
	p := path.Join(root, "op-program/bin/prestate-proof-interop.json")
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
	ownerFn               = w3.MustNewFunc("owner()", "address")
	proxyAdminFn          = w3.MustNewFunc("proxyAdmin()", "address")
	adminFn               = w3.MustNewFunc("admin()", "address")
	proxyAdminOwnerFn     = w3.MustNewFunc("proxyAdminOwner()", "address")
	ethLockboxFn          = w3.MustNewFunc("ethLockbox()", "address")
	anchorStateRegistryFn = w3.MustNewFunc("anchorStateRegistry()", "address")
	wethFn                = w3.MustNewFunc("weth()", "address")
	transferOwnershipFn   = w3.MustNewFunc("transferOwnership(address)", "")
)

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
	o *Orchestrator,
	l1ChainID *big.Int,
	ownerPrivateKey *ecdsa.PrivateKey,
	w3Client *w3.Client,
	client *ethclient.Client,
	delegateCallProxy common.Address,
	opChainConfigs []bindings.OPContractsManagerOpChainConfig,
) {
	l1PAO, err := o.keys.Address(devkeys.ChainOperatorKeys(l1ChainID)(devkeys.L1ProxyAdminOwnerRole))
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

	// The Proxy Admin owner is changed. Assert that the admin of other proxies are consistent
	var sharedAnchorStateRegistryProxy common.Address
	err = w3Client.Call(w3eth.CallFunc(portal0, anchorStateRegistryFn).Returns(&sharedAnchorStateRegistryProxy))
	t.Require().NoError(err)
	asrAAdminOwner := getProxyAdminOwner(t, w3Client, sharedAnchorStateRegistryProxy)
	t.Require().Equal(l1PAO, asrAAdminOwner, "sharedAnchorStateRegistryProxy proxy admin owner is not the L1PAO")

	gameTypes := []uint32{superPermissionedGameType, superCannonGameType}
	for _, gameType := range gameTypes {
		var game common.Address
		err = w3Client.Call(w3eth.CallFunc(sharedDGF, gameImplsFn, gameType).Returns(&game))
		t.Require().NoError(err)
		var wethProxy common.Address
		err = w3Client.Call(w3eth.CallFunc(game, wethFn).Returns(&wethProxy))
		t.Require().NoError(err, "failed to get weth proxy")
		wethAdminOwner := getProxyAdminOwner(t, w3Client, wethProxy)
		t.Require().Equal(l1PAO, wethAdminOwner, "wethProxy proxy admin owner is not the L1PAO")
	}
}

func resetOldDisputeGameFactories(
	t devtest.CommonT,
	o *Orchestrator,
	l1ChainID *big.Int,
	ownerPrivateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	delegateCallProxy common.Address,
	oldDisputeGameFactories map[eth.ChainID]common.Address,
) {
	for l2ChainID, oldDGF := range oldDisputeGameFactories {
		chainOpsForL2 := devkeys.ChainOperatorKeys(l2ChainID.ToBig())
		l1PAOForL2, err := o.keys.Address(chainOpsForL2(devkeys.L1ProxyAdminOwnerRole))
		t.Require().NoError(err, "must have configured L1 proxy admin owner private key")
		// Not required since the old DGFs are not used; but done to prevent surprises later
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
