package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type DeployImplementationsInput struct {
	WithdrawalDelaySeconds          *big.Int
	MinProposalSizeBytes            *big.Int
	ChallengePeriodSeconds          *big.Int
	ProofMaturityDelaySeconds       *big.Int
	DisputeGameFinalityDelaySeconds *big.Int
	MipsVersion                     *big.Int
	DevFeatureBitmap                common.Hash
	FaultGameV2MaxGameDepth         *big.Int
	FaultGameV2SplitDepth           *big.Int
	FaultGameV2ClockExtension       *big.Int
	FaultGameV2MaxClockDuration     *big.Int
	SuperchainConfigProxy           common.Address
	ProtocolVersionsProxy           common.Address
	SuperchainProxyAdmin            common.Address
	L1ProxyAdminOwner               common.Address
	Challenger                      common.Address
}

type DeployImplementationsOutput struct {
	Opcm                             common.Address `json:"opcmAddress"`
	OpcmContractsContainer           common.Address `json:"opcmContractsContainerAddress"`
	OpcmGameTypeAdder                common.Address `json:"opcmGameTypeAdderAddress"`
	OpcmDeployer                     common.Address `json:"opcmDeployerAddress"`
	OpcmUpgrader                     common.Address `json:"opcmUpgraderAddress"`
	OpcmInteropMigrator              common.Address `json:"opcmInteropMigratorAddress"`
	OpcmStandardValidator            common.Address `json:"opcmStandardValidatorAddress"`
	DelayedWETHImpl                  common.Address `json:"delayedWETHImplAddress"`
	OptimismPortalImpl               common.Address `json:"optimismPortalImplAddress"`
	OptimismPortalInteropImpl        common.Address `json:"optimismPortalInteropImplAddress"`
	ETHLockboxImpl                   common.Address `json:"ethLockboxImplAddress" abi:"ethLockboxImpl"`
	PreimageOracleSingleton          common.Address `json:"preimageOracleSingletonAddress"`
	MipsSingleton                    common.Address `json:"mipsSingletonAddress"`
	SystemConfigImpl                 common.Address `json:"systemConfigImplAddress"`
	L1CrossDomainMessengerImpl       common.Address `json:"l1CrossDomainMessengerImplAddress"`
	L1ERC721BridgeImpl               common.Address `json:"l1ERC721BridgeImplAddress"`
	L1StandardBridgeImpl             common.Address `json:"l1StandardBridgeImplAddress"`
	OptimismMintableERC20FactoryImpl common.Address `json:"optimismMintableERC20FactoryImplAddress"`
	DisputeGameFactoryImpl           common.Address `json:"disputeGameFactoryImplAddress"`
	AnchorStateRegistryImpl          common.Address `json:"anchorStateRegistryImplAddress"`
	SuperchainConfigImpl             common.Address `json:"superchainConfigImplAddress"`
	ProtocolVersionsImpl             common.Address `json:"protocolVersionsImplAddress"`
	FaultDisputeGameV2Impl           common.Address `json:"faultDisputeGameV2ImplAddress"`
	PermissionedDisputeGameV2Impl    common.Address `json:"permissionedDisputeGameV2ImplAddress"`
}

type DeployImplementationsScript script.DeployScriptWithOutput[DeployImplementationsInput, DeployImplementationsOutput]

// NewDeployImplementationsScript loads and validates the DeployImplementations script contract
func NewDeployImplementationsScript(host *script.Host) (DeployImplementationsScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployImplementationsInput, DeployImplementationsOutput](host, "DeployImplementations.s.sol", "DeployImplementations")
}

func NewDeployImplementationsForgeCaller(client *forge.Client) forge.ScriptCaller[DeployImplementationsInput, DeployImplementationsOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployImplementations.s.sol:DeployImplementations",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployImplementationsInput]{TypeName: "DeployImplementationsInput"},
		&forge.BytesScriptDecoder[DeployImplementationsOutput]{TypeName: "DeployImplementationsOutput"},
	)
}
