/* External Imports */
import { ethers } from 'hardhat'
import { constants, Wallet } from 'ethers'

/* Internal Imports */
import { remove0x, fromHexString } from '@eth-optimism/core-utils'

export interface EIP155Transaction {
  nonce: number
  gasLimit: number
  gasPrice: number
  to: string
  data: string
  chainId: number
}

export interface SignatureParameters {
  messageHash: string
  v: string
  r: string
  s: string
}

export const DEFAULT_EIP155_TX: EIP155Transaction = {
  to: `0x${'12'.repeat(20)}`,
  nonce: 100,
  gasLimit: 1000000,
  gasPrice: 100000000,
  data: `0x${'99'.repeat(10)}`,
  chainId: 420,
}

export const getRawSignedComponents = (signed: string): any[] => {
  return [signed.slice(130, 132), signed.slice(2, 66), signed.slice(66, 130)]
}

export const getSignedComponents = (signed: string): any[] => {
  return ethers.utils.RLP.decode(signed).slice(-3)
}
