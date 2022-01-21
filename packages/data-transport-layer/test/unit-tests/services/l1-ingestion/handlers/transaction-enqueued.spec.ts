/* Imports: External */
import { ethers, BigNumber } from 'ethers'

/* Imports: Internal */
import { expect } from '../../../../setup'
import { handleEventsTransactionEnqueued } from '../../../../../src/services/l1-ingestion/handlers/transaction-enqueued'

const MAX_ITERATIONS = 128

describe('Event Handlers: CanonicalTransactionChain.TransactionEnqueued', () => {
  describe('getExtraData', () => {
    it('should return null', async () => {
      const output1 = await handleEventsTransactionEnqueued.getExtraData()

      const expected1 = null

      expect(output1).to.equal(expected1)
    })
  })

  describe('parseEvent', () => {
    // TODO: Honestly this is the simplest `parseEvent` function we have and there isn't much logic
    // to test. We could add a lot more tests that guarantee the correctness of the provided input,
    // but it's probably better to get wider test coverage first.

    it('should have a ctcIndex equal to null', () => {
      const input1: [any, any, number] = [
        {
          blockNumber: 0,
          args: {
            _queueIndex: ethers.constants.Zero,
            _gasLimit: ethers.constants.Zero,
            _timestamp: ethers.constants.Zero,
          },
        },
        null,
        0,
      ]

      const output1 = handleEventsTransactionEnqueued.parseEvent(...input1)

      const expected1 = null

      expect(output1).to.have.property('ctcIndex', expected1)
    })

    it('should have a blockNumber equal to the integer value of the blockNumber parameter', () => {
      for (
        let i = 0;
        i < Number.MAX_SAFE_INTEGER;
        i += Math.floor(Number.MAX_SAFE_INTEGER / MAX_ITERATIONS)
      ) {
        const input1: [any, any, number] = [
          {
            blockNumber: i,
            args: {
              _queueIndex: ethers.constants.Zero,
              _gasLimit: ethers.constants.Zero,
              _timestamp: ethers.constants.Zero,
            },
          },
          null,
          0,
        ]

        const output1 = handleEventsTransactionEnqueued.parseEvent(...input1)

        const expected1 = BigNumber.from(i).toNumber()

        expect(output1).to.have.property('blockNumber', expected1)
      }
    })

    it('should have an index equal to the integer value of the _queueIndex argument', () => {
      for (
        let i = 0;
        i < Number.MAX_SAFE_INTEGER;
        i += Math.floor(Number.MAX_SAFE_INTEGER / MAX_ITERATIONS)
      ) {
        const input1: [any, any, number] = [
          {
            blockNumber: 0,
            args: {
              _queueIndex: BigNumber.from(i),
              _gasLimit: ethers.constants.Zero,
              _timestamp: ethers.constants.Zero,
            },
          },
          null,
          0,
        ]

        const output1 = handleEventsTransactionEnqueued.parseEvent(...input1)

        const expected1 = BigNumber.from(i).toNumber()

        expect(output1).to.have.property('index', expected1)
      }
    })

    it('should have a gasLimit equal to the string value of the _gasLimit argument', () => {
      for (
        let i = 0;
        i < Number.MAX_SAFE_INTEGER;
        i += Math.floor(Number.MAX_SAFE_INTEGER / MAX_ITERATIONS)
      ) {
        const input1: [any, any, number] = [
          {
            blockNumber: 0,
            args: {
              _queueIndex: ethers.constants.Zero,
              _gasLimit: BigNumber.from(i),
              _timestamp: ethers.constants.Zero,
            },
          },
          null,
          0,
        ]

        const output1 = handleEventsTransactionEnqueued.parseEvent(...input1)

        const expected1 = BigNumber.from(i).toString()

        expect(output1).to.have.property('gasLimit', expected1)
      }
    })

    it('should have a timestamp equal to the integer value of the _timestamp argument', () => {
      for (
        let i = 0;
        i < Number.MAX_SAFE_INTEGER;
        i += Math.floor(Number.MAX_SAFE_INTEGER / MAX_ITERATIONS)
      ) {
        const input1: [any, any, number] = [
          {
            blockNumber: 0,
            args: {
              _queueIndex: ethers.constants.Zero,
              _gasLimit: ethers.constants.Zero,
              _timestamp: BigNumber.from(i),
            },
          },
          null,
          0,
        ]

        const output1 = handleEventsTransactionEnqueued.parseEvent(...input1)

        const expected1 = BigNumber.from(i).toNumber()

        expect(output1).to.have.property('timestamp', expected1)
      }
    })
  })

  describe.skip('storeEvent', () => {
    // TODO: I don't know the best way to test this, plus it's just a single line. Going to ignore
    // it for now.
  })
})
