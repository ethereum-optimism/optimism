import { ethers } from 'ethers'

export const L1_TO_L2_ALIAS_OFFSET =
  '0x1111000000000000000000000000000000001111'

export const bnToAddress = (bn: ethers.BigNumber | number): string => {
  bn = ethers.BigNumber.from(bn)
  if (bn.isNegative()) {
    bn = ethers.BigNumber.from('0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF')
      .add(bn)
      .add(1)
  }

  const addr = bn.toHexString().slice(2).padStart(40, '0')
  return ethers.utils.getAddress(
    '0x' + addr.slice(addr.length - 40, addr.length)
  )
}

export const applyL1ToL2Alias = (address: string): string => {
  if (!ethers.utils.isAddress(address)) {
    throw new Error(`not a valid address: ${address}`)
  }

  return bnToAddress(ethers.BigNumber.from(address).add(L1_TO_L2_ALIAS_OFFSET))
}

export const undoL1ToL2Alias = (address: string): string => {
  if (!ethers.utils.isAddress(address)) {
    throw new Error(`not a valid address: ${address}`)
  }

  return bnToAddress(ethers.BigNumber.from(address).sub(L1_TO_L2_ALIAS_OFFSET))
}
