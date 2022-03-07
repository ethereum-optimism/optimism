import { BigNumber, ethers } from 'ethers'
import { sequencerBatch, add0x, BatchType } from '@eth-optimism/core-utils'

const compressBatchWithZlib = (calldata: string): string => {
  const batch = sequencerBatch.decode(calldata)
  batch.type = BatchType.ZLIB
  const encoded = sequencerBatch.encode(batch)
  return add0x(encoded)
}

/* Imports: Internal */
import { expect, readMockData } from '../../../../setup'
import { handleEventsSequencerBatchAppended } from '../../../../../src/services/l1-ingestion/handlers/sequencer-batch-appended'
import { SequencerBatchAppendedExtraData } from '../../../../../src/types'

describe('Event Handlers: CanonicalTransactionChain.SequencerBatchAppended', () => {
  const mockData = readMockData()

  describe('handleEventsSequencerBatchAppended.parseEvent', () => {
    // This tests the behavior of parsing a real mainnet transaction,
    // so it will break if the encoding scheme changes.

    // Transaction and extra data from
    // https://etherscan.io/tx/0x6effe006836b841205ace4d99d7ae1b74ee96aac499a3f358b97fccd32ee9af2
    const exampleExtraData = {
      timestamp: 1614862375,
      blockNumber: 11969713,
      submitter: '0xfd7d4de366850c08ee2cba32d851385a3071ec8d',
      l1TransactionHash:
        '0x6effe006836b841205ace4d99d7ae1b74ee96aac499a3f358b97fccd32ee9af2',
      gasLimit: '548976',
      prevTotalElements: BigNumber.from(73677),
      batchIndex: BigNumber.from(743),
      batchSize: BigNumber.from(101),
      batchRoot:
        '10B99425FB53AD7D40A939205C0F7B35CBB89AB4D67E7AE64BDAC5F1073943B4',
      batchExtraData: '',
    }

    it('should error on malformed transaction data', async () => {
      const input1: [any, SequencerBatchAppendedExtraData, number] = [
        {
          args: {
            _startingQueueIndex: ethers.constants.Zero,
            _numQueueElements: ethers.constants.Zero,
            _totalElements: ethers.constants.Zero,
          },
        },
        {
          l1TransactionData: '0x00000',
          ...exampleExtraData,
        },
        0,
      ]

      expect(() => {
        handleEventsSequencerBatchAppended.parseEvent(...input1)
      }).to.throw(
        `Block ${input1[1].blockNumber} transaction data is too small: ${input1[1].l1TransactionData.length}`
      )
    })

    describe('mainnet transactions', () => {
      for (const mock of mockData) {
        const { input, output } = mock
        const { event, extraData, l2ChainId } = input
        const hash = mock.input.extraData.l1TransactionHash

        it(`uncompressed: ${hash}`, () => {
          // Set the type to be legacy
          output.transactionBatchEntry.type = BatchType[BatchType.LEGACY]

          const res = handleEventsSequencerBatchAppended.parseEvent(
            event,
            extraData,
            l2ChainId
          )
          // Check all of the transaction entries individually
          for (const [i, got] of res.transactionEntries.entries()) {
            const expected = output.transactionEntries[i]
            expect(got).to.deep.eq(expected, `case ${i}`)
          }
          expect(res).to.deep.eq(output)
        })

        it(`compressed: ${hash}`, () => {
          // Set the type to be zlib
          output.transactionBatchEntry.type = BatchType[BatchType.ZLIB]

          const compressed = compressBatchWithZlib(
            input.extraData.l1TransactionData
          )

          const copy = { ...extraData }
          copy.l1TransactionData = compressed

          const res = handleEventsSequencerBatchAppended.parseEvent(
            event,
            copy,
            l2ChainId
          )

          expect(res).to.deep.eq(output)
        })
      }
    })
  })
})
