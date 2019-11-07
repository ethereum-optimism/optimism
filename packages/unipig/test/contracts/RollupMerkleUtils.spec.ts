import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { SparseMerkleTreeImpl, newInMemoryDB } from '@pigi/core-db'

import { hexStrToBuf, bufToHexString, BigNumber } from '@pigi/core-utils'

/* Internal Imports */
import { makeRepeatedBytes } from '../helpers'

/* Contract Imports */
import * as RollupMerkleUtils from '../../build/contracts/RollupMerkleUtils.json'

/* Logging */
import debug from 'debug'
const log = debug('test:info:merkle-utils')

async function createSMTfromDataBlocks(
  dataBlocks: Buffer[]
): Promise<SparseMerkleTreeImpl> {
  const treeHeight = Math.ceil(Math.log2(dataBlocks.length)) + 1 // The height should actually not be plus 1
  log('Creating tree of height:', treeHeight - 1)
  const tree = await getNewSMT(treeHeight)
  for (let i = 0; i < dataBlocks.length; i++) {
    await tree.update(new BigNumber(i, 10), dataBlocks[i])
  }
  return tree
}

async function getNewSMT(treeHeight: number): Promise<SparseMerkleTreeImpl> {
  return SparseMerkleTreeImpl.create(newInMemoryDB(), undefined, treeHeight)
}

function makeRandomBlockOfSize(blockSize: number): string[] {
  const block = []
  for (let i = 0; i < blockSize; i++) {
    block.push(makeRepeatedBytes('' + Math.floor(Math.random() * 500 + 1), 32))
  }
  return block
}

/* Begin tests */
describe('RollupMerkleUtils', () => {
  const provider = createMockProvider()
  const [wallet1] = getWallets(provider)
  let rollupMerkleUtils

  /* Deploy RollupMerkleUtils library before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet1, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  describe('getMerkleRoot() ', async () => {
    it('should not throw', async () => {
      await rollupMerkleUtils.getMerkleRoot(['0x1234', '0x4321'])
      // Did not throw... success!
    })

    it('should produce a correct merkle tree with two leaves', async () => {
      const block = ['0x1234', '0x4321']
      const bufBlock = block.map((data) => hexStrToBuf(data))
      // Create the Solidity tree, returning the root
      const result = await rollupMerkleUtils.getMerkleRoot(block)
      // Create a local tree
      const tree = await createSMTfromDataBlocks(bufBlock)
      // Get the root
      const root: Buffer = await tree.getRootHash()
      // Compare!
      result.should.equal(bufToHexString(root))
    })

    it('should produce a correct sparse merkle tree with three leaves', async () => {
      const block = ['0x1234', '0x4321', '0x0420']
      const bufBlock = block.map((data) => hexStrToBuf(data))
      // Create the Solidity tree, returning the root
      const result = await rollupMerkleUtils.getMerkleRoot(block)
      // Create a local tree
      const tree = await createSMTfromDataBlocks(bufBlock)
      // Get the root
      const root: Buffer = await tree.getRootHash()
      // Compare!
      result.should.equal(bufToHexString(root))
    })

    it('should produce correct merkle trees with leaves ranging from 1 to 10', async () => {
      for (let i = 1; i < 10; i++) {
        const block = []
        for (let j = 0; j < i; j++) {
          block.push(
            makeRepeatedBytes('' + Math.floor(Math.random() * 500 + 1), 32)
          )
        }
        const bufBlock = block.map((data) => hexStrToBuf(data))
        // Create the Solidity tree, returning the root
        const result = await rollupMerkleUtils.getMerkleRoot(block)
        // Create a local tree
        const tree = await createSMTfromDataBlocks(bufBlock)
        // Get the root
        const root: Buffer = await tree.getRootHash()
        // Compare!
        result.should.equal(bufToHexString(root))
      }
    })
  })

  describe('verify()', async () => {
    it('should verify all the nodes of trees at various heights', async () => {
      const maxBlockSize = 5
      const minBlockSize = 1
      // Create trees of multiple sizes tree
      for (
        let blockSize = minBlockSize;
        blockSize < maxBlockSize + 1;
        blockSize++
      ) {
        // Create the block we'll prove inclusion for
        const block = makeRandomBlockOfSize(blockSize)
        const bufBlock = block.map((data) => hexStrToBuf(data))
        const treeHeight = Math.ceil(Math.log2(bufBlock.length))
        // Create a local tree
        const tree = await createSMTfromDataBlocks(bufBlock)
        // Get the root
        const root: Buffer = await tree.getRootHash()

        // Now let's set the root in the contract
        await rollupMerkleUtils.setMerkleRootAndHeight(
          bufToHexString(root),
          treeHeight
        )
        // Now that the root is set, let's try verifying all the nodes
        for (let j = 0; j < block.length; j++) {
          const indexOfNode = j
          // Generate an inclusion proof
          const inclusionProof = await tree.getMerkleProof(
            new BigNumber(indexOfNode),
            bufBlock[indexOfNode]
          )
          // Extract the values we need for the proof in the form we need them
          const path = bufToHexString(inclusionProof.key.toBuffer('B', 32))
          const siblings = inclusionProof.siblings.map((sibBuf) =>
            bufToHexString(sibBuf)
          )
          const isValid = await rollupMerkleUtils.verify(
            bufToHexString(inclusionProof.rootHash),
            bufToHexString(inclusionProof.value),
            path,
            siblings
          )
          // Make sure that the verification was successful
          isValid.should.equal(true)
        }
      }
    })
  })

  describe('update()', async () => {
    it('should update all nodes correctly in trees of various heights', async () => {
      const minBlockSize = 1
      const maxBlockSize = 5
      for (
        let blockSize = minBlockSize;
        blockSize < maxBlockSize;
        blockSize++
      ) {
        const block = makeRandomBlockOfSize(blockSize)
        const bufBlock = block.map((data) => hexStrToBuf(data))
        const treeHeight = Math.ceil(Math.log2(bufBlock.length))
        // Create a local tree
        const tree = await createSMTfromDataBlocks(bufBlock)
        // Get the root
        const root: Buffer = await tree.getRootHash()
        // Set the root and the height of our stored tree
        await rollupMerkleUtils.setMerkleRootAndHeight(root, treeHeight)

        // Now that we've set everything up, let's store the full tree in Solidity
        for (let leafIndex = 0; leafIndex < block.length; leafIndex++) {
          const inclusionProof = await tree.getMerkleProof(
            new BigNumber(leafIndex),
            bufBlock[leafIndex]
          )
          // Extract the values we need for the proof in the form we need them
          const path = bufToHexString(inclusionProof.key.toBuffer('B', 32))
          const siblings = inclusionProof.siblings.map((sibBuf) =>
            bufToHexString(sibBuf)
          )
          await rollupMerkleUtils.store(
            bufToHexString(inclusionProof.value),
            path,
            siblings
          )
        }

        // Exciting! We've stored the full tree. Let's start updating everything!
        const newBlock = makeRandomBlockOfSize(blockSize)
        const newBufBlock = newBlock.map((data) => hexStrToBuf(data))
        // For each leaf in the tree let's call update and compare the results
        for (let leafIndex = 0; leafIndex < block.length; leafIndex++) {
          await tree.update(new BigNumber(leafIndex), newBufBlock[leafIndex])
          const inclusionProof = await tree.getMerkleProof(
            new BigNumber(leafIndex),
            newBufBlock[leafIndex]
          )
          // Extract the values we need for the proof in the form we need them
          const path = bufToHexString(inclusionProof.key.toBuffer('B', 32))
          await rollupMerkleUtils.update(
            bufToHexString(inclusionProof.value),
            path
          )
          const newContractRoot = await rollupMerkleUtils.getRoot()
          const newLocalRoot: Buffer = await tree.getRootHash()
          // Compare the updated roots! They should be equal.
          newContractRoot.should.equal(bufToHexString(newLocalRoot))
        }
      }
    }).timeout(5000)
  })
})
