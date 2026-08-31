package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type L2GenesisInput struct {
	L1ChainID                                *big.Int
	L2ChainID                                *big.Int
	L1CrossDomainMessengerProxy              common.Address
	L1StandardBridgeProxy                    common.Address
	L1ERC721BridgeProxy                      common.Address
	OpChainProxyAdminOwner                   common.Address
	SequencerFeeVaultRecipient               common.Address
	SequencerFeeVaultMinimumWithdrawalAmount *big.Int
	SequencerFeeVaultWithdrawalNetwork       *big.Int
	BaseFeeVaultRecipient                    common.Address
	BaseFeeVaultMinimumWithdrawalAmount      *big.Int
	BaseFeeVaultWithdrawalNetwork            *big.Int
	L1FeeVaultRecipient                      common.Address
	L1FeeVaultMinimumWithdrawalAmount        *big.Int
	L1FeeVaultWithdrawalNetwork              *big.Int
	OperatorFeeVaultRecipient                common.Address
	OperatorFeeVaultMinimumWithdrawalAmount  *big.Int
	OperatorFeeVaultWithdrawalNetwork        *big.Int
	GovernanceTokenOwner                     common.Address
	Fork                                     *big.Int
	EnableGovernance                         bool
	FundDevAccounts                          bool
	UseCustomGasToken                        bool
	UseInterop                               bool
	GasPayingTokenName                       string
	GasPayingTokenSymbol                     string
	NativeAssetLiquidityAmount               *big.Int
	LiquidityControllerOwner                 common.Address
	DevFeatureBitmap                         common.Hash
	// Private interop. Inert on every ordinary chain: which half (if any) a genesis renders is
	// decided by the PRIVATE_INTEROP_* bits in DevFeatureBitmap, and with neither bit set the
	// script never reads these.
	PrivateInteropOperator            common.Address
	PrivateInteropOperatorBalance     *big.Int
	PrivateInteropCounterpartyChainID *big.Int
	PrivateInteropLockVault           common.Address
}

type L2GenesisScript script.DeployScriptWithoutOutput[L2GenesisInput]

func NewL2GenesisScript(host *script.Host) (L2GenesisScript, error) {
	return script.NewDeployScriptWithoutOutputFromFile[L2GenesisInput](host, "L2Genesis.s.sol", "L2Genesis")
}
