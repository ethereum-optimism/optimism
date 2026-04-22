package sysgo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
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
) {
	require := t.Require()
	require.NotNil(migration, "interop migration state is required")
	require.NotEmpty(migration.opcmImpl, "must have an OPCM implementation")
	require.NotEmpty(migration.l2Deployments, "must have L2 deployments for interop upgrade")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)

	absoluteCannonPrestate := getInteropCannonAbsolutePrestate(t)
	absoluteCannonKonaPrestate := getCannonKonaAbsolutePrestate(t)

	l2Ops := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(l2Ops(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")
	challenger, err := keys.Address(l2Ops(devkeys.ChallengerRole))
	require.NoError(err, "must have configured challenger")

	l1PAO, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l1ChainID)

	anchorRootData := encodeStartingAnchorRoot(t, superRoot, superrootTime)
	respectedGameTypeData := encodeStartingRespectedGameType(t, superCannonGameType)

	artifactsFS, err := artifacts.Download(t.Ctx(), LocalArtifacts(t), ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	for _, l2Deployment := range migration.l2Deployments {
		executeOPCMUpgrade(t, rpcClient, client, l1PAOKey, artifactsFS, embedded.UpgradeOPChainInput{
			Prank: l1PAO,
			Opcm:  migration.opcmImpl,
			UpgradeInputV2: &embedded.UpgradeInputV2{
				SystemConfig: l2Deployment.SystemConfigProxyAddr(),
				DisputeGameConfigs: buildSuperRootUpgradeGameConfigs(
					absoluteCannonPrestate, absoluteCannonKonaPrestate, proposer, challenger,
				),
				ExtraInstructions: []embedded.ExtraInstruction{
					{Key: "overrides.cfg.startingAnchorRoot", Data: anchorRootData},
					{Key: "overrides.cfg.startingRespectedGameType", Data: respectedGameTypeData},
					{Key: "PermittedProxyDeployment", Data: []byte("DelayedWETH")},
				},
			},
		})
	}
}

func buildSuperRootUpgradeGameConfigs(
	absoluteCannonPrestate common.Hash,
	absoluteCannonKonaPrestate common.Hash,
	proposer common.Address,
	challenger common.Address,
) []embedded.DisputeGameConfig {
	return []embedded.DisputeGameConfig{
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypePermissionedCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannonKona},
		{
			Enabled: true, InitBond: new(big.Int), GameType: embedded.GameTypeSuperCannon,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{AbsolutePrestate: absoluteCannonPrestate},
		},
		{
			Enabled: true, InitBond: new(big.Int), GameType: embedded.GameTypeSuperPermCannon,
			PermissionedDisputeGameConfig: &embedded.PermissionedDisputeGameConfig{
				AbsolutePrestate: absoluteCannonPrestate,
				Proposer:         proposer,
				Challenger:       challenger,
			},
		},
		{
			Enabled: true, InitBond: new(big.Int), GameType: embedded.GameTypeSuperCannonKona,
			FaultDisputeGameConfig: &embedded.FaultDisputeGameConfig{AbsolutePrestate: absoluteCannonKonaPrestate},
		},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeZKDisputeGame},
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
