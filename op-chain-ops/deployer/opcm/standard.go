package opcm

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/ethereum-optimism/superchain-registry/superchain"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed standard-versions-mainnet.toml
var StandardVersionsMainnetData string

//go:embed standard-versions-sepolia.toml
var StandardVersionsSepoliaData string

var StandardVersionsSepolia StandardVersions

var StandardVersionsMainnet StandardVersions

type StandardVersions struct {
	Releases map[string]StandardVersionsReleases `toml:"releases"`
}

type StandardVersionsReleases struct {
	OptimismPortal               StandardVersionRelease `toml:"optimism_portal"`
	SystemConfig                 StandardVersionRelease `toml:"system_config"`
	AnchorStateRegistry          StandardVersionRelease `toml:"anchor_state_registry"`
	DelayedWETH                  StandardVersionRelease `toml:"delayed_weth"`
	DisputeGameFactory           StandardVersionRelease `toml:"dispute_game_factory"`
	FaultDisputeGame             StandardVersionRelease `toml:"fault_dispute_game"`
	PermissionedDisputeGame      StandardVersionRelease `toml:"permissioned_dispute_game"`
	MIPS                         StandardVersionRelease `toml:"mips"`
	PreimageOracle               StandardVersionRelease `toml:"preimage_oracle"`
	L1CrossDomainMessenger       StandardVersionRelease `toml:"l1_cross_domain_messenger"`
	L1ERC721Bridge               StandardVersionRelease `toml:"l1_erc721_bridge"`
	L1StandardBridge             StandardVersionRelease `toml:"l1_standard_bridge"`
	OptimismMintableERC20Factory StandardVersionRelease `toml:"optimism_mintable_erc20_factory"`
}

type StandardVersionRelease struct {
	Version               string         `toml:"version"`
	ImplementationAddress common.Address `toml:"implementation_address"`
	Address               common.Address `toml:"address"`
}

var _ embed.FS

func StandardVersionsFor(chainID uint64) (string, error) {
	switch chainID {
	case 1:
		return StandardVersionsMainnetData, nil
	case 11155111:
		return StandardVersionsSepoliaData, nil
	default:
		return "", fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func SuperchainFor(chainID uint64) (*superchain.Superchain, error) {
	switch chainID {
	case 1:
		return superchain.Superchains["mainnet"], nil
	case 11155111:
		return superchain.Superchains["sepolia"], nil
	default:
		return nil, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ManagerImplementationAddrFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 11155111:
		// Generated using the bootstrap command on 10/02/2024.
		return common.HexToAddress("0x0f29118caed0f72873701bcc079398c594b6f8e4"), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func ManagerOwnerAddrFor(chainID uint64) (common.Address, error) {
	switch chainID {
	case 1:
		// Set to superchain proxy admin
		return common.HexToAddress("0x543bA4AADBAb8f9025686Bd03993043599c6fB04"), nil
	case 11155111:
		// Set to development multisig
		return common.HexToAddress("0xDEe57160aAfCF04c34C887B5962D0a69676d3C8B"), nil
	default:
		return common.Address{}, fmt.Errorf("unsupported chain ID: %d", chainID)
	}
}

func init() {
	StandardVersionsMainnet = StandardVersions{}
	if err := toml.Unmarshal([]byte(StandardVersionsMainnetData), &StandardVersionsMainnet); err != nil {
		panic(err)
	}

	StandardVersionsSepolia = StandardVersions{}
	if err := toml.Unmarshal([]byte(StandardVersionsSepoliaData), &StandardVersionsSepolia); err != nil {
		panic(err)
	}
}
