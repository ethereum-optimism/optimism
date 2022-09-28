import { utils } from 'ethers'

// https://optimistic.etherscan.io/address/0x2501c477d0a35545a387aa4a3eee4292a9a8b3f0
export const l2MainnetMultisig = '0x2501c477D0A35545a387Aa4A3EEe4292A9a8B3F0'
// https://etherscan.io/address/0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A
export const l1MainnetMultisig = '0x9BA6e03D8B90dE867373Db8cF1A58d2F7F006b3A'
// https://goerli.etherscan.io/address/0xf80267194936da1E98dB10bcE06F3147D580a62e
export const goerliAdmin = '0xf80267194936da1E98dB10bcE06F3147D580a62e'
export const predeploy = '0x4200000000000000000000000000000000000014'
export const predeployDeployer = '0xdfc82d475833a50de90c642770f34a9db7deb725'

export const isTargetL2Network = (network: string): boolean => {
  switch (network) {
    case 'optimism':
    case 'optimism-goerli':
    case 'ops-l2':
      return true
    default:
      return false
  }
}

export const isTargetL1Network = (network: string): boolean => {
  switch (network) {
    case 'mainnet':
    case 'goerli':
    case 'ops-l1':
      return true
    default:
      return false
  }
}

export const getProxyAdmin = (network: string): string => {
  switch (network) {
    case 'optimism':
      return l2MainnetMultisig
    case 'mainnet':
      return l1MainnetMultisig
    case 'goerli':
    case 'optimism-goerli':
      return goerliAdmin
    case 'ops-l1':
    case 'ops-l2':
      return predeployDeployer
    default:
      throw new Error(`unknown network ${network}`)
  }
}

export const validateERC721Bridge = async (hre, address: string, expected) => {
  const L1ERC721Bridge = await hre.ethers.getContractAt('ERC721Bridge', address)

  const messenger = await L1ERC721Bridge.messenger()
  const otherBridge = await L1ERC721Bridge.otherBridge()

  if (utils.getAddress(messenger) !== utils.getAddress(expected.messenger)) {
    throw new Error(`messenger mismatch`)
  }

  if (
    utils.getAddress(otherBridge) !== utils.getAddress(expected.otherBridge)
  ) {
    throw new Error(`otherBridge mismatch`)
  }
}
