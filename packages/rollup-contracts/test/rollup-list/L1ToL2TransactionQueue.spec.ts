import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBlock } from './RLhelper'

/* Logging */
const log = getLogger('l1-to-l2-tx-queue')

/* Contract Imports */
import * as L1ToL2TransactionQueue from '../../build/L1ToL2TransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
describe('L1ToL2TransactionQueue', () => {
  const provider = createMockProvider()
  const [
    wallet,
    l1ToL2TransactionPasser,
    canonicalTransactionChain,
  ] = getWallets(provider)
  let l1ToL2TxQueue
  let rollupMerkleUtils

  /* Link libraries before tests */
  before(async () => {
    rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
      gasLimit: 6700000,
    })
  })

  /* Deploy a new RollupChain before each test */
  beforeEach(async () => {
    l1ToL2TxQueue = await deployContract(
      wallet,
      L1ToL2TransactionQueue,
      [
        rollupMerkleUtils.address,
        l1ToL2TransactionPasser.address,
        canonicalTransactionChain.address,
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  const enqueueAndGenerateBlock = async (
    block: string[],
    blockIndex: number,
    cumulativePrevElements: number
  ): Promise<DefaultRollupBlock> => {
    // Submit the rollup block on-chain
    const enqueueTx = await l1ToL2TxQueue
      .connect(l1ToL2TransactionPasser)
      .enqueueBlock(block)
    const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
    // Generate a local version of the rollup block
    const ethBlockNumber = txReceipt.blockNumber
    const localBlock = new DefaultRollupBlock(
      ethBlockNumber,
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
  describe('enqueueBlock() ', async () => {
    it('should allow enqueue from l1ToL2TransactionPasser', async () => {
      const block = ['0x1234']
      await l1ToL2TxQueue.connect(l1ToL2TransactionPasser).enqueueBlock(block) // Did not throw... success!
    })
    it('should not allow enqueue from other address', async () => {
      const block = ['0x1234']
      try {
        await l1ToL2TxQueue.enqueueBlock(block)
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error(
        'Allowed non-l1ToL2TransactionPasser account to enqueue block'
      )
    })
  })
  /*
   * Test dequeueBlock()
   */
  describe('dequeueBlock() ', async () => {
    it('should allow dequeue from canonicalTransactionChain', async () => {
      const block = ['0x1234']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      let blocksLength = await l1ToL2TxQueue.getBlocksLength()
      log.debug(`blocksLength before deletion: ${blocksLength}`)
      let front = await l1ToL2TxQueue.front()
      log.debug(`front before deletion: ${front}`)
      let firstBlockHash = await l1ToL2TxQueue.blocks(0)
      log.debug(`firstBlockHash before deletion: ${firstBlockHash}`)

      // delete the single appended block
      await l1ToL2TxQueue
        .connect(canonicalTransactionChain)
        .dequeueBeforeInclusive(blockIndex)

      blocksLength = await l1ToL2TxQueue.getBlocksLength()
      log.debug(`blocksLength after deletion: ${blocksLength}`)
      blocksLength.should.equal(1)
      firstBlockHash = await l1ToL2TxQueue.blocks(0)
      log.debug(`firstBlockHash after deletion: ${firstBlockHash}`)
      firstBlockHash.should.equal(
        '0x0000000000000000000000000000000000000000000000000000000000000000'
      )
      front = await l1ToL2TxQueue.front()
      log.debug(`front after deletion: ${front}`)
      front.should.equal(1)
    })
    it('should not allow dequeue from other address', async () => {
      const block = ['0x1234']
      const cumulativePrevElements = 0
      const blockIndex = 0
      const localBlock = await enqueueAndGenerateBlock(
        block,
        blockIndex,
        cumulativePrevElements
      )
      try {
        // delete the single appended block
        await l1ToL2TxQueue.dequeueBeforeInclusive(blockIndex)
      } catch (err) {
        // Success we threw an error!
        return
      }
      throw new Error(
        'Allowed non-canonicalTransactionChain account to dequeue block'
      )
    })
  })
})
