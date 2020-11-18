/* External Imports */
import { keccak256 } from '@ethersproject/keccak256'

/* Internal Imports */
import { fromHexString, toHexString } from './buffer-utils'

/**
 * Callback wrapper; returns a Merkle proof for an element within a trie.
 * @param trie Trie to generate the proof from.
 * @param key Key to prove existance of.
 * @param cb Callback for the returned value.
 */
const getTrieProofCb = (trie: any, key: Buffer, cb: any): void => {
  let nodes = []

  // tslint:disable-next-line:only-arrow-functions
  trie.findPath(key, function(err: any, node: any, remaining: any, stack: any) {
    if (err) {
      return cb(err)
    }

    if (remaining.length > 0) {
      return cb(
        new Error(`Node does not contain the key: ${key.toString('hex')}`)
      )
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
  // Some implementations use secure tries (hashed keys) but others don't.
  // Circumvent this by trying twice, once unhashed and once hashed.

  return new Promise<any>((resolve, reject) => {
    // tslint:disable-next-line:only-arrow-functions
    getTrieProofCb(trie, key, function(err: any, proof: any) {
      if (err) {
        key = fromHexString(keccak256(toHexString(key)))
        getTrieProofCb(trie, key, function(err: any, proof: any) {
          if (err) {
            reject(err)
          }

          resolve(proof)
        })
      } else {
        resolve(proof)
      }
    })
  })
}

/**
 * Returns the value for a given key in a trie.
 * @param trie Trie to get the value from.
 * @param key Key to get a value for.
 * @returns Value for the provided key.
 */
const getKeyValue = async (trie: any, key: Buffer): Promise<Buffer> => {
  return new Promise<any>((resolve, reject) => {
    trie.get(key, (err: any, value: Buffer) => {
      if (err) {
        reject(err)
      }

      resolve(value)
    })
  })
}

interface EthStorageProof {
  key: string
  value: string
  proof: string[]
}

export interface EthTrieProof {
  balance: string
  nonce: string
  storageHash: string
  codeHash: string
  stateRoot: string
  accountProof: string[]
  storageProof: EthStorageProof[]
}

/**
 * Returns a trie proof in the format of EIP-1186.
 * @param vm VM to generate the proof from.
 * @param address Address to generate the proof for.
 * @param slots Slots to get proofs for.
 * @returns A proof object in the format of EIP-1186.
 */
export const getEthTrieProofInternal = async (
  vm: any,
  address: Buffer,
  slots: Buffer[] = []
): Promise<EthTrieProof> => {
  // Generate the account proof using the state trie.
  const stateTrie = vm.stateManager._trie
  const account = await vm.pStateManager.getAccount(address)
  const accountProof = await getTrieProof(stateTrie, address)

  // Generate storage proofs for each of the requested slots.
  const storageTrie = await new Promise<any>((resolve, reject) => {
    vm.stateManager._getStorageTrie(address, (err, res) => {
      if (err) {
        reject(err)
      } else {
        resolve(res)
      }
    })
  })

  const storageProof: EthStorageProof[] = []
  for (const slot of slots) {
    const value = await getKeyValue(storageTrie, slot)
    const proof = await getTrieProof(storageTrie, slot)
    storageProof.push({
      key: toHexString(slot),
      value: toHexString(value),
      proof: proof.map((el: Buffer) => {
        return toHexString(el)
      }),
    })
  }

  return {
    balance: account.balance.length ? toHexString(account.balance) : '0x0',
    nonce: account.nonce.length ? toHexString(account.nonce) : '0x0',
    storageHash: toHexString(account.stateRoot),
    codeHash: toHexString(account.codeHash),
    stateRoot: toHexString(stateTrie.root),
    accountProof: accountProof.map((el: Buffer) => {
      return toHexString(el)
    }),
    storageProof,
  }
}
