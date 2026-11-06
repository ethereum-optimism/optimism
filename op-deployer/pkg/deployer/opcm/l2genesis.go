package opcm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
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
	L2ToL2MessageExpiryPeriod                *big.Int
	// SettlementChainID is written into the ChainConfig predeploy when the OP_PRIVATE dev feature
	// is on: 0 on a public chain, the chain's own ID on a settlement chain, and its settlement
	// chain's ID on a private chain.
	SettlementChainID *big.Int
}

func NewL2GenesisForgeCaller(client *forge.Client) forge.ScriptCaller[L2GenesisInput, forge.None] {
	return forge.NewScriptCaller(
		client,
		"scripts/L2Genesis.s.sol:L2Genesis",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[L2GenesisInput]{TypeName: "L2GenesisInput"},
		forge.NoneDecoder{},
	)
}

// L2Genesis generates the L2 genesis allocs using Forge. client runs in the L2 contracts.
func L2Genesis(
	ctx context.Context,
	client *forge.Client,
	deployer common.Address,
	noCodeSizeLimit bool,
	input L2GenesisInput,
) (*foundry.ForgeAllocs, error) {
	genesis := NewGenesisState(deployer, noCodeSizeLimit)
	env := &ForgeEnv{Client: client, Context: ctx, Genesis: genesis}
	if _, err := broadcastScript(env, NewL2GenesisForgeCaller, input); err != nil {
		return nil, fmt.Errorf("failed to generate the L2 genesis with Forge: %w", err)
	}
	// The script sends nothing from the deployer, so the deployer gets no L2 genesis account, even
	// when it is a dev account that the script funds.
	genesis.Wipe(deployer)
	return genesis.Allocs, nil
}
