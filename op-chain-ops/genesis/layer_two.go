package genesis

import (
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/core"
)

// L2Addresses represents L1 contract addresses
// that are required for the construction of an L2 state
type L2Addresses struct {
	// ProxyAdminOwner represents the admin of the L2 ProxyAdmin predeploy
	ProxyAdminOwner common.Address
	// L1StandardBridgeProxy represents the L1 contract address of the L1StandardBridgeProxy
	L1StandardBridgeProxy common.Address
	// L1CrossDomainMessengerProxy represents the L1 contract address of the L1CrossDomainMessengerProxy
	L1CrossDomainMessengerProxy common.Address
	// L1ERC721BridgeProxy represents the L1 contract address of the L1ERC721BridgeProxy
	L1ERC721BridgeProxy common.Address
	// SequencerFeeVaultRecipient represents the L1 address that the SequencerFeeVault can withdraw to
	SequencerFeeVaultRecipient common.Address
	// L1FeeVaultRecipient represents the L1 address that the L1FeeVault can withdraw to
	L1FeeVaultRecipient common.Address
	// BaseFeeVaultRecipient represents the L1 address that the BaseFeeVault can withdraw to
	BaseFeeVaultRecipient common.Address
}

// BuildL2DeveloperGenesis will build the developer Optimism Genesis
// Block. Suitable for devnets.
func BuildL2DeveloperGenesis(config *DeployConfig, l1StartBlock *types.Block, l2Addrs *L2Addresses) (*core.Genesis, error) {
	genspec, err := NewL2Genesis(config, l1StartBlock)
	if err != nil {
		return nil, err
	}

	db := state.NewMemoryStateDB(genspec)

	if config.FundDevAccounts {
		FundDevAccounts(db)
	}
	SetPrecompileBalances(db)

	// Use the known developer addresses if they are not set
	if l2Addrs == nil {
		l2Addrs = &L2Addresses{
			// Hardcoded address corresponds to m/44'/60'/0'/0/1 in the 'test test... junk' mnemonic
			ProxyAdminOwner:             common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
			L1StandardBridgeProxy:       predeploys.DevL1StandardBridgeAddr,
			L1CrossDomainMessengerProxy: predeploys.DevL1CrossDomainMessengerAddr,
			L1ERC721BridgeProxy:         predeploys.DevL1ERC721BridgeAddr,
			SequencerFeeVaultRecipient:  common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
			L1FeeVaultRecipient:         common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
			BaseFeeVaultRecipient:       common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
		}
	}

	return BuildL2Genesis(db, config, l1StartBlock, l2Addrs)
}

// BuildL2Genesis will build the L2 Optimism Genesis Block
func BuildL2Genesis(db *state.MemoryStateDB, config *DeployConfig, l1Block *types.Block, l2Addrs *L2Addresses) (*core.Genesis, error) {
	if err := SetL2Proxies(db); err != nil {
		return nil, err
	}

	storage, err := NewL2StorageConfig(config, l1Block, l2Addrs)
	if err != nil {
		return nil, err
	}

	immutable, err := NewL2ImmutableConfig(config, l1Block, l2Addrs)
	if err != nil {
		return nil, err
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return nil, err
	}

	return db.Genesis(), nil
}
