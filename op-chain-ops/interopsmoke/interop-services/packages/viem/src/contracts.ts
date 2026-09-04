import type { Chain } from 'viem'
import { chainConfig } from 'viem/op-stack'

/**
 * OP Stack Predeploy Addresses
 * @category Predeploy Addresses
 */
export const contracts = {
  ...chainConfig.contracts,
  crossL2Inbox: {
    address: '0x4200000000000000000000000000000000000022',
  },
  l2ToL2CrossDomainMessenger: {
    address: '0x4200000000000000000000000000000000000023',
  },
  optimismMintableERC20Factory: {
    address: '0x4200000000000000000000000000000000000012',
  },
  superchainTokenBridge: {
    address: '0x4200000000000000000000000000000000000028',
  },
  superchainETHBridge: {
    address: '0x4200000000000000000000000000000000000024',
  },
  weth: {
    address: '0x4200000000000000000000000000000000000006',
  },
} as const satisfies Chain['contracts']
