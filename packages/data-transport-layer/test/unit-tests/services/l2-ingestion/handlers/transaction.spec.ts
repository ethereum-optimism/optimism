/* Imports: Internal */
import { expect } from '../../../../setup'
import { l2Block } from '../../../examples/l2-data'
import { handleSequencerBlock } from '../../../../../src/services/l2-ingestion/handlers/transaction'

describe('Handlers: handleSequencerBlock', () => {
  describe('parseBlock', () => {
    it('should correctly extract key fields from an L2 mainnet transaction', async () => {
      const input1: [any, number] = [l2Block, 10]

      const output1 = await handleSequencerBlock.parseBlock(...input1)

      expect(output1.stateRootEntry.value).to.equal(l2Block.stateRoot)
      expect(output1.transactionEntry.decoded.data).to.equal(
        l2Block.transactions[0].input
      )
    })
  })
})
