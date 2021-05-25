import { expect } from '../setup'

/* Imports: External */
import { ethers } from 'ethers'

/* Imports: Internal */
import {
  makeRelayTransactionData,
  getMessageByTransactionHash,
} from '../../src/relay-tx'

describe('relay transaction generation functions', () => {
  describe('getMessageByTransactionHash', () => {
    it('should throw an error if a transaction with the given hash does not exist', async () => {})

    it('should return null if the transaction did not emit a SentMessage event', async () => {})

    it('should throw an error if the transaction emitted more than one SentMessage event', async () => {})

    it('should return the parsed event if the transaction emitted exactly one SentMessage event', async () => {})
  })

  describe('getStateBatchAppendedEventByTransactionIndex', () => {
    it('should return null if a batch for the index does not exist', async () => {})

    it('should return null when there are no batches yet', async () => {})

    it('should return the batch if the index is part of the last batch', async () => {})

    it('should return the batch if the index is part of teh first batch', async () => {})

    for (const numBatches of [1, 2, 8, 64, 128]) {
      describe(`when there are ${numBatches} batch(es)`, () => {
        for (const batchSize of [1, 2, 8, 64, 128]) {
          describe(`when there are ${batchSize} element(s) per batch`, () => {
            for (
              let i = batchSize - 1;
              i < batchSize * numBatches;
              i += batchSize
            ) {
              it(`should be able to get the correct batch for the ${batchSize}th/st/rd/whatever element`, async () => {})
            }
          })
        }
      })
    }
  })

  describe('getStateRootBatchByTransactionIndex', () => {
    it('should return null if a batch for the index does not exist', async () => {})

    it('should return the full batch for a given index when it exists', async () => {})
  })

  describe('makeRelayTransactionData', () => {
    it('should throw an error if the transaction does not exist', async () => {})

    it('should throw an error if the transaction did not send a message', async () => {})

    it('should throw an error if the corresponding state batch has not been submitted', async () => {})

    it('should otherwise return the encoded transaction data', () => {})
  })
})
