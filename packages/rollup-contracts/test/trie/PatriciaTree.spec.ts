import '../setup'

/* External Imports */
import {
  getLogger,
  numberToHexString,
  padToLength,
  TestUtils,
} from '@eth-optimism/core-utils'
import {
  createMockProvider,
  deployContract,
  getWallets,
  link,
} from 'ethereum-waffle'

/* Logging */
const log = getLogger('patricia-tree', true)

/* Contract Imports */
import * as FullPatriciaTreeImplementation from '../../build/FullPatriciaTreeImplementation.json'
import * as FullPatriciaTreeLibrary from '../../build/FullPatriciaTree.json'
import * as UtilsTest from '../../build/UtilsTest.json'

const insertSequentialKeys = async (
  treeContract: any,
  numKeysToInsert: number,
  startingIndex: number = 0
): Promise<Array<{
  key: string
  value: string
}>> => {
  const pairs = []
  for (let i = startingIndex; i < startingIndex + numKeysToInsert; i++) {
    const key = padToLength(numberToHexString(i), 32 * 2)
    const value = padToLength(numberToHexString(i * 32), 32 * 2)
    await treeContract.insert(key, value)
    pairs.push({
      key,
      value,
    })
  }
  return pairs
}

const insertAndVerifySequential = async (
  treeContract: any,
  numKeysToInsert: number,
  startingIndex: number = 0
) => {
  const KVPairs = await insertSequentialKeys(
    treeContract,
    numKeysToInsert,
    startingIndex
  )
  const rootHash = await treeContract.getRootHash()
  for (const pair of KVPairs) {
    const proof = await treeContract.getProof(pair.key)
    await treeContract.verifyProof(
      rootHash,
      pair.key,
      pair.value,
      proof.branchMask,
      proof._siblings
    )
  }
}

const getAndVerifyNonInclusionProof = async (
  treeContract: any,
  key: number
) => {
  const keyToUse = padToLength(numberToHexString(key), 32 * 2)
  const nonInclusionProof = await treeContract.getNonInclusionProof(keyToUse)

  const conflictingEdgeLabel = nonInclusionProof[0]
  const leafNode = nonInclusionProof[1]
  const branchMask = nonInclusionProof[2]
  const siblings = nonInclusionProof[3]

  const rootHash = await treeContract.getRootHash()

  await treeContract.verifyNonInclusionProof(
    rootHash,
    keyToUse,
    conflictingEdgeLabel[0],
    conflictingEdgeLabel[1],
    leafNode,
    branchMask,
    siblings
  )
}

describe('PatriciaTree (full, non-stateless version)', async () => {
  let fullTree
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)

  before(async () => {
    const treeLibrary = await deployContract(
      wallet1,
      FullPatriciaTreeLibrary,
      []
    )
    link(
      FullPatriciaTreeImplementation,
      'contracts/state-tree/FullPatriciaTree.sol:FullPatriciaTree',
      treeLibrary.address
    )
  })

  beforeEach('Deploy new PatriciaTree', async () => {
    fullTree = await deployContract(
      wallet1,
      FullPatriciaTreeImplementation,
      [],
      {
        gasLimit: 6700000,
      }
    )
  })

  describe('Works as a keystore', async () => {
    const FOO =
      '0x0000000000000000000000000000000000000000000000067320000000000000'
    const BAR =
      '0x0000000000000000000000000000000004578000000000000000000000000000'
    const FUZ =
      '0x0000000000000000157800000000000000000000000000000000000000000000'
    describe('get()', async () => {
      it('should return stored value for the given key', async () => {
        await fullTree.insert(FOO, BAR)
        const retrieved = await fullTree.get(FOO)
        retrieved.should.equal(BAR)
      })
    })

    describe('safeGet()', async () => {
      it('should return stored value for the given key', async () => {
        await fullTree.insert(FOO, BAR)
        const retrieved = await fullTree.safeGet(FOO)
        retrieved.should.equal(BAR)
      })
      it('should throw if the given key is not included', async () => {
        await fullTree.insert(FOO, BAR)
        TestUtils.assertThrowsAsync(async () => {
          await fullTree.safeGet(FUZ)
        })
      })
    })
  })

  describe('Inclusion proof generation and verification', async () => {
    it('should work for the single-key case', async () => {
      const key = 150
      const pairs = await insertAndVerifySequential(fullTree, 1, key)
    })
    it('should work for the two-key sequential case', async () => {
      const startKey = 150
      const pairs = await insertAndVerifySequential(fullTree, 2, startKey)
    })
    it('should work for 17-key sequential case', async () => {
      const startKey = 150
      const pairs = await insertAndVerifySequential(fullTree, 17, startKey)
    })
    it('should work for multiple non-sequential keys', async () => {
      const keyToVerify = 18
      await insertSequentialKeys(fullTree, 1, 5)
      await insertSequentialKeys(fullTree, 1, 13)
      await insertSequentialKeys(fullTree, 1, 27)
      await insertSequentialKeys(fullTree, 1, 100000)
      await insertSequentialKeys(fullTree, 1, 3000000345)
      const pairs = await insertSequentialKeys(fullTree, 1, keyToVerify)
      const pair = pairs[0]
      const rootHash = await fullTree.getRootHash()
      const proof = await fullTree.getProof(pair.key)
      await fullTree.verifyProof(
        rootHash,
        pair.key,
        pair.value,
        proof.branchMask,
        proof._siblings
      )
    })
  })
  describe('Non-inclusion proof generation and verification', async () => {
    it('Should work for an unset key next to a set one', async () => {
      await insertSequentialKeys(fullTree, 1, 0)
      await getAndVerifyNonInclusionProof(fullTree, 1)
    })
    it('Should work for an unset key between two set ones', async () => {
      await insertSequentialKeys(fullTree, 1, 0)
      await insertSequentialKeys(fullTree, 1, 2)
      await getAndVerifyNonInclusionProof(fullTree, 1)
    })
    it('Should work for an unset key next to some set ones', async () => {
      await insertSequentialKeys(fullTree, 7, 1)
      await getAndVerifyNonInclusionProof(fullTree, 0)
    })
    it('Should work for an unset key far away from some set ones', async () => {
      await insertSequentialKeys(fullTree, 3, 0)
      await insertSequentialKeys(fullTree, 3, 60)
      await getAndVerifyNonInclusionProof(fullTree, 17)
    })
  })

  describe('Binary Utils library', async () => {
    it('Legacy tests written in Solidity should all pass', async () => {
      const testerContract = await deployContract(wallet2, UtilsTest)
      await testerContract.test()
    })
  })
})
