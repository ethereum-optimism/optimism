import { ethers } from 'hardhat'
import ethSigUtil from 'eth-sig-util'

export const MAX_UINT256 = ethers.constants.MaxUint256.toString()

export const EIP712Domain = [
  { name: 'name', type: 'string' },
  { name: 'version', type: 'string' },
  { name: 'chainId', type: 'uint256' },
  { name: 'verifyingContract', type: 'address' },
]

export const Permit = [
  { name: 'owner', type: 'address' },
  { name: 'spender', type: 'address' },
  { name: 'value', type: 'uint256' },
  { name: 'nonce', type: 'uint256' },
  { name: 'deadline', type: 'uint256' },
]

export const Delegation = [
  { name: 'delegatee', type: 'address' },
  { name: 'nonce', type: 'uint256' },
  { name: 'expiry', type: 'uint256' },
]

export const buildDataPermit = (
  chainId: any,
  verifyingContract: any,
  owner: any,
  spender: any,
  value: any,
  nonce: any,
  deadline = MAX_UINT256
) => ({
  primaryType: 'Permit',
  types: { EIP712Domain, Permit },
  domain: { name: 'Optimism', version: '1', chainId, verifyingContract },
  message: { owner, spender, value, nonce, deadline },
})

export const buildDataDelegation = (
  chainId: any,
  verifyingContract: any,
  delegatee: any,
  nonce: any,
  expiry = MAX_UINT256
) => ({
  types: { EIP712Domain, Delegation },
  domain: { name: 'Optimism', version: '1', chainId, verifyingContract },
  primaryType: 'Delegation',
  message: { delegatee, nonce, expiry },
})

export const domainSeparator = (
  name: any,
  version: any,
  chainId: any,
  verifyingContract: any
) => {
  return (
    '0x' +
    ethSigUtil.TypedDataUtils.hashStruct(
      'EIP712Domain',
      { name, version, chainId, verifyingContract },
      { EIP712Domain }
    ).toString('hex')
  )
}
