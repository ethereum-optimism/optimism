import * as bre from '@nomiclabs/buidler'
import { keccak256 } from '@ethersproject/keccak256'
import { toHexBuffer } from './buffer-utils'

const getStateManager = (): any => {
  const ethModule = bre.ethereum['_ethModule' as any]
  const node = ethModule['_node' as any]
  return node['_stateManager' as any]
}

const getStateTrie = (stateManager: any): any => {
  const wrapped = stateManager['_wrapped' as any]
  return wrapped['_trie' as any]
}

const getTrieProofCb = (trie: any, key: Buffer, cb: any): void => {
  let nodes = []

  trie.findPath(key, function (err, node, remaining, stack) {
    if (err) return cb(err)

    if (remaining.length > 0) {
      return cb(new Error("Node does not contain the key"))
    }

    nodes = stack
    let p = [trie.root]
    for (let i = 0; i < nodes.length; i++) {
      console.log(nodes[i])
      let rlpNode = nodes[i].serialize()

      if ((rlpNode.length >= 32) || (i === 0)) {
        p.push(rlpNode)
      }
    }

    cb(null, p)
  })
}

const getTrieProof = async (trie: any, key: Buffer): Promise<any> => {
  key = toHexBuffer(keccak256('0x' + key.toString('hex')))
  return new Promise<any>((resolve, reject) => {
    getTrieProofCb(trie, key, function (err, proof) {
      if (err) reject(err)

      resolve(proof)
    })
  })
}

export const getStateTrieProof = async (address: string): Promise<any> => {
  const addressBuf = toHexBuffer(address)

  const stateManager = getStateManager()
  const stateTrie = getStateTrie(stateManager)

  const account = await stateManager.getAccount(addressBuf)
  const proof = await getTrieProof(stateTrie, addressBuf)

  return {
    account,
    proof
  }
}
