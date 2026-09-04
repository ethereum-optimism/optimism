import { interopAlphaNetwork } from '@/chains/interopAlpha.js'
import { interopRcAlphaNetwork } from '@/chains/interopRcAlpha.js'
import { interopV0Network } from '@/chains/interopV0.js'
import { interopJntV3Network } from '@/chains/interopJntV3.js'
import { mainnetNetwork } from '@/chains/mainnet.js'
import { sepoliaNetwork } from '@/chains/sepolia.js'
import { supersimNetwork } from '@/chains/supersim.js'
import type { Network, NetworkName } from '@/chains/types.js'

/**
 * Map of all unique networks configurations
 * @dev Multiple networks can share the same source chain.
 * @dev Chains can be apart of multiple networks.
 */
export const networks: Record<NetworkName, Network> = {
  mainnet: mainnetNetwork,
  sepolia: sepoliaNetwork,
  supersim: supersimNetwork,
  'interop-alpha': interopAlphaNetwork,
  'interop-rc-alpha': interopRcAlphaNetwork,
  'interop-v0': interopV0Network,
  'interop-jnt-v3': interopJntV3Network,
}
