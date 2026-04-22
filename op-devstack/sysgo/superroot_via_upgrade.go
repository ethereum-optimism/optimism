package sysgo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

// upgradeToSuperRoots calls OPCMv2.upgrade on each chain in the migration state
// to enable all three super-root game types with the supplied starting anchor.
func upgradeToSuperRoots(
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
	require.NotEmpty(migration.l2Deployments, "must have L2 deployments for interop upgrade")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)
	w3Client := w3.NewClient(rpcClient)

	absoluteCannonPrestate := getInteropCannonAbsolutePrestate(t)
	absoluteCannonKonaPrestate := getCannonKonaAbsolutePrestate(t)

	permissionedChainOps := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(permissionedChainOps(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")
	challenger, err := keys.Address(permissionedChainOps(devkeys.ChallengerRole))
	require.NoError(err, "must have configured challenger")

	l1Ops := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	l1PAO, err := keys.Address(l1Ops(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "must have configured L1 proxy admin owner")
	l1PAOKey, err := keys.Secret(l1Ops(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "must have configured L1 proxy admin owner key")

	anchorRootData := encodeStartingAnchorRoot(t, superRoot, superrootTime)
	respectedGameTypeData := encodeStartingRespectedGameType(t, superCannonGameType)

	loc := LocalArtifacts(t)
	artifactsFS, err := artifacts.Download(t.Ctx(), loc, ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	for _, l2Deployment := range migration.l2Deployments {
		configs := buildSuperRootUpgradeGameConfigs(
			absoluteCannonPrestate, absoluteCannonKonaPrestate, proposer, challenger,
		)

		upgradeInput := embedded.UpgradeOPChainInput{
			Prank: l1PAO,
			Opcm:  migration.opcmImpl,
			UpgradeInputV2: &embedded.UpgradeInputV2{
				SystemConfig:       l2Deployment.SystemConfigProxyAddr(),
				DisputeGameConfigs: configs,
				ExtraInstructions: []embedded.ExtraInstruction{
					{Key: "overrides.cfg.startingAnchorRoot", Data: anchorRootData},
					{Key: "overrides.cfg.startingRespectedGameType", Data: respectedGameTypeData},
					{Key: "PermittedProxyDeployment", Data: []byte("DelayedWETH")},
				},
			},
		}

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
		require.NoError(err, "failed to run upgrade script for super-root upgrade")

		calldata, err := bcaster.Dump()
		require.NoError(err, "failed to dump calldata")
		require.Len(calldata, 1, "calldata must contain one entry")

		t.Log("Executing opcmV2.upgrade via SetCode delegatecall for super-root upgrade")
		delegateCallWithSetCode(t, l1PAOKey, client, migration.opcmImpl, calldata[0].Data)
	}

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
	require.NotEmpty(getSuperGameImpl(t, w3Client, sharedDGF), "super-root game impl must be installed after upgrade")

	for chainID, l2Deployment := range migration.l2Deployments {
		l2Deployment.disputeGameFactoryProxy = sharedDGF
		migration.l2Deployments[chainID] = l2Deployment
	}
	t.Log("Interop super-root upgrade complete")
	return sharedDGF
}

func buildSuperRootUpgradeGameConfigs(
	absoluteCannonPrestate common.Hash,
	absoluteCannonKonaPrestate common.Hash,
	proposer common.Address,
	challenger common.Address,
) []embedded.DisputeGameConfig {
	zero := func() *big.Int { return new(big.Int) }
	initBond := zero()

	return []embedded.DisputeGameConfig{
		{Enabled: false, InitBond: zero(), GameType: embedded.GameTypeCannon},
		{Enabled: false, InitBond: zero(), GameType: embedded.GameTypePermissionedCannon},
		{Enabled: false, InitBond: zero(), GameType: embedded.GameTypeCannonKona},
		{
			Enabled: true, InitBond: initBond, GameType: embedded.GameTypeSuperCannon,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{AbsolutePrestate: absoluteCannonPrestate},
		},
		{
			Enabled: true, InitBond: initBond, GameType: embedded.GameTypeSuperPermCannon,
			PermissionedDisputeGameConfig: &embedded.PermissionedDisputeGameConfig{
				AbsolutePrestate: absoluteCannonPrestate,
				Proposer:         proposer,
				Challenger:       challenger,
			},
		},
		{
			Enabled: true, InitBond: initBond, GameType: embedded.GameTypeSuperCannonKona,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{AbsolutePrestate: absoluteCannonKonaPrestate},
		},
		{Enabled: false, InitBond: zero(), GameType: embedded.GameTypeZKDisputeGame},
	}
}

func encodeStartingAnchorRoot(t devtest.T, superRoot eth.Bytes32, superrootTime uint64) []byte {
	require := t.Require()
	proposalTy, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "root", Type: "bytes32"},
		{Name: "l2SequenceNumber", Type: "uint256"},
	})
	require.NoError(err, "failed to build Proposal ABI type")
	data, err := (abi.Arguments{{Type: proposalTy}}).Pack(
		struct {
			Root             common.Hash
			L2SequenceNumber *big.Int
		}{
			Root:             common.Hash(superRoot),
			L2SequenceNumber: new(big.Int).SetUint64(superrootTime),
		},
	)
	require.NoError(err, "failed to encode startingAnchorRoot override")
	return data
}

func encodeStartingRespectedGameType(t devtest.T, gameType uint32) []byte {
	require := t.Require()
	uint32Ty, err := abi.NewType("uint32", "", nil)
	require.NoError(err, "failed to build uint32 ABI type")
	data, err := (abi.Arguments{{Type: uint32Ty}}).Pack(gameType)
	require.NoError(err, "failed to encode startingRespectedGameType override")
	return data
}
