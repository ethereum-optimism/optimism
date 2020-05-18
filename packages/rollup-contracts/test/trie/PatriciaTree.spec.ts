import '../setup'

/* External Imports */
import { getLogger, numberToHexString, bufToHexString, padToLength, hexStrToNumber } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets, link } from 'ethereum-waffle'

/* Logging */
const log = getLogger('patricia-tree', true)

/* Contract Imports */
import * as PatriciaTreeImplementation from '../../build/PatriciaTreeImplementation.json'
import * as PatriciaTreeLibrary from '../../build/PatriciaTree.json'

const ZERO_KEY = padToLength(numberToHexString(0), 32*2)
const BIG_RANDOM_KEY = padToLength(numberToHexString(1234567), 32*2)

const insertSequentialKeys = async (
  treeContract: any,
  numKeysToInsert: number,
  startingIndex: number = 0
): Promise<{
  key: string,
  value: string
}[]> => {
  const pairs = []
  for (let i = startingIndex; i < startingIndex + numKeysToInsert; i++) {
    const key = padToLength(numberToHexString(i), 32*2)
    const value = padToLength(numberToHexString(i*32), 32*2)
    await treeContract.insert(
      key,
      value
    )
    pairs.push({
      key,
      value
    })
  }
  return pairs
}

describe.only('PatriciaTree', async () => {
    let fullTree
    const provider = createMockProvider()
    const [wallet1, wallet2] = getWallets(provider)

    before(async () => {
      const treeLibrary = await deployContract(wallet1, PatriciaTreeLibrary, [])
      link(PatriciaTreeImplementation, 'contracts/trie/tree.sol:PatriciaTree', treeLibrary.address)
    })

    beforeEach('deploy new PatriciaTree', async () => {
      fullTree = await deployContract(wallet1, PatriciaTreeImplementation, [], {
        gasLimit: 6700000,
      })
    })
    describe('getProof() & verifyProof()', async () => {
      it('should be able to verify proofs for some keys', async () => {
        const numLeaves = 5
        const KVPairs = await insertSequentialKeys(fullTree, numLeaves)
        const rootHash = await fullTree.getRootHash()
        for (const {key, value} of KVPairs) {
          const proof = await fullTree.getProof(key)
          log.debug(`Got proof for key: ${key}, value: ${value}.  It has:`)
          log.debug(`    proof bitmask (in binary): ${
            hexStrToNumber(proof.branchMask._hex)
            .toString(2)
          }`)
          log.debug(`    proof siblings: ${proof._siblings}`)
          await fullTree.verifyProof(
            rootHash,
            key,
            value,
            proof.branchMask,
            proof._siblings
          )
        }
    })

    it.only('non inclusion messing around', async () => {
      const numLeaves = 21
      const offset = 1
      const KVPairs = await insertSequentialKeys(fullTree, numLeaves, offset)

      const keyToUse = ZERO_KEY

      log.debug(`getting  non-inclusion proof for key ${keyToUse}`)
      const nonInclusionProof = await fullTree.getNonInclusionProof(
        keyToUse
      )
      
      const leafLabel = nonInclusionProof[0]
      const leafNode = nonInclusionProof[1]
      const branchMask = nonInclusionProof[2]
      const siblings = nonInclusionProof[3]
      const leafLength = nonInclusionProof[4]

      log.debug(`here is the non inclusion proof:`)
      log.debug(`leaf label: ${leafLabel}`)
      log.debug(`leaf label(this time in binary): ${hexStrToNumber(leafLabel).toString(2)}`)
      log.debug(`leaf node/hash: ${leafNode}`)
      log.debug(`branch mask (in binary): ${hexStrToNumber(branchMask._hex).toString(2)}`)
      log.debug(`siblings: ${siblings}`)
      log.debug(`leaf length: ${leafLength}`)

      const ONE_KEY = padToLength(numberToHexString(1), 32*2)
      const proof = await fullTree.getProof(ONE_KEY)


      const onekeyval = await fullTree.get(ONE_KEY)
      const hashedonekeyval = await fullTree.getHash(onekeyval);
      
      log.debug(`Got proof for key: ${ONE_KEY}, It has:`)
      log.debug(`    proof bitmask (in binary): ${
        hexStrToNumber(proof.branchMask._hex)
        .toString(2)
      }`)
      log.debug(`    proof edge commitment: ${hashedonekeyval}`)
      log.debug(`    proof siblings: ${proof._siblings}`)
      log.debug(` and the value for ${ONE_KEY} is: ${onekeyval}, which hashes to: ${hashedonekeyval}`)


      let rootHash = await fullTree.getRootHash()



      await fullTree.verifyProof(
        rootHash,
        ONE_KEY,
        onekeyval,
        proof.branchMask,
        proof._siblings
      )


      const doesConflict = await fullTree.verifyNonInclusionProof2(
        rootHash,
        keyToUse,
        ONE_KEY,
        256,
        leafNode,
        branchMask,
        siblings
      )
      log.debug(`branches at conflict?: ${JSON.stringify(doesConflict)}`)
    })
  })
})
