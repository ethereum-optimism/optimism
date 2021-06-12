import '../setup'

/* Internal Imports */
import {
  encodeAppendSequencerBatch,
  decodeAppendSequencerBatch,
  sequencerBatch,
} from '../../src'
import { expect } from 'chai'

describe('BatchEncoder', () => {
  describe('appendSequencerBatch', () => {
    it('should work with the simple case', () => {
      const batch = {
        shouldStartAtElement: 0,
        totalElementsToAppend: 0,
        contexts: [],
        transactions: [],
      }
      const encoded = encodeAppendSequencerBatch(batch)
      const decoded = decodeAppendSequencerBatch(encoded)
      expect(decoded).to.deep.equal(batch)
    })

    it('should work with more complex case', () => {
      const batch = {
        shouldStartAtElement: 10,
        totalElementsToAppend: 1,
        contexts: [
          {
            numSequencedTransactions: 2,
            numSubsequentQueueTransactions: 1,
            timestamp: 100,
            blockNumber: 200,
          },
        ],
        transactions: ['0x45423400000011', '0x45423400000012'],
      }
      const encoded = encodeAppendSequencerBatch(batch)
      const decoded = decodeAppendSequencerBatch(encoded)
      expect(decoded).to.deep.equal(batch)
    })

    it('should work with mainnet calldata', () => {
      const data = require('../fixtures/appendSequencerBatch.json')
      for (const calldata of data.calldata) {
        const decoded = sequencerBatch.decode(calldata)
        const encoded = sequencerBatch.encode(decoded)
        expect(encoded).to.equal(calldata)
      }
    })
  })
})
