import '../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { DefaultRollupBlock } from './RLhelper'

/* Logging */
const log = getLogger('rollup-tx-queue', true)

/* Contract Imports */
import * as RollupTransactionQueue from '../../build/RollupTransactionQueue.json'
import * as RollupMerkleUtils from '../../build/RollupMerkleUtils.json'

/* Begin tests */
// describe('RollupTransactionQueue', () => {
//   const provider = createMockProvider()
//   const [wallet, sequencer, canonicalTransactionChain] = getWallets(provider)
//   let rollupTxQueue
//   let rollupMerkleUtils

//   /* Link libraries before tests */
//   before(async () => {
//     rollupMerkleUtils = await deployContract(wallet, RollupMerkleUtils, [], {
//       gasLimit: 6700000,
//     })
//   })

//   /* Deploy a new RollupChain before each test */
//   beforeEach(async () => {
//     rollupTxQueue = await deployContract(
//       wallet,
//       RollupTransactionQueue,
//       [
//         rollupMerkleUtils.address,
//         sequencer.address,
//         canonicalTransactionChain.address,
//       ],
//       {
//         gasLimit: 6700000,
//       }
//     )
//   })

//   const enqueueAndGenerateBlock = async (
//     block: string[],
//     blockIndex: number,
//     cumulativePrevElements: number
//   ): Promise<DefaultRollupBlock> => {
//     // Submit the rollup block on-chain
//     const enqueueTx = await rollupTxQueue.connect(sequencer).enqueueBlock(block)
//     const txReceipt = await provider.getTransactionReceipt(enqueueTx.hash)
//     // Generate a local version of the rollup block
//     const ethBlockNumber = txReceipt.blockNumber
//     const localBlock = new DefaultRollupBlock(
//       ethBlockNumber,
//       blockIndex,
//       cumulativePrevElements,
//       block
//     )
//     await localBlock.generateTree()
//     return localBlock
//   }

//   /*
//    * Test enqueueBlock()
//    */
//   describe('enqueueBlock() ', async () => {
//     it('should allow enqueue from sequencer', async () => {
//       const block = ['0x1234']
//       await rollupTxQueue.connect(sequencer).enqueueBlock(block) // Did not throw... success!
//     })
//     it('should not allow enqueue from other address', async () => {
//       const block = ['0x1234']
//       await rollupTxQueue
//         .enqueueBlock(block)
//         .should.be.revertedWith(
//           'VM Exception while processing transaction: revert Message sender does not have permission to enqueue'
//         )
//     })
//   })
//   /*
//    * Test dequeueBlock()
//    */
//   describe('dequeueBlock() ', async () => {
//     it('should allow dequeue from canonicalTransactionChain', async () => {
//       const block = ['0x1234']
//       const cumulativePrevElements = 0
//       const blockIndex = 0
//       const localBlock = await enqueueAndGenerateBlock(
//         block,
//         blockIndex,
//         cumulativePrevElements
//       )
//       let blocksLength = await rollupTxQueue.getBlocksLength()
//       log.debug(`blocksLength before deletion: ${blocksLength}`)
//       let front = await rollupTxQueue.front()
//       log.debug(`front before deletion: ${front}`)
//       let firstBlockHash = await rollupTxQueue.blocks(0)
//       log.debug(`firstBlockHash before deletion: ${firstBlockHash}`)

//       // delete the single appended block
//       await rollupTxQueue
//         .connect(canonicalTransactionChain)
//         .dequeueBeforeInclusive(blockIndex)

//       blocksLength = await rollupTxQueue.getBlocksLength()
//       log.debug(`blocksLength after deletion: ${blocksLength}`)
//       blocksLength.should.equal(1)
//       firstBlockHash = await rollupTxQueue.blocks(0)
//       log.debug(`firstBlockHash after deletion: ${firstBlockHash}`)
//       firstBlockHash.should.equal(
//         '0x0000000000000000000000000000000000000000000000000000000000000000'
//       )
//       front = await rollupTxQueue.front()
//       log.debug(`front after deletion: ${front}`)
//       front.should.equal(1)
//     })
//     it('should not allow dequeue from other address', async () => {
//       const block = ['0x1234']
//       const cumulativePrevElements = 0
//       const blockIndex = 0
//       const localBlock = await enqueueAndGenerateBlock(
//         block,
//         blockIndex,
//         cumulativePrevElements
//       )
//       await rollupTxQueue
//         .dequeueBeforeInclusive(blockIndex)
//         .should.be.revertedWith(
//           'VM Exception while processing transaction: revert Message sender does not have permission to dequeue'
//         )
//     })
//   })
// })
