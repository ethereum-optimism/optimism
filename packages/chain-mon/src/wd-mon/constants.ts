import { L2ChainID } from '@eth-optimism/sdk'

// TODO: Consider moving to `@eth-optimism/constants` and generating from superchain registry.
// @see https://github.com/ethereum-optimism/optimism/pull/9041

/**
 * Mapping of L2ChainIDs to the L1 block numbers where the wd-mon service should start looking for
 * withdrawals by default. L1 block numbers here are based on the block number in which the
 * OptimismPortal proxy contract was deployed to L1.
 */
export const DEFAULT_STARTING_BLOCK_NUMBERS: {
  [ChainID in L2ChainID]?: number
} = {
  [L2ChainID.OPTIMISM]: 17365802 as const,
  [L2ChainID.OPTIMISM_GOERLI]: 8299684 as const,
  [L2ChainID.OPTIMISM_SEPOLIA]: 4071248 as const,
  [L2ChainID.BASE_MAINNET]: 17482143 as const,
  [L2ChainID.BASE_GOERLI]: 8411116 as const,
  [L2ChainID.BASE_SEPOLIA]: 4370901 as const,
  [L2ChainID.ZORA_MAINNET]: 17473938 as const,
}
