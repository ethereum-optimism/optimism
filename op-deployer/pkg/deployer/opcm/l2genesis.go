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
	GovernanceTokenOwner                     common.Address
	Fork                                     *big.Int
	DeployCrossL2Inbox                       bool
	EnableGovernance                         bool
	FundDevAccounts                          bool
}

type L2GenesisScript script.DeployScriptWithoutOutput[L2GenesisInput]

func NewL2GenesisScript(host *script.Host) (L2GenesisScript, error) {
	return script.NewDeployScriptWithoutOutputFromFile[L2GenesisInput](host, "L2Genesis.s.sol", "L2Genesis")
}
