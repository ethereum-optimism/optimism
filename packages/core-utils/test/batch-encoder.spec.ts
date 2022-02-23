import './setup'

/* Internal Imports */
import { expect } from 'chai'

import {
  encodeAppendSequencerBatch,
  decodeAppendSequencerBatch,
  sequencerBatch,
  BatchType,
  remove0x,
} from '../src'

describe('BatchEncoder', () => {
  describe('appendSequencerBatch', () => {
    it('should work with the simple case', () => {
      const batch = {
        shouldStartAtElement: 0,
        totalElementsToAppend: 0,
        contexts: [],
        transactions: [],
        type: BatchType.LEGACY,
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
        type: BatchType.LEGACY,
      }
      const encoded = encodeAppendSequencerBatch(batch)
      const decoded = decodeAppendSequencerBatch(encoded)
      expect(decoded).to.deep.equal(batch)
    })

    it('should work with mainnet calldata', () => {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      const data = require('./fixtures/appendSequencerBatch.json')
      for (const calldata of data.calldata) {
        const decoded = sequencerBatch.decode(calldata)
        const encoded = sequencerBatch.encode(decoded)
        expect(encoded).to.deep.equal(calldata)

        // Ensure that passing as buffer works
        const decodedBuf = sequencerBatch.decode(
          Buffer.from(remove0x(calldata), 'hex')
        )
        expect(decodedBuf).to.deep.eq(decoded)

        const encodedBuf = sequencerBatch.encode(decodedBuf, { buffer: true })
        expect(Buffer.isBuffer(encodedBuf))

        expect(
          Buffer.compare(
            encodedBuf as Buffer,
            Buffer.from(remove0x(encoded.toString('hex')), 'hex')
          )
        )
        expect('0x' + encodedBuf.toString('hex')).to.eq(encoded)
      }
    })

    it('should work with mainnet calldata (compressed)', () => {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      const data = require('./fixtures/appendSequencerBatch.json')
      for (const calldata of data.calldata) {
        const decoded = sequencerBatch.decode(calldata)
        // Set the batch type to be zlib so that the batch
        // is compressed
        decoded.type = BatchType.ZLIB
        // Encode a compressed batch
        const encodedCompressed = sequencerBatch.encode(decoded)
        expect(decoded.type).to.eq(BatchType.ZLIB)

        // Decode the compressed batch
        const decodedPostCompressed = sequencerBatch.decode(encodedCompressed)
        expect(decoded.type).to.eq(BatchType.ZLIB)
        expect(decoded.contexts).to.deep.equal(decodedPostCompressed.contexts)

        // Set the batch type to legacy so that when it is encoded,
        // it is not compressed
        decodedPostCompressed.type = BatchType.LEGACY

        const encoded = sequencerBatch.encode(decodedPostCompressed)
        expect(encoded).to.deep.equal(calldata)
      }
    })

    it('should throw an error', () => {
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
        transactions: ['0x454234000000112', '0x45423400000012'],
      }
      expect(() => encodeAppendSequencerBatch(batch)).to.throw(
        'Unexpected uneven hex string value!'
      )

      expect(() => sequencerBatch.decode('0x')).to.throw(
        'Incorrect function signature'
      )
    })
  })
})
