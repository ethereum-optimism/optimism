import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBlock } from './RLhelper'

/* Logging */
const log = getLogger('rollup-tx-queue', true)

/* Contract Imports */
import * as CanonicalTransactionChain from '../../build/CanonicalTransactionChain.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('CanonicalTransactionChain', () => {
  const provider = createMockProvider()
  const [wallet, sequencer, canonicalTransactionChain] = getWallets(provider)
  let canonicalTxChain
  let rollupMerkleUtils

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    canonicalTxChain = await deployContract(
      wallet,
      CanonicalTransactionChain,
      [rollupMerkleUtils.address, sequencer.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  const enqueueAndGenerateBlock = async (
    block: string[],
    timestamp: number,
    blockIndex: number,
    cumulativePrevElements: number
  ): Promise<DefaultRollupBlock> => {
    // Submit the rollup block on-chain
    await canonicalTxChain
      .connect(sequencer)
      .appendTransactionBatch(block, timestamp)
    // Generate a local version of the rollup block
    const localBlock = new DefaultRollupBlock(
      timestamp,
      false,
      blockIndex,
      cumulativePrevElements,
      block
    )
    await localBlock.generateTree()
    return localBlock
  }

  /*
   * Test enqueueBlock()
   */
  describe('appendTransactionBatch() ', async () => {
    it('should not throw as long as it gets a bytes array (even if its invalid)', async () => {
      const block = ['0x1234', '0x1234']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(block, timestamp) // Did not throw... success!
    })

    it('should throw if submitting an empty block', async () => {
      const emptyBlock = []
      const timestamp = 0
      try {
        await canonicalTxChain
          .connect(sequencer)
          .appendTransactionBatch(emptyBlock, timestamp)
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error('Allowed an empty block to be appended')
    })

    it('should add to blocks array', async () => {
      const block = ['0x1234', '0x6578']
      const timestamp = 0
      const output = await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(block, timestamp)
      log.debug('enqueue block output', JSON.stringify(output))
      const blocksLength = await canonicalTxChain.getBlocksLength()
      blocksLength.toNumber().should.equal(1)
    })

    it('should update cumulativeNumElements correctly', async () => {
      const block = ['0x1234', '0x5678']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(block, timestamp)
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(2)
    })
    it('should allow appendTransactionBatch from sequencer', async () => {
      const block = ['0x1234', '0x6578']
      const timestamp = 0
      await canonicalTxChain
        .connect(sequencer)
        .appendTransactionBatch(block, timestamp) // Did not throw... success!
    })
    it('should not allow appendTransactionBatch from other address', async () => {
      const block = ['0x1234', '0x6578']
      const timestamp = 0
      await canonicalTxChain
        .appendTransactionBatch(block, timestamp)
        .should.be.revertedWith(
          'VM Exception while processing transaction: revert Message sender does not have permission to enqueue'
        )
    })
    it('should calculate blockHeaderHash correctly', async () => {
      const block = ['0x1234', '0x5678']
      const blockIndex = 0
      const cumulativePrevElements = 0
      const timestamp = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        timestamp,
        blockIndex,
        cumulativePrevElements
      )
      //Check blockHeaderHash
      const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
      const calculatedBlockHeaderHash = await canonicalTxChain.blocks(0)
      calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
    })
    it('should add multiple blocks correctly', async () => {
      const block = ['0x1234', '0x5678']
      const numBlocks = 10
      for (let blockIndex = 0; blockIndex < numBlocks; blockIndex++) {
        const timestamp = blockIndex
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(
          block,
          timestamp,
          blockIndex,
          cumulativePrevElements
        )
        //Check blockHeaderHash
        const expectedBlockHeaderHash = await localBlock.hashBlockHeader()
        const calculatedBlockHeaderHash = await canonicalTxChain.blocks(
          blockIndex
        )
        calculatedBlockHeaderHash.should.equal(expectedBlockHeaderHash)
      }
      //check cumulativeNumElements
      const cumulativeNumElements = await canonicalTxChain.cumulativeNumElements.call()
      cumulativeNumElements.toNumber().should.equal(numBlocks * block.length)
      //check blocks length
      const blocksLength = await canonicalTxChain.getBlocksLength()
      blocksLength.toNumber().should.equal(numBlocks)
    })
    //TODO test with actual transitions and actual state roots
    //TODO test above with multiple blocks with different # elements and different size elements
  })

  /*
   * Test verifyElement()
   */
  describe('verifyElement() ', async () => {
    it('should return true for valid elements for different blockIndexs', async () => {
      const maxBlockNumber = 5
      const minBlockNumber = 0
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      for (
        let blockIndex = minBlockNumber;
        blockIndex < maxBlockNumber + 1;
        blockIndex++
      ) {
        log.debug(`testing valid proof for block #: ${blockIndex}`)
        const timestamp = blockIndex
        const cumulativePrevElements = block.length * blockIndex
        const localBlock = await enqueueAndGenerateBlock(
          block,
          timestamp,
          blockIndex,
          cumulativePrevElements
        )
        // Create inclusion proof for the element at elementIndex
        const elementIndex = 3
        const element = block[elementIndex]
        const position = localBlock.getPosition(elementIndex)
        const elementInclusionProof = await localBlock.getElementInclusionProof(
          elementIndex
        )
        log.debug(
          `trying to correctly verify this inclusion proof: ${JSON.stringify(
            elementInclusionProof
          )}`
        )
        //run verifyElement()
        //
        const isIncluded = await canonicalTxChain.verifyElement(
          element,
          position,
          elementInclusionProof
        )
        log.debug('isIncluded: ', JSON.stringify(isIncluded))
        isIncluded.should.equal(true)
      }
    })

    it('should return false for wrong position with wrong indexInBlock', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const timestamp = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        timestamp,
        blockIndex,
        cumulativePrevElements
      )
      const elementIndex = 1
      const element = block[elementIndex]
      const position = localBlock.getPosition(elementIndex)
      const elementInclusionProof = await localBlock.getElementInclusionProof(
        elementIndex
      )
      log.debug(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //run verifyElement()
      //
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log.debug('isIncluded: ', JSON.stringify(isIncluded))
      isIncluded.should.equal(false)
    })

    it('should return false for wrong position and matching indexInBlock', async () => {
      const block = ['0x1234', '0x4567', '0x890a', '0xabcd']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const timestamp = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        timestamp,
        blockIndex,
        cumulativePrevElements
      )
      //generate inclusion proof
      const elementIndex = 1
      const element = block[elementIndex]
      const position = localBlock.getPosition(elementIndex)
      const elementInclusionProof = await localBlock.getElementInclusionProof(
        elementIndex
      )
      //Give wrong position so inclusion proof is wrong
      const wrongPosition = position + 1
      //Change index to also be false (so position = index + cumulative)
      elementInclusionProof.indexInBlock++
      log.debug(
        `trying to falsely verify this inclusion proof: ${JSON.stringify(
          elementInclusionProof
        )}`
      )
      //run verifyElement()
      //
      const isIncluded = await canonicalTxChain.verifyElement(
        element,
        wrongPosition,
        elementInclusionProof
      )
      log.debug('isIncluded: ', JSON.stringify(isIncluded))
      isIncluded.should.equal(false)
    })
  })
})
