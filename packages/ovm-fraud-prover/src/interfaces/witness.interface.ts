export interface MerkleTrieWitness {
  root: string
  proof: string
  key: string
  value: any
}

export interface EncodedTrieWitness extends MerkleTrieWitness {
  value: string
}

export interface StateTrieWitness extends MerkleTrieWitness {
  value: {
    nonce: number
    balance: number
    storageRoot: string
    codeHash: string
  }
}

export interface AccountTrieWitness {
  stateTrieWitness: StateTrieWitness
  accountTrieWitness: EncodedTrieWitness
}

export type FraudProofWitness = StateTrieWitness | AccountTrieWitness

export const isAccountTrieWitness = (
  witness: FraudProofWitness
): witness is AccountTrieWitness => {
  return witness.hasOwnProperty('stateTrieWitness')
}
