/* External Imports */
import { BigNumber } from '@ethersproject/bignumber'

export interface MerkleTrieWitness {
  root: string
  proof: Buffer[]
  key: string
  value: string
}

export interface AccountState {
  nonce: number
  balance: BigNumber
  storageRoot: string
  codeHash: string
}

export interface StateTrieWitness {
  root: string
  proof: Buffer[]
  ovmContractAddress: string
  codeContractAddress: string
  value: AccountState
}

export interface AccountTrieWitness {
  stateTrieWitness: StateTrieWitness
  accountTrieWitness: MerkleTrieWitness
}

export type FraudProofWitness = StateTrieWitness | AccountTrieWitness

export const isAccountTrieWitness = (
  witness: FraudProofWitness
): witness is AccountTrieWitness => {
  return witness.hasOwnProperty('stateTrieWitness')
}