/* External Imports */
import { BaseTrie } from 'merkle-patricia-tree'
import * as rlp from 'rlp'

/* Internal Imports */
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