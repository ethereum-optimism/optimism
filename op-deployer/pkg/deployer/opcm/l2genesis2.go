package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type L2Genesis2Input struct {
	L1ChainID                                *big.Int
	L2ChainID                                *big.Int
	L1CrossDomainMessengerProxy              common.Address
	L1StandardBridgeProxy                    common.Address
	L1ERC721BridgeProxy                      common.Address
	L2ProxyAdminOwner                        common.Address
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
	UseInterop                               bool
	EnableGovernance                         bool
	FundDevAccounts                          bool
}

type L2Genesis2Script script.DeployScriptWithoutOutput[L2Genesis2Input]

func NewL2Genesis2Script(host *script.Host) (L2Genesis2Script, error) {
	return script.NewDeployScriptWithoutOutputFromFile[L2Genesis2Input](host, "L2Genesis2.s.sol", "L2Genesis")
}
