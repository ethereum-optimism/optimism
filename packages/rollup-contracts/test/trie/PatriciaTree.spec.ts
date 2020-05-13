import '../setup'

/* External Imports */
import { getLogger, numberToHexString, bufToHexString, padToLength, hexStrToNumber } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets, link } from 'ethereum-waffle'

/* Logging */
const log = getLogger('patricia-tree', true)

/* Contract Imports */
import * as PatriciaTreeImplementation from '../../build/PatriciaTreeImplementation.json'
import * as PatriciaTreeLibrary from '../../build/PatriciaTree.json'
import { Z_TREES } from 'zlib'

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
        const numLeaves = 3
        let keys = []
        let values = []
        for (let i = 0; i < numLeaves; i++) {
          keys.push(
            padToLength(numberToHexString(i), 32*2)
          )
          values.push(
            padToLength(numberToHexString(i*32), 32*2)
          )
          await fullTree.insert(keys[i], values[i])
        }
        for (let i = 2; i < 3; i++) {
          const key = keys[i]
          const value = values[i]
          let proof = await fullTree.getProof(key)
          log.debug(`key: ${key}, value: ${value}`)
          log.debug(`proof bitmask (in binary): ${
            hexStrToNumber(proof.branchMask._hex)
            .toString(2)
          }`)
          log.debug(`proof siblings: ${proof._siblings}`)
          let rootHash = await fullTree.getRootHash()
          await fullTree.verifyProof(rootHash, key, value, proof.branchMask, proof._siblings)
        }
    })

    it.only('non inclusion messing around', async () => {
      const numLeaves = 2
      let keys = []
      let values = []
      for (let i = 0; i < numLeaves; i++) {
        keys.push(
          padToLength(numberToHexString(i+1), 32*2)
        )
        values.push(
          padToLength(numberToHexString(i*32), 32*2)
        )
        await fullTree.insert(keys[i], values[i])
      }
      const ZERO_KEY = padToLength(numberToHexString(0), 32*2)

      const BIG_RANDOM_KEY = padToLength(numberToHexString(1345), 32*2)

      log.debug(`gettingnoninclusion prooffor big key ${BIG_RANDOM_KEY}`)
      const nonInclusionProof = await fullTree.getNonInclusionProof(
        BIG_RANDOM_KEY
      )
      
      const leafLabel = nonInclusionProof[0]
      const leafNode = nonInclusionProof[1]
      const branchMask = nonInclusionProof[2]
      const siblings = nonInclusionProof[3]

      log.debug(`here is the non inclusion proof:`)
      log.debug(`leaf label(binary): ${hexStrToNumber(leafLabel).toString(2)}`)
      log.debug(`leaf node/hash: ${leafNode}`)
      log.debug(`branch mask (in binary): ${hexStrToNumber(branchMask._hex).toString(2)}`)
      log.debug(`siblings: ${siblings}`)

      let rootHash = await fullTree.getRootHash()
      await fullTree.verifyNonInclusionProof(
        rootHash,
        ZERO_KEY,
        leafLabel,
        leafNode,
        branchMask,
        siblings
      )
  })
  })
})
