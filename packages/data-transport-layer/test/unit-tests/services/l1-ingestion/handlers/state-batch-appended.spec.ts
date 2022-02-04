/* Imports: External */
import { BigNumber } from 'ethers'

/* Imports: Internal */
import { expect } from '../../../../setup'
import { handleEventsStateBatchAppended } from '../../../../../src/services/l1-ingestion/handlers/state-batch-appended'
import { StateBatchAppendedExtraData } from '../../../../../src/types'
import { l1StateBatchData } from '../../../examples/l1-data'

describe('Event Handlers: CanonicalTransactionChain.StateBatchAppended', () => {
  describe('getExtraData', () => {
    it('should return event block and transaction', async () => {
      // Source: https://etherscan.io/tx/0x4ca72484e93cdb50fe1089984db152258c2bbffc2534dcafbfe032b596bd5b49
      const l1Transaction = {
        hash: '0x4ca72484e93cdb50fe1089984db152258c2bbffc2534dcafbfe032b596bd5b49',
        from: '0xfd7d4de366850c08ee2cba32d851385a3071ec8d',
        data: l1StateBatchData,
      }
      // Source: https://etherscan.io/block/12106615
      const eventBlock = {
        timestamp: 1616680530,
        number: 12106615,
        hash: '0x9c40310e19e943ad38e170329465c4489f6aba5895e9cacdac236be181aea31f',
        parentHash:
          '0xc7707a04c287a22ff4e43e5d9316e45ab342dcd405e7e0284eb51ce71a3a29ac',
        miner: '0xea674fdde714fd979de3edf0f56aa9716b898ec8',
        nonce: '0x40e6174f521a7cd8',
        difficulty: 5990647962682594,
        gasLimit: BigNumber.from(548976),
        gasUsed: BigNumber.from(12495850),
        extraData: '0x65746865726d696e652d6575726f70652d7765737433',
        transactions: [l1Transaction.hash],
      }

      const input1: [any] = [
        {
          getBlock: () => eventBlock,
          getTransaction: () => l1Transaction,
        },
      ]
      const output1 = await handleEventsStateBatchAppended.getExtraData(
        ...input1
      )

      expect(output1.timestamp).to.equal(eventBlock.timestamp)
      expect(output1.blockNumber).to.equal(eventBlock.number)
      expect(output1.submitter).to.equal(l1Transaction.from)
      expect(output1.l1TransactionHash).to.equal(l1Transaction.hash)
      expect(output1.l1TransactionData).to.equal(l1Transaction.data)
    })
  })

  describe('parseEvent', () => {
    it('should have a ctcIndex equal to null', () => {
      // Source: https://etherscan.io/tx/0x4ca72484e93cdb50fe1089984db152258c2bbffc2534dcafbfe032b596bd5b49#eventlog
      const event = {
        args: {
          _batchIndex: BigNumber.from(144),
          _batchRoot:
            'AD2039C6E9A8EE58817252CF16AB720BF3ED20CC4B53184F5B11DE09639AA123',
          _batchSize: BigNumber.from(522),
          _prevTotalElements: BigNumber.from(96000),
          _extraData:
            '00000000000000000000000000000000000000000000000000000000605C33E2000000000000000000000000FD7D4DE366850C08EE2CBA32D851385A3071EC8D',
        },
      }
      const extraData: StateBatchAppendedExtraData = {
        l1TransactionData: l1StateBatchData,
        timestamp: 1616680530,
        blockNumber: 12106615,
        submitter: '0xfd7d4de366850c08ee2cba32d851385a3071ec8d',
        l1TransactionHash:
          '0x4ca72484e93cdb50fe1089984db152258c2bbffc2534dcafbfe032b596bd5b49',
      }
      const input1: [any, StateBatchAppendedExtraData, number] = [
        event,
        extraData,
        0,
      ]

      const output1 = handleEventsStateBatchAppended.parseEvent(...input1)

      expect(output1.stateRootEntries.length).to.eq(
        event.args._batchSize.toNumber()
      )
      output1.stateRootEntries.forEach((entry, i) => {
        expect(entry.index).to.eq(
          event.args._prevTotalElements.add(BigNumber.from(i)).toNumber()
        )
        expect(entry.batchIndex).to.eq(event.args._batchIndex.toNumber())
        expect(entry.confirmed).to.be.true
      })

      const batchEntry = output1.stateRootBatchEntry
      expect(batchEntry.index).to.eq(event.args._batchIndex.toNumber())
      expect(batchEntry.blockNumber).to.eq(extraData.blockNumber)
      expect(batchEntry.timestamp).to.eq(extraData.timestamp)
      expect(batchEntry.submitter).to.eq(extraData.submitter)
      expect(batchEntry.size).to.eq(event.args._batchSize.toNumber())
      expect(batchEntry.root).to.eq(event.args._batchRoot)
      expect(batchEntry.prevTotalElements).to.eq(
        event.args._prevTotalElements.toNumber()
      )
      expect(batchEntry.extraData).to.eq(event.args._extraData)
      expect(batchEntry.l1TransactionHash).to.eq(extraData.l1TransactionHash)
    })
  })
})
