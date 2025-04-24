package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployImplementationsInput struct {
	WithdrawalDelaySeconds          *big.Int
	MinProposalSizeBytes            *big.Int
	ChallengePeriodSeconds          *big.Int
	ProofMaturityDelaySeconds       *big.Int
	DisputeGameFinalityDelaySeconds *big.Int
	MipsVersion                     *big.Int
	// Release version to set OPCM implementations for, of the format `op-contracts/vX.Y.Z`.
	L1ContractsRelease    string
	SuperchainConfigProxy common.Address
	ProtocolVersionsProxy common.Address
	SuperchainProxyAdmin  common.Address
	UpgradeController     common.Address
}

type DeployImplementationsOutput struct {
	Opcm                             common.Address `json:"opcmAddress"`
	OpcmContractsContainer           common.Address `json:"opcmContractsContainerAddress"`
	OpcmGameTypeAdder                common.Address `json:"opcmGameTypeAdderAddress"`
	OpcmDeployer                     common.Address `json:"opcmDeployerAddress"`
	OpcmUpgrader                     common.Address `json:"opcmUpgraderAddress"`
	OpcmInteropMigrator              common.Address `json:"opcmInteropMigratorAddress"`
	DelayedWETHImpl                  common.Address `json:"delayedWETHImplAddress"`
	OptimismPortalImpl               common.Address `json:"optimismPortalImplAddress"`
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
}

type DeployImplementationsScript script.DeployScriptWithOutput[DeployImplementationsInput, DeployImplementationsOutput]

// NewDeployImplementationsScript loads and validates the DeployImplementations script contract
func NewDeployImplementationsScript(host *script.Host) (DeployImplementationsScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployImplementationsInput, DeployImplementationsOutput](host, "DeployImplementations.s.sol", "DeployImplementations")
}
