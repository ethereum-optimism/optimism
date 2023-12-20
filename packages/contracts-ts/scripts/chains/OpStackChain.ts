import { Address } from 'abitype'
import { Chain } from 'viem'

/**
 * Extension to the viem/chains Chain type to include all OP contracts
 */
export type OpStackChain = Chain
  // All l2 chains have a sourceId (l1)
  & Pick<Required<Chain>, 'sourceId'>
  // Add additional contracts to l2
  & {
    contracts: Chain['contracts'] & {
      weth9: {
        address: '0x4200000000000000000000000000000000000006',
        blockCreated: number
      },
      l2CrossDomainMessenger: {
        address: '0x4200000000000000000000000000000000000007',
      },
      l2StandardBridge: {
        address: '0x4200000000000000000000000000000000000010',
        blockCreated: number
      },
      sequencerFeeVault: {
        address: '0x4200000000000000000000000000000000000011',
        blockCreated: number
      },
      optimismMintableERC20Factory: {
        address: '0x4200000000000000000000000000000000000012',
        blockCreated: number
      },
      gasPriceOracle: {
        address: '0x420000000000000000000000000000000000000F',
        blockCreated: number
      },
      governanceToken: {
        address: '0x4200000000000000000000000000000000000042',
        blockCreated: number
      },
      l1Block: {
        address: '0x4200000000000000000000000000000000000015',
        blockCreated: number
      },
      l2ToL1MessagePasser: {
        address: '0x4200000000000000000000000000000000000016',
        blockCreated: number
      },
      l2ERC721Bridge: {
        address: '0x4200000000000000000000000000000000000014',
        blockCreated: number
      },
      optimismMintableERC721Factory: {
        address: '0x4200000000000000000000000000000000000017',
        blockCreated: number
      },
      proxyAdmin: {
        address: '0x4200000000000000000000000000000000000018',
        blockCreated: number
      },
      baseFeeVault: {
        address: '0x4200000000000000000000000000000000000019',
        blockCreated: number
      },
      l1FeeVault: {
        address: Address,
        blockCreated: number
      },
      portal: {
        address: Address,
        blockCreated: number
      },
      addressManager: {
        address: Address,
        blockCreated: number
      },
      l1ERC721Bridge: {
        address: Address,
        blockCreated: number
      },
      l2OutputOracle: {
        address: Address,
        blockCreated: number
      },
      l1StandardBridge: {
        address: Address,
        blockCreated: number
      },
      l1CrossDomainMessenger: {
        address: Address,
        blockCreated: number
      },
      l2ERC20Factory: {
        address: Address,
        blockCreated: number
      },
    }
  }

