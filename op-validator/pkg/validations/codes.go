package validations

var descriptions = map[string]string{
	// SuperchainConfig validations
	"SPRCFG-10": "SuperchainConfig is paused",

	// ProxyAdmin validations
	"PROXYA-10": "ProxyAdmin owner is not set to L1 PAO multisig",

	// SystemConfig validations
	"SYSCON-10":  "SystemConfig version mismatch",
	"SYSCON-20":  "SystemConfig gas limit is not set to 60,000,000",
	"SYSCON-30":  "SystemConfig scalar is set to zero",
	"SYSCON-40":  "SystemConfig implementation address mismatch",
	"SYSCON-50":  "SystemConfig maxResourceLimit is not set to 20,000,000",
	"SYSCON-60":  "SystemConfig elasticityMultiplier is not set to 10",
	"SYSCON-70":  "SystemConfig baseFeeMaxChangeDenominator is not set to 8",
	"SYSCON-80":  "SystemConfig systemTxMaxGas is not set to 1,000,000",
	"SYSCON-90":  "SystemConfig minimumBaseFee is not set to 1 gwei",
	"SYSCON-100": "SystemConfig maximumBaseFee is not set to max uint128",
	"SYSCON-110": "SystemConfig operatorFeeScalar is not set to 0",
	"SYSCON-120": "SystemConfig operatorFeeConstant is not set to 0",
	"SYSCON-130": "SystemConfig proxyAdmin is invalid",
	"SYSCON-140": "SystemConfig superchainConfig is invalid",

	// L1 Cross Domain Messenger validations
	"L1xDM-10": "L1CrossDomainMessenger version mismatch",
	"L1xDM-20": "L1CrossDomainMessenger implementation address mismatch",
	"L1xDM-30": "L1CrossDomainMessenger OTHER_MESSENGER address mismatch",
	"L1xDM-40": "L1CrossDomainMessenger otherMessenger address mismatch",
	"L1xDM-50": "L1CrossDomainMessenger PORTAL address mismatch",
	"L1xDM-60": "L1CrossDomainMessenger portal address mismatch",
	"L1xDM-70": "L1CrossDomainMessenger systemConfig address mismatch",
	"L1xDM-80": "L1CrossDomainMessenger proxyAdmin is invalid",

	// L1 Standard Bridge validations
	"L1SB-10": "L1StandardBridge version mismatch",
	"L1SB-20": "L1StandardBridge implementation address mismatch",
	"L1SB-30": "L1StandardBridge MESSENGER address mismatch",
	"L1SB-40": "L1StandardBridge messenger address mismatch",
	"L1SB-50": "L1StandardBridge OTHER_BRIDGE address mismatch",
	"L1SB-60": "L1StandardBridge otherBridge address mismatch",
	"L1SB-70": "L1StandardBridge systemConfig address mismatch",
	"L1SB-80": "L1StandardBridge proxyAdmin is invalid",

	// Optimism Mintable ERC20 Factory validations
	"MERC20F-10": "OptimismMintableERC20Factory version mismatch",
	"MERC20F-20": "OptimismMintableERC20Factory implementation address mismatch",
	"MERC20F-30": "OptimismMintableERC20Factory BRIDGE address mismatch",
	"MERC20F-40": "OptimismMintableERC20Factory bridge address mismatch",

	// L1 ERC721 Bridge validations
	"L721B-10": "L1ERC721Bridge version mismatch",
	"L721B-20": "L1ERC721Bridge implementation address mismatch",
	"L721B-30": "L1ERC721Bridge OTHER_BRIDGE address mismatch",
	"L721B-40": "L1ERC721Bridge otherBridge address mismatch",
	"L721B-50": "L1ERC721Bridge MESSENGER address mismatch",
	"L721B-60": "L1ERC721Bridge messenger address mismatch",
	"L721B-70": "L1ERC721Bridge systemConfig address mismatch",
	"L721B-80": "L1ERC721Bridge proxyAdmin is invalid",

	// Optimism Portal validations
	"PORTAL-10": "OptimismPortal version mismatch",
	"PORTAL-20": "OptimismPortal implementation address mismatch",
	"PORTAL-30": "OptimismPortal disputeGameFactory address mismatch",
	"PORTAL-40": "OptimismPortal systemConfig address mismatch",
	"PORTAL-80": "OptimismPortal l2Sender not set to default value",
	"PORTAL-90": "OptimismPortal proxyAdmin is invalid",

	// Dispute Factory validations
	"DF-10": "DisputeGameFactory version mismatch",
	"DF-20": "DisputeGameFactory implementation address mismatch",
	"DF-30": "DisputeGameFactory owner is not set to L1 PAO multisig",
	"DF-40": "DisputeGameFactory proxyAdmin is invalid",

	// ETHLockbox validations
	"LOCKBOX-10": "ETHLockbox version mismatch",
	"LOCKBOX-20": "ETHLockbox implementation address mismatch",
	"LOCKBOX-30": "ETHLockbox proxyAdmin is invalid",
	"LOCKBOX-40": "ETHLockbox systemConfig address mismatch",
	"LOCKBOX-50": "ETHLockbox authorizedPortals mismatch",

	// Permissioned Dispute Game validations
	"PDDG-10":  "Permissioned dispute game implementation not found",
	"PDDG-20":  "Permissioned dispute game version mismatch",
	"PDDG-30":  "Permissioned dispute game type mismatch",
	"PDDG-40":  "Permissioned dispute game absolute prestate mismatch",
	"PDDG-50":  "Permissioned dispute game VM address mismatch",
	"PDDG-60":  "Permissioned dispute game L2 chain ID mismatch",
	"PDDG-70":  "Permissioned dispute game L2 block number not set to 0",
	"PDDG-80":  "Permissioned dispute game clock extension not set to 10800",
	"PDDG-90":  "Permissioned dispute game split depth not set to 30",
	"PDDG-100": "Permissioned dispute game max game depth not set to 73",
	"PDDG-110": "Permissioned dispute game max clock duration not set to 302400",
	"PDDG-120": "Permissioned dispute game challenger address mismatch",

	// Cannon Dispute Game validations
	"CDG-10":  "Permissionless cannon dispute game implementation not found",
	"CDG-20":  "Permissionless cannon dispute game version mismatch",
	"CDG-30":  "Permissionless cannon dispute game type mismatch",
	"CDG-40":  "Permissionless cannon dispute game absolute prestate mismatch",
	"CDG-50":  "Permissionless cannon dispute game VM address mismatch",
	"CDG-60":  "Permissionless cannon dispute game L2 chain ID mismatch",
	"CDG-70":  "Permissionless cannon dispute game L2 block number not set to 0",
	"CDG-80":  "Permissionless cannon dispute game clock extension not set to 10800",
	"CDG-90":  "Permissionless cannon dispute game split depth not set to 30",
	"CDG-100": "Permissionless cannon dispute game max game depth not set to 73",
	"CDG-110": "Permissionless cannon dispute game max clock duration not set to 302400",

	// Cannon Kona Dispute Game validations
	"CKDG-10":  "Permissionless cannon-kona dispute game implementation not found",
	"CKDG-20":  "Permissionless cannon-kona dispute game version mismatch",
	"CKDG-30":  "Permissionless cannon-kona dispute game type mismatch",
	"CKDG-40":  "Permissionless cannon-kona dispute game absolute prestate mismatch",
	"CKDG-50":  "Permissionless cannon-kona dispute game VM address mismatch",
	"CKDG-60":  "Permissionless cannon-kona dispute game L2 chain ID mismatch",
	"CKDG-70":  "Permissionless cannon-kona dispute game L2 block number not set to 0",
	"CKDG-80":  "Permissionless cannon-kona dispute game clock extension not set to 10800",
	"CKDG-90":  "Permissionless cannon-kona dispute game split depth not set to 30",
	"CKDG-100": "Permissionless cannon-kona dispute game max game depth not set to 73",
	"CKDG-110": "Permissionless cannon-kona dispute game max clock duration not set to 302400",

	// Delayed WETH validations (for CDG, CKDG and PLDG)
	"PDDG-DWETH-10": "Permissioned dispute game delayed WETH version mismatch",
	"PDDG-DWETH-20": "Permissioned dispute game delayed WETH implementation address mismatch",
	"PDDG-DWETH-30": "Permissioned dispute game delayed WETH owner mismatch",
	"PDDG-DWETH-40": "Permissioned dispute game delayed WETH delay not set to 1 week",
	"PDDG-DWETH-50": "Permissioned dispute game delayed WETH system config address mismatch",
	"PDDG-DWETH-60": "Permissioned dispute game delayed WETH proxy admin mismatch",
	"CDG-DWETH-10":  "Permissionless cannon dispute game delayed WETH version mismatch",
	"CDG-DWETH-20":  "Permissionless cannon dispute game delayed WETH implementation address mismatch",
	"CDG-DWETH-30":  "Permissionless cannon dispute game delayed WETH owner mismatch",
	"CDG-DWETH-40":  "Permissionless cannon dispute game delayed WETH delay not set to 1 week",
	"CDG-DWETH-50":  "Permissionless cannon dispute game delayed WETH system config address mismatch",
	"CDG-DWETH-60":  "Permissionless cannon dispute game delayed WETH proxy admin mismatch",
	"CKDG-DWETH-10": "Permissionless cannon-kona dispute game delayed WETH version mismatch",
	"CKDG-DWETH-20": "Permissionless cannon-kona dispute game delayed WETH implementation address mismatch",
	"CKDG-DWETH-30": "Permissionless cannon-kona dispute game delayed WETH owner mismatch",
	"CKDG-DWETH-40": "Permissionless cannon-kona dispute game delayed WETH delay not set to 1 week",
	"CKDG-DWETH-50": "Permissionless cannon-kona dispute game delayed WETH system config address mismatch",
	"CKDG-DWETH-60": "Permissionless cannon-kona dispute game delayed WETH proxy admin mismatch",

	// Anchor State Registry validations (for both PDDG and PLDG)
	"PDDG-ANCHORP-10": "Permissioned dispute game anchor state registry version mismatch",
	"PDDG-ANCHORP-20": "Permissioned dispute game anchor state registry implementation address mismatch",
	"PDDG-ANCHORP-30": "Permissioned dispute game anchor state registry dispute game factory address mismatch",
	"PDDG-ANCHORP-40": "Permissioned dispute game anchor state registry root hash mismatch",
	"PDDG-ANCHORP-50": "Permissioned dispute game anchor state registry superchain config address mismatch",
	"PDDG-ANCHORP-60": "Permissioned dispute game anchor state registry retirement timestamp is not set",
	"CDG-ANCHORP-10":  "Permissionless cannon dispute game anchor state registry version mismatch",
	"CDG-ANCHORP-20":  "Permissionless cannon dispute game anchor state registry implementation address mismatch",
	"CDG-ANCHORP-30":  "Permissionless cannon dispute game anchor state registry dispute game factory address mismatch",
	"CDG-ANCHORP-40":  "Permissionless cannon dispute game anchor state registry root hash mismatch",
	"CDG-ANCHORP-50":  "Permissionless cannon dispute game anchor state registry superchain config address mismatch",
	"CDG-ANCHORP-60":  "Permissionless cannon dispute game anchor state registry retirement timestamp is not set",
	"CKDG-ANCHORP-10": "Permissionless cannon-kona dispute game anchor state registry version mismatch",
	"CKDG-ANCHORP-20": "Permissionless cannon-kona dispute game anchor state registry implementation address mismatch",
	"CKDG-ANCHORP-30": "Permissionless cannon-kona dispute game anchor state registry dispute game factory address mismatch",
	"CKDG-ANCHORP-40": "Permissionless cannon-kona dispute game anchor state registry root hash mismatch",
	"CKDG-ANCHORP-50": "Permissionless cannon-kona dispute game anchor state registry superchain config address mismatch",
	"CKDG-ANCHORP-60": "Permissionless cannon-kona dispute game anchor state registry retirement timestamp is not set",

	// Preimage Oracle validations (for both PDDG and PLDG)
	"PDDG-PIMGO-10": "Permissioned dispute game preimage oracle version mismatch",
	"PDDG-PIMGO-20": "Permissioned dispute game preimage oracle challenge period not set to 86400",
	"PDDG-PIMGO-30": "Permissioned dispute game preimage oracle min proposal size not set to 126000",
	"CDG-PIMGO-10":  "Permissionless cannon dispute game preimage oracle version mismatch",
	"CDG-PIMGO-20":  "Permissionless cannon dispute game preimage oracle challenge period not set to 86400",
	"CDG-PIMGO-30":  "Permissionless cannon dispute game preimage oracle min proposal size not set to 126000",
	"CKDG-PIMGO-10": "Permissionless cannon-kona dispute game preimage oracle version mismatch",
	"CKDG-PIMGO-20": "Permissionless cannon-kona dispute game preimage oracle challenge period not set to 86400",
	"CKDG-PIMGO-30": "Permissionless cannon-kona dispute game preimage oracle min proposal size not set to 126000",
}

func ErrorDescription(code string) string {
	return descriptions[code]
}
