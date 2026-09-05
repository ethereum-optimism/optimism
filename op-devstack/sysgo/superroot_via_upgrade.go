package sysgo

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

var (
	// defaultInitBond matches Deploy.s.sol DEFAULT_INIT_BOND (0.08 ether).
	defaultInitBond = big.NewInt(8e16)
	// defaultZKChallengerBond is independent from the bond required to create a ZK game.
	defaultZKChallengerBond = big.NewInt(5e17)
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

	absoluteCannonKonaPrestate := getCannonKonaAbsolutePrestate(t)

	l2Ops := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(l2Ops(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")

	l1PAO, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l1ChainID)

	anchorRootData := encodeStartingAnchorRoot(t, superRoot, superrootTime)
	respectedGameTypeData := encodeStartingRespectedGameType(t, superCannonKonaGameType)

	artifactsFS, err := artifacts.Download(t.Ctx(), LocalArtifacts(t), ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	for _, l2Deployment := range migration.l2Deployments {
		executeOPCMUpgrade(t, rpcClient, client, l1PAOKey, artifactsFS, embedded.UpgradeOPChainInput{
			Prank: l1PAO,
			Opcm:  migration.opcmImpl,
			UpgradeInputV2: &embedded.UpgradeInputV2{
				SystemConfig: l2Deployment.SystemConfigProxyAddr(),
				DisputeGameConfigs: buildSuperRootUpgradeGameConfigs(
					absoluteCannonKonaPrestate, proposer,
				),
				ExtraInstructions: []embedded.ExtraInstruction{
					{Key: "overrides.cfg.startingAnchorRoot", Data: anchorRootData},
					{Key: "overrides.cfg.startingRespectedGameType", Data: respectedGameTypeData},
				},
			},
		})
	}
}

// setInteropZKDisputeGameViaUpgrade re-points the shared dispute games of a migrated interop set
// to the ZK dispute game. The chains in the set share an AnchorStateRegistry and
// DisputeGameFactory, so upgrading a single chain applies the new dispute game config to the set.
func setInteropZKDisputeGameViaUpgrade(
	t devtest.T,
	keys devkeys.Keys,
	migration *interopMigrationState,
	l1ChainID eth.ChainID,
	l1EL L1ELNode,
	sharedDGF common.Address,
	programVKey common.Hash,
	cfg ZKDisputeGameConfig,
	primaryL2 eth.ChainID,
) {
	require := t.Require()
	require.NoError(cfg.validate())
	require.NotNil(migration, "interop migration state is required")
	require.NotEmpty(migration.opcmImpl, "must have an OPCM implementation")
	require.NotEmpty(migration.l2Deployments, "must have L2 deployments for interop upgrade")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)
	w3Client := w3.NewClient(rpcClient)

	l2Ops := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(l2Ops(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")

	l1PAO, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l1ChainID)

	artifactsFS, err := artifacts.Download(t.Ctx(), LocalArtifacts(t), ioutil.NoopProgressor(), t.TempDir())
	require.NoError(err, "failed to download artifacts")

	executeOPCMUpgrade(t, rpcClient, client, l1PAOKey, artifactsFS, embedded.UpgradeOPChainInput{
		Prank: l1PAO,
		Opcm:  migration.opcmImpl,
		UpgradeInputV2: &embedded.UpgradeInputV2{
			SystemConfig:       sortedChainSystemConfigs(migration)[0],
			DisputeGameConfigs: buildZKUpgradeGameConfigs(programVKey, cfg, proposer),
			ExtraInstructions: []embedded.ExtraInstruction{
				{
					Key:  "overrides.cfg.startingRespectedGameType",
					Data: encodeStartingRespectedGameType(t, uint32(gameTypes.ZKDisputeGameType)),
				},
			},
		},
	})

	require.Equal(common.Address{}, getGameImpl(t, w3Client, sharedDGF, superCannonKonaGameType), "retired super game must be disabled")
	require.NotEqual(common.Address{}, getGameImpl(t, w3Client, sharedDGF, uint32(gameTypes.ZKDisputeGameType)), "ZK dispute game must be installed")
}

// buildZKUpgradeGameConfigs keeps SuperPermissioned as the permissioned liveness backup, retires
// the permissionless super fault game, and enables the ZK dispute game in its place.
func buildZKUpgradeGameConfigs(
	programVKey common.Hash,
	cfg ZKDisputeGameConfig,
	proposer common.Address,
) []embedded.DisputeGameConfig {
	return []embedded.DisputeGameConfig{
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypePermissionedCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannonKona},
		{
			Enabled: true, InitBond: new(big.Int), GameType: embedded.GameTypeSuperPermissioned,
			SuperPermissionedDisputeGameConfig: &embedded.SuperPermissionedDisputeGameConfig{
				Proposer: proposer,
			},
		},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeSuperCannonKona},
		{
			Enabled: true, InitBond: new(big.Int).Set(defaultInitBond), GameType: embedded.GameTypeZKDisputeGame,
			ZKDisputeGameConfig: &embedded.ZKDisputeGameConfig{
				AbsolutePrestate:     programVKey,
				MaxChallengeDuration: uint64(cfg.MaxChallengeDuration / time.Second),
				MaxProveDuration:     uint64(cfg.MaxProveDuration / time.Second),
				ChallengerBond:       new(big.Int).Set(defaultZKChallengerBond),
			},
		},
	}
}

func buildSuperRootUpgradeGameConfigs(
	absoluteCannonKonaPrestate common.Hash,
	proposer common.Address,
) []embedded.DisputeGameConfig {
	return []embedded.DisputeGameConfig{
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypePermissionedCannon},
		{Enabled: false, InitBond: new(big.Int), GameType: embedded.GameTypeCannonKona},
		{
			Enabled: true, InitBond: new(big.Int), GameType: embedded.GameTypeSuperPermissioned,
			SuperPermissionedDisputeGameConfig: &embedded.SuperPermissionedDisputeGameConfig{
				Proposer: proposer,
			},
		},
		{
			Enabled: true, InitBond: new(big.Int).Set(defaultInitBond), GameType: embedded.GameTypeSuperCannonKona,
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
