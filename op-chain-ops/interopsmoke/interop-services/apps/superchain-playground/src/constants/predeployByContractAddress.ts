import { contracts } from '@eth-optimism/viem'
import type { Abi, Address } from 'viem'

import { superchainTokenBridgeAbi } from '@/constants/superchainTokenBridgeAbi'

export const predeployByContractAddress: Record<
  Address,
  { abi: Abi; name: string }
> = {
  [contracts.superchainTokenBridge.address]: {
    abi: superchainTokenBridgeAbi,
    name: 'SuperchainTokenBridge',
  },
}
