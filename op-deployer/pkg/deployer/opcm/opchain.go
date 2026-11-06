package opcm

import (
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

// PermissionedCannonFallbackPrestatePlaceholder matches the historical bytes32(hex"dead") anchor.
// Kona prestates cannot select it.
var PermissionedCannonFallbackPrestatePlaceholder = common.HexToHash(
	"0xdead000000000000000000000000000000000000000000000000000000000000",
)

// PermissionedGameStartingAnchorRoot contains the permissioned placeholder at block 0 and no permissionless root.
var PermissionedGameStartingAnchorRoot = append(
	PermissionedCannonFallbackPrestatePlaceholder.Bytes(),
	make([]byte, common.HashLength)...,
)

// Proposal mirrors the Solidity Proposal tuple used for the starting anchor root.
type Proposal struct {
	Root             common.Hash
	L2SequenceNumber *big.Int
}

// DefaultStartingAnchorProposal returns the permissioned placeholder proposal.
// Each call returns an independent sequence number that callers may safely mutate.
func DefaultStartingAnchorProposal() Proposal {
	return Proposal{
		Root:             DefaultStartingAnchorRoot.Root,
		L2SequenceNumber: new(big.Int),
	}
}

type DeployOPChainInput struct {
	OpChainProxyAdminOwner common.Address
	SystemConfigOwner      common.Address
	Batcher                common.Address
	UnsafeBlockSigner      common.Address
	Proposer               common.Address
	Challenger             common.Address

	BasefeeScalar     uint32
	BlobBaseFeeScalar uint32
	L2ChainId         *big.Int
	Opcm              common.Address
	SaltMixer         string
	GasLimit          uint64

	DisputeGameType         uint32
	DisputeAbsolutePrestate common.Hash // Selected game prestate.
	StartingAnchorRoot      Proposal
	// CannonAbsolutePrestate configures the CANNON_KONA guardian fallback.
	// PERMISSIONED_CANNON mirrors the selected prestate. The super types (SUPER_CANNON_KONA,
	// SUPER_PERMISSIONED) leave it zero.
	CannonAbsolutePrestate       common.Hash
	DisputeMaxGameDepth          *big.Int
	DisputeSplitDepth            *big.Int
	DisputeClockExtension        uint64
	DisputeMaxClockDuration      uint64
	AllowCustomDisputeParameters bool

	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
	SuperchainConfig    common.Address

	UseCustomGasToken bool

	ProofMaturityDelaySeconds       *big.Int
	DisputeGameFinalityDelaySeconds *big.Int
	WithdrawalDelaySeconds          *big.Int

	// BridgeHook is the hook the portal screens deposits and withdrawals through, or the zero
	// address for no hook.
	BridgeHook common.Address
}

type DeployOPChainOutput struct {
	OpChainProxyAdmin                 common.Address
	AddressManager                    common.Address
	L1ERC721BridgeProxy               common.Address
	SystemConfigProxy                 common.Address
	OptimismMintableERC20FactoryProxy common.Address
	L1StandardBridgeProxy             common.Address
	L1CrossDomainMessengerProxy       common.Address
	// Fault proof contracts below.
	OptimismPortalProxy                common.Address
	EthLockboxProxy                    common.Address `evm:"ethLockboxProxy"`
	DisputeGameFactoryProxy            common.Address
	AnchorStateRegistryProxy           common.Address
	FaultDisputeGame                   common.Address
	PermissionedDisputeGame            common.Address
	DelayedWETHPermissionedGameProxy   common.Address
	DelayedWETHPermissionlessGameProxy common.Address
}

type DeployOPChainScript script.DeployScriptWithOutput[DeployOPChainInput, DeployOPChainOutput]

// NewDeployOPChainScript loads and validates the DeployOPChain script contract
func NewDeployOPChainScript(host *script.Host) (DeployOPChainScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployOPChainInput, DeployOPChainOutput](host, "DeployOPChain.s.sol", "DeployOPChain")
}

func NewDeployOPChainForgeCaller(client *forge.Client) forge.ScriptCaller[DeployOPChainInput, DeployOPChainOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployOPChain.s.sol:DeployOPChain",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployOPChainInput]{TypeName: "DeployOPChainInput"},
		&forge.BytesScriptDecoder[DeployOPChainOutput]{TypeName: "DeployOPChainOutput"},
	)
}

// DeployOPChainViaForge deploys OP Chain contracts using Forge
func DeployOPChainViaForge(env *ForgeEnv, input DeployOPChainInput) (DeployOPChainOutput, error) {
	var output DeployOPChainOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployOPChainForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy OP Chain with Forge: %w", err)
	}
	return output, nil
}
