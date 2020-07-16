/* External Imports */
import { BaseTrie } from 'merkle-patricia-tree'
import * as rlp from 'rlp'

/* Internal Imports */
import { WorldState, FraudProofWitness, isAccountTrieWitness, StateTrieWitness, MerkleTrieWitness } from '../interfaces'
import { toHexString } from './encoding'

/**
 * Utility; updates a trie and returns the proof for the updated value.
 * @param trie Trie to update.
 * @param key Key to insert or update.
 * @param value Value to insert at the given key.
 * @returns Proof for the updated k/v pair.
 */
export const updateAndProve = async (
  trie: BaseTrie,
  key: Buffer,
  value: Buffer
): Promise<string> => {
  await trie.put(key, value)
  const proof = await BaseTrie.prove(trie, key)
  return toHexString(rlp.encode(proof))
}

const makeTrieFromWitnesses = async (
  witnesses: Array<StateTrieWitness | MerkleTrieWitness>
): Promise<BaseTrie> => {
  let rootNode: Buffer
  let nonRootNodes: Buffer[] = []

  for (const witness of witnesses) {
    const nodes = witness.proof

    if (rootNode === undefined) {
      rootNode = nodes[0]
    }

    if (!rootNode.equals(nodes[0])) {
      throw new Error("All root nodes in provided proofs must match.")
    }

    nonRootNodes = nonRootNodes.concat(...nodes.slice(1))
  }

  const allNodes = [rootNode].concat(nonRootNodes)

  return BaseTrie.fromProof(allNodes)
}

export const makeWorldStateFromWitnesses = async (
  witnesses: FraudProofWitness[]
): Promise<WorldState> => {
  const stateTrieWitnesses = witnesses.map((witness) => {
    return isAccountTrieWitness(witness) ? witness.stateTrieWitness : witness
  })
  const stateTrie = await makeTrieFromWitnesses(stateTrieWitnesses)

  const accountTrieWitnessMap = witnesses.reduce((map, witness) => {
    if (!isAccountTrieWitness(witness)) {
      return map
    }

    const address = witness.stateTrieWitness.ovmContractAddress
    if (!(address in map)) {
      map[address] = []
    }

    map[address] = map[address].concat(witness.accountTrieWitness)
    return map
  }, {})

  let accountTries = {}
  for (const address of Object.keys(accountTrieWitnessMap)) {
    const accountTrieWitnesses = accountTrieWitnessMap[address]
    accountTries[address] = await makeTrieFromWitnesses(accountTrieWitnesses)
  }

  return {
    stateTrie,
    accountTries
  }
}