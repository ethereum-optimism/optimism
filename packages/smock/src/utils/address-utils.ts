import { ethers } from 'ethers'

export const makeRandomAddress = (): string => {
  return ethers.utils.getAddress(
    '0x' +
      [...Array(40)]
        .map(() => {
          return Math.floor(Math.random() * 16).toString(16)
        })
        .join('')
  )
}
