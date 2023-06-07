import '../setup'

/* Internal Imports */
import { expect } from 'chai'

import {
  encodeAppendSequencerBatch,
  decodeAppendSequencerBatch,
  sequencerBatch,
  BatchType,
  SequencerBatch,
} from '../../src'

describe('BatchEncoder', function () {
  this.timeout(10_000)

  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const data = require('../fixtures/calldata.json')

  describe('appendSequencerBatch', () => {
    it('legacy: should work with the simple case', () => {
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

    it('legacy: should work with more complex case', () => {
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

    describe('mainnet data', () => {
      for (const [hash, calldata] of Object.entries(data)) {
        // Deserialize the raw calldata
        const decoded = SequencerBatch.fromHex<SequencerBatch>(
          calldata as string
        )

        it(`${hash}`, () => {
          const encoded = decoded.toHex()
          expect(encoded).to.deep.equal(calldata)

          const batch = SequencerBatch.decode(decoded.encode())
          expect(decoded).to.deep.eq(batch)
        })

        it(`${hash} (compressed)`, () => {
          // Set the batch type to be zlib so that the batch
          // is compressed
          decoded.type = BatchType.ZLIB
          // Encode a compressed batch
          const encodedCompressed = decoded.encode()
          // Decode a compressed batch
          const decodedPostCompressed =
            SequencerBatch.decode<SequencerBatch>(encodedCompressed)
          // Expect that the batch type is detected
          expect(decodedPostCompressed.type).to.eq(BatchType.ZLIB)
          // Expect that the contexts match
          expect(decoded.contexts).to.deep.equal(decodedPostCompressed.contexts)
          for (const [i, tx] of decoded.transactions.entries()) {
            const got = decodedPostCompressed.transactions[i]
            expect(got).to.deep.eq(tx)
          }
          // Reserialize the batch as legacy
          decodedPostCompressed.type = BatchType.LEGACY
          // Ensure that the original data can be recovered
          const encoded = decodedPostCompressed.toHex()
          expect(encoded).to.deep.equal(calldata)
        })

        it(`${hash}: serialize txs`, () => {
          for (const tx of decoded.transactions) {
            tx.toTransaction()
          }
        })
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
