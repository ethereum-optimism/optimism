/* External Imports */
import * as bre from '@nomiclabs/buidler'
import { keccak256 } from '@ethersproject/keccak256'

/* Internal Imports */
import { toHexBuffer } from '../../src/utils'

/**
 * Used to access buidler's state manager.
 * @returns Buidler's state manager.
 */
const getStateManager = (): any => {
  const ethModule = bre.ethereum['_ethModule' as any]
  const node = ethModule['_node' as any]
  return node['_stateManager' as any]
}

/**
 * Retrieves buidler's state trie.
 * @param stateManager Buidler's state manager.
 * @returns Buidler's state trie.
 */
const getStateTrie = (stateManager: any): any => {
  const wrapped = stateManager['_wrapped' as any]
  return wrapped['_trie' as any]
}

/**
 * Retrieves buidler's state cache.
 * @param stateManager Buidler's state manager.
 * @returns Buidler's state cache.
 */
const getStateCache = (stateManager: any): any => {
  const wrapped = stateManager['_wrapped' as any]
  return wrapped['_cache' as any]
}

/**
 * Callback wrapper; retrieves a storage trie for a given address.
 * @param stateManager Buidler's state manager.
 * @param addressBuf Address to get a trie for, as a Buffer.
 * @param cb Callback for the returned value.
 */
const getStorageTrieCb = (
  stateManager: any,
  addressBuf: Buffer,
  cb: any
): void => {
  const wrapped = stateManager['_wrapped' as any]
  // tslint:disable-next-line:only-arrow-functions
  wrapped['_getStorageTrie' as any](addressBuf, function(err, trie) {
    if (err) {
      return cb(err)
    }

    return cb(null, trie)
  })
}

/**
 * Retrieves a storage trie for a given address.
 * @param stateManager Buidler's state manager.
 * @param addressBuf Address to get a trie for, as a Buffer.
 * @returns Storage trie for the address.
 */
const getStorageTrie = async (
  stateManager: any,
  addressBuf: Buffer
): Promise<any> => {
  return new Promise<any>((resolve, reject) => {
    // tslint:disable-next-line:only-arrow-functions
    getStorageTrieCb(stateManager, addressBuf, function(err, trie) {
      if (err) {
        reject(err)
      }

      resolve(trie)
    })
  })
}

/**
 * Callback wrapper; returns a Merkle proof for an element within a trie.
 * @param trie Trie to generate the proof from.
 * @param key Key to prove existance of.
 * @param cb Callback for the returned value.
 */
const getTrieProofCb = (trie: any, key: Buffer, cb: any): void => {
  let nodes = []

  // tslint:disable-next-line:only-arrow-functions
  trie.findPath(key, function(err, node, remaining, stack) {
    if (err) {
      return cb(err)
    }

    if (remaining.length > 0) {
      return cb(new Error('Node does not contain the key'))
    }

    nodes = stack
    const p = []
    for (let i = 0; i < nodes.length; i++) {
      const rlpNode = nodes[i].serialize()

      if (rlpNode.length >= 32 || i === 0) {
        p.push(rlpNode)
      }
    }

    cb(null, p)
  })
}

/**
 * Returns a Merkle proof for an element within a trie.
 * @param trie Trie to generate the proof from.
 * @param key Key to prove existance of.
 * @returns Merkle proof for the provided element.
 */
const getTrieProof = async (trie: any, key: Buffer): Promise<any> => {
  key = toHexBuffer(keccak256('0x' + key.toString('hex')))
  return new Promise<any>((resolve, reject) => {
    // tslint:disable-next-line:only-arrow-functions
    getTrieProofCb(trie, key, function(err, proof) {
      if (err) {
        reject(err)
      }

      resolve(proof)
    })
  })
}

/**
 * Callback wrapper; looks up an account in the state.
 * @param cache Buidler's state cache.
 * @param key Address to search for.
 * @param cb Callback for the returned value.
 */
const getAccountCb = (cache: any, key: Buffer, cb: any): void => {
  // tslint:disable-next-line:only-arrow-functions
  cache._lookupAccount(key, function(err, account) {
    if (err) {
      return cb(err)
    }

    return cb(null, account)
  })
}

/**
 * Looks up an account in the state.
 * @param cache Buidler's state cache
 * @param key Address to search for.
 * @returns Account object for the given address.
 */
const getAccount = async (cache: any, key: Buffer): Promise<any> => {
  return new Promise<any>((resolve, reject) => {
    // tslint:disable-next-line:only-arrow-functions
    getAccountCb(cache, key, function(err, account) {
      if (err) {
        reject(err)
      }

      resolve(account)
    })
  })
}

/**
 * Generates a Merkle proof for a given address within the state trie.
 * @param address Address to generate a proof for.
 * @returns Merkle proof information for the given address.
 */
export const getStateTrieProof = async (address: string): Promise<any> => {
  const addressBuf = toHexBuffer(address)

  const stateManager = getStateManager()
  const stateTrie = getStateTrie(stateManager)
  const stateCache = getStateCache(stateManager)

  const account = await getAccount(stateCache, addressBuf)
  const proof = await getTrieProof(stateTrie, addressBuf)

  return {
    account,
    address,
    root: stateTrie.root,
    proof,
  }
}

/**
 * Generates a Merkle proof for a slot of an account's storage trie.
 * @param address Address of the account to get a proof for.
 * @param slot Specific slot to prove.
 * @returns Merkle proof information for the given address/slot pair.
 */
export const getStorageTrieProof = async (
  address: string,
  slot: string
): Promise<any> => {
  const addressBuf = toHexBuffer(address)
  const slotBuf = toHexBuffer(slot)

  const stateManager = getStateManager()
  const stateTrie = getStateTrie(stateManager)
  const stateCache = getStateCache(stateManager)
  const storageTrie = await getStorageTrie(stateManager, addressBuf)

  const account = await getAccount(stateCache, addressBuf)
  const proof = await getTrieProof(storageTrie, slotBuf)

  return {
    account,
    address,
    slot,
    stateTrieRoot: stateTrie.root,
    storageTrieRoot: storageTrie.root,
    proof,
  }
}
