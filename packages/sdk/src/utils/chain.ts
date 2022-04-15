import { BigNumber, ethers } from 'ethers'
import { EthereumProvider } from 'hardhat/types'

import { Chain, L1Chain } from '../interfaces'

export const OptimismKovan = {
  chainId: Chain.OPTIMISM_KOVAN,
  chainName: 'OptimismKovan',
  isTestChain: true,
  isLocalChain: false,
  multicallAddress: '0xE71bf4622578c7d1526A88CD3060f03030E99a04',
  getExplorerAddressLink: (address: string) =>
    `https://kovan-optimistic.etherscan.io/address/${address}`,
  getExplorerTransactionLink: (transactionHash: string) =>
    `https://kovan-optimistic.etherscan.io/tx/${transactionHash}`,
} as const

export const Optimism = {
  chainId: Chain.OPTIMISM,
  chainName: 'Optimism',
  isTestChain: false,
  isLocalChain: false,
  multicallAddress: '0x35A6Cdb2C9AD4a45112df4a04147EB07dFA01aB7',
  getExplorerAddressLink: (address: string) =>
    `https://optimistic.etherscan.io/address/${address}`,
  getExplorerTransactionLink: (transactionHash: string) =>
    `https://optimistic.etherscan.io/tx/${transactionHash}`,
} as const

export const OptimismLocal = {
  chainId: Chain.OPTIMISM_LOCAL,
  chainName: 'Optimism Local',
  isTestChain: true,
  isLocalChain: true,
  // this is just copy paste
  multicallAddress: '',
}

export const L1ToL2Mapping = {
  [Chain.MAINNET]: Chain.OPTIMISM,
  [Chain.KOVAN]: Chain.OPTIMISM_KOVAN,
  [Chain.HARDHAT_LOCAL]: Chain.OPTIMISM_LOCAL,
} as const

export const L2ToL1Mapping = {
  [Chain.OPTIMISM]: Chain.MAINNET,
  [Chain.OPTIMISM_KOVAN]: Chain.KOVAN,
  [Chain.OPTIMISM_LOCAL]: Chain.HARDHAT_LOCAL,
} as const

export const addOptimismNetworkToProvider = async <
  TProvider extends Pick<EthereumProvider, 'request'>
>(
  provider: TProvider,
  l1Chain: L1Chain = Chain.MAINNET
) => {
  const l2Network = L1ToL2Mapping[l1Chain]
  const optimismNetworkConfig = [Optimism, OptimismKovan].find(
    ({ chainId }) => chainId === l2Network
  )
  if (!optimismNetworkConfig) {
    throw new Error(`Invalid l1 network id ${l1Chain}`)
  }
  return provider.request({
    method: 'wallet_addEthereumChain',
    params: [optimismNetworkConfig],
  })
}

export const isTestChain = (id: number) =>
  [Chain.KOVAN, Chain.OPTIMISM_KOVAN].includes(id)

export const isL1 = (id: number) =>
  [Chain.MAINNET, Chain.KOVAN, Chain.HARDHAT_LOCAL].includes(id)

export const isL2 = (id: number) =>
  [Chain.OPTIMISM, Chain.OPTIMISM_KOVAN, Chain.OPTIMISM_LOCAL].includes(id)

const MISSING_NETWORK_ERROR_CODE = 4902

export const toggleLayer = async <
  TProvider extends Pick<EthereumProvider, 'request'>
>(
  provider: TProvider
) => {
  const currentChain = (await provider.request({
    method: 'eth_chainId',
  })) as number
  if (!Number.isInteger(currentChain)) {
    throw new Error('Not connect to a current chain')
  }

  const otherLayer: number | undefined =
    L1ToL2Mapping[currentChain] ?? L2ToL1Mapping[currentChain]
  if (!otherLayer) {
    throw new Error(`Unknown current chain id ${currentChain}`)
  }
  try {
    const formattedChainId = ethers.utils.hexStripZeros(
      BigNumber.from(otherLayer).toHexString()
    )
    return provider.request({
      method: 'wallet_switchEthereumChain',
      params: [{ chainId: formattedChainId }],
    })
  } catch (e) {
    if (e?.code === MISSING_NETWORK_ERROR_CODE) {
      addOptimismNetworkToProvider(provider)
      return
    }
    throw e
  }
}
