import { Address } from 'abitype'
import { Chain } from 'viem'
import { type L2Predeploys } from './getL2Predeploys'

export type L1Contracts<SourceId extends number> = {
  l1FeeVault: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  portal: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  addressManager: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  l1ERC721Bridge: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  l2OutputOracle: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  l1StandardBridge: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  l1CrossDomainMessenger: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
  l2ERC20Factory: Record<
    SourceId,
    {
      address: Address
      blockCreated: number
    }
  >
}

/**
 * Extension to the viem/chains Chain type to include all OP contracts
 */
export type OpStackChain<SourceId extends number> = Chain & {
  // SourceId is an optional property on a normal viem chain representing the source chain // All l2 chains have a sourceId (l1)
  sourceId: SourceId
} & {
  // Add additional contracts to l2
  contracts: Chain['contracts'] & L1Contracts<SourceId> & L2Predeploys
}
