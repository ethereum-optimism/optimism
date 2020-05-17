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
describe.only('CanonicalTransactionChain', () => {
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
      [
        rollupMerkleUtils.address,
        sequencer.address,
        canonicalTransactionChain.address,
      ],
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
    it.only('should calculate blockHeaderHash correctly', async () => {
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
  })
})
