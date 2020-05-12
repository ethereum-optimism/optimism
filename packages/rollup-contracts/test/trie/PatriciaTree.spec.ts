import '../setup'

/* External Imports */
import { getLogger, numberToHexString } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets, link } from 'ethereum-waffle'

/* Logging */
const log = getLogger('patricia-tree', true)

/* Contract Imports */
import * as PatriciaTreeImplementation from '../../build/PatriciaTreeImplementation.json'
import * as PatriciaTreeLibrary from '../../build/PatriciaTree.json'

describe.only('PatriciaTree', async () => {
    let fullTree
    const provider = createMockProvider()
    const [wallet1, wallet2] = getWallets(provider)

    beforeEach('deploy PatriciaTree', async () => {
      const treeLibrary = await deployContract(wallet1, PatriciaTreeLibrary, [])
      link(PatriciaTreeImplementation, 'contracts/trie/tree.sol:PatriciaTree', treeLibrary.address)
      fullTree = await deployContract(wallet1, PatriciaTreeImplementation, [], {
        gasLimit: 6700000,
      })
    })
    describe('getProof() & verifyProof()', async () => {
      it('should be able to verify merkle proof for a given key', async () => {
        const numLeaves = 5
        let keys = []
        let values = []
        for (let i = 0; i < numLeaves; i++) {
          keys.push(numberToHexString(i*100))
          values.push(numberToHexString(i + 100))
          await fullTree.insert(keys[i], values[i])
        }
        for (let i = 0; i < numLeaves; i++) {
          const key = keys[i]
          const value = values[i]
          log.debug(`key: ${key}, value: ${value}`)
          let proof = await fullTree.getProof(key)
          console.log(proof)
          let rootHash = await fullTree.getRootHash()
          await fullTree.verifyProof(rootHash, key, value, proof.branchMask, proof._siblings)
        }
    })
  })
})
