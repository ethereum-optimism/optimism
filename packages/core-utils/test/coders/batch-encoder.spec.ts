import '../setup'

/* Internal Imports */
import {
  ctcCoder,
  encodeAppendSequencerBatch,
  decodeAppendSequencerBatch,
  TxType,
  sequencerBatch,
} from '../../src'
import { expect } from 'chai'

describe('BatchEncoder', () => {
  describe('eip155TxData', () => {
    it('should encode & then decode to the correct value', () => {
      const eip155TxData = {
        sig: {
          v: 1,
          r: '0x' + '11'.repeat(32),
          s: '0x' + '22'.repeat(32),
        },
        gasLimit: 500,
        gasPrice: 1000000,
        nonce: 100,
        target: '0x' + '12'.repeat(20),
        data: '0x' + '99'.repeat(10),
        type: TxType.EIP155,
      }
      const encoded = ctcCoder.eip155TxData.encode(eip155TxData)
      const decoded = ctcCoder.eip155TxData.decode(encoded)
      expect(eip155TxData).to.deep.equal(decoded)
    })

    it('should fail encoding a bad gas price', () => {
      const badGasPrice = 1000001
      const eip155TxData = {
        sig: {
          v: 1,
          r: '0x' + '11'.repeat(32),
          s: '0x' + '22'.repeat(32),
        },
        gasLimit: 500,
        gasPrice: badGasPrice,
        nonce: 100,
        target: '0x' + '12'.repeat(20),
        data: '0x' + '99'.repeat(10),
        type: TxType.EIP155,
      }

      let error
      try {
        ctcCoder.eip155TxData.encode(eip155TxData)
      } catch (e) {
        error = e
      }
      expect(error.message).to.equal(
        `Gas Price ${badGasPrice} cannot be encoded`
      )
    })
  })

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

  describe('generic ctcCoder', () => {
    it('should decode EIP155 txs to the correct value', () => {
      const eip155TxData = {
        sig: {
          v: 1,
          r: '0x' + '11'.repeat(32),
          s: '0x' + '22'.repeat(32),
        },
        gasLimit: 500,
        gasPrice: 1000000,
        nonce: 100,
        target: '0x' + '12'.repeat(20),
        data: '0x' + '99'.repeat(10),
        type: TxType.EIP155,
      }
      const encoded = ctcCoder.encode(eip155TxData)
      const decoded = ctcCoder.decode(encoded)
      expect(eip155TxData).to.deep.equal(decoded)
    })

    it('should return null when encoding an unknown type', () => {
      const weirdTypeTxData = {
        sig: {
          v: 1,
          r: '0x' + '11'.repeat(32),
          s: '0x' + '22'.repeat(32),
        },
        gasLimit: 500,
        gasPrice: 100,
        nonce: 100,
        target: '0x' + '12'.repeat(20),
        data: '0x' + '99'.repeat(10),
        type: 420,
      }
      const encoded = ctcCoder.encode(weirdTypeTxData)
      expect(encoded).to.be.null
    })
  })
})
